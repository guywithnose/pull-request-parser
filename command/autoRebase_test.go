package command_test

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/guywithnose/pull-request-parser/command"
	"github.com/guywithnose/runner"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestCmdAutoRebase(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	var testCases = []struct {
		name             string
		expectedCommands []*runner.ExpectedCommand
		output           []string
		verbose          bool
		expectedError    bool
	}{
		{
			"Normal",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
				runner.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 0),
				runner.NewExpectedCommand(repoDir, "git fetch origin", "", 0),
				runner.NewExpectedCommand(repoDir, "git fetch upstream", "", 0),
				runner.NewExpectedCommand(repoDir, "git symbolic-ref HEAD", "currentBranch", 0),
				runner.NewExpectedCommand(repoDir, "git checkout -b prp-ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git reset --hard origin/ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git rebase upstream/baseRef1", "", 0),
				runner.NewExpectedCommand(repoDir, "git push origin prp-ref1:ref1 --force", "", 0),
				runner.NewExpectedCommand(repoDir, "git checkout currentBranch", "", 0),
				runner.NewExpectedCommand(repoDir, "git branch -D prp-ref1", "", 0),
			},
			[]string{""},
			false,
			false,
		},
		{
			"Verbose",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
				runner.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 0),
				runner.NewExpectedCommand(repoDir, "git fetch origin", "", 0),
				runner.NewExpectedCommand(repoDir, "git fetch upstream", "", 0),
				runner.NewExpectedCommand(repoDir, "git symbolic-ref HEAD", "currentBranch", 0),
				runner.NewExpectedCommand(repoDir, "git checkout -b prp-ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git reset --hard origin/ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git rebase upstream/baseRef1", "", 0),
				runner.NewExpectedCommand(repoDir, "git push origin prp-ref1:ref1 --force", "", 0),
				runner.NewExpectedCommand(repoDir, "git checkout currentBranch", "", 0),
				runner.NewExpectedCommand(repoDir, "git branch -D prp-ref1", "", 0),
			},
			[]string{
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
			},
			true,
			false,
		},
		{
			"LocalChanges",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
				runner.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 1),
				runner.NewExpectedCommand(repoDir, "git fetch origin", "", 0),
				runner.NewExpectedCommand(repoDir, "git fetch upstream", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash", "", 0),
				runner.NewExpectedCommand(repoDir, "git symbolic-ref HEAD", "currentBranch", 0),
				runner.NewExpectedCommand(repoDir, "git checkout -b prp-ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git reset --hard origin/ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git rebase upstream/baseRef1", "", 0),
				runner.NewExpectedCommand(repoDir, "git push origin prp-ref1:ref1 --force", "", 0),
				runner.NewExpectedCommand(repoDir, "git checkout currentBranch", "", 0),
				runner.NewExpectedCommand(repoDir, "git branch -D prp-ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash pop", "", 0),
			},
			[]string{
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
			},
			true,
			false,
		},
		{
			"FailureDetectingChanges",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
				runner.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", -1),
			},
			[]string{
				"Requesting repo data from config",
				"Analyzing remotes",
				"Checking for local changes",
				fmt.Sprintf("Could not rebase PR #1 in own/rep because: Unable to detect local changes in %s", repoDir),
				"",
			},
			true,
			true,
		},
		{
			"AnalyzeOwnedRemoteNotFound",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "upstream\tbaseLabel1SSHURL (fetch)", 0),
			},
			[]string{
				"Requesting repo data from config",
				"Analyzing remotes",
				"Could not rebase PR #1 in own/rep because: No remote exists in /tmp/repo that points to labelSSHURL",
				"",
			},
			true,
			true,
		},
		{
			"AnalyzeUpstreamRemoteNotFound",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)", 0),
			},
			[]string{
				"Requesting repo data from config",
				"Analyzing remotes",
				"Could not rebase PR #1 in own/rep because: No remote exists in /tmp/repo that points to baseLabel1SSHURL",
				"",
			},
			true,
			true,
		},
		{
			"AnalyzeRemotesFailure",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "failure analyzing remotes", 1),
			},
			[]string{
				"Requesting repo data from config",
				"Analyzing remotes",
				"Could not rebase PR #1 in own/rep because: Unable to analyze remotes in /tmp/repo",
				"failure analyzing remotes",
				"",
			},
			true,
			true,
		},
		{
			"RestoreOriginalBranchFailure",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
				runner.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 1),
				runner.NewExpectedCommand(repoDir, "git fetch origin", "", 0),
				runner.NewExpectedCommand(repoDir, "git fetch upstream", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash", "", 0),
				runner.NewExpectedCommand(repoDir, "git symbolic-ref HEAD", "currentBranch", 0),
				runner.NewExpectedCommand(repoDir, "git checkout -b prp-ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git reset --hard origin/ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git rebase upstream/baseRef1", "", 0),
				runner.NewExpectedCommand(repoDir, "git push origin prp-ref1:ref1 --force", "", 0),
				runner.NewExpectedCommand(repoDir, "git checkout currentBranch", "checkout failure", 1),
				runner.NewExpectedCommand(repoDir, "git stash pop", "", 0),
			},
			[]string{
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
				"",
				"checkout failure",
				"Warning: Could not go back to branch currentBranch in /tmp/repo",
				"Popping the stash",
				"",
			},
			true,
			false,
		},
		{
			"DeleteTempBranchFailure",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
				runner.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 1),
				runner.NewExpectedCommand(repoDir, "git fetch origin", "", 0),
				runner.NewExpectedCommand(repoDir, "git fetch upstream", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash", "", 0),
				runner.NewExpectedCommand(repoDir, "git symbolic-ref HEAD", "currentBranch", 0),
				runner.NewExpectedCommand(repoDir, "git checkout -b prp-ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git reset --hard origin/ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git rebase upstream/baseRef1", "", 0),
				runner.NewExpectedCommand(repoDir, "git push origin prp-ref1:ref1 --force", "", 0),
				runner.NewExpectedCommand(repoDir, "git checkout currentBranch", "", 0),
				runner.NewExpectedCommand(repoDir, "git branch -D prp-ref1", "delete branch failure", 1),
				runner.NewExpectedCommand(repoDir, "git stash pop", "", 0),
			},
			[]string{
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
				"",
				"delete branch failure",
				"Warning: Could not delete temporary branch prp-ref1 in /tmp/repo",
				"Popping the stash",
				"",
			},
			true,
			false,
		},
		{
			"StashPopFailure",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
				runner.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 1),
				runner.NewExpectedCommand(repoDir, "git fetch origin", "", 0),
				runner.NewExpectedCommand(repoDir, "git fetch upstream", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash", "", 0),
				runner.NewExpectedCommand(repoDir, "git symbolic-ref HEAD", "currentBranch", 0),
				runner.NewExpectedCommand(repoDir, "git checkout -b prp-ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git reset --hard origin/ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git rebase upstream/baseRef1", "", 0),
				runner.NewExpectedCommand(repoDir, "git push origin prp-ref1:ref1 --force", "", 0),
				runner.NewExpectedCommand(repoDir, "git checkout currentBranch", "", 0),
				runner.NewExpectedCommand(repoDir, "git branch -D prp-ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash pop", "stash pop failure", 1),
			},
			[]string{
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
				"stash pop failure",
				"Warning: Could not pop stash in /tmp/repo",
				"",
			},
			true,
			false,
		},
		{
			"PushFailure",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
				runner.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 1),
				runner.NewExpectedCommand(repoDir, "git fetch origin", "", 0),
				runner.NewExpectedCommand(repoDir, "git fetch upstream", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash", "", 0),
				runner.NewExpectedCommand(repoDir, "git symbolic-ref HEAD", "currentBranch", 0),
				runner.NewExpectedCommand(repoDir, "git checkout -b prp-ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git reset --hard origin/ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git rebase upstream/baseRef1", "", 0),
				runner.NewExpectedCommand(repoDir, "git push origin prp-ref1:ref1 --force", "push failure", 1),
				runner.NewExpectedCommand(repoDir, "git checkout currentBranch", "", 0),
				runner.NewExpectedCommand(repoDir, "git branch -D prp-ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash pop", "", 0),
			},
			[]string{
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
				"Could not rebase PR #1 in own/rep because: Unable to push to origin/ref1",
				"push failure",
				"",
			},
			true,
			true,
		},
		{
			"StashFailure",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
				runner.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 1),
				runner.NewExpectedCommand(repoDir, "git fetch origin", "", 0),
				runner.NewExpectedCommand(repoDir, "git fetch upstream", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash", "stash error", 1),
			},
			[]string{
				"Requesting repo data from config",
				"Analyzing remotes",
				"Checking for local changes",
				"Fetching from remote: origin",
				"Fetching from remote: upstream",
				"Local changes found... stashing",
				fmt.Sprintf("Could not rebase PR #1 in own/rep because: Unable to stash changes in %s", repoDir),
				"stash error",
				"",
			},
			true,
			true,
		},
		{
			"GetCurrentBranchFailure",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
				runner.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 1),
				runner.NewExpectedCommand(repoDir, "git fetch origin", "", 0),
				runner.NewExpectedCommand(repoDir, "git fetch upstream", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash", "", 0),
				runner.NewExpectedCommand(repoDir, "git symbolic-ref HEAD", "Invalid output", 1),
				runner.NewExpectedCommand(repoDir, "git stash pop", "", 0),
			},
			[]string{
				"Requesting repo data from config",
				"Analyzing remotes",
				"Checking for local changes",
				"Fetching from remote: origin",
				"Fetching from remote: upstream",
				"Local changes found... stashing",
				"Saving current branch name",
				"Popping the stash",
				fmt.Sprintf("Could not rebase PR #1 in own/rep because: Unable to get current branch name in %s", repoDir),
				"exit status 1",
				"Invalid output",
				"",
			},
			true,
			true,
		},
		{
			"NoBranchCheckedOut",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
				runner.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 1),
				runner.NewExpectedCommand(repoDir, "git fetch origin", "", 0),
				runner.NewExpectedCommand(repoDir, "git fetch upstream", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash", "", 0),
				runner.NewExpectedCommand(repoDir, "git symbolic-ref HEAD", "Not a branch", 128),
				runner.NewExpectedCommand(repoDir, "git stash pop", "", 0),
			},
			[]string{
				"Requesting repo data from config",
				"Analyzing remotes",
				"Checking for local changes",
				"Fetching from remote: origin",
				"Fetching from remote: upstream",
				"Local changes found... stashing",
				"Saving current branch name",
				"Popping the stash",
				fmt.Sprintf("Could not rebase PR #1 in own/rep because: Unable to get current branch name in %s", repoDir),
				"No branch checked out in /tmp/repo",
				"Not a branch",
				"",
			},
			true,
			true,
		},
		{
			"TempBranchFailure",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
				runner.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 1),
				runner.NewExpectedCommand(repoDir, "git fetch origin", "", 0),
				runner.NewExpectedCommand(repoDir, "git fetch upstream", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash", "", 0),
				runner.NewExpectedCommand(repoDir, "git symbolic-ref HEAD", "currentBranch", 0),
				runner.NewExpectedCommand(repoDir, "git checkout -b prp-ref1", "checkout error", 1),
				runner.NewExpectedCommand(repoDir, "git stash pop", "", 0),
			},
			[]string{
				"Requesting repo data from config",
				"Analyzing remotes",
				"Checking for local changes",
				"Fetching from remote: origin",
				"Fetching from remote: upstream",
				"Local changes found... stashing",
				"Saving current branch name",
				"Current branch name is currentBranch",
				"Checking out temporary branch: prp-ref1",
				"Popping the stash",
				fmt.Sprintf("Could not rebase PR #1 in own/rep because: Unable to checkout temporary branch prp-ref1 in %s", repoDir),
				"checkout error",
				"",
			},
			true,
			true,
		},
		{
			"TempBranchExists",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
				runner.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 1),
				runner.NewExpectedCommand(repoDir, "git fetch origin", "", 0),
				runner.NewExpectedCommand(repoDir, "git fetch upstream", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash", "", 0),
				runner.NewExpectedCommand(repoDir, "git symbolic-ref HEAD", "currentBranch", 0),
				runner.NewExpectedCommand(repoDir, "git checkout -b prp-ref1", "branch exists", 128),
				runner.NewExpectedCommand(repoDir, "git stash pop", "", 0),
			},
			[]string{
				"Requesting repo data from config",
				"Analyzing remotes",
				"Checking for local changes",
				"Fetching from remote: origin",
				"Fetching from remote: upstream",
				"Local changes found... stashing",
				"Saving current branch name",
				"Current branch name is currentBranch",
				"Checking out temporary branch: prp-ref1",
				"Popping the stash",
				"Could not rebase PR #1 in own/rep because: Branch prp-ref1 already exists",
				"branch exists",
				"",
			},
			true,
			true,
		},
		{
			"ResetFailure",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
				runner.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 1),
				runner.NewExpectedCommand(repoDir, "git fetch origin", "", 0),
				runner.NewExpectedCommand(repoDir, "git fetch upstream", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash", "", 0),
				runner.NewExpectedCommand(repoDir, "git symbolic-ref HEAD", "currentBranch", 0),
				runner.NewExpectedCommand(repoDir, "git checkout -b prp-ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git reset --hard origin/ref1", "reset failure", 1),
				runner.NewExpectedCommand(repoDir, "git checkout currentBranch", "", 0),
				runner.NewExpectedCommand(repoDir, "git branch -D prp-ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash pop", "", 0),
			},
			[]string{
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
				"Going back to branch currentBranch",
				"Deleting temporary branch prp-ref1",
				"Popping the stash",
				"Could not rebase PR #1 in own/rep because: Unable to reset the code to origin/ref1",
				"reset failure",
				"",
			},
			true,
			true,
		},
		{
			"RebaseFailure",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
				runner.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 1),
				runner.NewExpectedCommand(repoDir, "git fetch origin", "", 0),
				runner.NewExpectedCommand(repoDir, "git fetch upstream", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash", "", 0),
				runner.NewExpectedCommand(repoDir, "git symbolic-ref HEAD", "currentBranch", 0),
				runner.NewExpectedCommand(repoDir, "git checkout -b prp-ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git reset --hard origin/ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git rebase upstream/baseRef1", "rebase failure", 1),
				runner.NewExpectedCommand(repoDir, "git rebase --abort", "", 0),
				runner.NewExpectedCommand(repoDir, "git checkout currentBranch", "", 0),
				runner.NewExpectedCommand(repoDir, "git branch -D prp-ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash pop", "", 0),
			},
			[]string{
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
				"Going back to branch currentBranch",
				"Deleting temporary branch prp-ref1",
				"Popping the stash",
				"Could not rebase PR #1 in own/rep because: Unable to rebase against upstream/baseRef1, there may be a conflict",
				"rebase failure",
				"",
			},
			true,
			true,
		},
		{
			"RebaseFailureAbortFailure",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
				runner.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 1),
				runner.NewExpectedCommand(repoDir, "git fetch origin", "", 0),
				runner.NewExpectedCommand(repoDir, "git fetch upstream", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash", "", 0),
				runner.NewExpectedCommand(repoDir, "git symbolic-ref HEAD", "currentBranch", 0),
				runner.NewExpectedCommand(repoDir, "git checkout -b prp-ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git reset --hard origin/ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git rebase upstream/baseRef1", "rebase failure", 1),
				runner.NewExpectedCommand(repoDir, "git rebase --abort", "", 1),
				runner.NewExpectedCommand(repoDir, "git checkout currentBranch", "", 0),
				runner.NewExpectedCommand(repoDir, "git branch -D prp-ref1", "", 0),
				runner.NewExpectedCommand(repoDir, "git stash pop", "", 0),
			},
			[]string{
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
				"Could not abort rebase PR #1 in own/rep because: exit status 1",
				"Going back to branch currentBranch",
				"Deleting temporary branch prp-ref1",
				"Popping the stash",
				"Could not rebase PR #1 in own/rep because: Unable to rebase against upstream/baseRef1, there may be a conflict",
				"rebase failure",
				"",
			},
			true,
			true,
		},
		{
			"OwnedRemoteFetchFailure",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
				runner.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 1),
				runner.NewExpectedCommand(repoDir, "git fetch origin", "fetch failure", 1),
			},
			[]string{
				"Requesting repo data from config",
				"Analyzing remotes",
				"Checking for local changes",
				"Fetching from remote: origin",
				"Could not rebase PR #1 in own/rep because: Unable to fetch code from origin",
				"fetch failure",
				"",
			},
			true,
			true,
		},
		{
			"UpstreamRemoteFetchFailure",
			[]*runner.ExpectedCommand{
				runner.NewExpectedCommand(repoDir, "git remote -v", "origin\tlabelSSHURL (push)\nupstream\tbaseLabel1SSHURL (fetch)", 0),
				runner.NewExpectedCommand(repoDir, "git diff-index --quiet HEAD", "", 1),
				runner.NewExpectedCommand(repoDir, "git fetch origin", "", 0),
				runner.NewExpectedCommand(repoDir, "git fetch upstream", "fetch failure", 1),
			},
			[]string{
				"Requesting repo data from config",
				"Analyzing remotes",
				"Checking for local changes",
				"Fetching from remote: origin",
				"Fetching from remote: upstream",
				"Could not rebase PR #1 in own/rep because: Unable to fetch code from upstream",
				"fetch failure",
				"",
			},
			true,
			true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runner := &runner.Test{ExpectedCommands: tc.expectedCommands}
			writer := runBaseCommand(t, repoDir, runner, tc.verbose, tc.expectedError)
			assert.Equal(t, tc.output, strings.Split(writer.String(), "\n"))
			removeFile(t, repoDir)
		})
	}
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
	cb := &runner.Test{ExpectedCommands: []*runner.ExpectedCommand{}}
	assert.Nil(t, command.CmdAutoRebase(cb)(cli.NewContext(app, set, nil)))
	assert.Equal(t, []*runner.ExpectedCommand{}, cb.ExpectedCommands)
	assert.Equal(t, []error(nil), cb.Errors)
	assert.Equal(t, "", writer.String())
}

func TestCmdAutoRebaseBadAPIURL(t *testing.T) {
	_, configFileName := getConfigWithAPIURL(t, "%s/mockApi")
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app := cli.NewApp()
	cb := &runner.Test{}
	err := command.CmdAutoRebase(cb)(cli.NewContext(app, set, nil))
	assert.EqualError(t, err, "parse %s/mockApi: invalid URL escape \"%s/\"")
}

func TestCmdAutoRebaseUserFailure(t *testing.T) {
	ts := getAutoRebaseTestServer("/user")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app := cli.NewApp()
	cb := &runner.Test{}
	err := command.CmdAutoRebase(cb)(cli.NewContext(app, set, nil))
	assert.EqualError(t, err, fmt.Sprintf("GET %s/user: 500  []", ts.URL))
}

func TestCmdAutoRebaseUsage(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"foo"}))
	app := cli.NewApp()
	cb := &runner.Test{}
	err := command.CmdAutoRebase(cb)(cli.NewContext(app, set, nil))
	assert.EqualError(t, err, "Usage: \"prp auto-rebase\"")
}

func TestCmdAutoRebaseNoConfig(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	app := cli.NewApp()
	cb := &runner.Test{}
	err := command.CmdAutoRebase(cb)(cli.NewContext(app, set, nil))
	assert.EqualError(t, err, "You must specify a config file")
}

func TestCmdAutoRebaseNoPath(t *testing.T) {
	ts := getAutoRebaseTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, _, writer := appWithTestWriters()
	cb := &runner.Test{}
	err := command.CmdAutoRebase(cb)(cli.NewContext(app, set, nil))
	assert.EqualError(t, err, "Unable to rebase all pull requests")
	assert.Equal(t, "Could not rebase PR #1 in own/rep because: Path was not set for repo: own/rep\n", writer.String())
}

func TestCmdAutoRebaseInvalidPath(t *testing.T) {
	repoDir := fmt.Sprintf("%s/repo", os.TempDir())
	ts := getAutoRebaseTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURLAndPath(t, ts.URL, repoDir)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, _, writer := appWithTestWriters()
	cb := &runner.Test{}
	err := command.CmdAutoRebase(cb)(cli.NewContext(app, set, nil))
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
	cb := &runner.Test{}
	err := command.CmdAutoRebase(cb)(cli.NewContext(app, set, nil))
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
	command.CompleteAutoRebase(cli.NewContext(app, set, nil))
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
	command.CompleteAutoRebase(cli.NewContext(app, set, nil))
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
	command.CompleteAutoRebase(cli.NewContext(app, set, nil))
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
	command.CompleteAutoRebase(cli.NewContext(app, set, nil))
	assert.Equal(t, "1\n", writer.String())
}

func TestCompleteAutoRebaseNoConfig(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	app, writer, _ := appWithTestWriters()
	os.Args = []string{"auto-rebase", "--pull-request-number", "--completion"}
	command.CompleteAutoRebase(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}

func TestCompleteAutoRebaseBadAPIURL(t *testing.T) {
	_, configFileName := getConfigWithAPIURL(t, "%s/mockApi")
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, writer, _ := appWithTestWriters()
	os.Args = []string{"auto-rebase", "--pull-request-number", "--completion"}
	command.CompleteAutoRebase(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
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

		response = handleCommitsComparisonRequests(r, w, server)
		if response != nil {
			fmt.Fprint(w, *response)
			return
		}

		panic(r.URL.String())
	}))

	return server
}

func runBaseCommand(t *testing.T, repoDir string, cb *runner.Test, verbose, expectedError bool) *bytes.Buffer {
	t.Helper()
	assert.Nil(t, os.MkdirAll(fmt.Sprintf("%s/.git", repoDir), 0777))
	ts := getAutoRebaseTestServer("")
	_, configFileName := getConfigWithAPIURLAndPath(t, ts.URL, repoDir)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	set.Bool("verbose", verbose, "doc")
	app, _, writer := appWithTestWriters()
	err := command.CmdAutoRebase(cb)(cli.NewContext(app, set, nil))
	ts.Close()
	if expectedError {
		assert.EqualError(t, err, "Unable to rebase all pull requests")
	} else {
		assert.Nil(t, err)
	}

	assert.Equal(t, []*runner.ExpectedCommand{}, cb.ExpectedCommands)
	assert.Equal(t, []error(nil), cb.Errors)
	return writer
}

func TestHelperProcess(*testing.T) {
	runner.ErrorCodeHelper()
}
