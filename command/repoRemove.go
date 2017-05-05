package command

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"
)

// CmdRepoRemove parses the pull requests
func CmdRepoRemove(c *cli.Context) error {
	configData, profileName, err := loadProfile(c)
	if err != nil {
		return err
	}

	if c.NArg() != 1 {
		return cli.NewExitError("Usage: \"prp profile repo remove {repoName}\"", 1)
	}

	repoName := c.Args().Get(0)

	profile := configData.Profiles[*profileName]
	_, repoIndex, err := loadRepo(&profile, repoName)
	if err != nil {
		return err
	}

	profile.TrackedRepos[repoIndex] = profile.TrackedRepos[len(profile.TrackedRepos)-1]
	profile.TrackedRepos = profile.TrackedRepos[:len(profile.TrackedRepos)-1]
	configData.Profiles[*profileName] = profile

	return configData.Write(c.GlobalString("config"))
}

// CompleteRepoRemove handles bash autocompletion for the 'profile repo remove' command
func CompleteRepoRemove(c *cli.Context) {
	if c.NArg() > 0 {
		return
	}

	configData, profileName, err := loadProfile(c)
	if err != nil {
		return
	}

	profile := configData.Profiles[*profileName]

	fmt.Fprintln(c.App.Writer, strings.Join(sortRepoNames(&profile), "\n"))
}
