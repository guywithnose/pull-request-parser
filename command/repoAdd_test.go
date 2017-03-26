package command

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/google/go-github/github"
	"github.com/guywithnose/pull-request-parser/config"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestCmdRepoAdd(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"foo", "bar"}))
	assert.Nil(t, CmdRepoAdd(cli.NewContext(nil, set, nil)))
	assert.Nil(t, set.Parse([]string{"own", "rep"}))
	assert.Nil(t, CmdRepoAdd(cli.NewContext(nil, set, nil)))

	modifiedConfigData, err := config.LoadConfigFromFile(configFileName)
	assert.Nil(t, err)

	expectedConfigFile, configFileName := getConfigWithTwoRepos(t)
	defer removeFile(t, configFileName)
	if !reflect.DeepEqual(*modifiedConfigData, expectedConfigFile) {
		t.Fatalf("File was \n%v\n, expected \n%v\n", *modifiedConfigData, expectedConfigFile)
	}
}

func TestCmdRepoAddNoConfig(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	err := CmdRepoAdd(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "You must specify a config file")
}

func TestCmdRepoAddUsage(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := flag.NewFlagSet("test", 0)
	set.String("config", configFileName, "doc")
	set.String("profile", "foo", "doc")
	err := CmdRepoAdd(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "Usage: \"prp profile repo add {owner} {repoName}\"")
}

func TestCmdRepoAddAlreadyTracked(t *testing.T) {
	_, configFileName := getConfigWithTwoRepos(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"own", "rep"}))
	err := CmdRepoAdd(cli.NewContext(nil, set, nil))
	assert.EqualError(t, err, "own/rep is already tracked")
}

func TestCompleteRepoAddOwner(t *testing.T) {
	ts := getRepoAddTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	os.Args = []string{"repo", "add", "--completion"}
	app, writer := appWithTestWriter()
	CompleteRepoAdd(cli.NewContext(app, set, nil))
	assertOutput(t, writer, "source\n")
}

func TestCompleteRepoAddName(t *testing.T) {
	ts := getRepoAddTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"source"}))
	os.Args = []string{"repo", "add", "source", "--completion"}
	app, writer := appWithTestWriter()
	CompleteRepoAdd(cli.NewContext(app, set, nil))
	assertOutput(t, writer, "rep\n")
}

func TestCompleteRepoAddNameAlreadyTracked(t *testing.T) {
	ts := getRepoAddTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"foo"}))
	os.Args = []string{"repo", "add", "foo", "--completion"}
	app, writer := appWithTestWriter()
	CompleteRepoAdd(cli.NewContext(app, set, nil))
	assertOutput(t, writer, "\n")
}

func TestCompleteRepoAddDone(t *testing.T) {
	ts := getRepoAddTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"foo", "bar"}))
	os.Args = []string{"repo", "add", "foo", "bar", "--completion"}
	app, writer := appWithTestWriter()
	CompleteRepoAdd(cli.NewContext(app, set, nil))
	assertOutput(t, writer, "")
}

func TestCompleteRepoAddNoConfig(t *testing.T) {
	ts := getRepoAddTestServer("")
	defer ts.Close()
	set := flag.NewFlagSet("test", 0)
	set.String("profile", "foo", "doc")
	os.Args = []string{"repo", "add", "--completion"}
	app, writer := appWithTestWriter()
	CompleteRepoAdd(cli.NewContext(app, set, nil))
	assertOutput(t, writer, "")
}

func TestCompleteRepoAddUserFailure(t *testing.T) {
	ts := getRepoAddTestServer("/user")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	os.Args = []string{"repo", "add", "--completion"}
	app, writer := appWithTestWriter()
	CompleteRepoAdd(cli.NewContext(app, set, nil))
	assertOutput(t, writer, "")
}

func TestCompleteRepoAddReposFailure(t *testing.T) {
	ts := getRepoAddTestServer("/users/own/repos?per_page=100")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	os.Args = []string{"repo", "add", "--completion"}
	app, writer := appWithTestWriter()
	CompleteRepoAdd(cli.NewContext(app, set, nil))
	assertOutput(t, writer, "\n")
}

func TestCompleteRepoAddRepoFailure(t *testing.T) {
	ts := getRepoAddTestServer("/repos/own/rep")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	os.Args = []string{"repo", "add", "--completion"}
	app, writer := appWithTestWriter()
	CompleteRepoAdd(cli.NewContext(app, set, nil))
	assertOutput(t, writer, "\n")
}

func TestCompleteRepoAddBadApiUrl(t *testing.T) {
	_, configFileName := getConfigWithAPIURL(t, "%s/mockApi")
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	os.Args = []string{"repo", "add", "--completion"}
	app, writer := appWithTestWriter()
	CompleteRepoAdd(cli.NewContext(app, set, nil))
	assertOutput(t, writer, "")
}

func getRepoAddTestServer(failureURL string) *httptest.Server {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == failureURL {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response := handleUserRequest(r)
		if response != nil {
			fmt.Fprint(w, *response)
			return
		}

		if r.URL.String() == "/users/own/repos?per_page=100" {
			repos := []*github.Repository{
				newRepository("own", "rep", true),
				newRepository("own", "bar", true),
				newRepository("own", "foo", false),
			}
			bytes, _ := json.Marshal(repos)
			w.Header().Set("Link", fmt.Sprintf(`<%s/mockApi/users/own/repos?per_page=100&page=2>; rel="next", <%s/mockApi/users/own/repos?per_page=100&page=2>; rel="last"`, server.URL, server.URL))
			fmt.Fprint(w, string(bytes))
			return
		}

		if r.URL.String() == "/users/own/repos?page=2&per_page=100" {
			bytes, _ := json.Marshal([]*github.Repository{})
			fmt.Fprint(w, string(bytes))
			return
		}

		if r.URL.String() == "/repos/own/bar" {
			sourceRepo := newRepository("foo", "bar", false)
			bytes, _ := json.Marshal(newRepositoryWithSource("own", "rep", true, sourceRepo))
			fmt.Fprint(w, string(bytes))
			return
		}

		if r.URL.String() == "/repos/own/rep" {
			sourceRepo := newRepository("source", "rep", false)
			bytes, _ := json.Marshal(newRepositoryWithSource("own", "rep", true, sourceRepo))
			fmt.Fprint(w, string(bytes))
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

func newRepositoryWithSource(owner, name string, fork bool, source *github.Repository) *github.Repository {
	return &github.Repository{
		Source: source,
		Owner:  &github.User{Login: &owner},
		Fork:   &fork,
		Name:   &name,
	}
}
