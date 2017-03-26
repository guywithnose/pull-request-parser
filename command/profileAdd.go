package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/guywithnose/pull-request-parser/config"
	"github.com/urfave/cli"
)

// CmdProfileAdd parses the pull requests
func CmdProfileAdd(c *cli.Context) error {
	configData, err := loadConfig(c)
	if err != nil {
		return err
	}

	if c.NArg() != 1 {
		return cli.NewExitError("Usage: \"prp profile add {profileName} --token {token}\"", 1)
	}

	profileName := c.Args().Get(0)

	token := c.String("token")
	if token == "" {
		return cli.NewExitError("You must specify a token", 1)
	}

	if _, ok := configData.Profiles[profileName]; ok {
		return cli.NewExitError(fmt.Sprintf("Profile %s already exists", profileName), 1)
	}

	newProfile := config.PrpConfigProfile{Token: token, TrackedRepos: []config.PrpConfigRepo{}, APIURL: c.String("apiUrl")}

	configData.Profiles[profileName] = newProfile

	return config.WriteConfig(c.GlobalString("config"), configData)
}

// CompleteProfileAdd handles bash autocompletion for the 'profile add' command
func CompleteProfileAdd(c *cli.Context) {
	lastParam := os.Args[len(os.Args)-2]
	if lastParam != "--token" && lastParam != "--apiUrl" {
		for _, flag := range c.App.Command("add").Flags {
			name := strings.Split(flag.GetName(), ",")[0]
			if !c.IsSet(name) {
				fmt.Fprintf(c.App.Writer, "--%s\n", name)
			}
		}
		return
	}

	configData, err := loadConfig(c)
	if err != nil {
		return
	}

	suggestionList := []string{}
	for _, profile := range configData.Profiles {
		if lastParam == "--token" {
			suggestionList = append(suggestionList, profile.Token)
		}

		if lastParam == "--apiUrl" {
			if profile.APIURL != "" {
				suggestionList = append(suggestionList, strings.Replace(profile.APIURL, ":", "\\:", -1))
			}
		}
	}

	fmt.Fprintln(c.App.Writer, strings.Join(suggestionList, "\n"))
}
