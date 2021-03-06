package command_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/go-github/github"
	"github.com/guywithnose/pull-request-parser/command"
	"github.com/guywithnose/pull-request-parser/config"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestCmdRepoAdd(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"foo", "bar"}))
	assert.Nil(t, command.CmdRepoAdd(cli.NewContext(nil, set, nil)))
	assert.Nil(t, set.Parse([]string{"own", "rep"}))
	assert.Nil(t, command.CmdRepoAdd(cli.NewContext(nil, set, nil)))

	modifiedConfigData, err := config.LoadFromFile(configFileName)
	assert.Nil(t, err)

	expectedConfigFile, configFileName := getConfigWithTwoRepos(t)
	defer removeFile(t, configFileName)
	assert.Equal(t, expectedConfigFile, *modifiedConfigData)
}

func TestCmdRepoAddNoConfig(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	err := command.CmdRepoAdd(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "You must specify a config file")
}

func TestCmdRepoAddUsage(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := flag.NewFlagSet("test", 0)
	set.String("config", configFileName, "doc")
	set.String("profile", "foo", "doc")
	err := command.CmdRepoAdd(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "Usage: \"prp profile repo add {owner} {repoName}\"")
}

func TestCmdRepoAddAlreadyTracked(t *testing.T) {
	_, configFileName := getConfigWithTwoRepos(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"own", "rep"}))
	err := command.CmdRepoAdd(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "own/rep is already tracked")
}

func TestCompleteRepoAddOwner(t *testing.T) {
	ts := getRepoAddTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	os.Args = []string{"repo", "add", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteRepoAdd(cli.NewContext(app, set, nil))
	assert.Equal(t, "bar\nown\nsource\n", writer.String())
}

func TestCompleteRepoAddName(t *testing.T) {
	ts := getRepoAddTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"source"}))
	os.Args = []string{"repo", "add", "source", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteRepoAdd(cli.NewContext(app, set, nil))
	assert.Equal(t, "rep\n", writer.String())
}

func TestCompleteRepoAddNameOwnRepos(t *testing.T) {
	ts := getRepoAddTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"own"}))
	os.Args = []string{"repo", "add", "own", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteRepoAdd(cli.NewContext(app, set, nil))
	assert.Equal(t, "foo\n", writer.String())
}

func TestCompleteRepoAddNameAlreadyTracked(t *testing.T) {
	ts := getRepoAddTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"foo"}))
	os.Args = []string{"repo", "add", "foo", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteRepoAdd(cli.NewContext(app, set, nil))
	assert.Equal(t, "\n", writer.String())
}

func TestCompleteRepoAddDone(t *testing.T) {
	ts := getRepoAddTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"foo", "bar"}))
	os.Args = []string{"repo", "add", "foo", "bar", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteRepoAdd(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}

func TestCompleteRepoAddNoConfig(t *testing.T) {
	ts := getRepoAddTestServer("")
	defer ts.Close()
	set := flag.NewFlagSet("test", 0)
	set.String("profile", "foo", "doc")
	os.Args = []string{"repo", "add", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteRepoAdd(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}

func TestCompleteRepoAddUserFailure(t *testing.T) {
	ts := getRepoAddTestServer("/user")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	os.Args = []string{"repo", "add", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteRepoAdd(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}

func TestCompleteRepoAddReposFailure(t *testing.T) {
	ts := getRepoAddTestServer("/users/own/repos?per_page=100")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	os.Args = []string{"repo", "add", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteRepoAdd(cli.NewContext(app, set, nil))
	assert.Equal(t, "\n", writer.String())
}

func TestCompleteRepoAddRepoFailure(t *testing.T) {
	ts := getRepoAddTestServer("/repos/own/rep")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	os.Args = []string{"repo", "add", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteRepoAdd(cli.NewContext(app, set, nil))
	assert.Equal(t, "bar\nown\n", writer.String())
}

func TestCompleteRepoAddBadApiUrl(t *testing.T) {
	_, configFileName := getConfigWithAPIURL(t, "%s/mockApi")
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	os.Args = []string{"repo", "add", "--completion"}
	app, writer, _ := appWithTestWriters()
	command.CompleteRepoAdd(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}

func getRepoAddTestServer(failureURL string) *httptest.Server {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == failureURL {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response := handleUserRequest(r, "own")
		if response != nil {
			fmt.Fprint(w, *response)
			return
		}

		responses := make(map[string]string)
		repos := []*github.Repository{
			newRepository("own", "rep", true),
			newRepository("own", "bar", true),
			newRepository("own", "foo", false),
			newRepository("bar", "goo", false),
		}
		bytes, _ := json.Marshal(repos)
		if r.URL.String() == "/users/own/repos?per_page=100" {
			w.Header().Set(
				"Link",
				fmt.Sprintf(
					`<%s/mockApi/users/own/repos?per_page=100&page=2>; rel="next", `+
						`<%s/mockApi/users/own/repos?per_page=100&page=2>; rel="last"`,
					server.URL,
					server.URL,
				),
			)
		}
		responses["/users/own/repos?per_page=100"] = string(bytes)

		bytes, _ = json.Marshal([]*github.Repository{})
		responses["/users/own/repos?page=2&per_page=100"] = string(bytes)

		sourceRepo := newRepository("foo", "bar", false)
		bytes, _ = json.Marshal(newRepositoryWithSource(sourceRepo))
		responses["/repos/own/bar"] = string(bytes)

		sourceRepo = newRepository("source", "rep", false)
		bytes, _ = json.Marshal(newRepositoryWithSource(sourceRepo))
		responses["/repos/own/rep"] = string(bytes)

		resp, ok := responses[r.URL.String()]
		if ok {
			fmt.Fprint(w, resp)
			return
		}

		panic(r.URL.String())
	}))

	return server
}

func newRepository(owner, name string, fork bool) *github.Repository {
	return &github.Repository{
		Owner: &github.User{Login: &owner},
		Fork:  &fork,
		Name:  &name,
	}
}

func newRepositoryWithSource(source *github.Repository) *github.Repository {
	owner := "own"
	name := "rep"
	fork := true
	return &github.Repository{
		Source: source,
		Owner:  &github.User{Login: &owner},
		Fork:   &fork,
		Name:   &name,
	}
}
