package command

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/google/go-github/github"
	"github.com/guywithnose/pull-request-parser/config"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func removeFile(t *testing.T, fileName string) {
	assert.Nil(t, os.RemoveAll(fileName))
}

func getConfigWithFooProfile(t *testing.T) (config.PrpConfig, string) {
	conf := config.PrpConfig{
		Profiles: map[string]config.PrpConfigProfile{
			"foo": config.PrpConfigProfile{
				TrackedRepos: []config.PrpConfigRepo{},
			},
		},
	}

	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)
	assert.Nil(t, config.WriteConfig(configFile.Name(), &conf))
	return conf, configFile.Name()
}

func getConfigWithTwoRepos(t *testing.T) (config.PrpConfig, string) {
	conf := config.PrpConfig{
		Profiles: map[string]config.PrpConfigProfile{
			"foo": config.PrpConfigProfile{
				TrackedRepos: []config.PrpConfigRepo{
					{
						Owner:         "foo",
						Name:          "bar",
						IgnoredBuilds: []string{},
					},
					{
						Owner:         "own",
						Name:          "rep",
						IgnoredBuilds: []string{},
					},
				},
			},
		},
	}

	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)
	assert.Nil(t, config.WriteConfig(configFile.Name(), &conf))
	return conf, configFile.Name()
}

func getConfigWithIgnoredBuild(t *testing.T) (config.PrpConfig, string) {
	conf, configFileName := getConfigWithTwoRepos(t)
	profile := conf.Profiles["foo"]
	profile.TrackedRepos[0].IgnoredBuilds = []string{"goo"}
	conf.Profiles["foo"] = profile
	assert.Nil(t, config.WriteConfig(configFileName, &conf))
	return conf, configFileName
}

func appWithTestWriter() (*cli.App, *bytes.Buffer) {
	app := cli.NewApp()
	writer := new(bytes.Buffer)
	app.Writer = writer
	return app, writer
}

func assertOutput(t *testing.T, writer *bytes.Buffer, expectedOutput string) {
	if writer.String() != expectedOutput {
		t.Fatalf("Output was \n%s\n, expected \n%s\n", writer.String(), expectedOutput)
	}
}

func getBaseFlagSet(configFileName string) *flag.FlagSet {
	set := flag.NewFlagSet("test", 0)
	set.String("config", configFileName, "doc")
	set.String("profile", "foo", "doc")
	return set
}

func assertConfigFile(t *testing.T, expectedConfigFile config.PrpConfig, configFileName string) {
	modifiedConfigData, err := config.LoadConfigFromFile(configFileName)
	assert.Nil(t, err)

	if !reflect.DeepEqual(*modifiedConfigData, expectedConfigFile) {
		t.Fatalf("File was \n%v\n, expected \n%v\n", *modifiedConfigData, expectedConfigFile)
	}
}

func getConfigWithAPIURL(t *testing.T, url string) (config.PrpConfig, string) {
	conf, configFileName := getConfigWithIgnoredBuild(t)
	profile := conf.Profiles["foo"]
	profile.APIURL = url
	profile.Token = "abc"
	conf.Profiles["foo"] = profile
	assert.Nil(t, config.WriteConfig(configFileName, &conf))
	return conf, configFileName
}

func newUser(login string) *github.User {
	return &github.User{Login: &login}
}

func handleUserRequest(r *http.Request) *string {
	if r.URL.String() == "/user" {
		bytes, _ := json.Marshal(newUser("own"))
		response := string(bytes)
		return &response
	}

	return nil
}
