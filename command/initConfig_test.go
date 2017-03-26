package command

import (
	"flag"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestCmdInitConfig(t *testing.T) {
	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)
	removeFile(t, configFile.Name())
	defer removeFile(t, configFile.Name())
	set := getBaseFlagSet(configFile.Name())
	assert.Nil(t, CmdInitConfig(cli.NewContext(nil, set, nil)))

	configJSON, err := ioutil.ReadFile(configFile.Name())
	assert.Nil(t, err)

	expectedConfigFile := `{}`
	if string(configJSON) != expectedConfigFile {
		t.Fatalf("File was \n%v\n, expected \n%v\n", string(configJSON), expectedConfigFile)
	}
}

func TestCmdInitConfigNoConfigParameter(t *testing.T) {
	err := CmdInitConfig(cli.NewContext(nil, flag.NewFlagSet("test", 0), nil))
	assert.EqualError(t, err, "You must specify a config file")
}

func TestCmdInitConfigFileExists(t *testing.T) {
	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)
	defer removeFile(t, configFile.Name())
	set := getBaseFlagSet(configFile.Name())
	err = CmdInitConfig(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, fmt.Sprintf("File already exists: %s", configFile.Name()))
}
