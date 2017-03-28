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
	client, err := getGithubClient(ctx, &profile.Token, &profile.APIURL)
	if err != nil {
		return err
	}

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return err
	}

	outputs := getBasePrData(ctx, client, user, &profile)

	filteredOutputs := filterOutputs(outputs, c.String("owner"), c.StringSlice("repo"))

	sort.Sort(filteredOutputs)

	getExtraData(ctx, client, user, c.Bool("need-rebase"), filteredOutputs)

	return printResults(filteredOutputs, c.Bool("verbose"), c.App.Writer)
}

func getExtraData(ctx context.Context, client *github.Client, user *github.User, filterRebased bool, outputs []*prInfo) {
	wg := sync.WaitGroup{}
	for _, output := range outputs {
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
	}

	wg.Wait()
}

// CompleteParse handles bash autocompletion for the 'parse' command
func CompleteParse(c *cli.Context) {
	lastParam := os.Args[len(os.Args)-2]
	if lastParam != "--user" && lastParam != "--repo" {
		for _, flag := range c.App.Command("parse").Flags {
			name := strings.Split(flag.GetName(), ",")[0]
			if !c.IsSet(name) {
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
		handleUserCompletion(&profile, c.App.Writer)
		return
	}

	for _, repo := range profile.TrackedRepos {
		fmt.Fprintf(c.App.Writer, "%s/%s\n", repo.Owner, repo.Name)
	}
}

func handleUserCompletion(profile *config.PrpConfigProfile, writer io.Writer) {
	ctx := context.Background()
	client, err := getGithubClient(ctx, &profile.Token, &profile.APIURL)
	if err != nil {
		return
	}

	suggestionList := []string{}
	suggestionChan := make(chan string)
	wg := sync.WaitGroup{}
	for _, repo := range profile.TrackedRepos {
		wg.Add(1)
		go func(repo config.PrpConfigRepo) {
			repoPrs, err := getRepoPullRequests(ctx, client, repo.Owner, repo.Name)
			if err != nil {
				wg.Done()
				return
			}

			for _, pr := range repoPrs {
				wg.Add(1)
				suggestionChan <- pr.Head.User.GetLogin()
			}

			wg.Done()
		}(repo)
	}

	go func() {
		for {
			newSuggestion := <-suggestionChan
			suggestionList = append(suggestionList, newSuggestion)
			wg.Done()
		}
	}()

	wg.Wait()

	suggestionList = unique(suggestionList)
	sort.Strings(suggestionList)
	fmt.Fprintln(writer, strings.Join(suggestionList, "\n"))
}
