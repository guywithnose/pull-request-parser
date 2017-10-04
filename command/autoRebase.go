package command

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/guywithnose/pull-request-parser/config"
	"github.com/guywithnose/runner"
	"github.com/urfave/cli"
)

// CmdAutoRebase parses the pull requests
func CmdAutoRebase(cmdWrapper runner.Builder) func(*cli.Context) error {
	return func(c *cli.Context) error {
		return cmdAutoRebaseHelper(c, cmdWrapper)
	}
}

// CmdAutoRebase parses the pull requests
func cmdAutoRebaseHelper(c *cli.Context, cmdWrapper runner.Builder) error {
	configData, profileName, err := loadProfile(c)
	if err != nil {
		return err
	}

	if c.NArg() != 0 {
		return cli.NewExitError("Usage: \"prp auto-rebase\"", 1)
	}

	profile := configData.Profiles[*profileName]

	pullRequests, err := getValidPullRequests(&profile, c.StringSlice("repo"), c.Bool("use-cache"), c.App.ErrWriter)
	if err != nil {
		return err
	}

	verboseWriter := ioutil.Discard
	if c.Bool("verbose") {
		verboseWriter = c.App.ErrWriter
	}

	return newRebaser(c.App.ErrWriter, verboseWriter, cmdWrapper).rebasePullRequests(pullRequests, c.Int("pull-request-number"))
}

func getValidPullRequests(profile *config.Profile, repos []string, useCache bool, errWriter io.Writer) (<-chan *pullRequest, error) {
	client, err := getGithubClient(&profile.Token, &profile.APIURL, useCache)
	if err != nil {
		return nil, err
	}

	user, _, err := client.Users.Get(context.Background(), "")
	if err != nil {
		return nil, err
	}

	prs := newParser(client, user, profile).getBasePullRequestData(errWriter)
	prs = filterPullRequestsByRepo(prs, *user.Login, repos)

	filteredPullRequests := make(chan *pullRequest, 5)
	go func() {
		wg := sync.WaitGroup{}
		for pr := range prs {
			wg.Add(1)
			go func(pr *pullRequest) {
				pr.getCommitComparison()
				if !pr.Rebased {
					filteredPullRequests <- pr
				}
				wg.Done()
			}(pr)
		}

		wg.Wait()
		close(filteredPullRequests)
	}()

	return filteredPullRequests, nil
}

// CompleteAutoRebase handles bash autocompletion for the 'auto-rebase' command
func CompleteAutoRebase(c *cli.Context) {
	lastParam := os.Args[len(os.Args)-2]
	if lastParam != "--repo" && lastParam != "--pull-request-number" {
		completeAutoRebaseFlags(c)
		return
	}

	handleCompletion(c)
}

func completeAutoRebaseFlags(c *cli.Context) {
	for _, flag := range c.App.Command("auto-rebase").Flags {
		name := strings.Split(flag.GetName(), ",")[0]
		if !c.IsSet(name) || name == "repo" {
			fmt.Fprintf(c.App.Writer, "--%s\n", name)
		}
	}
}

func handleCompletion(c *cli.Context) {
	configData, profileName, err := loadProfile(c)
	if err != nil {
		return
	}

	profile := configData.Profiles[*profileName]
	prs, err := getValidPullRequests(&profile, []string{}, true, c.App.ErrWriter)
	if err != nil {
		return
	}

	selectedRepos := c.StringSlice("repo")
	completions := completeRepoValues(profile, prs, selectedRepos)

	completions = unique(completions)
	sort.Strings(completions)
	fmt.Fprintln(c.App.Writer, strings.Join(completions, "\n"))
}

func completeRepoValues(profile config.Profile, prs <-chan *pullRequest, selectedRepos []string) []string {
	lastParam := os.Args[len(os.Args)-2]
	completions := make([]string, 0, len(profile.TrackedRepos))
	for pr := range prs {
		fullRepoName := fmt.Sprintf("%s/%s", pr.Repo.Owner, pr.Repo.Name)
		if lastParam == "--pull-request-number" {
			if len(selectedRepos) == 0 || stringSliceContains(fullRepoName, selectedRepos) {
				completions = append(completions, strconv.Itoa(pr.PullRequestID))
			}
		} else if !stringSliceContains(fullRepoName, selectedRepos) {
			completions = append(completions, fullRepoName)
		}
	}

	return completions
}
