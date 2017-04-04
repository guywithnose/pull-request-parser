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

	client, err := getGithubClient(&token, &profile.APIURL, true)
	if err != nil {
		return
	}

	user, _, err := client.Users.Get(context.Background(), "")
	if err != nil {
		return
	}

	allRepos := getAllRepos(client, *user.Login)

	suggestionList := []string{}
	suggestionChan := make(chan *string, 5)
	for _, repo := range allRepos {
		go func(repo *github.Repository) {
			firstArg := c.NArg() == 0

			login := *repo.Owner.Login
			name := *repo.Name
			suggestionChan <- parseRepository(client, login, name, c.Args().Get(0), *repo.Fork, firstArg, profile.TrackedRepos)
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

func parseRepository(client *github.Client, login, name, ownerParam string, isFork, firstArg bool, trackedRepos []config.PrpConfigRepo) *string {
	if !isFork {
		if firstArg {
			return &login
		} else if ownerParam == login {
			return &name
		}

		return nil
	}

	fullRepository, _, err := client.Repositories.Get(context.Background(), login, name)
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
	} else if ownerParam == *fullRepository.Source.Owner.Login {
		return fullRepository.Source.Name
	}

	return nil
}
