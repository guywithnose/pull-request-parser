package command

import (
	"github.com/guywithnose/runner"
	"github.com/urfave/cli"
)

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
		Action:  CmdInitConfig,
	},
	{
		Name:         "parse",
		Aliases:      []string{"pa"},
		Usage:        "Parse your pull requests",
		Action:       CmdParse,
		BashComplete: CompleteParse,
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
			cli.BoolFlag{
				Name:  "use-cache, uc, c",
				Usage: "Use file cache",
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
				Action:       CmdProfileAdd,
				BashComplete: CompleteProfileAdd,
				Flags:        profilCrudFlags,
			},
			{
				Name:         "update",
				Aliases:      []string{"u"},
				Usage:        "Update a profile",
				Action:       CmdProfileUpdate,
				BashComplete: CompleteProfileAdd,
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
				Action:       CmdRepoAdd,
				BashComplete: CompleteRepoAdd,
			},
			{
				Name:         "remove",
				Aliases:      []string{"r"},
				Usage:        "Remove a repo",
				Action:       CmdRepoRemove,
				BashComplete: CompleteRepoRemove,
			},
			{
				Name:         "ignore-build",
				Aliases:      []string{"ib"},
				Usage:        "Ignore a build.",
				Action:       CmdRepoIgnoreBuild,
				BashComplete: CompleteRepoIgnoreBuild,
			},
			{
				Name:         "remove-ignored-build",
				Aliases:      []string{"rib"},
				Usage:        "Remove a build from the list of ignored builds.",
				Action:       CmdRepoRemoveIgnoredBuild,
				BashComplete: CompleteRepoRemoveIgnoredBuild,
			},
			{
				Name:         "set-path",
				Aliases:      []string{"sp"},
				Usage:        "Set the path of the local clone.",
				Action:       CmdRepoSetPath,
				BashComplete: CompleteRepoSetPath,
			},
		},
	},
	{
		Name:         "auto-rebase",
		Aliases:      []string{"a", "auto"},
		Usage:        "Automatically rebase your pull requests with local path set",
		Action:       CmdAutoRebase(runner.Real{}),
		BashComplete: CompleteAutoRebase,
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
			cli.BoolFlag{
				Name:  "use-cache, uc, c",
				Usage: "Use file cache",
			},
		},
	},
}
