package config

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteConfig(t *testing.T) {
	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)

	configData := getTestingConfig()

	err = WriteConfig(configFile.Name(), configData)
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
	err := WriteConfig("/doesntexist", configData)
	assert.EqualError(t, err, "open /doesntexist: permission denied")
}

func TestLoadConfigFromFile(t *testing.T) {
	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)

	err = ioutil.WriteFile(configFile.Name(), []byte(getTestingConfigJSONString()), 0644)
	assert.Nil(t, err)

	configData, err := LoadConfigFromFile(configFile.Name())
	assert.Nil(t, err)

	expectedConfigData := getTestingConfig()

	if !reflect.DeepEqual(configData, expectedConfigData) {
		t.Fatalf("File was %v, expected %v", configData, expectedConfigData)
	}

	assert.Nil(t, os.Remove(configFile.Name()))
}

func TestLoadConfigFromFileInvalidFile(t *testing.T) {
	_, err := LoadConfigFromFile("/doesntexist")
	assert.EqualError(t, err, "open /doesntexist: no such file or directory")
}

func TestLoadConfigFromFileInvalidJSON(t *testing.T) {
	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)

	err = ioutil.WriteFile(configFile.Name(), []byte("{"), 0644)
	assert.Nil(t, err)

	_, err = LoadConfigFromFile(configFile.Name())
	assert.EqualError(t, err, "unexpected end of JSON input")
	assert.Nil(t, os.Remove(configFile.Name()))
}

func TestLoadEmptyConfigAndWrite(t *testing.T) {
	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)

	err = ioutil.WriteFile(configFile.Name(), []byte("{}"), 0644)
	assert.Nil(t, err)

	configData, err := LoadConfigFromFile(configFile.Name())
	assert.Nil(t, err)

	err = WriteConfig(configFile.Name(), configData)
	assert.Nil(t, err)

	configBytes, err := ioutil.ReadFile(configFile.Name())
	assert.Nil(t, err)

	expectedJSONString := `{}`
	if string(configBytes) != expectedJSONString {
		t.Fatalf("File was %s, expected %s", configBytes, expectedJSONString)
	}

	assert.Nil(t, os.Remove(configFile.Name()))
}

func TestLoadEmptyProfile(t *testing.T) {
	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)

	err = ioutil.WriteFile(configFile.Name(), []byte(`{"profiles":{"foo": {}}}`), 0644)
	assert.Nil(t, err)

	configData, err := LoadConfigFromFile(configFile.Name())
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

	configData, err := LoadConfigFromFile(configFile.Name())
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

	configData, err := LoadConfigFromFile(configFile.Name())
	assert.Nil(t, err)

	err = WriteConfig(configFile.Name(), configData)
	assert.Nil(t, err)

	configBytes, err := ioutil.ReadFile(configFile.Name())
	assert.Nil(t, err)

	expectedJSONString := `{}`
	if string(configBytes) != expectedJSONString {
		t.Fatalf("File was %s, expected %s", configBytes, expectedJSONString)
	}

	assert.Nil(t, os.Remove(configFile.Name()))
}

func getTestingConfig() *PrpConfig {
	return &PrpConfig{
		Profiles: map[string]PrpConfigProfile{
			"foo": {
				Token:  "abc",
				APIURL: "https://api.com",
				TrackedRepos: []PrpConfigRepo{
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
