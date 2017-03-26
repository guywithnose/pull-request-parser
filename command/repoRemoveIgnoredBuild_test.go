package command

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestCmdRepoRemoveIgnoredBuild(t *testing.T) {
	_, configFileName := getConfigWithIgnoredBuild(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"foo/bar", "goo"}))
	assert.Nil(t, CmdRepoRemoveIgnoredBuild(cli.NewContext(nil, set, nil)))

	expectedConfigFile, disposableConfigFile := getConfigWithTwoRepos(t)
	removeFile(t, disposableConfigFile)
	assertConfigFile(t, expectedConfigFile, configFileName)
}

func TestCmdRepoRemoveIgnoredBuildNoConfig(t *testing.T) {
	err := CmdRepoRemoveIgnoredBuild(cli.NewContext(nil, flag.NewFlagSet("test", 0), nil))
	assert.EqualError(t, err, "You must specify a config file")
}

func TestCmdRepoRemoveIgnoredBuildInvalidRepo(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"own/rep", "goo"}))

	err := CmdRepoRemoveIgnoredBuild(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "Not a valid Repo: own/rep")
}

func TestCmdRepoRemoveIgnoredBuildUsage(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)

	err := CmdRepoRemoveIgnoredBuild(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "Usage: \"prp profile repo remove-ignored-build {repoName} {buildName}\"")
}

func TestCmdRepoRemoveIgnoredBuildNotIgnored(t *testing.T) {
	_, configFileName := getConfigWithTwoRepos(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"own/rep", "goo"}))
	err := CmdRepoRemoveIgnoredBuild(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "own/rep is not ignoring goo")
}

func TestCompleteRepoRemoveIgnoredBuildRepos(t *testing.T) {
	_, configFileName := getConfigWithTwoRepos(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	os.Args = []string{"repo", "ignore-build", "--completion"}
	app, writer := appWithTestWriter()
	CompleteRepoRemoveIgnoredBuild(cli.NewContext(app, set, nil))
	assertOutput(t, writer, "foo/bar\nown/rep\n")
}

func TestCompleteRepoRemoveIgnoredBuildInvalidRepo(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"foo/bar"}))
	os.Args = []string{"repo", "remove-ignored-build", "foo/bar", "--completion"}
	app, writer := appWithTestWriter()
	CompleteRepoRemoveIgnoredBuild(cli.NewContext(app, set, nil))
	assertOutput(t, writer, "")
}

func TestCompleteRepoRemoveIgnoredBuildIgnoredBuilds(t *testing.T) {
	_, configFileName := getConfigWithIgnoredBuild(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"foo/bar"}))
	os.Args = []string{"repo", "ignore-build", "own/rep", "--completion"}
	app, writer := appWithTestWriter()
	CompleteRepoRemoveIgnoredBuild(cli.NewContext(app, set, nil))
	assertOutput(t, writer, "goo\n")
}

func TestCompleteRepoRemoveIgnoredBuildDone(t *testing.T) {
	_, configFileName := getConfigWithIgnoredBuild(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"own/rep", "goo"}))
	os.Args = []string{"repo", "ignore-build", "own/rep", "goo", "--completion"}
	app, writer := appWithTestWriter()
	CompleteRepoRemoveIgnoredBuild(cli.NewContext(app, set, nil))
	assertOutput(t, writer, "")
}

func TestCompleteRepoRemoveIgnoredBuildNoConfig(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	os.Args = []string{"repo", "ignore-build", "--completion"}
	app, writer := appWithTestWriter()
	CompleteRepoRemoveIgnoredBuild(cli.NewContext(app, set, nil))
	assertOutput(t, writer, "")
}
