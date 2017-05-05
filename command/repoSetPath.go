package command

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"
)

// CmdRepoSetPath parses the pull requests
func CmdRepoSetPath(c *cli.Context) error {
	configData, profileName, err := loadProfile(c)
	if err != nil {
		return err
	}

	if c.NArg() != 2 {
		return cli.NewExitError("Usage: \"prp profile repo set-path {repoName} {localPath}\"", 1)
	}

	repoName := c.Args().Get(0)
	localPath := c.Args().Get(1)

	profile := configData.Profiles[*profileName]
	repo, repoIndex, err := loadRepo(&profile, repoName)
	if err != nil {
		return err
	}

	err = checkPath(localPath)
	if err != nil {
		return err
	}

	repo.LocalPath = localPath
	profile.TrackedRepos[repoIndex] = *repo
	configData.Profiles[*profileName] = profile

	return configData.Write(c.GlobalString("config"))
}

// CompleteRepoSetPath handles bash autocompletion for the 'profile repo set-path' command
func CompleteRepoSetPath(c *cli.Context) {
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
		fmt.Fprintln(c.App.Writer, "fileCompletion")
	}
}
