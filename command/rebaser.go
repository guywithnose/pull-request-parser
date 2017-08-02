package command

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"

	"github.com/guywithnose/runner"
	"github.com/urfave/cli"
)

type rebaser struct {
	errorWriter   io.Writer
	verboseWriter io.Writer
	cmdWrapper    runner.Builder
}

func newRebaser(errorWriter, verboseWriter io.Writer, cmdWrapper runner.Builder) *rebaser {
	return &rebaser{
		errorWriter:   errorWriter,
		verboseWriter: verboseWriter,
		cmdWrapper:    cmdWrapper,
	}
}

func (r rebaser) rebasePullRequests(pullRequests <-chan *pullRequest, pullRequestNumber int) error {
	var completeError error
	for pullRequest := range pullRequests {
		if pullRequestNumber != 0 && pullRequest.PullRequestID != pullRequestNumber {
			continue
		}

		err := r.rebasePullRequest(pullRequest)
		if err != nil {
			fmt.Fprintf(r.errorWriter, "Could not rebase PR #%d in %s/%s because: %v\n", pullRequest.PullRequestID, pullRequest.Repo.Owner, pullRequest.Repo.Name, err)
			completeError = cli.NewExitError("Unable to rebase all pull requests", 1)
		}
	}

	return completeError
}

func (r rebaser) rebasePullRequest(pr *pullRequest) error {
	path, ownedRemote, upstreamRemote, localChanges, err := r.getRepoData(pr)
	if err != nil {
		return err
	}

	if localChanges {
		fmt.Fprintln(r.verboseWriter, "Local changes found... stashing")
		err = r.runCommand(path, "git", "stash")
		if err != nil {
			return wrapExitError(err, fmt.Sprintf("Unable to stash changes in %s", path))
		}

		defer r.popStash(path)
	}

	currentBranchName, tempBranch, err := r.checkoutTempBranch(path, pr.Branch)
	if err != nil {
		return err
	}

	defer r.cleanUp(currentBranchName, tempBranch, path)

	return r.doRebase(path, ownedRemote, upstreamRemote, tempBranch, pr)
}

func (r rebaser) doRebase(path, ownedRemote, upstreamRemote, tempBranch string, pr *pullRequest) error {
	myRemoteBranch := fmt.Sprintf("%s/%s", ownedRemote, pr.Branch)
	fmt.Fprintf(r.verboseWriter, "Resetting code to %s\n", myRemoteBranch)
	err := r.runCommand(path, "git", "reset", "--hard", myRemoteBranch)
	if err != nil {
		return wrapExitError(err, fmt.Sprintf("Unable to reset the code to %s", myRemoteBranch))
	}

	upstreamBranch := fmt.Sprintf("%s/%s", upstreamRemote, pr.TargetBranch)
	fmt.Fprintf(r.verboseWriter, "Rebasing against %s\n", upstreamBranch)
	err = r.runCommand(path, "git", "rebase", upstreamBranch)
	if err != nil {
		abortErr := r.runCommand(path, "git", "rebase", "--abort")
		if abortErr != nil {
			fmt.Fprintf(r.errorWriter, "Could not abort rebase PR #%d in %s/%s because: %v\n", pr.PullRequestID, pr.Repo.Owner, pr.Repo.Name, abortErr)
		}

		return wrapExitError(err, fmt.Sprintf("Unable to rebase against %s, there may be a conflict", upstreamBranch))
	}

	fmt.Fprintf(r.verboseWriter, "Pushing to %s\n", myRemoteBranch)
	err = r.runCommand(path, "git", "push", ownedRemote, fmt.Sprintf("%s:%s", tempBranch, pr.Branch), "--force")
	if err != nil {
		return wrapExitError(err, fmt.Sprintf("Unable to push to %s", myRemoteBranch))
	}

	return nil
}

func (r rebaser) detectLocalChanges(path string) (bool, error) {
	localChangesCommand := r.cmdWrapper.New(path, "git", "diff-index", "--quiet", "HEAD")
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

func (r rebaser) getRepoData(pr *pullRequest) (string, string, string, bool, error) {
	fmt.Fprintln(r.verboseWriter, "Requesting repo data from config")
	err := pr.checkLocalPath()
	if err != nil {
		return "", "", "", false, err
	}

	fmt.Fprintln(r.verboseWriter, "Analyzing remotes")
	ownedRemote, upstreamRemote, err := r.getRemotes(pr.Repo.LocalPath, pr)
	if err != nil {
		return "", "", "", false, err
	}

	fmt.Fprintln(r.verboseWriter, "Checking for local changes")
	localChanges, err := r.detectLocalChanges(pr.Repo.LocalPath)
	if err != nil {
		return "", "", "", false, wrapExitError(err, fmt.Sprintf("Unable to detect local changes in %s", pr.Repo.LocalPath))
	}

	err = r.fetchRemotes(pr.Repo.LocalPath, ownedRemote, upstreamRemote)
	if err != nil {
		return "", "", "", false, err
	}

	return pr.Repo.LocalPath, ownedRemote, upstreamRemote, localChanges, nil
}

func (r rebaser) cleanUp(currentBranchName, tempBranch, path string) {
	fmt.Fprintf(r.verboseWriter, "Going back to branch %s\n", currentBranchName)
	err := r.runCommand(path, "git", "checkout", currentBranchName)
	if err != nil {
		fmt.Fprintf(r.errorWriter, "%v\nWarning: Could not go back to branch %s in %s\n", wrapExitError(err, ""), currentBranchName, path)
		return
	}

	fmt.Fprintf(r.verboseWriter, "Deleting temporary branch %s\n", tempBranch)
	err = r.runCommand(path, "git", "branch", "-D", tempBranch)
	if err != nil {
		fmt.Fprintf(r.errorWriter, "%v\nWarning: Could not delete temporary branch %s in %s\n", wrapExitError(err, ""), tempBranch, path)
	}
}

func (r rebaser) popStash(path string) {
	fmt.Fprintln(r.verboseWriter, "Popping the stash")
	err := r.runCommand(path, "git", "stash", "pop")
	if err != nil {
		fmt.Fprintf(r.errorWriter, "%v\nWarning: Could not pop stash in %s\n", wrapExitError(err, ""), path)
	}
}

func (r rebaser) runCommand(path string, command ...string) error {
	cmd := r.cmdWrapper.New(path, command...)
	_, err := cmd.Output()
	return err
}

func wrapExitError(err error, extra string) error {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return fmt.Errorf("%s\n%s", extra, string(exitErr.Stderr))
	}

	return errors.New(extra)
}

func (r rebaser) checkoutTempBranch(path, branch string) (string, string, error) {
	fmt.Fprintln(r.verboseWriter, "Saving current branch name")
	currentBranchName, err := r.getCurrentBranch(path)
	if err != nil {
		return "", "", fmt.Errorf("Unable to get current branch name in %s\n%v", path, err)
	}

	fmt.Fprintf(r.verboseWriter, "Current branch name is %s\n", currentBranchName)

	tempBranch := fmt.Sprintf("prp-%s", branch)
	fmt.Fprintf(r.verboseWriter, "Checking out temporary branch: %s\n", tempBranch)
	err = r.runCommand(path, "git", "checkout", "-b", tempBranch)
	if err != nil {
		code := getErrorCode(err)
		if code == 128 {
			return "", "", wrapExitError(err, fmt.Sprintf("Branch %s already exists", tempBranch))
		}

		return "", "", wrapExitError(err, fmt.Sprintf("Unable to checkout temporary branch %s in %s", tempBranch, path))
	}

	return currentBranchName, tempBranch, nil
}

func (r rebaser) getCurrentBranch(path string) (string, error) {
	getCurrentBranch := r.cmdWrapper.New(path, "git", "symbolic-ref", "HEAD")
	currentBranchOutput, err := getCurrentBranch.CombinedOutput()
	if err != nil {
		code := getErrorCode(err)
		if code == 128 {
			getCurrentBranch = r.cmdWrapper.New(path, "git", "rev-parse", "HEAD")
			currentBranchOutput, err = getCurrentBranch.CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("No branch checked out in %s\n%s", path, string(currentBranchOutput))
			}

			return strings.Replace(strings.Replace(string(currentBranchOutput), "refs/heads/", "", -1), "\n", "", -1), nil
		}

		return "", fmt.Errorf("%s\n%s", err, string(currentBranchOutput))
	}

	return strings.Replace(strings.Replace(string(currentBranchOutput), "refs/heads/", "", -1), "\n", "", -1), nil
}

func (r rebaser) fetchRemotes(path, ownedRemote, upstreamRemote string) error {
	err := r.fetchRemote(path, ownedRemote)
	if err != nil {
		return err
	}

	return r.fetchRemote(path, upstreamRemote)
}

func (r rebaser) fetchRemote(path, remoteName string) error {
	fmt.Fprintf(r.verboseWriter, "Fetching from remote: %s\n", remoteName)
	err := r.runCommand(path, "git", "fetch", remoteName)
	if err != nil {
		return wrapExitError(err, fmt.Sprintf("Unable to fetch code from %s", remoteName))
	}

	return nil
}

func (r rebaser) getRemotes(path string, pr *pullRequest) (string, string, error) {
	getRemotes := r.cmdWrapper.New(path, "git", "remote", "-v")
	remotesOutput, err := getRemotes.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("Unable to analyze remotes in %s\n%s", path, string(remotesOutput))
	}

	remotes := parseRemotes(strings.Split(string(remotesOutput), "\n"))

	ownedRemote, ok := remotes[fmt.Sprintf("%s (push)", pr.HeadSSHURL)]
	if !ok {
		return "", "", fmt.Errorf("No remote exists in %s that points to %s", path, pr.HeadSSHURL)
	}

	upstreamRemote, ok := remotes[fmt.Sprintf("%s (fetch)", pr.BaseSSHURL)]
	if !ok {
		return "", "", fmt.Errorf("No remote exists in %s that points to %s", path, pr.BaseSSHURL)
	}

	return ownedRemote, upstreamRemote, nil
}

func parseRemotes(lines []string) map[string]string {
	remotes := make(map[string]string)
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) == 2 {
			remotes[parts[1]] = parts[0]
		}
	}

	return remotes
}

func getErrorCode(err error) int {
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}

	return -1
}
