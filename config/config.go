package config

import (
	"encoding/json"
	"io/ioutil"
)

// PrpConfig defines the structure of the pull request parser config file
type PrpConfig struct {
	Profiles map[string]Profile `json:"profiles,omitempty"`
}

// Profile defines the structure of pull request parser profile
type Profile struct {
	TrackedRepos []Repo `json:"trackedRepos,omitempty"`
	Token        string `json:"token,omitempty"`
	APIURL       string `json:"apiUrl,omitempty"`
}

// Repo defines the structure of pull request parser tracked repo entry
type Repo struct {
	Owner         string   `json:"owner,omitempty"`
	Name          string   `json:"name,omitempty"`
	LocalPath     string   `json:"localPath,omitempty"`
	IgnoredBuilds []string `json:"ignoredBuilds,omitempty"`
}

// LoadFromFile loads a PrpConfig from a file
func LoadFromFile(fileName string) (*PrpConfig, error) {
	configJSON, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var configData = new(PrpConfig)
	err = json.Unmarshal(configJSON, configData)
	if err != nil {
		return nil, err
	}

	configData.Validate()

	return configData, nil
}

// Validate makes sure all of the config values are initialized
func (configData *PrpConfig) Validate() {
	if configData.Profiles == nil {
		configData.Profiles = make(map[string]Profile)
	}

	for profileName, profile := range configData.Profiles {
		profile.validate()
		configData.Profiles[profileName] = profile
	}
}

func (p *Profile) validate() {
	if p.TrackedRepos == nil {
		p.TrackedRepos = make([]Repo, 0)
	}

	for repoName, repo := range p.TrackedRepos {
		if repo.IgnoredBuilds == nil {
			repo.IgnoredBuilds = []string{}
			p.TrackedRepos[repoName] = repo
		}
	}
}

// Update updates values in the profile
func (p *Profile) Update(token, APIURL string) {
	if token != "" {
		p.Token = token
	}

	if APIURL != "" {
		p.APIURL = APIURL
	}
}

// Write saves a PrpConfig to a file
func (configData PrpConfig) Write(outputFile string) error {
	formattedConfig, _ := json.MarshalIndent(configData, "", "  ")
	return ioutil.WriteFile(outputFile, formattedConfig, 0644)
}
