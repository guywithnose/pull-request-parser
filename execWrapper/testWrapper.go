package execWrapper

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// TestCommandBuilder is used for testing code that runs system commands without actually running the commands
type TestCommandBuilder struct {
	ExpectedCommands []*ExpectedCommand
	Errors           []error
}

// TestCommand emulates an os.exec.Cmd, but returns mock data
type TestCommand struct {
	cmdBuilder      *TestCommandBuilder
	Dir             string
	expectedCommand *ExpectedCommand
}

// ExpectedCommand represents a command that will be handled by a TestCommand
type ExpectedCommand struct {
	command string
	output  []byte
	path    string
	error   error
}

// CreateCommand returns a TestCommand
func (testBuilder *TestCommandBuilder) CreateCommand(path string, command ...string) Command {
	var expectedCommand *ExpectedCommand
	commandString := strings.Join(command, " ")
	if len(testBuilder.ExpectedCommands) == 0 {
		testBuilder.Errors = append(testBuilder.Errors, fmt.Errorf("More commands were run than expected.  Extra command: %s", commandString))
	} else {
		expectedCommand = testBuilder.ExpectedCommands[0]
		if expectedCommand.path != path {
			testBuilder.Errors = append(testBuilder.Errors, fmt.Errorf("Path %s did not match expected path %s", path, expectedCommand.path))
		} else if expectedCommand.command != commandString {
			testBuilder.Errors = append(testBuilder.Errors, fmt.Errorf("Command '%s' did not match expected command '%s'", commandString, expectedCommand.command))
		} else {
			testBuilder.ExpectedCommands = testBuilder.ExpectedCommands[1:]
		}
	}

	return TestCommand{cmdBuilder: testBuilder, Dir: path, expectedCommand: expectedCommand}
}

// Output returns the expected mock data
func (cmd TestCommand) Output() ([]byte, error) {
	if cmd.expectedCommand == nil {
		return nil, nil
	}

	return cmd.expectedCommand.output, cmd.expectedCommand.error
}

// CombinedOutput returns the expected mock data
func (cmd TestCommand) CombinedOutput() ([]byte, error) {
	if cmd.expectedCommand == nil {
		return nil, nil
	}

	return cmd.expectedCommand.output, cmd.expectedCommand.error
}

// NewExpectedCommand returns a new ExpectedCommand
func NewExpectedCommand(path, command, output string, exitCode int) *ExpectedCommand {
	var exitError error
	if exitCode == -1 {
		exitError = errors.New("Error running command")
	} else if exitCode != 0 {
		script := []byte("#!/bin/bash\nexit $1")
		scriptFileName := fmt.Sprintf("%s/returnCode", os.TempDir())
		err := ioutil.WriteFile(scriptFileName, script, 0777)
		if err != nil {
			panic(err)
		}

		//TODO this doesn't seem to work
		cmd := exec.Command(scriptFileName, strconv.Itoa(exitCode))
		exitError = cmd.Run()
		if exitErr, ok := exitError.(*exec.ExitError); ok {
			exitErr.Stderr = []byte(output)
			exitError = exitErr
		}
		err = os.Remove(scriptFileName)
		if err != nil {
			panic(err)
		}
	}

	return &ExpectedCommand{
		command: command,
		output:  []byte(output),
		path:    path,
		error:   exitError,
	}
}
