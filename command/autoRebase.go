package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/guywithnose/commandBuilder"
	"github.com/guywithnose/pull-request-parser/config"
	"github.com/urfave/cli"
)

// CmdAutoRebase parses the pull requests
func CmdAutoRebase(cmdWrapper commandBuilder.Builder) func(*cli.Context) error {
	return func(c *cli.Context) error {
		return cmdAutoRebaseHelper(c, cmdWrapper)
	}
}

// CmdAutoRebase parses the pull requests
func cmdAutoRebaseHelper(c *cli.Context, cmdWrapper commandBuilder.Builder) error {
	configData, profileName, err := loadProfile(c)
	if err != nil {
		return err
	}

	if c.NArg() != 0 {
		return cli.NewExitError("Usage: \"prp auto-rebase\"", 1)
	}

	profile := configData.Profiles[*profileName]

	outputs, err := getValidPullRequests(&profile, c.StringSlice("repo"), c.Bool("use-cache"), c.App.ErrWriter)
	if err != nil {
		return err
	}

	verboseWriter := ioutil.Discard
	if c.Bool("verbose") {
		verboseWriter = c.App.ErrWriter
	}

	success := true
	for output := range outputs {
		if c.Int("pull-request-number") != 0 && output.PullRequestID != c.Int("pull-request-number") {
			continue
		}

		err = rebasePullRequest(output, c.App.ErrWriter, verboseWriter, cmdWrapper)
		if err != nil {
			fmt.Fprintf(c.App.ErrWriter, "Could not rebase PR #%d in %s/%s because: %v\n", output.PullRequestID, output.Repo.Owner, output.Repo.Name, err)
			success = false
		}
	}

	if !success {
		return cli.NewExitError("Unable to rebase all pull requests", 1)
	}

	return nil
}

func getValidPullRequests(profile *config.PrpConfigProfile, repos []string, useCache bool, errWriter io.Writer) (<-chan *prInfo, error) {
	client, err := getGithubClient(&profile.Token, &profile.APIURL, useCache)
	if err != nil {
		return nil, err
	}

	user, _, err := client.Users.Get(context.Background(), "")
	if err != nil {
		return nil, err
	}

	outputs := getBasePrData(client, user, profile, errWriter)

	filteredOutputs := make(chan *prInfo, 5)
	go func() {
		wg := sync.WaitGroup{}
		for output := range outputs {
			if !filterOutput(output, *user.Login, repos) {
				continue
			}

			wg.Add(1)
			go func(output *prInfo) {
				handleCommitComparision(client, output, false)
				if !output.Rebased {
					filteredOutputs <- output
				}
				wg.Done()
			}(output)
		}

		wg.Wait()
		close(filteredOutputs)
	}()

	return filteredOutputs, nil
}

// CompleteAutoRebase handles bash autocompletion for the 'auto-rebase' command
func CompleteAutoRebase(c *cli.Context) {
	lastParam := os.Args[len(os.Args)-2]
	if lastParam != "--repo" && lastParam != "--pull-request-number" {
		for _, flag := range c.App.Command("auto-rebase").Flags {
			name := strings.Split(flag.GetName(), ",")[0]
			if !c.IsSet(name) || name == "repo" {
				fmt.Fprintf(c.App.Writer, "--%s\n", name)
			}
		}
		return
	}

	handleCompletion(c)
}

func handleCompletion(c *cli.Context) {
	lastParam := os.Args[len(os.Args)-2]
	configData, profileName, err := loadProfile(c)
	if err != nil {
		return
	}

	profile := configData.Profiles[*profileName]
	outputs, err := getValidPullRequests(&profile, c.StringSlice("repo"), true, c.App.ErrWriter)
	if err != nil {
		return
	}

	selectedRepos := c.StringSlice("repo")
	completions := make([]string, 0, len(profile.TrackedRepos))
	for output := range outputs {
		fullRepoName := fmt.Sprintf("%s/%s", output.Repo.Owner, output.Repo.Name)
		if lastParam == "--pull-request-number" {
			completions = append(completions, strconv.Itoa(output.PullRequestID))
		} else if !stringSliceContains(fullRepoName, selectedRepos) {
			completions = append(completions, fullRepoName)
		}
	}

	completions = unique(completions)
	sort.Strings(completions)
	fmt.Fprintln(c.App.Writer, strings.Join(completions, "\n"))
}

func rebasePullRequest(output *prInfo, errorWriter, verboseWriter io.Writer, cmdWrapper commandBuilder.Builder) error {
	path, ownedRemote, upstreamRemote, localChanges, err := getRepoData(output, verboseWriter, cmdWrapper)
	if err != nil {
		return err
	}

	if localChanges {
		fmt.Fprintln(verboseWriter, "Local changes found... stashing")
		err = runCommand(path, cmdWrapper, "git", "stash")
		if err != nil {
			return wrapExitError(err, fmt.Sprintf("Unable to stash changes in %s", path))
		}

		defer popStash(path, errorWriter, verboseWriter, cmdWrapper)
	}

	currentBranchName, tempBranch, err := checkoutTempBranch(path, output.Branch, verboseWriter, cmdWrapper)
	if err != nil {
		return err
	}

	defer cleanUp(currentBranchName, tempBranch, path, errorWriter, verboseWriter, cmdWrapper)

	return handleRebase(path, ownedRemote, upstreamRemote, tempBranch, output, verboseWriter, cmdWrapper)
}

func handleRebase(path, ownedRemote, upstreamRemote, tempBranch string, output *prInfo, verboseWriter io.Writer, cmdWrapper commandBuilder.Builder) error {
	myRemoteBranch := fmt.Sprintf("%s/%s", ownedRemote, output.Branch)
	fmt.Fprintf(verboseWriter, "Resetting code to %s\n", myRemoteBranch)
	err := runCommand(path, cmdWrapper, "git", "reset", "--hard", myRemoteBranch)
	if err != nil {
		return wrapExitError(err, fmt.Sprintf("Unable to reset the code to %s", myRemoteBranch))
	}

	upstreamBranch := fmt.Sprintf("%s/%s", upstreamRemote, output.TargetBranch)
	fmt.Fprintf(verboseWriter, "Rebasing against %s\n", upstreamBranch)
	err = runCommand(path, cmdWrapper, "git", "rebase", upstreamBranch)
	if err != nil {
		return wrapExitError(err, fmt.Sprintf("Unable to rebase against %s, there may be a conflict", upstreamBranch))
	}

	fmt.Fprintf(verboseWriter, "Pushing to %s\n", myRemoteBranch)
	err = runCommand(path, cmdWrapper, "git", "push", ownedRemote, fmt.Sprintf("%s:%s", tempBranch, output.Branch), "--force")
	if err != nil {
		return wrapExitError(err, fmt.Sprintf("Unable to push to %s", myRemoteBranch))
	}

	return nil
}

func fetchRemote(path, remoteName string, verboseWriter io.Writer, cmdWrapper commandBuilder.Builder) error {
	fmt.Fprintf(verboseWriter, "Fetching from remote: %s\n", remoteName)
	err := runCommand(path, cmdWrapper, "git", "fetch", remoteName)
	if err != nil {
		return wrapExitError(err, fmt.Sprintf("Unable to fetch code from %s", remoteName))
	}

	return nil
}

func wrapExitError(err error, extra string) error {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return fmt.Errorf("%s\n%s", extra, string(exitErr.Stderr))
	}

	return errors.New(extra)
}

func checkoutTempBranch(path, branch string, verboseWriter io.Writer, cmdWrapper commandBuilder.Builder) (string, string, error) {
	fmt.Fprintln(verboseWriter, "Saving current branch name")
	currentBranchName, err := getCurrentBranch(path, cmdWrapper)
	if err != nil {
		return "", "", fmt.Errorf("Unable to get current branch name in %s\n%v", path, err)
	}

	fmt.Fprintf(verboseWriter, "Current branch name is %s\n", currentBranchName)

	tempBranch := fmt.Sprintf("prp-%s", branch)
	fmt.Fprintf(verboseWriter, "Checking out temporary branch: %s\n", tempBranch)
	err = runCommand(path, cmdWrapper, "git", "checkout", "-b", tempBranch)
	if err != nil {
		code := getErrorCode(err)
		if code == 128 {
			return "", "", wrapExitError(err, fmt.Sprintf("Branch %s already exists", tempBranch))
		}

		return "", "", wrapExitError(err, fmt.Sprintf("Unable to checkout temporary branch %s in %s", tempBranch, path))
	}

	return currentBranchName, tempBranch, nil
}

func getRepoData(output *prInfo, verboseWriter io.Writer, cmdWrapper commandBuilder.Builder) (string, string, string, bool, error) {
	fmt.Fprintln(verboseWriter, "Requesting repo data from config")
	if output.Repo.LocalPath == "" {
		return "", "", "", false, errors.New("Path was not set for this repo")
	}

	if _, err := os.Stat(output.Repo.LocalPath); os.IsNotExist(err) {
		return "", "", "", false, fmt.Errorf("Path does not exist: %s", output.Repo.LocalPath)
	}

	if _, err := os.Stat(fmt.Sprintf("%s/.git", output.Repo.LocalPath)); os.IsNotExist(err) {
		return "", "", "", false, fmt.Errorf("Path is not a git repo: %s", output.Repo.LocalPath)
	}

	fmt.Fprintln(verboseWriter, "Analyzing remotes")
	ownedRemote, upstreamRemote, err := getRemotes(output.Repo.LocalPath, cmdWrapper, output)
	if err != nil {
		return "", "", "", false, err
	}

	fmt.Fprintln(verboseWriter, "Checking for local changes")
	localChanges, err := detectLocalChanges(output.Repo.LocalPath, cmdWrapper)
	if err != nil {
		return "", "", "", false, wrapExitError(err, fmt.Sprintf("Unable to detect local changes in %s", output.Repo.LocalPath))
	}

	err = fetchRemote(output.Repo.LocalPath, ownedRemote, verboseWriter, cmdWrapper)
	if err != nil {
		return "", "", "", false, err
	}

	err = fetchRemote(output.Repo.LocalPath, upstreamRemote, verboseWriter, cmdWrapper)
	if err != nil {
		return "", "", "", false, err
	}

	return output.Repo.LocalPath, ownedRemote, upstreamRemote, localChanges, nil
}

func cleanUp(currentBranchName, tempBranch, path string, errorWriter, verboseWriter io.Writer, cmdWrapper commandBuilder.Builder) {
	fmt.Fprintf(verboseWriter, "Going back to branch %s\n", currentBranchName)
	err := runCommand(path, cmdWrapper, "git", "checkout", currentBranchName)
	if err != nil {
		fmt.Fprintf(errorWriter, "%v\nWarning: Could not go back to branch %s in %s\n", wrapExitError(err, ""), currentBranchName, path)
		return
	}

	fmt.Fprintf(verboseWriter, "Deleting temporary branch %s\n", tempBranch)
	err = runCommand(path, cmdWrapper, "git", "branch", "-D", tempBranch)
	if err != nil {
		fmt.Fprintf(errorWriter, "%v\nWarning: Could not delete temporary branch %s in %s\n", wrapExitError(err, ""), tempBranch, path)
	}
}

func popStash(path string, errorWriter, verboseWriter io.Writer, cmdWrapper commandBuilder.Builder) {
	fmt.Fprintln(verboseWriter, "Popping the stash")
	err := runCommand(path, cmdWrapper, "git", "stash", "pop")
	if err != nil {
		fmt.Fprintf(errorWriter, "%v\nWarning: Could not pop stash in %s\n", wrapExitError(err, ""), path)
	}
}

func detectLocalChanges(path string, cmdWrapper commandBuilder.Builder) (bool, error) {
	localChangesCommand := cmdWrapper.CreateCommand(path, "git", "diff-index", "--quiet", "HEAD")
	_, err := localChangesCommand.Output()
	if err != nil {
		code := getErrorCode(err)
		if code == 1 {
			return true, nil
		}

		return true, err
	}

	return false, nil
}

func runCommand(path string, cmdWrapper commandBuilder.Builder, command ...string) error {
	cmd := cmdWrapper.CreateCommand(path, command...)
	_, err := cmd.Output()
	return err
}

func getCurrentBranch(path string, cmdWrapper commandBuilder.Builder) (string, error) {
	getCurrentBranch := cmdWrapper.CreateCommand(path, "git", "symbolic-ref", "HEAD")
	currentBranchOutput, err := getCurrentBranch.CombinedOutput()
	if err != nil {
		code := getErrorCode(err)
		if code == 128 {
			return "", fmt.Errorf("No branch checked out in %s\n%s", path, string(currentBranchOutput))
		}

		return "", fmt.Errorf("%s\n%s", err, string(currentBranchOutput))
	}

	return strings.Replace(strings.Replace(string(currentBranchOutput), "refs/heads/", "", -1), "\n", "", -1), nil
}

func getErrorCode(err error) int {
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}

	return -1
}

func getRemotes(path string, cmdWrapper commandBuilder.Builder, output *prInfo) (string, string, error) {
	getRemotes := cmdWrapper.CreateCommand(path, "git", "remote", "-v")
	remotesOutput, err := getRemotes.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("Unable to analyze remotes in %s\n%s", path, string(remotesOutput))
	}

	remotes := make(map[string]string)
	lines := strings.Split(string(remotesOutput), "\n")
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) == 2 {
			remotes[parts[1]] = parts[0]
		}
	}

	ownedRemote, ok := remotes[fmt.Sprintf("%s (push)", output.HeadSSHURL)]
	if !ok {
		return "", "", fmt.Errorf("No remote exists in %s that points to %s", path, output.HeadSSHURL)
	}

	upstreamRemote, ok := remotes[fmt.Sprintf("%s (fetch)", output.BaseSSHURL)]
	if !ok {
		return "", "", fmt.Errorf("No remote exists in %s that points to %s", path, output.BaseSSHURL)
	}

	return ownedRemote, upstreamRemote, nil
}
