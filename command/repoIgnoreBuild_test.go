package command_test

import (
	"flag"
	"os"
	"testing"

	"github.com/guywithnose/pull-request-parser/command"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestCmdRepoIgnoreBuild(t *testing.T) {
	_, configFileName := getConfigWithTwoRepos(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"foo/bar", "goo"}))
	assert.Nil(t, command.CmdRepoIgnoreBuild(cli.NewContext(nil, set, nil)))

	expectedConfigFile, disposableConfigFile := getConfigWithIgnoredBuild(t)
	removeFile(t, disposableConfigFile)
	assertConfigFile(t, expectedConfigFile, configFileName)
}

func TestCmdRepoIgnoreBuildNoConfig(t *testing.T) {
	err := command.CmdRepoIgnoreBuild(cli.NewContext(nil, flag.NewFlagSet("test", 0), nil))
	assert.EqualError(t, err, "You must specify a config file")
}

func TestCmdRepoIgnoreBuildInvalidRepo(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"own/rep", "goo"}))

	err := command.CmdRepoIgnoreBuild(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "Not a valid Repo: own/rep")
}

func TestCmdRepoIgnoreBuildUsage(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	err := command.CmdRepoIgnoreBuild(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "Usage: \"prp profile repo ignore-build {repoName} {buildName}\"")
}

func TestCmdRepoIgnoreBuildAlreadyIgnored(t *testing.T) {
	_, configFileName := getConfigWithIgnoredBuild(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"foo/bar", "goo"}))
	err := command.CmdRepoIgnoreBuild(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "goo is already being ignored by foo/bar")
}

func TestCompleteRepoIgnoreBuildRepos(t *testing.T) {
	_, configFileName := getConfigWithTwoRepos(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	os.Args = []string{"repo", "ignore-build", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteRepoIgnoreBuild(cli.NewContext(app, set, nil))
	assert.Equal(t, "foo/bar\nown/rep\n", writer.String())
}

func TestCompleteRepoIgnoreBuildIgnoredBuilds(t *testing.T) {
	_, configFileName := getConfigWithIgnoredBuild(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"own/rep"}))
	os.Args = []string{"repo", "ignore-build", "own/rep", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteRepoIgnoreBuild(cli.NewContext(app, set, nil))
	assert.Equal(t, "goo\n", writer.String())
}

func TestCompleteRepoIgnoreBuildDone(t *testing.T) {
	_, configFileName := getConfigWithIgnoredBuild(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"own/rep", "goo"}))
	os.Args = []string{"repo", "ignore-build", "own/rep", "goo", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteRepoIgnoreBuild(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}

func TestCompleteRepoIgnoreBuildNoConfig(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	os.Args = []string{"repo", "ignore-build", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteRepoIgnoreBuild(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}
