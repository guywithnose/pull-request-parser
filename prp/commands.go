package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/guywithnose/pull-request-parser/command"
	"github.com/guywithnose/pull-request-parser/config"
	"github.com/guywithnose/pull-request-parser/execWrapper"
	"github.com/urfave/cli"
)

var globalFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "config, c",
		Usage:  "The config file",
		EnvVar: "PRP_CONFIG_FILE",
	},
	cli.StringFlag{
		Name:  "profile, p",
		Usage: "The current profile",
		Value: "default",
	},
}

var profilCrudFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "token, t",
		Usage: "The github access token for this profile",
	},
	cli.StringFlag{
		Name:  "apiUrl, a",
		Usage: "The url for accessing the github API (You only need to specify this for Enterprise Github)",
	},
}

// Commands defines the commands that can be called on hostBuilder
var Commands = []cli.Command{
	{
		Name:    "init-config",
		Aliases: []string{"ic"},
		Usage:   "Initialize a configuration file",
		Action:  command.CmdInitConfig,
	},
	{
		Name:         "parse",
		Aliases:      []string{"pa"},
		Usage:        "Parse your pull requests",
		Action:       command.CmdParse,
		BashComplete: command.CompleteParse,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "user, owner, u",
				Usage: "Only show pull requests by owner.",
			},
			cli.StringSliceFlag{
				Name:  "repo, r",
				Usage: "Only show pull requests on a repository.",
			},
			cli.BoolFlag{
				Name:  "need-rebase, nr",
				Usage: "Only show pull requests that need a rebase.",
			},
			cli.BoolFlag{
				Name:  "verbose, v",
				Usage: "Output more info",
			},
		},
	},
	{
		Name:    "profile",
		Aliases: []string{"pr"},
		Usage:   "Manage profiles",
		Subcommands: []cli.Command{
			{
				Name:         "add",
				Aliases:      []string{"a"},
				Usage:        "Add a profile",
				Action:       command.CmdProfileAdd,
				BashComplete: command.CompleteProfileAdd,
				Flags:        profilCrudFlags,
			},
			{
				Name:         "update",
				Aliases:      []string{"u"},
				Usage:        "Update a profile",
				Action:       command.CmdProfileUpdate,
				BashComplete: command.CompleteProfileAdd,
				Flags:        profilCrudFlags,
			},
		},
	},
	{
		Name:    "repo",
		Aliases: []string{"r"},
		Usage:   "Manage repos.",
		Subcommands: []cli.Command{
			{
				Name:         "add",
				Aliases:      []string{"a"},
				Usage:        "Add a repo",
				Action:       command.CmdRepoAdd,
				BashComplete: command.CompleteRepoAdd,
			},
			{
				Name:         "remove",
				Aliases:      []string{"r"},
				Usage:        "Remove a repo",
				Action:       command.CmdRepoRemove,
				BashComplete: command.CompleteRepoRemove,
			},
			{
				Name:         "ignore-build",
				Aliases:      []string{"ib"},
				Usage:        "Ignore a build.",
				Action:       command.CmdRepoIgnoreBuild,
				BashComplete: command.CompleteRepoIgnoreBuild,
			},
			{
				Name:         "remove-ignored-build",
				Aliases:      []string{"rib"},
				Usage:        "Remove a build from the list of ignored builds.",
				Action:       command.CmdRepoRemoveIgnoredBuild,
				BashComplete: command.CompleteRepoRemoveIgnoredBuild,
			},
			{
				Name:         "set-path",
				Aliases:      []string{"sp"},
				Usage:        "Set the path of the local clone.",
				Action:       command.CmdRepoSetPath,
				BashComplete: command.CompleteRepoSetPath,
			},
		},
	},
	{
		Name:         "auto-rebase",
		Aliases:      []string{"a", "auto"},
		Usage:        "Automatically rebase your pull requests with local path set",
		Action:       command.CmdAutoRebase(execWrapper.Wrapper{}),
		BashComplete: command.CompleteAutoRebase,
		Flags: []cli.Flag{
			cli.StringSliceFlag{
				Name:  "repo, r",
				Usage: "Only rebase these repos.",
			},
			cli.IntFlag{
				Name:  "pull-request-number, prNum, n",
				Usage: "A specific pull request number",
			},
			cli.BoolFlag{
				Name:  "verbose, v",
				Usage: "Output more info",
			},
		},
	},
}

// CommandNotFound runs when hostBuilder is invoked with an invalid command
func CommandNotFound(c *cli.Context, command string) {
	fmt.Fprintf(c.App.Writer, "%s: '%s' is not a %s command. See '%s --help'.", c.App.Name, command, c.App.Name, c.App.Name)
	os.Exit(2)
}

// RootCompletion prints the list of root commands as the root completion method
// This is similar to the deafult method, but it excludes aliases
func RootCompletion(c *cli.Context) {
	lastParam := os.Args[len(os.Args)-2]
	if lastParam == "--config" {
		fmt.Println("fileCompletion")
		return
	}

	if lastParam == "--profile" {
		configFile := c.GlobalString("config")
		if configFile == "" {
			return
		}

		configData, err := config.LoadConfigFromFile(configFile)
		if err != nil {
			return
		}

		for profileName := range configData.Profiles {
			fmt.Fprintln(c.App.Writer, profileName)
		}

		return
	}

	for _, command := range c.App.Commands {
		if command.Hidden {
			continue
		}

		fmt.Fprintf(c.App.Writer, "%s:%s\n", command.Name, command.Usage)
	}

	for _, flag := range c.App.Flags {
		name := strings.Split(flag.GetName(), ",")[0]
		if !c.IsSet(name) {
			fmt.Fprintf(c.App.Writer, "--%s\n", name)
		}
	}
}
