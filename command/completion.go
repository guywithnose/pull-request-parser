package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/guywithnose/pull-request-parser/config"
	"github.com/urfave/cli"
)

// RootCompletion prints the list of root commands as the root completion method
// This is similar to the default method, but it excludes aliases
func RootCompletion(c *cli.Context) {
	lastParam := os.Args[len(os.Args)-2]
	if lastParam == "--config" {
		fmt.Fprintln(c.App.Writer, "fileCompletion")
		return
	}

	if lastParam == "--profile" {
		completeProfile(c)
		return
	}

	for _, command := range c.App.Commands {
		if command.Hidden {
			continue
		}

		fmt.Fprintf(c.App.Writer, "%s:%s\n", command.Name, command.Usage)
	}

	completeRootFlags(c)
}

func completeProfile(c *cli.Context) {
	configFile := c.GlobalString("config")
	if configFile == "" {
		return
	}

	configData, err := config.LoadFromFile(configFile)
	if err != nil {
		return
	}

	for profileName := range configData.Profiles {
		fmt.Fprintln(c.App.Writer, profileName)
	}

}

func completeRootFlags(c *cli.Context) {
	for _, flag := range c.App.Flags {
		name := strings.Split(flag.GetName(), ",")[0]
		if !c.IsSet(name) {
			fmt.Fprintf(c.App.Writer, "--%s\n", name)
		}
	}
}
