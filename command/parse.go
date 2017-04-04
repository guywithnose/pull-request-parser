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

	outputs := getBasePrData(client, user, &profile, c.App.ErrWriter)

	results := make(chan *prInfo, 10)
	go func() {
		wg := sync.WaitGroup{}
		for output := range outputs {
			if !filterOutput(output, c.String("owner"), c.StringSlice("repo")) {
				continue
			}

			wg.Add(1)
			go func(output *prInfo) {
				getExtraData(client, user, c.Bool("need-rebase"), output)
				results <- output
				wg.Done()
			}(output)
		}

		wg.Wait()
		close(results)
	}()

	return printResults(results, c.Bool("verbose"), c.App.Writer)
}

func getExtraData(client *github.Client, user *github.User, filterRebased bool, output *prInfo) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func(output *prInfo) {
		handleCommitComparision(client, output, filterRebased)
		wg.Done()
	}(output)

	wg.Add(1)
	go func(output *prInfo) {
		handleComments(client, user, output)
		wg.Done()
	}(output)

	wg.Add(1)
	go func(output *prInfo) {
		handleLabels(client, output)
		wg.Done()
	}(output)

	wg.Add(1)
	go func(output *prInfo) {
		handleStatuses(client, output)
		wg.Done()
	}(output)

	wg.Wait()
}

// CompleteParse handles bash autocompletion for the 'parse' command
func CompleteParse(c *cli.Context) {
	lastParam := os.Args[len(os.Args)-2]
	if lastParam != "--user" && lastParam != "--repo" {
		for _, flag := range c.App.Command("parse").Flags {
			name := strings.Split(flag.GetName(), ",")[0]
			if !c.IsSet(name) || name == "repo" {
				fmt.Fprintf(c.App.Writer, "--%s\n", name)
			}
		}
		return
	}

	configData, profileName, err := loadProfile(c)
	if err != nil {
		return
	}

	profile := configData.Profiles[*profileName]
	if lastParam == "--user" {
		handleUserCompletion(&profile, c.App.Writer, c.App.ErrWriter)
		return
	}

	selectedRepos := c.StringSlice("repo")
	for _, repo := range profile.TrackedRepos {
		fullRepoName := fmt.Sprintf("%s/%s", repo.Owner, repo.Name)
		if stringSliceContains(fullRepoName, selectedRepos) {
			continue
		}

		fmt.Fprintln(c.App.Writer, fullRepoName)
	}
}

func handleUserCompletion(profile *config.PrpConfigProfile, writer, errWriter io.Writer) {
	client, err := getGithubClient(&profile.Token, &profile.APIURL, true)
	if err != nil {
		return
	}

	suggestionList := []string{}
	suggestionChan := make(chan string, 5)
	go func() {
		wg := sync.WaitGroup{}
		for _, repo := range profile.TrackedRepos {
			wg.Add(1)
			go func(repo config.PrpConfigRepo) {
				repoPrs, errors := getRepoPullRequests(client, repo.Owner, repo.Name)
				go func() {
					for {
						err := <-errors
						if err == nil {
							return
						}

						fmt.Fprintln(errWriter, err)
					}
				}()

				for pr := range repoPrs {
					suggestionChan <- pr.Head.User.GetLogin()
				}

				wg.Done()
			}(repo)
		}
		wg.Wait()
		close(suggestionChan)
	}()

	for newSuggestion := range suggestionChan {
		suggestionList = append(suggestionList, newSuggestion)
	}

	suggestionList = unique(suggestionList)
	sort.Strings(suggestionList)
	fmt.Fprintln(writer, strings.Join(suggestionList, "\n"))
}
