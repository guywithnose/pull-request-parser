package command

import (
	"flag"
	"io/ioutil"
	"reflect"
	"testing"

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
	assert.Nil(t, CmdProfileUpdate(cli.NewContext(nil, set, nil)))

	modifiedConfigData, err := config.LoadConfigFromFile(configFileName)
	assert.Nil(t, err)

	expectedConfigFile := config.PrpConfig{
		Profiles: map[string]config.PrpConfigProfile{
			"foo": config.PrpConfigProfile{
				Token:        "abc",
				TrackedRepos: []config.PrpConfigRepo{},
				APIURL:       "https://api.com",
			},
		},
	}
	if !reflect.DeepEqual(*modifiedConfigData, expectedConfigFile) {
		t.Fatalf("File was \n%v\n, expected \n%v\n", *modifiedConfigData, expectedConfigFile)
	}
}

func TestCmdProfileUpdateUsage(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"foo"}))
	err := CmdProfileUpdate(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "Usage: \"prp profile update\"")
}

func TestCmdProfileUpdateNoParameters(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	err := CmdProfileUpdate(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "An update parameter is required")
}

func TestCmdProfileUpdateNotExists(t *testing.T) {
	conf := config.PrpConfig{
		Profiles: map[string]config.PrpConfigProfile{},
	}
	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)
	assert.Nil(t, config.WriteConfig(configFile.Name(), &conf))
	defer removeFile(t, configFile.Name())
	set := getBaseFlagSet(configFile.Name())
	assert.Nil(t, set.Parse([]string{"foo"}))
	err = CmdProfileUpdate(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "Invalid Profile: foo")
}

func TestCmdProfileUpdateNoConfig(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	assert.Nil(t, set.Parse([]string{"foo"}))
	err := CmdProfileUpdate(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "You must specify a config file")
}
