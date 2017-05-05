package command

import (
	"fmt"
	"strings"

	"github.com/guywithnose/pull-request-parser/config"
	"github.com/urfave/cli"
)

func loadConfig(c *cli.Context) (*config.PrpConfig, error) {
	configFile := c.GlobalString("config")
	if configFile == "" {
		return nil, cli.NewExitError("You must specify a config file", 1)
	}

	configData, err := config.LoadFromFile(configFile)
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

func loadRepo(profile *config.Profile, repoName string) (*config.Repo, int, error) {
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
