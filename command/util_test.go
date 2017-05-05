package command_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
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
		Profiles: map[string]config.Profile{
			"foo": {
				TrackedRepos: []config.Repo{},
			},
		},
	}

	configFile, err := ioutil.TempFile("/tmp", "config")
	assert.Nil(t, err)
	assert.Nil(t, conf.Write(configFile.Name()))
	return conf, configFile.Name()
}

func getConfigWithTwoRepos(t *testing.T) (config.PrpConfig, string) {
	conf := config.PrpConfig{
		Profiles: map[string]config.Profile{
			"foo": {
				TrackedRepos: []config.Repo{
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
	assert.Nil(t, conf.Write(configFile.Name()))
	return conf, configFile.Name()
}

func getConfigWithIgnoredBuild(t *testing.T) (config.PrpConfig, string) {
	conf, configFileName := getConfigWithTwoRepos(t)
	profile := conf.Profiles["foo"]
	profile.TrackedRepos[0].IgnoredBuilds = []string{"goo"}
	conf.Profiles["foo"] = profile
	assert.Nil(t, conf.Write(configFileName))
	return conf, configFileName
}

func appWithTestWriters() (*cli.App, *bytes.Buffer, *bytes.Buffer) {
	app := cli.NewApp()
	writer := new(bytes.Buffer)
	errWriter := new(bytes.Buffer)
	app.Writer = writer
	app.ErrWriter = errWriter
	return app, writer, errWriter
}

func getBaseFlagSet(configFileName string) *flag.FlagSet {
	set := flag.NewFlagSet("test", 0)
	set.String("config", configFileName, "doc")
	set.String("profile", "foo", "doc")
	return set
}

func assertConfigFile(t *testing.T, expectedConfigFile config.PrpConfig, configFileName string) {
	modifiedConfigData, err := config.LoadFromFile(configFileName)
	assert.Nil(t, err)
	assert.Equal(t, *modifiedConfigData, expectedConfigFile)
}

func getConfigWithAPIURL(t *testing.T, url string) (config.PrpConfig, string) {
	conf, configFileName := getConfigWithIgnoredBuild(t)
	profile := conf.Profiles["foo"]
	profile.APIURL = url
	profile.Token = "abc"
	conf.Profiles["foo"] = profile
	assert.Nil(t, conf.Write(configFileName))
	return conf, configFileName
}

func getConfigWithAPIURLAndPath(t *testing.T, url, path string) (config.PrpConfig, string) {
	conf, configFileName := getConfigWithAPIURL(t, url)
	profile := conf.Profiles["foo"]
	profile.TrackedRepos[1].LocalPath = path
	conf.Profiles["foo"] = profile
	assert.Nil(t, conf.Write(configFileName))
	return conf, configFileName
}

func newUser(login string) *github.User {
	return &github.User{Login: &login}
}

func handleUserRequest(r *http.Request, owner string) *string {
	if r.URL.String() == "/user" {
		bytes, _ := json.Marshal(newUser(owner))
		response := string(bytes)
		return &response
	}

	return nil
}

func handlePullRequestRequests(r *http.Request, w http.ResponseWriter, server *httptest.Server) *string {
	if r.URL.String() == "/repos/own/rep/pulls?per_page=100" {
		bytes, _ := json.Marshal([]*github.PullRequest{
			newPullRequest(1, "prOne", "guy", "label", "ref1", "sha1", "baseLabel1", "baseRef1"),
		})
		w.Header().Set(
			"Link",
			fmt.Sprintf(
				`<%s/mockApi/repos/own/rep/pulls?per_page=100&page=2>; rel="next", `+
					`<%s/mockApi/repos/own/rep/pulls?per_page=100&page=2>; rel="last"`,
				server.URL,
				server.URL,
			),
		)
		response := string(bytes)
		return &response
	}

	if r.URL.String() == "/repos/own/rep/pulls?page=2&per_page=100" {
		bytes, _ := json.Marshal([]*github.PullRequest{
			newPullRequest(2, "Really long Pull Request Title", "guy2", "label", "ref2", "sha2", "baseLabel2", "baseRef2"),
		})
		response := string(bytes)
		return &response
	}

	if r.URL.String() == "/repos/foo/bar/pulls?per_page=100" {
		bytes, _ := json.Marshal([]*github.PullRequest{
			newPullRequest(1, "fooPrOne", "fooGuy", "fooLabel", "fooRef1", "fooSha1", "fooBaseLabel1", "fooBaseRef1"),
			newPullRequest(2, "fooPrTwo", "fooGuy2", "fooLabel", "fooRef2", "fooSha2", "fooBaseLabel2", "fooBaseRef2"),
		})
		response := string(bytes)
		return &response
	}

	return nil
}

func handleCommentRequests(r *http.Request, w http.ResponseWriter, server *httptest.Server) *string {
	responses := make(map[string]string)
	bytes, _ := json.Marshal([]*github.IssueComment{
		newComment("foo", "guy"),
	})
	if r.URL.String() == "/repos/own/rep/issues/1/comments?per_page=100" {
		w.Header().Set(
			"Link",
			fmt.Sprintf(
				`<%s/mockApi/repos/own/rep/issues/1/comments?per_page=100&page=2>; rel="next", `+
					`<%s/mockApi/repos/own/rep/issues/1/comments?per_page=100&page=2>; rel="last"`,
				server.URL,
				server.URL,
			),
		)
	}
	responses["/repos/own/rep/issues/1/comments?per_page=100"] = string(bytes)

	bytes, _ = json.Marshal([]*github.IssueComment{
		newComment(":thumbsup:", "own"),
	})
	responses["/repos/own/rep/issues/1/comments?page=2&per_page=100"] = string(bytes)

	bytes, _ = json.Marshal([]*github.IssueComment{
		newComment(":+1:", "guy"),
		newComment(":thumbsup:", "guy2"),
		newComment("LGTM", "guy"),
	})
	responses["/repos/foo/bar/issues/1/comments?per_page=100"] = string(bytes)

	bytes, _ = json.Marshal([]*github.IssueComment{
		newComment(":+1:", "guy"),
		newComment("LGTM", "guy2"),
	})
	responses["/repos/own/rep/issues/2/comments?per_page=100"] = string(bytes)

	bytes, _ = json.Marshal([]*github.IssueComment{
		newComment("foo", "guy"),
	})
	responses["/repos/foo/bar/issues/2/comments?per_page=100"] = string(bytes)

	response, ok := responses[r.URL.String()]
	if ok {
		return &response
	}

	return nil
}

func handleReviewRequests(r *http.Request, w http.ResponseWriter, server *httptest.Server) *string {
	if r.URL.String() == "/repos/own/rep/pulls/1/reviews" {
		bytes, _ := json.Marshal([]*github.PullRequestReview{
			newReview("guy", "APPROVED"),
		})
		w.Header().Set(
			"Link",
			fmt.Sprintf(
				`<%s/mockApi/repos/own/rep/issues/1/comments?per_page=100&page=2>; rel="next", `+
					`<%s/mockApi/repos/own/rep/issues/1/comments?per_page=100&page=2>; rel="last"`,
				server.URL,
				server.URL,
			),
		)
		response := string(bytes)
		return &response
	}

	if r.URL.String() == "/repos/foo/bar/pulls/1/reviews" {
		bytes, _ := json.Marshal([]*github.PullRequestReview{
			newReview("guy", "APPROVED"),
			newReview("own", "APPROVED"),
			newReview("guy2", "APPROVED"),
			newReview("guy3", "APPROVED"),
		})
		response := string(bytes)
		return &response
	}

	if r.URL.String() == "/repos/own/rep/pulls/2/reviews" {
		bytes, _ := json.Marshal([]*github.PullRequestReview{
			newReview("guy", "APPROVED"),
		})
		response := string(bytes)
		return &response
	}

	if r.URL.String() == "/repos/foo/bar/pulls/2/reviews" {
		bytes, _ := json.Marshal([]*github.PullRequestReview{
			newReview("guy2", "APPROVED"),
		})
		response := string(bytes)
		return &response
	}

	return nil
}

func handleLabelRequests(r *http.Request, _ http.ResponseWriter, _ *httptest.Server) *string {
	if r.URL.String() == "/repos/own/rep/issues/1/labels" {
		bytes, _ := json.Marshal([]*github.Label{
			newLabel("label1"),
		})
		response := string(bytes)
		return &response
	}

	if r.URL.String() == "/repos/foo/bar/issues/1/labels" {
		bytes, _ := json.Marshal([]*github.Label{
			newLabel("label2"),
			newLabel("label3"),
		})
		response := string(bytes)
		return &response
	}

	if r.URL.String() == "/repos/own/rep/issues/2/labels" {
		bytes, _ := json.Marshal([]*github.Label{})
		response := string(bytes)
		return &response
	}

	if r.URL.String() == "/repos/foo/bar/issues/2/labels" {
		bytes, _ := json.Marshal([]*github.Label{
			newLabel("label4"),
			newLabel("label5"),
			newLabel("really-long-label"),
		})
		response := string(bytes)
		return &response
	}

	return nil
}

func handleStatusRequests(r *http.Request, _ http.ResponseWriter, _ *httptest.Server) *string {
	if r.URL.String() == "/repos/own/rep/commits/sha1/statuses" {
		bytes, _ := json.Marshal([]*github.RepoStatus{
			newStatus("build1", "success"),
			newStatus("build1", "pending"),
		})
		response := string(bytes)
		return &response
	}

	if r.URL.String() == "/repos/foo/bar/commits/fooSha1/statuses" {
		bytes, _ := json.Marshal([]*github.RepoStatus{
			newStatus("build1", "pending"),
			newStatus("build2", "success"),
			newStatus("build2", "pending"),
			newStatus("goo", "failure"),
			newStatus("goo", "pending"),
		})
		response := string(bytes)
		return &response
	}

	if r.URL.String() == "/repos/own/rep/commits/sha2/statuses" {
		bytes, _ := json.Marshal([]*github.RepoStatus{
			newStatus("build1", "failure"),
			newStatus("build1", "pending"),
		})
		response := string(bytes)
		return &response
	}

	if r.URL.String() == "/repos/foo/bar/commits/fooSha2/statuses" {
		bytes, _ := json.Marshal([]*github.RepoStatus{})
		response := string(bytes)
		return &response
	}

	return nil
}

func handleCommitsComparisonRequests(r *http.Request, _ http.ResponseWriter, _ *httptest.Server) *string {
	if r.URL.String() == "/repos/own/rep/compare/label...baseLabel1" {
		bytes, _ := json.Marshal(newCommitsComparison(1))
		response := string(bytes)
		return &response
	}

	if r.URL.String() == "/repos/foo/bar/compare/fooLabel...fooBaseLabel1" {
		bytes, _ := json.Marshal(newCommitsComparison(0))
		response := string(bytes)
		return &response
	}

	if r.URL.String() == "/repos/own/rep/compare/label...baseLabel2" {
		bytes, _ := json.Marshal(newCommitsComparison(0))
		response := string(bytes)
		return &response
	}

	if r.URL.String() == "/repos/foo/bar/compare/fooLabel...fooBaseLabel2" {
		bytes, _ := json.Marshal(newCommitsComparison(1))
		response := string(bytes)
		return &response
	}

	return nil
}

func newPullRequest(number int, title, owner, label, ref, sha, baseLabel, baseRef string) *github.PullRequest {
	headSSHURL := fmt.Sprintf("%sSSHURL", label)
	baseSSHURL := fmt.Sprintf("%sSSHURL", baseLabel)
	return &github.PullRequest{
		Number: &number,
		Title:  &title,
		Head: &github.PullRequestBranch{
			Label: &label,
			Ref:   &ref,
			SHA:   &sha,
			User:  &github.User{Login: &owner},
			Repo:  &github.Repository{SSHURL: &headSSHURL},
		},
		Base: &github.PullRequestBranch{
			Label: &baseLabel,
			Ref:   &baseRef,
			Repo:  &github.Repository{SSHURL: &baseSSHURL},
		},
	}
}

func newCommitsComparison(aheadBy int) *github.CommitsComparison {
	return &github.CommitsComparison{
		AheadBy: &aheadBy,
	}
}

func newComment(body, user string) *github.IssueComment {
	return &github.IssueComment{
		Body: &body,
		User: newUser(user),
	}
}

func newReview(user, state string) *github.PullRequestReview {
	return &github.PullRequestReview{
		User:  newUser(user),
		State: &state,
	}
}

func newLabel(name string) *github.Label {
	return &github.Label{
		Name: &name,
	}
}

func newStatus(context, state string) *github.RepoStatus {
	return &github.RepoStatus{
		Context: &context,
		State:   &state,
	}
}
