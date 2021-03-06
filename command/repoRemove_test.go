package command_test

import (
	"flag"
	"os"
	"testing"

	"github.com/guywithnose/pull-request-parser/command"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestCmdRepoRemove(t *testing.T) {
	_, configFileName := getConfigWithTwoRepos(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"own/rep"}))
	assert.Nil(t, command.CmdRepoRemove(cli.NewContext(nil, set, nil)))
	assert.Nil(t, set.Parse([]string{"foo/bar"}))
	assert.Nil(t, command.CmdRepoRemove(cli.NewContext(nil, set, nil)))

	expectedConfigFile, disposableConfigFile := getConfigWithFooProfile(t)
	removeFile(t, disposableConfigFile)
	assertConfigFile(t, expectedConfigFile, configFileName)
}

func TestCmdRepoRemoveNoConfig(t *testing.T) {
	err := command.CmdRepoRemove(cli.NewContext(nil, flag.NewFlagSet("test", 0), nil))
	assert.EqualError(t, err, "You must specify a config file")
}

func TestCmdRepoRemoveInvalidRepo(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"own/rep"}))

	err := command.CmdRepoRemove(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "Not a valid Repo: own/rep")
}

func TestCmdRepoRemoveUsage(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	err := command.CmdRepoRemove(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "Usage: \"prp profile repo remove {repoName}\"")
}

func TestCompleteRepoRemoveRepos(t *testing.T) {
	_, configFileName := getConfigWithTwoRepos(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	os.Args = []string{"repo", "remove", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteRepoRemove(cli.NewContext(app, set, nil))
	assert.Equal(t, "foo/bar\nown/rep\n", writer.String())
}

func TestCompleteRepoRemoveDone(t *testing.T) {
	_, configFileName := getConfigWithIgnoredBuild(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"own/rep"}))
	os.Args = []string{"repo", "remove", "own/rep", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteRepoRemove(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}

func TestCompleteRepoRemoveNoConfig(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	os.Args = []string{"repo", "ignore-build", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteRepoRemove(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}
