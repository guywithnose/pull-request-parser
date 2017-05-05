package command

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/google/go-github/github"
	"github.com/guywithnose/pull-request-parser/config"
	"github.com/urfave/cli"
)

// CmdParse parses the pull requests
func CmdParse(c *cli.Context) error {
	configData, profileName, err := loadProfile(c)
	if err != nil {
		return err
	}

	if c.NArg() != 0 {
		return cli.NewExitError("Usage: \"prp parse\"", 1)
	}

	profile := configData.Profiles[*profileName]
	client, err := getGithubClient(&profile.Token, &profile.APIURL, c.Bool("use-cache"))
	if err != nil {
		return err
	}

	user, _, err := client.Users.Get(context.Background(), "")
	if err != nil {
		return err
	}

	parser := newParser(client, user, &profile)
	prs := parser.getBasePullRequestData(c.App.ErrWriter)

	results := parser.parsePullRequests(prs, c.String("owner"), c.StringSlice("repo"), c.Bool("need-rebase"))

	return printResults(results, c.Bool("verbose"), c.App.Writer)
}

// CompleteParse handles bash autocompletion for the 'parse' command
func CompleteParse(c *cli.Context) {
	lastParam := os.Args[len(os.Args)-2]
	if lastParam != "--user" && lastParam != "--repo" {
		completeFlags(c)
		return
	}

	configData, profileName, err := loadProfile(c)
	if err != nil {
		return
	}

	profile := configData.Profiles[*profileName]
	if lastParam == "--user" {
		completeUser(&profile, c.App.Writer, c.App.ErrWriter)
		return
	}

	completeRepo(c.StringSlice("repo"), profile, c.App.Writer)
}

func completeRepo(selectedRepos []string, profile config.Profile, writer io.Writer) {
	for _, repo := range profile.TrackedRepos {
		fullRepoName := fmt.Sprintf("%s/%s", repo.Owner, repo.Name)
		if stringSliceContains(fullRepoName, selectedRepos) {
			continue
		}

		fmt.Fprintln(writer, fullRepoName)
	}
}

func completeFlags(c *cli.Context) {
	for _, flag := range c.App.Command("parse").Flags {
		name := strings.Split(flag.GetName(), ",")[0]
		if !c.IsSet(name) || name == "repo" {
			fmt.Fprintf(c.App.Writer, "--%s\n", name)
		}
	}
}

func completeUser(profile *config.Profile, writer, errWriter io.Writer) {
	client, err := getGithubClient(&profile.Token, &profile.APIURL, true)
	if err != nil {
		return
	}

	suggestionList := []string{}
	suggestionChan := getUsersForAllRepos(client, profile, errWriter)

	for newSuggestion := range suggestionChan {
		suggestionList = append(suggestionList, newSuggestion)
	}

	suggestionList = unique(suggestionList)
	sort.Strings(suggestionList)
	fmt.Fprintln(writer, strings.Join(suggestionList, "\n"))
}

func getUsersForAllRepos(client *github.Client, profile *config.Profile, errWriter io.Writer) <-chan string {
	suggestionChan := make(chan string, 5)
	go func() {
		wg := sync.WaitGroup{}
		for _, repo := range profile.TrackedRepos {
			wg.Add(1)
			go func(repo config.Repo) {
				repoPrs := getRepoPullRequestsAndReportErrors(client, repo.Owner, repo.Name, errWriter)

				for pr := range repoPrs {
					suggestionChan <- pr.Head.User.GetLogin()
				}

				wg.Done()
			}(repo)
		}
		wg.Wait()
		close(suggestionChan)
	}()

	return suggestionChan
}
