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

	for _, existingBuildName := range repo.IgnoredBuilds {
		if existingBuildName == buildName {
			return cli.NewExitError(fmt.Sprintf("%s is already being ignored by %s", buildName, repoName), 1)
		}
	}

	repo.IgnoredBuilds = append(repo.IgnoredBuilds, buildName)
	profile.TrackedRepos[repoIndex] = *repo
	configData.Profiles[*profileName] = profile

	return config.WriteConfig(c.GlobalString("config"), configData)
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
		buildNames := []string{}
		for _, profile := range configData.Profiles {
			for _, repo := range profile.TrackedRepos {
				if fmt.Sprintf("%s/%s", repo.Owner, repo.Name) == repoName {
					continue
				}

				for _, buildName := range repo.IgnoredBuilds {
					buildNames = append(buildNames, buildName)
				}
			}
		}

		sort.Strings(buildNames)
		fmt.Fprintln(c.App.Writer, strings.Join(buildNames, "\n"))
	}
}
