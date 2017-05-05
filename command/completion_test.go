package command_test

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/guywithnose/pull-request-parser/command"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestRootCompletion(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	app, writer, _ := appWithTestWriters()
	app.Commands = append(command.Commands, cli.Command{Hidden: true, Name: "don't show"})
	app.Flags = command.GlobalFlags
	command.RootCompletion(cli.NewContext(app, set, nil))
	assert.Equal(
		t,
		strings.Join(
			[]string{
				"init-config:Initialize a configuration file",
				"parse:Parse your pull requests",
				"profile:Manage profiles",
				"repo:Manage repos.",
				"auto-rebase:Automatically rebase your pull requests with local path set",
				"--config",
				"--profile",
				"",
			},
			"\n",
		),
		writer.String(),
	)
}

func TestRootCompletionConfig(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	app, writer, _ := appWithTestWriters()
	app.Commands = command.Commands
	os.Args = []string{os.Args[0], "prp", "--config", "--completion"}
	command.RootCompletion(cli.NewContext(app, set, nil))
	assert.Equal(t, "fileCompletion\n", writer.String())
}

func TestRootCompletionProfile(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	app, writer, _ := appWithTestWriters()
	app.Commands = command.Commands
	os.Args = []string{os.Args[0], "prp", "--profile", "--completion"}
	_, configFileName := getConfigWithAPIURL(t, "")
	set.String("config", configFileName, "doc")
	command.RootCompletion(cli.NewContext(app, set, nil))
	assert.Equal(t, "foo\n", writer.String())
}

func TestRootCompletionProfileNoConfig(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	app, writer, _ := appWithTestWriters()
	app.Commands = command.Commands
	os.Args = []string{os.Args[0], "prp", "--profile", "--completion"}
	command.RootCompletion(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}

func TestRootCompletionProfileInvalidConfigConfig(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	app, writer, _ := appWithTestWriters()
	app.Commands = command.Commands
	os.Args = []string{os.Args[0], "prp", "--profile", "--completion"}
	set.String("config", "/notafile", "doc")
	command.RootCompletion(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}
