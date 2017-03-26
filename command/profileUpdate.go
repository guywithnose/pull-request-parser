package command

import (
	"github.com/guywithnose/pull-request-parser/config"
	"github.com/urfave/cli"
)

// CmdProfileUpdate parses the pull requests
func CmdProfileUpdate(c *cli.Context) error {
	configData, profileName, err := loadProfile(c)
	if err != nil {
		return err
	}

	if c.NArg() != 0 {
		return cli.NewExitError("Usage: \"prp profile update\"", 1)
	}

	profile := configData.Profiles[*profileName]

	token := c.String("token")
	APIURL := c.String("apiUrl")

	if token == "" && APIURL == "" {
		return cli.NewExitError("An update parameter is required", 1)
	}

	if token != "" {
		profile.Token = token
	}

	if APIURL != "" {
		profile.APIURL = APIURL
	}

	configData.Profiles[*profileName] = profile

	return config.WriteConfig(c.GlobalString("config"), configData)
}
