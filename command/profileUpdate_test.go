package command_test

import (
	"flag"
	"io/ioutil"
	"testing"

	"github.com/guywithnose/pull-request-parser/command"
	"github.com/guywithnose/pull-request-parser/config"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestCmdProfileUpdate(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	set.String("token", "abc", "doc")
	set.String("apiUrl", "https://api.com", "doc")
	assert.Nil(t, command.CmdProfileUpdate(cli.NewContext(nil, set, nil)))

	modifiedConfigData, err := config.LoadFromFile(configFileName)
	assert.Nil(t, err)

	expectedConfigFile := config.PrpConfig{
		Profiles: map[string]config.Profile{
			"foo": {
				Token:        "abc",
				TrackedRepos: []config.Repo{},
				APIURL:       "https://api.com",
			},
		},
	}

	assert.Equal(t, *modifiedConfigData, expectedConfigFile)
}

func TestCmdProfileUpdateUsage(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"foo"}))
	err := command.CmdProfileUpdate(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "Usage: \"prp profile update\"")
}

func TestCmdProfileUpdateNoParameters(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	err := command.CmdProfileUpdate(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "An update parameter is required")
}

func TestCmdProfileUpdateNotExists(t *testing.T) {
	conf := config.PrpConfig{
		Profiles: map[string]config.Profile{},
	}
	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)
	assert.Nil(t, conf.Write(configFile.Name()))
	defer removeFile(t, configFile.Name())
	set := getBaseFlagSet(configFile.Name())
	assert.Nil(t, set.Parse([]string{"foo"}))
	err = command.CmdProfileUpdate(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "Invalid Profile: foo")
}

func TestCmdProfileUpdateNoConfig(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	assert.Nil(t, set.Parse([]string{"foo"}))
	err := command.CmdProfileUpdate(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "You must specify a config file")
}
