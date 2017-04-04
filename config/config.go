package config

import (
	"encoding/json"
	"io/ioutil"
)

// PrpConfig defines the structure of the pull request parser config file
type PrpConfig struct {
	Profiles map[string]PrpConfigProfile `json:"profiles,omitempty"`
}

// PrpConfigProfile defines the structure of pull request parser profile
type PrpConfigProfile struct {
	TrackedRepos []PrpConfigRepo `json:"trackedRepos,omitempty"`
	Token        string          `json:"token,omitempty"`
	APIURL       string          `json:"apiUrl,omitempty"`
}

// PrpConfigRepo defines the structure of pull request parser tracked repo entry
type PrpConfigRepo struct {
	Owner         string   `json:"owner,omitempty"`
	Name          string   `json:"name,omitempty"`
	LocalPath     string   `json:"localPath,omitempty"`
	IgnoredBuilds []string `json:"ignoredBuilds,omitempty"`
}

// LoadConfigFromFile loads a PrpConfig from a file
func LoadConfigFromFile(fileName string) (*PrpConfig, error) {
	configJSON, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var configData = new(PrpConfig)
	err = json.Unmarshal(configJSON, configData)
	if err != nil {
		return nil, err
	}

	Validate(configData)

	return configData, nil
}

// Validate makes sure all of the config values are initialized
func Validate(configData *PrpConfig) {
	if configData.Profiles == nil {
		configData.Profiles = make(map[string]PrpConfigProfile)
	}

	for profileName, profile := range configData.Profiles {
		if profile.TrackedRepos == nil {
			profile.TrackedRepos = make([]PrpConfigRepo, 0)
		}

		for repoName, repo := range profile.TrackedRepos {
			if repo.IgnoredBuilds == nil {
				repo.IgnoredBuilds = []string{}
				profile.TrackedRepos[repoName] = repo
			}
		}

		configData.Profiles[profileName] = profile
	}
}

// WriteConfig saves a PrpConfig to a file
func WriteConfig(outputFile string, configData *PrpConfig) error {
	formattedConfig, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		// This should never happen
		panic(err)
	}

	return ioutil.WriteFile(outputFile, formattedConfig, 0644)
}
