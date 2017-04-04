package command

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestCmdRepoSetPath(t *testing.T) {
	_, configFileName := getConfigWithTwoRepos(t)
	defer removeFile(t, configFileName)
	repoDir, err := ioutil.TempDir("", "repo")
	assert.Nil(t, err)
	assert.Nil(t, os.MkdirAll(fmt.Sprintf("%s/.git", repoDir), 0777))
	defer removeFile(t, repoDir)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"own/rep", repoDir}))
	assert.Nil(t, CmdRepoSetPath(cli.NewContext(nil, set, nil)))

	expectedConfigFile, disposableConfigFile := getConfigWithTwoRepos(t)
	removeFile(t, disposableConfigFile)
	profile := expectedConfigFile.Profiles["foo"]
	profile.TrackedRepos[1].LocalPath = repoDir
	expectedConfigFile.Profiles["foo"] = profile
	assertConfigFile(t, expectedConfigFile, configFileName)
}

func TestCmdRepoSetPathInvalidPath(t *testing.T) {
	_, configFileName := getConfigWithTwoRepos(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"own/rep", "/notadir"}))
	err := CmdRepoSetPath(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "Path does not exist: /notadir")
}

func TestCmdRepoSetPathNotGit(t *testing.T) {
	_, configFileName := getConfigWithTwoRepos(t)
	defer removeFile(t, configFileName)
	repoDir, err := ioutil.TempDir("", "repo")
	assert.Nil(t, err)
	defer removeFile(t, repoDir)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"own/rep", repoDir}))
	err = CmdRepoSetPath(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, fmt.Sprintf("Path is not a git repo: %s", repoDir))
}

func TestCmdRepoSetPathNoConfig(t *testing.T) {
	err := CmdRepoSetPath(cli.NewContext(nil, flag.NewFlagSet("test", 0), nil))
	assert.EqualError(t, err, "You must specify a config file")
}

func TestCmdRepoSetPathInvalidRepo(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"own/rep", "goo"}))

	err := CmdRepoSetPath(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "Not a valid Repo: own/rep")
}

func TestCmdRepoSetPathUsage(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	err := CmdRepoSetPath(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "Usage: \"prp profile repo set-path {repoName} {localPath}\"")
}

func TestCompleteRepoSetPathRepos(t *testing.T) {
	_, configFileName := getConfigWithTwoRepos(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	os.Args = []string{"repo", "ignore-build", "--completion"}
	app, writer, _ := appWithTestWriters()
	CompleteRepoSetPath(cli.NewContext(app, set, nil))
	assert.Equal(t, "foo/bar\nown/rep\n", writer.String())
}

func TestCompleteRepoSetPathFileCompletion(t *testing.T) {
	_, configFileName := getConfigWithTwoRepos(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"own/rep"}))
	os.Args = []string{"repo", "set-path", "own/rep", "--completion"}
	app, writer, _ := appWithTestWriters()
	CompleteRepoSetPath(cli.NewContext(app, set, nil))
	assert.Equal(t, "fileCompletion\n", writer.String())
}

func TestCompleteRepoSetPathDone(t *testing.T) {
	_, configFileName := getConfigWithTwoRepos(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"own/rep", "goo"}))
	os.Args = []string{"repo", "ignore-build", "own/rep", "goo", "--completion"}
	app, writer, _ := appWithTestWriters()
	CompleteRepoSetPath(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}

func TestCompleteRepoSetPathNoConfig(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	os.Args = []string{"repo", "ignore-build", "--completion"}
	app, writer, _ := appWithTestWriters()
	CompleteRepoSetPath(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}
