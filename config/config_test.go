package config_test

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/guywithnose/pull-request-parser/config"
	"github.com/stretchr/testify/assert"
)

func TestWriteConfig(t *testing.T) {
	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)

	configData := getTestingConfig()

	err = configData.Write(configFile.Name())
	assert.Nil(t, err)

	configBytes, err := ioutil.ReadFile(configFile.Name())
	assert.Nil(t, err)

	if string(configBytes) != getTestingConfigJSONString() {
		t.Fatalf("File was %s, expected %s", configBytes, getTestingConfigJSONString())
	}

	assert.Nil(t, os.Remove(configFile.Name()))
}

func TestWriteConfigInvalidFile(t *testing.T) {
	configData := getTestingConfig()
	err := configData.Write("/doesntexist")
	assert.EqualError(t, err, "open /doesntexist: permission denied")
}

func TestLoadConfigFromFile(t *testing.T) {
	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)

	err = ioutil.WriteFile(configFile.Name(), []byte(getTestingConfigJSONString()), 0644)
	assert.Nil(t, err)

	configData, err := config.LoadFromFile(configFile.Name())
	assert.Nil(t, err)

	expectedConfigData := getTestingConfig()

	if !reflect.DeepEqual(configData, expectedConfigData) {
		t.Fatalf("File was %v, expected %v", configData, expectedConfigData)
	}

	assert.Nil(t, os.Remove(configFile.Name()))
}

func TestLoadConfigFromFileInvalidFile(t *testing.T) {
	_, err := config.LoadFromFile("/doesntexist")
	assert.EqualError(t, err, "open /doesntexist: no such file or directory")
}

func TestLoadConfigFromFileInvalidJSON(t *testing.T) {
	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)

	err = ioutil.WriteFile(configFile.Name(), []byte("{"), 0644)
	assert.Nil(t, err)

	_, err = config.LoadFromFile(configFile.Name())
	assert.EqualError(t, err, "unexpected end of JSON input")
	assert.Nil(t, os.Remove(configFile.Name()))
}

func TestLoadEmptyProfile(t *testing.T) {
	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)

	err = ioutil.WriteFile(configFile.Name(), []byte(`{"profiles":{"foo": {}}}`), 0644)
	assert.Nil(t, err)

	configData, err := config.LoadFromFile(configFile.Name())
	assert.Nil(t, err)

	if configData.Profiles["foo"].TrackedRepos == nil {
		t.Fatal("TrackedRepos was not auto initialized")
	}

	assert.Nil(t, os.Remove(configFile.Name()))
}

func TestLoadEmptyIgnoredBuilds(t *testing.T) {
	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)

	err = ioutil.WriteFile(configFile.Name(), []byte(`{"profiles":{"foo": {"trackedRepos": [{}]}}}`), 0644)
	assert.Nil(t, err)

	configData, err := config.LoadFromFile(configFile.Name())
	assert.Nil(t, err)

	if configData.Profiles["foo"].TrackedRepos[0].IgnoredBuilds == nil {
		t.Fatal("TrackedRepos was not auto initialized")
	}

	assert.Nil(t, os.Remove(configFile.Name()))
}

func TestLoadEmptyHostConfigAndWrite(t *testing.T) {
	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)

	err = ioutil.WriteFile(configFile.Name(), []byte(`{}`), 0644)
	assert.Nil(t, err)

	configData, err := config.LoadFromFile(configFile.Name())
	assert.Nil(t, err)

	err = configData.Write(configFile.Name())
	assert.Nil(t, err)

	configBytes, err := ioutil.ReadFile(configFile.Name())
	assert.Nil(t, err)

	expectedJSONString := `{}`
	if string(configBytes) != expectedJSONString {
		t.Fatalf("File was %s, expected %s", configBytes, expectedJSONString)
	}

	assert.Nil(t, os.Remove(configFile.Name()))
}

func TestProfileUpdate(t *testing.T) {
	profile := config.Profile{
		Token:  "abc",
		APIURL: "https://api.com",
	}

	profile.Update("foo", "bar")
	assert.Equal(t, "foo", profile.Token)
	assert.Equal(t, "bar", profile.APIURL)
}

func getTestingConfig() *config.PrpConfig {
	return &config.PrpConfig{
		Profiles: map[string]config.Profile{
			"foo": {
				Token:  "abc",
				APIURL: "https://api.com",
				TrackedRepos: []config.Repo{
					{
						Owner:         "own",
						Name:          "rep",
						LocalPath:     "/foo",
						IgnoredBuilds: []string{"broken"},
					},
				},
			},
		},
	}
}

func getTestingConfigJSONString() string {
	return `{
  "profiles": {
    "foo": {
      "trackedRepos": [
        {
          "owner": "own",
          "name": "rep",
          "localPath": "/foo",
          "ignoredBuilds": [
            "broken"
          ]
        }
      ],
      "token": "abc",
      "apiUrl": "https://api.com"
    }
  }
}`
}
