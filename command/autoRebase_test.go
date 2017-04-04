package command

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/guywithnose/pull-request-parser/execWrapper"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestCmdAutoRebase(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := &execWrapper.TestCommandBuilder{ExpectedCommands: getExpectedCommands(repoDir)}
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, false, false)
	defer ts.Close()
	assert.Equal(t, "", writer.String())
}

func TestCmdAutoRebaseVerbose(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := &execWrapper.TestCommandBuilder{ExpectedCommands: getExpectedCommands(repoDir)}
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, false)
	defer ts.Close()
	assert.Equal(t, getVerboseOutput(), strings.Split(writer.String(), "\n"))
}

func TestCmdAutoRebaseLocalChanges(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := &execWrapper.TestCommandBuilder{ExpectedCommands: getLocalChangesExpectedCommands(repoDir)}
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, false)
	defer ts.Close()
	assert.Equal(t, getLocalChangesVerboseOutput(), strings.Split(writer.String(), "\n"))
}

func TestCmdAutoRebaseCommandError(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := &execWrapper.TestCommandBuilder{
		ExpectedCommands: []*execWrapper.ExpectedCommand{
			execWrapper.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
			execWrapper.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", -1),
		},
	}
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, true)
	defer ts.Close()
	assert.Equal(
		t,
		append(
			getVerboseOutput()[:3],
			fmt.Sprintf("Could not rebase PR #1 in own/rep because: Unable to detect local changes in %s", repoDir),
			"",
		),
		strings.Split(writer.String(), "\n"),
	)
}

func TestCmdAutoRebasePullRequestNumber(t *testing.T) {
	ts := getAutoRebaseTestServer("")
	defer ts.Close()
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	assert.Nil(t, os.MkdirAll(fmt.Sprintf("%s/.git", repoDir), 0777))
	defer removeFile(t, repoDir)
	_, configFileName := getConfigWithAPIURLAndPath(t, ts.URL, repoDir)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	set.Int("pull-request-number", 2, "doc")
	app, writer, _ := appWithTestWriters()
	commandBuilder := &execWrapper.TestCommandBuilder{ExpectedCommands: []*execWrapper.ExpectedCommand{}}
	assert.Nil(t, CmdAutoRebase(commandBuilder)(cli.NewContext(app, set, nil)))
	assert.Equal(t, []*execWrapper.ExpectedCommand{}, commandBuilder.ExpectedCommands)
	assert.Equal(t, []error(nil), commandBuilder.Errors)
	assert.Equal(t, "", writer.String())
}

func TestCmdAutoRebaseStashError(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := &execWrapper.TestCommandBuilder{ExpectedCommands: getLocalChangesExpectedCommands(repoDir)}
	commandBuilder.ExpectedCommands[4] = execWrapper.NewExpectedCommand(repoDir, "git stash", "stash error", 1)
	commandBuilder.ExpectedCommands = commandBuilder.ExpectedCommands[:5]
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, true)
	defer ts.Close()
	assert.Equal(
		t,
		append(
			getLocalChangesVerboseOutput()[:6],
			fmt.Sprintf("Could not rebase PR #1 in own/rep because: Unable to stash changes in %s", repoDir),
			"stash error",
			"",
		),
		strings.Split(writer.String(), "\n"),
	)
}

func TestCmdAutoRebaseCurrentBranchError(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := getFailureCommands(repoDir, "git symbolic-ref HEAD", "Invalid output", 5, 12, 1)
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, true)
	defer ts.Close()
	assert.Equal(
		t,
		append(
			getLocalChangesVerboseOutput()[:7],
			"Popping the stash",
			fmt.Sprintf("Could not rebase PR #1 in own/rep because: Unable to get current branch name in %s", repoDir),
			"exit status 1",
			"Invalid output",
			"",
		),
		strings.Split(writer.String(), "\n"),
	)
}

func TestCmdAutoRebaseNoBranchCheckedOut(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := getFailureCommands(repoDir, "git symbolic-ref HEAD", "Not a branch", 5, 12, 128)
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, true)
	defer ts.Close()
	assert.Equal(
		t,
		append(
			getLocalChangesVerboseOutput()[:7],
			"Popping the stash",
			fmt.Sprintf("Could not rebase PR #1 in own/rep because: Unable to get current branch name in %s", repoDir),
			"No branch checked out in /tmp/repo",
			"Not a branch",
			"",
		),
		strings.Split(writer.String(), "\n"),
	)
}

func TestCmdAutoRebaseTempBranchError(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := getFailureCommands(repoDir, "git checkout -b prp-ref1", "checkout error", 6, 12, 1)
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, true)
	defer ts.Close()
	assert.Equal(
		t,
		append(
			getLocalChangesVerboseOutput()[:9],
			"Popping the stash",
			fmt.Sprintf("Could not rebase PR #1 in own/rep because: Unable to checkout temporary branch prp-ref1 in %s", repoDir),
			"checkout error",
			"",
		),
		strings.Split(writer.String(), "\n"),
	)
}

func TestCmdAutoRebaseTempBranchExists(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := getFailureCommands(repoDir, "git checkout -b prp-ref1", "branch exists", 6, 12, 128)
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, true)
	defer ts.Close()
	assert.Equal(
		t,
		append(
			getLocalChangesVerboseOutput()[:9],
			"Popping the stash",
			"Could not rebase PR #1 in own/rep because: Branch prp-ref1 already exists",
			"branch exists",
			"",
		),
		strings.Split(writer.String(), "\n"),
	)
}

func TestCmdAutoRebaseGoBackToOriginalBranchFailure(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := &execWrapper.TestCommandBuilder{ExpectedCommands: getLocalChangesExpectedCommands(repoDir)}
	commandBuilder.ExpectedCommands[10] = execWrapper.NewExpectedCommand(repoDir, "git checkout currentBranch", "checkout failure", 1)
	commandBuilder.ExpectedCommands = append(commandBuilder.ExpectedCommands[:11], commandBuilder.ExpectedCommands[12])
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, false)
	defer ts.Close()
	assert.Equal(
		t,
		append(
			getLocalChangesVerboseOutput()[:13],
			"",
			"checkout failure",
			"Warning: Could not go back to branch currentBranch in /tmp/repo",
			"Popping the stash",
			"",
		),
		strings.Split(writer.String(), "\n"),
	)
}

func TestCmdAutoRebaseDeleteTempBranchFailure(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := &execWrapper.TestCommandBuilder{ExpectedCommands: getLocalChangesExpectedCommands(repoDir)}
	commandBuilder.ExpectedCommands[11] = execWrapper.NewExpectedCommand(repoDir, "git branch -D prp-ref1", "delete branch failure", 1)
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, false)
	defer ts.Close()
	assert.Equal(
		t,
		append(
			getLocalChangesVerboseOutput()[:14],
			"",
			"delete branch failure",
			"Warning: Could not delete temporary branch prp-ref1 in /tmp/repo",
			"Popping the stash",
			"",
		),
		strings.Split(writer.String(), "\n"),
	)
}

func TestCmdAutoRebaseStashPopFailure(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := &execWrapper.TestCommandBuilder{ExpectedCommands: getLocalChangesExpectedCommands(repoDir)}
	commandBuilder.ExpectedCommands[12] = execWrapper.NewExpectedCommand(repoDir, "git stash pop", "stash pop failure", 1)
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, false)
	defer ts.Close()
	assert.Equal(
		t,
		append(
			getLocalChangesVerboseOutput()[:15],
			"",
			"stash pop failure",
			"Warning: Could not pop stash in /tmp/repo",
			"",
		),
		strings.Split(writer.String(), "\n"),
	)
}

func TestCmdAutoRebaseResetFailure(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := getFailureCommands(repoDir, "git reset --hard origin/ref1", "reset failure", 7, 10, 1)
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, true)
	defer ts.Close()
	assert.Equal(
		t,
		append(
			getLocalChangesVerboseOutput()[:10],
			"Going back to branch currentBranch",
			"Deleting temporary branch prp-ref1",
			"Popping the stash",
			"Could not rebase PR #1 in own/rep because: Unable to reset the code to origin/ref1",
			"reset failure",
			"",
		),
		strings.Split(writer.String(), "\n"),
	)
}

func TestCmdAutoRebaseRebaseFailure(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := getFailureCommands(repoDir, "git rebase upstream/baseRef1", "rebase failure", 8, 10, 1)
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, true)
	defer ts.Close()
	assert.Equal(
		t,
		append(
			getLocalChangesVerboseOutput()[:11],
			"Going back to branch currentBranch",
			"Deleting temporary branch prp-ref1",
			"Popping the stash",
			"Could not rebase PR #1 in own/rep because: Unable to rebase against upstream/baseRef1, there may be a conflict",
			"rebase failure",
			"",
		),
		strings.Split(writer.String(), "\n"),
	)
}

func TestCmdAutoRebasePushFailure(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := &execWrapper.TestCommandBuilder{ExpectedCommands: getLocalChangesExpectedCommands(repoDir)}
	commandBuilder.ExpectedCommands[9] = execWrapper.NewExpectedCommand(repoDir, "git push origin prp-ref1:ref1 --force", "push failure", 1)
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, true)
	defer ts.Close()
	assert.Equal(
		t,
		append(
			getLocalChangesVerboseOutput()[:15],
			"Could not rebase PR #1 in own/rep because: Unable to push to origin/ref1",
			"push failure",
			"",
		),
		strings.Split(writer.String(), "\n"),
	)
}

func TestCmdAutoRebaseOwnedRemoteFetchFailure(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := getFailureCommands(repoDir, "git fetch origin", "fetch failure", 2, 13, 1)
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, true)
	defer ts.Close()
	assert.Equal(
		t,
		append(
			getLocalChangesVerboseOutput()[:4],
			"Could not rebase PR #1 in own/rep because: Unable to fetch code from origin",
			"fetch failure",
			"",
		),
		strings.Split(writer.String(), "\n"),
	)
}

func TestCmdAutoRebaseUpstreamRemoteFetchFailure(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := getFailureCommands(repoDir, "git fetch upstream", "fetch upstream failure", 3, 13, 1)
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, true)
	defer ts.Close()
	assert.Equal(
		t,
		append(
			getLocalChangesVerboseOutput()[:5],
			"Could not rebase PR #1 in own/rep because: Unable to fetch code from upstream",
			"fetch upstream failure",
			"",
		),
		strings.Split(writer.String(), "\n"),
	)
}

func getFailureCommands(repoDir, command, stdErr string, index, end, exitStatus int) *execWrapper.TestCommandBuilder {
	commandBuilder := &execWrapper.TestCommandBuilder{ExpectedCommands: getLocalChangesExpectedCommands(repoDir)}
	commandBuilder.ExpectedCommands[index] = execWrapper.NewExpectedCommand(repoDir, command, stdErr, exitStatus)
	commandBuilder.ExpectedCommands = append(commandBuilder.ExpectedCommands[:index+1], commandBuilder.ExpectedCommands[end:]...)
	return commandBuilder
}

func TestCmdAutoRebaseAnalyzeRemotesOwnedNotFound(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := &execWrapper.TestCommandBuilder{
		ExpectedCommands: []*execWrapper.ExpectedCommand{
			execWrapper.NewExpectedCommand(repoDir, "git remote -v", "upstream\tbaseLabel1SSHURL (fetch)", 0),
		},
	}
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, true)
	defer ts.Close()
	assert.Equal(
		t,
		append(
			getLocalChangesVerboseOutput()[:2],
			"Could not rebase PR #1 in own/rep because: No remote exists in /tmp/repo that points to labelSSHURL",
			"",
		),
		strings.Split(writer.String(), "\n"),
	)
}

func TestCmdAutoRebaseAnalyzeRemotesUpstreamNotFound(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := &execWrapper.TestCommandBuilder{
		ExpectedCommands: []*execWrapper.ExpectedCommand{
			execWrapper.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)", 0),
		},
	}
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, true)
	defer ts.Close()
	assert.Equal(
		t,
		append(
			getLocalChangesVerboseOutput()[:2],
			"Could not rebase PR #1 in own/rep because: No remote exists in /tmp/repo that points to baseLabel1SSHURL",
			"",
		),
		strings.Split(writer.String(), "\n"),
	)
}

func TestCmdAutoRebaseAnalyzeRemotesFailure(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	defer removeFile(t, repoDir)
	commandBuilder := &execWrapper.TestCommandBuilder{
		ExpectedCommands: []*execWrapper.ExpectedCommand{
			execWrapper.NewExpectedCommand(repoDir, "git remote -v", "failure analyzing remotes", 1),
		},
	}
	ts, writer := runBaseCommand(t, "", repoDir, commandBuilder, true, true)
	defer ts.Close()
	assert.Equal(
		t,
		append(
			getLocalChangesVerboseOutput()[:2],
			"Could not rebase PR #1 in own/rep because: Unable to analyze remotes in /tmp/repo",
			"failure analyzing remotes",
			"",
		),
		strings.Split(writer.String(), "\n"),
	)
}

func TestCmdAutoRebaseBadAPIURL(t *testing.T) {
	_, configFileName := getConfigWithAPIURL(t, "%s/mockApi")
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app := cli.NewApp()
	commandBuilder := &execWrapper.TestCommandBuilder{}
	err := CmdAutoRebase(commandBuilder)(cli.NewContext(app, set, nil))
	assert.EqualError(t, err, "parse %s/mockApi: invalid URL escape \"%s/\"")
}

func TestCmdAutoRebaseUserFailure(t *testing.T) {
	ts := getAutoRebaseTestServer("/user")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app := cli.NewApp()
	commandBuilder := &execWrapper.TestCommandBuilder{}
	err := CmdAutoRebase(commandBuilder)(cli.NewContext(app, set, nil))
	assert.EqualError(t, err, fmt.Sprintf("GET %s/user: 500  []", ts.URL))
}

func TestCmdAutoRebaseUsage(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"foo"}))
	app := cli.NewApp()
	commandBuilder := &execWrapper.TestCommandBuilder{}
	err := CmdAutoRebase(commandBuilder)(cli.NewContext(app, set, nil))
	assert.EqualError(t, err, "Usage: \"prp auto-rebase\"")
}

func TestCmdAutoRebaseNoConfig(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	app := cli.NewApp()
	commandBuilder := &execWrapper.TestCommandBuilder{}
	err := CmdAutoRebase(commandBuilder)(cli.NewContext(app, set, nil))
	assert.EqualError(t, err, "You must specify a config file")
}

func TestCmdAutoRebaseNoPath(t *testing.T) {
	ts := getAutoRebaseTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, _, writer := appWithTestWriters()
	commandBuilder := &execWrapper.TestCommandBuilder{}
	err := CmdAutoRebase(commandBuilder)(cli.NewContext(app, set, nil))
	assert.EqualError(t, err, "Unable to rebase all pull requests")
	assert.Equal(t, "Could not rebase PR #1 in own/rep because: Path was not set for this repo\n", writer.String())
}

func TestCmdAutoRebaseInvalidPath(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	ts := getAutoRebaseTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURLAndPath(t, ts.URL, repoDir)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, _, writer := appWithTestWriters()
	commandBuilder := &execWrapper.TestCommandBuilder{}
	err := CmdAutoRebase(commandBuilder)(cli.NewContext(app, set, nil))
	assert.EqualError(t, err, "Unable to rebase all pull requests")
	assert.Equal(t, "Could not rebase PR #1 in own/rep because: Path does not exist: /tmp/repo\n", writer.String())
}

func TestCmdAutoRebaseNonGitPath(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	assert.Nil(t, os.MkdirAll(repoDir, 0777))
	defer removeFile(t, repoDir)
	ts := getAutoRebaseTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURLAndPath(t, ts.URL, repoDir)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, _, writer := appWithTestWriters()
	commandBuilder := &execWrapper.TestCommandBuilder{}
	err := CmdAutoRebase(commandBuilder)(cli.NewContext(app, set, nil))
	assert.EqualError(t, err, "Unable to rebase all pull requests")
	assert.Equal(t, "Could not rebase PR #1 in own/rep because: Path is not a git repo: /tmp/repo\n", writer.String())
}

func TestCompleteAutoRebaseFlags(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, writer, _ := appWithTestWriters()
	app.Commands = []cli.Command{
		{
			Name: "auto-rebase",
			Flags: []cli.Flag{
				cli.StringSliceFlag{Name: "repo, r"},
				cli.IntFlag{Name: "pull-request-number, prNum, n"},
				cli.BoolFlag{Name: "verbose, v"},
			},
		},
	}
	os.Args = []string{"auto-rebase", "--completion"}
	CompleteAutoRebase(cli.NewContext(app, set, nil))
	assert.Equal(t, "--repo\n--pull-request-number\n--verbose\n", writer.String())
}

func TestCompleteAutoRebaseRepo(t *testing.T) {
	ts := getAutoRebaseTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, writer, _ := appWithTestWriters()
	os.Args = []string{"auto-rebase", "--repo", "--completion"}
	CompleteAutoRebase(cli.NewContext(app, set, nil))
	assert.Equal(t, "own/rep\n", writer.String())
}

func TestCompleteAutoRebaseRepoMulti(t *testing.T) {
	ts := getAutoRebaseTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	repoFlag := cli.StringSlice{"own/rep"}
	set.Var(&repoFlag, "repo", "doc")
	app, _, writer := appWithTestWriters()
	os.Args = []string{"auto-rebase", "--repo", "own/rep", "--repo", "--completion"}
	CompleteAutoRebase(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}

func TestCompleteAutoRebasePullRequestNumber(t *testing.T) {
	ts := getAutoRebaseTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, writer, _ := appWithTestWriters()
	os.Args = []string{"auto-rebase", "--pull-request-number", "--completion"}
	CompleteAutoRebase(cli.NewContext(app, set, nil))
	assert.Equal(t, "1\n", writer.String())
}

func TestCompleteAutoRebaseNoConfig(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	app, writer, _ := appWithTestWriters()
	os.Args = []string{"auto-rebase", "--pull-request-number", "--completion"}
	CompleteAutoRebase(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}

func TestCompleteAutoRebaseBadAPIURL(t *testing.T) {
	_, configFileName := getConfigWithAPIURL(t, "%s/mockApi")
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, writer, _ := appWithTestWriters()
	os.Args = []string{"auto-rebase", "--pull-request-number", "--completion"}
	CompleteAutoRebase(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}

func getExpectedCommands(repoDir string) []*execWrapper.ExpectedCommand {
	return []*execWrapper.ExpectedCommand{
		execWrapper.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
		execWrapper.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 0),
		execWrapper.NewExpectedCommand(repoDir, "git fetch origin", "", 0),
		execWrapper.NewExpectedCommand(repoDir, "git fetch upstream", "", 0),
		execWrapper.NewExpectedCommand(repoDir, "git symbolic-ref HEAD", "currentBranch", 0),
		execWrapper.NewExpectedCommand(repoDir, "git checkout -b prp-ref1", "", 0),
		execWrapper.NewExpectedCommand(repoDir, "git reset --hard origin/ref1", "", 0),
		execWrapper.NewExpectedCommand(repoDir, "git rebase upstream/baseRef1", "", 0),
		execWrapper.NewExpectedCommand(repoDir, "git push origin prp-ref1:ref1 --force", "", 0),
		execWrapper.NewExpectedCommand(repoDir, "git checkout currentBranch", "", 0),
		execWrapper.NewExpectedCommand(repoDir, "git branch -D prp-ref1", "", 0),
	}
}

func getLocalChangesExpectedCommands(repoDir string) []*execWrapper.ExpectedCommand {
	return []*execWrapper.ExpectedCommand{
		execWrapper.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
		execWrapper.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 1),
		execWrapper.NewExpectedCommand(repoDir, "git fetch origin", "", 0),
		execWrapper.NewExpectedCommand(repoDir, "git fetch upstream", "", 0),
		execWrapper.NewExpectedCommand(repoDir, "git stash", "", 0),
		execWrapper.NewExpectedCommand(repoDir, "git symbolic-ref HEAD", "currentBranch", 0),
		execWrapper.NewExpectedCommand(repoDir, "git checkout -b prp-ref1", "", 0),
		execWrapper.NewExpectedCommand(repoDir, "git reset --hard origin/ref1", "", 0),
		execWrapper.NewExpectedCommand(repoDir, "git rebase upstream/baseRef1", "", 0),
		execWrapper.NewExpectedCommand(repoDir, "git push origin prp-ref1:ref1 --force", "", 0),
		execWrapper.NewExpectedCommand(repoDir, "git checkout currentBranch", "", 0),
		execWrapper.NewExpectedCommand(repoDir, "git branch -D prp-ref1", "", 0),
		execWrapper.NewExpectedCommand(repoDir, "git stash pop", "", 0),
	}
}

func getAutoRebaseTestServer(failureURL string) *httptest.Server {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == failureURL {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response := handleUserRequest(r, "guy")
		if response != nil {
			fmt.Fprint(w, *response)
			return
		}

		response = handlePullRequestRequests(r, w, server)
		if response != nil {
			fmt.Fprint(w, *response)
			return
		}

		response = handleCommitsComparisonRequests(r)
		if response != nil {
			fmt.Fprint(w, *response)
			return
		}

		panic(r.URL.String())
	}))

	return server
}

func getVerboseOutput() []string {
	return []string{
		"Requesting repo data from config",
		"Analyzing remotes",
		"Checking for local changes",
		"Fetching from remote: origin",
		"Fetching from remote: upstream",
		"Saving current branch name",
		"Current branch name is currentBranch",
		"Checking out temporary branch: prp-ref1",
		"Resetting code to origin/ref1",
		"Rebasing against upstream/baseRef1",
		"Pushing to origin/ref1",
		"Going back to branch currentBranch",
		"Deleting temporary branch prp-ref1",
		"",
	}
}

func getLocalChangesVerboseOutput() []string {
	return []string{
		"Requesting repo data from config",
		"Analyzing remotes",
		"Checking for local changes",
		"Fetching from remote: origin",
		"Fetching from remote: upstream",
		"Local changes found... stashing",
		"Saving current branch name",
		"Current branch name is currentBranch",
		"Checking out temporary branch: prp-ref1",
		"Resetting code to origin/ref1",
		"Rebasing against upstream/baseRef1",
		"Pushing to origin/ref1",
		"Going back to branch currentBranch",
		"Deleting temporary branch prp-ref1",
		"Popping the stash",
		"",
	}
}

func runBaseCommand(
	t *testing.T,
	failureURL,
	repoDir string,
	commandBuilder *execWrapper.TestCommandBuilder,
	verbose,
	expectedError bool,
) (*httptest.Server, *bytes.Buffer) {
	ts := getAutoRebaseTestServer(failureURL)
	assert.Nil(t, os.MkdirAll(fmt.Sprintf("%s/.git", repoDir), 0777))
	_, configFileName := getConfigWithAPIURLAndPath(t, ts.URL, repoDir)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	set.Bool("verbose", verbose, "doc")
	app, _, writer := appWithTestWriters()
	err := CmdAutoRebase(commandBuilder)(cli.NewContext(app, set, nil))
	if expectedError {
		assert.EqualError(t, err, "Unable to rebase all pull requests")
	} else {
		assert.Nil(t, err)
	}

	assert.Equal(t, []*execWrapper.ExpectedCommand{}, commandBuilder.ExpectedCommands)
	assert.Equal(t, []error(nil), commandBuilder.Errors)
	return ts, writer
}
