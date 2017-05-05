package command

import (
	"fmt"
	"sort"
	"strings"

	"github.com/guywithnose/pull-request-parser/config"
	"github.com/urfave/cli"
)

// CmdRepoIgnoreBuild parses the pull requests
func CmdRepoIgnoreBuild(c *cli.Context) error {
	configData, profileName, err := loadProfile(c)
	if err != nil {
		return err
	}

	if c.NArg() != 2 {
		return cli.NewExitError("Usage: \"prp profile repo ignore-build {repoName} {buildName}\"", 1)
	}

	repoName := c.Args().Get(0)
	buildName := c.Args().Get(1)

	profile := configData.Profiles[*profileName]
	repo, repoIndex, err := loadRepo(&profile, repoName)
	if err != nil {
		return err
	}

	err = checkExistingIgnoredBuilds(repo, buildName, repoName)
	if err != nil {
		return err
	}

	repo.IgnoredBuilds = append(repo.IgnoredBuilds, buildName)
	profile.TrackedRepos[repoIndex] = *repo
	configData.Profiles[*profileName] = profile

	return configData.Write(c.GlobalString("config"))
}

func checkExistingIgnoredBuilds(repo *config.Repo, buildName, repoName string) error {
	for _, existingBuildName := range repo.IgnoredBuilds {
		if existingBuildName == buildName {
			return cli.NewExitError(fmt.Sprintf("%s is already being ignored by %s", buildName, repoName), 1)
		}
	}

	return nil
}

// CompleteRepoIgnoreBuild handles bash autocompletion for the 'profile repo ignore-build' command
func CompleteRepoIgnoreBuild(c *cli.Context) {
	if c.NArg() >= 2 {
		return
	}

	configData, profileName, err := loadProfile(c)
	if err != nil {
		return
	}

	profile := configData.Profiles[*profileName]

	if c.NArg() == 0 {
		fmt.Fprintln(c.App.Writer, strings.Join(sortRepoNames(&profile), "\n"))
	} else {
		repoName := c.Args().Get(0)
		ignoredBuilds := getExistingIgnoredBuilds(configData, repoName)
		sort.Strings(ignoredBuilds)
		fmt.Fprintln(c.App.Writer, strings.Join(ignoredBuilds, "\n"))
	}
}

func getExistingIgnoredBuilds(configData *config.PrpConfig, repoName string) []string {
	buildNames := []string{}
	for _, profile := range configData.Profiles {
		for _, repo := range profile.TrackedRepos {
			if fmt.Sprintf("%s/%s", repo.Owner, repo.Name) == repoName {
				continue
			}

			buildNames = append(buildNames, repo.IgnoredBuilds...)
		}
	}

	return buildNames
}
