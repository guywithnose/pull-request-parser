package command

import (
	"fmt"
	"os"

	"github.com/guywithnose/pull-request-parser/config"
	"github.com/urfave/cli"
)

// CmdInitConfig creates a new config file
func CmdInitConfig(c *cli.Context) error {
	configFile := c.GlobalString("config")
	if configFile == "" {
		return cli.NewExitError("You must specify a config file", 1)
	}

	if _, err := os.Stat(configFile); err == nil {
		return cli.NewExitError(fmt.Sprintf("File already exists: %s", configFile), 1)
	}

	configData := &config.PrpConfig{}
	configData.Validate()

	return configData.Write(configFile)
}
