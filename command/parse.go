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
	ctx := context.Background()
	client, err := getGithubClient(ctx, &profile.Token, &profile.APIURL, false)
	if err != nil {
		return err
	}

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return err
	}

	outputs := getBasePrData(ctx, client, user, &profile, c.App.ErrWriter)

	results := make(chan *prInfo)
	go func() {
		wg := sync.WaitGroup{}
		for output := range outputs {
			if !filterOutput(output, c.String("owner"), c.StringSlice("repo")) {
				continue
			}

			wg.Add(1)
			go func(output *prInfo) {
				getExtraData(ctx, client, user, c.Bool("need-rebase"), output)
				results <- output
				wg.Done()
			}(output)
		}

		wg.Wait()
		close(results)
	}()

	return printResults(results, c.Bool("verbose"), c.App.Writer)
}

func getExtraData(ctx context.Context, client *github.Client, user *github.User, filterRebased bool, output *prInfo) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func(output *prInfo) {
		handleCommitComparision(ctx, client, output, filterRebased)
		wg.Done()
	}(output)

	wg.Add(1)
	go func(output *prInfo) {
		handleComments(ctx, client, user, output)
		wg.Done()
	}(output)

	wg.Add(1)
	go func(output *prInfo) {
		handleLabels(ctx, client, output)
		wg.Done()
	}(output)

	wg.Add(1)
	go func(output *prInfo) {
		handleStatuses(ctx, client, output)
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
	ctx := context.Background()
	client, err := getGithubClient(ctx, &profile.Token, &profile.APIURL, true)
	if err != nil {
		return
	}

	suggestionList := []string{}
	suggestionChan := make(chan string)
	go func() {
		wg := sync.WaitGroup{}
		for _, repo := range profile.TrackedRepos {
			wg.Add(1)
			go func(repo config.PrpConfigRepo) {
				repoPrs, errors := getRepoPullRequests(ctx, client, repo.Owner, repo.Name)
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
