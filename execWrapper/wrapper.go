package execWrapper

import "os/exec"

// CommandBuilder builds a command that can then be run
type CommandBuilder interface {
	CreateCommand(path string, command ...string) Command
}

// Command is used to run commands.  It is a sub-interface of os.exec.Cmd
type Command interface {
	Output() ([]byte, error)
	CombinedOutput() ([]byte, error)
}

// Wrapper is the standard CommandBuilder that will return instances of os.exec.Cmd
type Wrapper struct{}

// CreateCommand creates an instance os.exec.Cmd
func (w Wrapper) CreateCommand(path string, command ...string) Command {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = path
	return cmd
}
