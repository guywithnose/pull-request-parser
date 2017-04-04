package command

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestCmdParse(t *testing.T) {
	ts := getParseTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, writer, _ := appWithTestWriters()
	assert.Nil(t, CmdParse(cli.NewContext(app, set, nil)))
	output := strings.Split(writer.String(), "\n")
	sort.Strings(output[1:5])
	assert.Equal(
		t,
		[]string{
			"Repo|ID|Title     |Owner  |Branch |Target     |+1|UTD|Status|Review|Labels",
			"bar |1 |fooPrOne  |fooGuy |fooRef1|fooBaseRef1|3 |Y  |N/Y   |Y     |L,L",
			"bar |2 |fooPrTwo  |fooGuy2|fooRef2|fooBaseRef2|0 |N  |      |Y     |L,L,RLL",
			"rep |1 |prOne     |guy    |ref1   |baseRef1   |1 |N  |Y     |N     |L",
			"rep |2 |Really lon|guy2   |ref2   |baseRef2   |2 |Y  |N     |Y     |",
			"",
		},
		output,
	)
}

func TestCmdParseVerbose(t *testing.T) {
	ts := getParseTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	set.Bool("verbose", true, "doc")
	app, writer, _ := appWithTestWriters()
	assert.Nil(t, CmdParse(cli.NewContext(app, set, nil)))
	output := strings.Split(writer.String(), "\n")
	sort.Strings(output[1:5])
	assert.Equal(
		t,
		[]string{
			"Repo|ID|Title                         |Owner  |Branch |Target     |+1|UTD|Status|Review|Labels",
			"bar |1 |fooPrOne                      |fooGuy |fooRef1|fooBaseRef1|3 |Y  |N/Y   |Y     |label2,label3",
			"bar |2 |fooPrTwo                      |fooGuy2|fooRef2|fooBaseRef2|0 |N  |      |Y     |label4,label5,really-long-label",
			"rep |1 |prOne                         |guy    |ref1   |baseRef1   |1 |N  |Y     |N     |label1",
			"rep |2 |Really long Pull Request Title|guy2   |ref2   |baseRef2   |2 |Y  |N     |Y     |",
			"",
		},
		output,
	)
}

func TestCmdParseNeedRebase(t *testing.T) {
	ts := getParseTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	set.Bool("need-rebase", true, "doc")
	app, writer, _ := appWithTestWriters()
	assert.Nil(t, CmdParse(cli.NewContext(app, set, nil)))
	output := strings.Split(writer.String(), "\n")
	sort.Strings(output[1:3])
	assert.Equal(
		t,
		[]string{
			"Repo|ID|Title   |Owner  |Branch |Target     |+1|UTD|Status|Review|Labels",
			"bar |2 |fooPrTwo|fooGuy2|fooRef2|fooBaseRef2|0 |N  |      |Y     |L,L,RLL",
			"rep |1 |prOne   |guy    |ref1   |baseRef1   |1 |N  |Y     |N     |L",
			"",
		},
		output,
	)
}

func TestCmdParseUserFilter(t *testing.T) {
	ts := getParseTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	set.String("owner", "guy", "doc")
	app, writer, _ := appWithTestWriters()
	assert.Nil(t, CmdParse(cli.NewContext(app, set, nil)))
	output := strings.Split(writer.String(), "\n")
	sort.Strings(output[1:2])
	assert.Equal(
		t,
		[]string{
			"Repo|ID|Title|Owner|Branch|Target  |+1|UTD|Status|Review|Labels",
			"rep |1 |prOne|guy  |ref1  |baseRef1|1 |N  |Y     |N     |L",
			"",
		},
		output,
	)
}

func TestCmdParseRepoFilter(t *testing.T) {
	ts := getParseTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	repoFlag := cli.StringSlice{"foo/bar"}
	set.Var(&repoFlag, "repo", "doc")
	app, writer, _ := appWithTestWriters()
	assert.Nil(t, CmdParse(cli.NewContext(app, set, nil)))
	output := strings.Split(writer.String(), "\n")
	sort.Strings(output[1:3])
	assert.Equal(
		t,
		[]string{
			"Repo|ID|Title   |Owner  |Branch |Target     |+1|UTD|Status|Review|Labels",
			"bar |1 |fooPrOne|fooGuy |fooRef1|fooBaseRef1|3 |Y  |N/Y   |Y     |L,L",
			"bar |2 |fooPrTwo|fooGuy2|fooRef2|fooBaseRef2|0 |N  |      |Y     |L,L,RLL",
			"",
		},
		output,
	)
}

func TestCmdParseRepoFilterMultiple(t *testing.T) {
	ts := getParseTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	repoFlag := cli.StringSlice{"foo/bar", "own/rep"}
	set.Var(&repoFlag, "repo", "doc")
	app, writer, _ := appWithTestWriters()
	assert.Nil(t, CmdParse(cli.NewContext(app, set, nil)))
	output := strings.Split(writer.String(), "\n")
	sort.Strings(output[1:5])
	assert.Equal(
		t,
		[]string{
			"Repo|ID|Title     |Owner  |Branch |Target     |+1|UTD|Status|Review|Labels",
			"bar |1 |fooPrOne  |fooGuy |fooRef1|fooBaseRef1|3 |Y  |N/Y   |Y     |L,L",
			"bar |2 |fooPrTwo  |fooGuy2|fooRef2|fooBaseRef2|0 |N  |      |Y     |L,L,RLL",
			"rep |1 |prOne     |guy    |ref1   |baseRef1   |1 |N  |Y     |N     |L",
			"rep |2 |Really lon|guy2   |ref2   |baseRef2   |2 |Y  |N     |Y     |",
			"",
		},
		output,
	)
}

func TestCmdParseNoConfig(t *testing.T) {
	set := flag.NewFlagSet("test", 0)
	app := cli.NewApp()
	err := CmdParse(cli.NewContext(app, set, nil))
	assert.EqualError(t, err, "You must specify a config file")
}

func TestCmdParseNoProfile(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := flag.NewFlagSet("test", 0)
	set.String("config", configFileName, "doc")
	set.String("profile", "bar", "doc")
	app := cli.NewApp()
	err := CmdParse(cli.NewContext(app, set, nil))
	assert.EqualError(t, err, "Invalid Profile: bar")
}

func TestCmdParseUsage(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	assert.Nil(t, set.Parse([]string{"foo"}))
	app := cli.NewApp()
	err := CmdParse(cli.NewContext(app, set, nil))
	assert.EqualError(t, err, "Usage: \"prp parse\"")
}

func TestCmdParseBadApiUrl(t *testing.T) {
	_, configFileName := getConfigWithAPIURL(t, "%s/mockApi")
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app := cli.NewApp()
	err := CmdParse(cli.NewContext(app, set, nil))
	assert.EqualError(t, err, "parse %s/mockApi: invalid URL escape \"%s/\"")
}

func TestCmdParseUserFailure(t *testing.T) {
	ts := getParseTestServer("/user")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app := cli.NewApp()
	err := CmdParse(cli.NewContext(app, set, nil))
	assert.EqualError(t, err, fmt.Sprintf("GET %s/user: 500  []", ts.URL))
}

func TestCmdParsePullRequestFailure(t *testing.T) {
	ts := getParseTestServer("/repos/own/rep/pulls?per_page=100")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, writer, _ := appWithTestWriters()
	assert.Nil(t, CmdParse(cli.NewContext(app, set, nil)))
	output := strings.Split(writer.String(), "\n")
	sort.Strings(output[1:3])
	assert.Equal(
		t,
		[]string{
			"Repo|ID|Title   |Owner  |Branch |Target     |+1|UTD|Status|Review|Labels",
			"bar |1 |fooPrOne|fooGuy |fooRef1|fooBaseRef1|3 |Y  |N/Y   |Y     |L,L",
			"bar |2 |fooPrTwo|fooGuy2|fooRef2|fooBaseRef2|0 |N  |      |Y     |L,L,RLL",
			"",
		},
		output,
	)
}

func TestCmdParseStatusFailure(t *testing.T) {
	ts := getParseTestServer("/repos/own/rep/commits/sha1/statuses")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, writer, _ := appWithTestWriters()
	assert.Nil(t, CmdParse(cli.NewContext(app, set, nil)))
	output := strings.Split(writer.String(), "\n")
	sort.Strings(output[1:5])
	assert.Equal(
		t,
		[]string{
			"Repo|ID|Title     |Owner  |Branch |Target     |+1|UTD|Status|Review|Labels",
			"bar |1 |fooPrOne  |fooGuy |fooRef1|fooBaseRef1|3 |Y  |N/Y   |Y     |L,L",
			"bar |2 |fooPrTwo  |fooGuy2|fooRef2|fooBaseRef2|0 |N  |      |Y     |L,L,RLL",
			"rep |1 |prOne     |guy    |ref1   |baseRef1   |1 |N  |      |N     |L",
			"rep |2 |Really lon|guy2   |ref2   |baseRef2   |2 |Y  |N     |Y     |",
			"",
		},
		output,
	)
}

func TestCmdParseLabelFailure(t *testing.T) {
	ts := getParseTestServer("/repos/own/rep/issues/1/labels")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, writer, _ := appWithTestWriters()
	assert.Nil(t, CmdParse(cli.NewContext(app, set, nil)))
	output := strings.Split(writer.String(), "\n")
	sort.Strings(output[1:5])
	assert.Equal(
		t,
		[]string{
			"Repo|ID|Title     |Owner  |Branch |Target     |+1|UTD|Status|Review|Labels",
			"bar |1 |fooPrOne  |fooGuy |fooRef1|fooBaseRef1|3 |Y  |N/Y   |Y     |L,L",
			"bar |2 |fooPrTwo  |fooGuy2|fooRef2|fooBaseRef2|0 |N  |      |Y     |L,L,RLL",
			"rep |1 |prOne     |guy    |ref1   |baseRef1   |1 |N  |Y     |N     |",
			"rep |2 |Really lon|guy2   |ref2   |baseRef2   |2 |Y  |N     |Y     |",
			"",
		},
		output,
	)
}

func TestCmdParseCommentFailure(t *testing.T) {
	ts := getParseTestServer("/repos/own/rep/issues/1/comments?per_page=100")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, writer, _ := appWithTestWriters()
	assert.Nil(t, CmdParse(cli.NewContext(app, set, nil)))
	output := strings.Split(writer.String(), "\n")
	sort.Strings(output[1:5])
	assert.Equal(
		t,
		[]string{
			"Repo|ID|Title     |Owner  |Branch |Target     |+1|UTD|Status|Review|Labels",
			"bar |1 |fooPrOne  |fooGuy |fooRef1|fooBaseRef1|3 |Y  |N/Y   |Y     |L,L",
			"bar |2 |fooPrTwo  |fooGuy2|fooRef2|fooBaseRef2|0 |N  |      |Y     |L,L,RLL",
			"rep |1 |prOne     |guy    |ref1   |baseRef1   |0 |N  |Y     |Y     |L",
			"rep |2 |Really lon|guy2   |ref2   |baseRef2   |2 |Y  |N     |Y     |",
			"",
		},
		output,
	)
}

func TestCmdParseCommitCompareFailure(t *testing.T) {
	ts := getParseTestServer("/repos/foo/bar/compare/fooLabel...fooBaseLabel1")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, writer, _ := appWithTestWriters()
	assert.Nil(t, CmdParse(cli.NewContext(app, set, nil)))
	output := strings.Split(writer.String(), "\n")
	sort.Strings(output[1:5])
	assert.Equal(
		t,
		[]string{
			"Repo|ID|Title     |Owner  |Branch |Target     |+1|UTD|Status|Review|Labels",
			"bar |1 |fooPrOne  |fooGuy |fooRef1|fooBaseRef1|3 |N  |N/Y   |Y     |L,L",
			"bar |2 |fooPrTwo  |fooGuy2|fooRef2|fooBaseRef2|0 |N  |      |Y     |L,L,RLL",
			"rep |1 |prOne     |guy    |ref1   |baseRef1   |1 |N  |Y     |N     |L",
			"rep |2 |Really lon|guy2   |ref2   |baseRef2   |2 |Y  |N     |Y     |",
			"",
		},
		output,
	)
}

func TestCompleteParseFlags(t *testing.T) {
	_, configFileName := getConfigWithFooProfile(t)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, writer, _ := appWithTestWriters()
	app.Commands = []cli.Command{
		{
			Name: "parse",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "owner, o"},
				cli.StringFlag{Name: "repo, r"},
				cli.BoolFlag{Name: "need-rebase, nr"},
				cli.BoolFlag{Name: "verbose, v"},
			},
		},
	}
	os.Args = []string{"parse", "--completion"}
	CompleteParse(cli.NewContext(app, set, nil))
	assert.Equal(t, "--owner\n--repo\n--need-rebase\n--verbose\n", writer.String())
}

func TestCompleteParseUser(t *testing.T) {
	ts := getParseTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, writer, _ := appWithTestWriters()
	os.Args = []string{"parse", "--user", "--completion"}
	CompleteParse(cli.NewContext(app, set, nil))
	assert.Equal(t, "fooGuy\nfooGuy2\nguy\nguy2\n", writer.String())
}

func TestCompleteParseUserNoConfig(t *testing.T) {
	ts := getParseTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := flag.NewFlagSet("test", 0)
	app, writer, _ := appWithTestWriters()
	os.Args = []string{"parse", "--user", "--completion"}
	CompleteParse(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}

func TestCompleteParseUserBadApiUrl(t *testing.T) {
	_, configFileName := getConfigWithAPIURL(t, "%s/mockApi")
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, writer, _ := appWithTestWriters()
	os.Args = []string{"parse", "--user", "--completion"}
	CompleteParse(cli.NewContext(app, set, nil))
	assert.Equal(t, "", writer.String())
}

func TestCompleteParseUserPullRequestFailure(t *testing.T) {
	ts := getParseTestServer("/repos/own/rep/pulls?per_page=100")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, writer, _ := appWithTestWriters()
	os.Args = []string{"parse", "--user", "--completion"}
	CompleteParse(cli.NewContext(app, set, nil))
	assert.Equal(t, "fooGuy\nfooGuy2\n", writer.String())
}

func TestCompleteParseRepo(t *testing.T) {
	ts := getParseTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	app, writer, _ := appWithTestWriters()
	os.Args = []string{"parse", "--repo", "--completion"}
	CompleteParse(cli.NewContext(app, set, nil))
	assert.Equal(t, "foo/bar\nown/rep\n", writer.String())
}

func TestCompleteParseRepoMulti(t *testing.T) {
	ts := getParseTestServer("")
	defer ts.Close()
	_, configFileName := getConfigWithAPIURL(t, ts.URL)
	defer removeFile(t, configFileName)
	set := getBaseFlagSet(configFileName)
	repoFlag := cli.StringSlice{"foo/bar"}
	set.Var(&repoFlag, "repo", "doc")
	app, writer, _ := appWithTestWriters()
	os.Args = []string{"parse", "--repo", "foo/bar", "--repo", "--completion"}
	CompleteParse(cli.NewContext(app, set, nil))
	assert.Equal(t, "own/rep\n", writer.String())
}

func getParseTestServer(failureURL string) *httptest.Server {
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

		response = handlePullRequestRequests(r, w, server)
		if response != nil {
			fmt.Fprint(w, *response)
			return
		}

		response = handleCommentRequests(r, w, server)
		if response != nil {
			fmt.Fprint(w, *response)
			return
		}

		response = handleLabelRequests(r)
		if response != nil {
			fmt.Fprint(w, *response)
			return
		}

		response = handleStatusRequests(r)
		if response != nil {
			fmt.Fprint(w, *response)
			return
		}

		response = handleCommitsComparisonRequests(r)
		if response != nil {
			fmt.Fprint(w, *response)
			return
		}

		panic(r.URL.String())
	}))

	return server
}
