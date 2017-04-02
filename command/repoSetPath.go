package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/guywithnose/pull-request-parser/config"
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

	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return cli.NewExitError(fmt.Sprintf("Path does not exist: %s", localPath), 1)
	}

	if _, err := os.Stat(fmt.Sprintf("%s/.git", localPath)); os.IsNotExist(err) {
		return cli.NewExitError(fmt.Sprintf("Path is not a git repo: %s", localPath), 1)
	}

	repo.LocalPath = localPath
	profile.TrackedRepos[repoIndex] = *repo
	configData.Profiles[*profileName] = profile

	return config.WriteConfig(c.GlobalString("config"), configData)
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
