package command

import (
	"fmt"
	"sort"
	"strings"

	"github.com/guywithnose/pull-request-parser/config"
	"github.com/urfave/cli"
)

// CmdRepoRemoveIgnoredBuild parses the pull requests
func CmdRepoRemoveIgnoredBuild(c *cli.Context) error {
	configData, profileName, err := loadProfile(c)
	if err != nil {
		return err
	}

	if c.NArg() != 2 {
		return cli.NewExitError("Usage: \"prp profile repo remove-ignored-build {repoName} {buildName}\"", 1)
	}

	repoName := c.Args().Get(0)
	buildName := c.Args().Get(1)

	profile := configData.Profiles[*profileName]
	repo, repoIndex, err := loadRepo(&profile, repoName)
	if err != nil {
		return err
	}

	foundIndex := -1
	for index, build := range repo.IgnoredBuilds {
		if build == buildName {
			foundIndex = index
			break
		}
	}

	if foundIndex == -1 {
		return cli.NewExitError(fmt.Sprintf("%s is not ignoring %s", repoName, buildName), 1)
	}

	repo.IgnoredBuilds[foundIndex] = repo.IgnoredBuilds[len(repo.IgnoredBuilds)-1]
	repo.IgnoredBuilds = repo.IgnoredBuilds[:len(repo.IgnoredBuilds)-1]

	profile.TrackedRepos[repoIndex] = *repo
	configData.Profiles[*profileName] = profile

	return config.WriteConfig(c.GlobalString("config"), configData)
}

// CompleteRepoRemoveIgnoredBuild handles bash autocompletion for the 'profile repo remove-ignored-build' command
func CompleteRepoRemoveIgnoredBuild(c *cli.Context) {
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
		buildNames := []string{}
		repo, _, err := loadRepo(&profile, repoName)
		if err != nil {
			return
		}

		for _, buildName := range repo.IgnoredBuilds {
			buildNames = append(buildNames, buildName)
		}

		sort.Strings(buildNames)
		fmt.Fprintln(c.App.Writer, strings.Join(buildNames, "\n"))
	}
}