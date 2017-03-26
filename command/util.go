package command

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/guywithnose/pull-request-parser/config"
	"github.com/urfave/cli"
)

func loadConfig(c *cli.Context) (*config.PrpConfig, error) {
	configFile := c.GlobalString("config")
	if configFile == "" {
		return nil, cli.NewExitError("You must specify a config file", 1)
	}

	configData, err := config.LoadConfigFromFile(configFile)
	if err != nil {
		return nil, err
	}

	return configData, nil
}

func loadProfile(c *cli.Context) (*config.PrpConfig, *string, error) {
	configData, err := loadConfig(c)
	if err != nil {
		return nil, nil, err
	}

	profileName := c.GlobalString("profile")

	_, ok := configData.Profiles[profileName]
	if !ok {
		return nil, nil, cli.NewExitError(fmt.Sprintf("Invalid Profile: %s", profileName), 1)
	}

	return configData, &profileName, nil
}

func loadRepo(profile *config.PrpConfigProfile, repoName string) (*config.PrpConfigRepo, int, error) {
	repoNameParts := strings.Split(repoName, "/")
	if len(repoNameParts) == 2 {
		for index, repo := range profile.TrackedRepos {
			if repo.Owner == repoNameParts[0] && repo.Name == repoNameParts[1] {
				return &repo, index, nil
			}
		}
	}

	return nil, -1, cli.NewExitError(fmt.Sprintf("Not a valid Repo: %s", repoName), 1)
}

func sortRepoNames(profile *config.PrpConfigProfile) []string {
	repoNames := []string{}
	for _, repo := range profile.TrackedRepos {
		repoNames = append(repoNames, fmt.Sprintf("%s/%s", repo.Owner, repo.Name))
	}

	sort.Strings(repoNames)
	return repoNames
}

func getGithubClient(ctx context.Context, token, apiURL *string) (*github.Client, error) {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: *token})
	tokenClient := oauth2.NewClient(ctx, tokenSource)

	cache := diskcache.New(fmt.Sprintf("%s/prpCache", os.TempDir()))
	transport := httpcache.NewTransport(cache)
	transport.Transport = tokenClient.Transport
	tokenClient.Transport = transport

	client := github.NewClient(tokenClient)
	if apiURL != nil && *apiURL != "" {
		url, err := url.Parse(*apiURL)
		if err != nil {
			return nil, err
		}

		client.BaseURL = url
	}

	return client, nil
}

func unique(in []string) []string {
	set := make(map[string]bool, len(in))
	for _, value := range in {
		set[value] = true
	}

	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}

	return out
}
