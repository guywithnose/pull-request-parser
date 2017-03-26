package command

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/go-github/github"
	"github.com/guywithnose/pull-request-parser/config"
	"github.com/urfave/cli"
)

// CmdRepoAdd parses the pull requests
func CmdRepoAdd(c *cli.Context) error {
	configData, profileName, err := loadProfile(c)
	if err != nil {
		return err
	}

	if c.NArg() != 2 {
		return cli.NewExitError("Usage: \"prp profile repo add {owner} {repoName}\"", 1)
	}

	owner := c.Args().Get(0)
	repoName := c.Args().Get(1)

	profile := configData.Profiles[*profileName]
	newRepo := config.PrpConfigRepo{Owner: owner, Name: repoName, IgnoredBuilds: []string{}}

	for _, repo := range profile.TrackedRepos {
		if repo.Owner == owner && repo.Name == repoName {
			return cli.NewExitError(fmt.Sprintf("%s/%s is already tracked", owner, repoName), 1)
		}
	}

	profile.TrackedRepos = append(profile.TrackedRepos, newRepo)
	configData.Profiles[*profileName] = profile

	return config.WriteConfig(c.GlobalString("config"), configData)
}

// CompleteRepoAdd handles bash autocompletion for the 'profile repo add' command
func CompleteRepoAdd(c *cli.Context) {
	if c.NArg() >= 2 {
		return
	}

	configData, profileName, err := loadProfile(c)
	if err != nil {
		return
	}

	profile := configData.Profiles[*profileName]
	token := profile.Token

	ctx := context.Background()
	client, err := getGithubClient(ctx, &token, &profile.APIURL)
	if err != nil {
		return
	}

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return
	}

	allRepos := getAllRepos(ctx, client, *user.Login)

	suggestionList := []string{}
	suggestionChan := make(chan *string)
	for _, repo := range allRepos {
		go func(repo *github.Repository) {
			if !*repo.Fork {
				suggestionChan <- nil
				return
			}

			login := *repo.Owner.Login
			name := *repo.Name
			suggestionChan <- parseRepository(ctx, client, login, name, c.Args().Get(0), c.NArg() == 0, profile.TrackedRepos)
		}(repo)
	}

	for i := 0; i < len(allRepos); i++ {
		newSuggestion := <-suggestionChan
		if newSuggestion != nil {
			suggestionList = append(suggestionList, *newSuggestion)
		}
	}

	suggestionList = unique(suggestionList)
	sort.Strings(suggestionList)
	fmt.Fprintln(c.App.Writer, strings.Join(suggestionList, "\n"))
}

func getAllRepos(ctx context.Context, client *github.Client, login string) []*github.Repository {
	allRepos := make([]*github.Repository, 0, 30)
	opt := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		repositories, resp, err := client.Repositories.List(ctx, login, opt)
		if err != nil {
			return []*github.Repository{}
		}

		allRepos = append(allRepos, repositories...)
		if resp.NextPage == 0 {
			return allRepos
		}

		opt.ListOptions.Page = resp.NextPage
	}
}

func parseRepository(ctx context.Context, client *github.Client, login, name, repoName string, firstArg bool, trackedRepos []config.PrpConfigRepo) *string {
	fullRepository, _, err := client.Repositories.Get(ctx, login, name)
	if err != nil {
		return nil
	}

	for _, repo := range trackedRepos {
		if repo.Owner == *fullRepository.Source.Owner.Login && repo.Name == *fullRepository.Source.Name {
			return nil
		}
	}

	if firstArg {
		return fullRepository.Source.Owner.Login
	} else if repoName == *fullRepository.Source.Owner.Login {
		return fullRepository.Source.Name
	}

	return nil
}
