package command_test

import (
	"flag"
	"io/ioutil"
	"os"
	"testing"

	"github.com/guywithnose/pull-request-parser/command"
	"github.com/guywithnose/pull-request-parser/config"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestCmdProfileAdd(t *testing.T) {
	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)
	defer removeFile(t, configFile.Name())
	conf := config.PrpConfig{}
	assert.Nil(t, conf.Write(configFile.Name()))
	set := getBaseFlagSet(configFile.Name())
	set.String("token", "abc", "doc")
	assert.Nil(t, set.Parse([]string{"foo"}))
	assert.Nil(t, command.CmdProfileAdd(cli.NewContext(nil, set, nil)))

	modifiedConfigData, err := config.LoadFromFile(configFile.Name())
	assert.Nil(t, err)

	expectedConfigFile := config.PrpConfig{
		Profiles: map[string]config.Profile{
			"foo": {
				Token:        "abc",
				TrackedRepos: []config.Repo{},
			},
		},
	}

	assert.Equal(t, *modifiedConfigData, expectedConfigFile)
}

func TestCmdProfileAddUsage(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	set.String("token", "abc", "doc")
	err := command.CmdProfileAdd(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "Usage: \"prp profile add {profileName} --token {token}\"")
}

func TestCmdProfileAddNoToken(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"foo"}))
	err := command.CmdProfileAdd(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "You must specify a token")
}

func TestCmdProfileAddExists(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	set.String("token", "abc", "doc")
	assert.Nil(t, set.Parse([]string{"foo"}))
	err := command.CmdProfileAdd(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "Profile foo already exists")
}

func TestCmdProfileAddNoConfig(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	assert.Nil(t, set.Parse([]string{"foo"}))
	err := command.CmdProfileAdd(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "You must specify a config file")
}

func TestCmdProfileAddInvalidConfig(t *testing.T) {
	set := getBaseFlagSet("/notafile")
	assert.Nil(t, set.Parse([]string{"foo"}))
	err := command.CmdProfileAdd(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "open /notafile: no such file or directory")
}

func TestCompleteProfileAddToken(t *testing.T) {
	configWithToken, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	profile := configWithToken.Profiles["foo"]
	profile.Token = "abc"
	configWithToken.Profiles["foo"] = profile
	assert.Nil(t, configWithToken.Write(configFileName))
	set := getBaseFlagSet(configFileName)
	os.Args = []string{"profile", "add", "--token", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteProfileAdd(cli.NewContext(app, set, nil))
	assert.Equal(t, "abc\n", writer.String())
}

func TestCompleteProfileAddFlags(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	os.Args = []string{"profile", "add", "--completion"}
	app, writer, _ := appWithTestWriters()
	app.Commands = []cli.Command{
		{
			Name: "add",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "token, t"},
				cli.StringFlag{Name: "apiUrl, a"},
			},
		},
	}
	command.CompleteProfileAdd(cli.NewContext(app, set, nil))
	assert.Equal(t, "--token\n--apiUrl\n", writer.String())
}

func TestCompleteProfileAddApiUrl(t *testing.T) {
	conf, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	profile := conf.Profiles["foo"]
	profile.APIURL = "http://api.com"
	conf.Profiles["foo"] = profile
	assert.Nil(t, conf.Write(configFileName))
	os.Args = []string{"profile", "add", "--apiUrl", "--completion"}
	set := getBaseFlagSet(configFileName)
	app, writer, _ := appWithTestWriters()
	command.CompleteProfileAdd(cli.NewContext(app, set, nil))
	assert.Equal(t, "http\\://api.com\n", writer.String())
}

func TestCompleteProfileAddNoConfig(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	os.Args = []string{"profile", "add", "--apiUrl", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteProfileAdd(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}
