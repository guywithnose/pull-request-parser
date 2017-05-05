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
	newRepo := config.Repo{Owner: owner, Name: repoName, IgnoredBuilds: []string{}}

	if isTracked(profile.TrackedRepos, owner, repoName) {
		return cli.NewExitError(fmt.Sprintf("%s/%s is already tracked", owner, repoName), 1)
	}

	profile.TrackedRepos = append(profile.TrackedRepos, newRepo)
	configData.Profiles[*profileName] = profile

	return configData.Write(c.GlobalString("config"))
}

func isTracked(trackedRepos []config.Repo, owner, repoName string) bool {
	for _, repo := range trackedRepos {
		if repo.Owner == owner && repo.Name == repoName {
			return true
		}
	}

	return false
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

	suggestionList := getSuggestions(client, allRepos, c.NArg(), c.Args().Get(0), profile.TrackedRepos)

	suggestionList = unique(suggestionList)
	sort.Strings(suggestionList)
	fmt.Fprintln(c.App.Writer, strings.Join(suggestionList, "\n"))
}

func getSuggestions(client *github.Client, allRepos []*github.Repository, numArgs int, owner string, trackedRepos []config.Repo) []string {
	suggestionList := []string{}
	suggestionChan := make(chan *string, 5)
	for _, repo := range allRepos {
		go func(repo *github.Repository) {
			isFirstArg := numArgs == 0

			login := *repo.Owner.Login
			name := *repo.Name
			if *repo.Fork {
				suggestionChan <- handleFork(client, login, name, owner, isFirstArg, trackedRepos)
			} else {
				suggestionChan <- handleNonFork(isFirstArg, login, owner, name)
			}
		}(repo)
	}

	for i := 0; i < len(allRepos); i++ {
		newSuggestion := <-suggestionChan
		if newSuggestion != nil {
			suggestionList = append(suggestionList, *newSuggestion)
		}
	}

	return suggestionList
}

func handleNonFork(isFirstArg bool, login, owner, name string) *string {
	if isFirstArg {
		return &login
	} else if owner == login {
		return &name
	}

	return nil
}

func handleFork(client *github.Client, login, name, ownerParam string, isFirstArg bool, trackedRepos []config.Repo) *string {
	fullRepository, _, err := client.Repositories.Get(context.Background(), login, name)
	if err != nil {
		return nil
	}

	if repoIsTracked(trackedRepos, fullRepository) {
		return nil
	}

	if isFirstArg {
		return fullRepository.Source.Owner.Login
	} else if ownerParam == *fullRepository.Source.Owner.Login {
		return fullRepository.Source.Name
	}

	return nil
}

func repoIsTracked(trackedRepos []config.Repo, fullRepository *github.Repository) bool {
	for _, repo := range trackedRepos {
		if repo.Owner == *fullRepository.Source.Owner.Login && repo.Name == *fullRepository.Source.Name {
			return true
		}
	}

	return false
}
