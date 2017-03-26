package command

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/google/go-github/github"
	"github.com/guywithnose/pull-request-parser/config"
	"github.com/urfave/cli"
)

type prInfo struct {
	RepoName        string
	RepoOwner       string
	PullRequestID   int
	Title           string
	Owner           string
	Branch          string
	TargetBranch    string
	HeadLabel       string
	BaseLabel       string
	SHA             string
	Approvals       int
	Rebased         bool
	BuildInfo       map[string]bool
	NeedsMyApproval bool
	Labels          []string
	IgnoredBuilds   []string
	Hidden          bool
}

type sortablePrs []*prInfo

func (s sortablePrs) Len() int {
	return len(s)
}

func (s sortablePrs) Less(i, j int) bool {
	if s[i].RepoName == s[j].RepoName {
		return s[i].PullRequestID < s[j].PullRequestID
	}

	return s[i].RepoName < s[j].RepoName
}

func (s sortablePrs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// CmdParse parses the pull requests
func CmdParse(c *cli.Context) error {
	configData, profileName, err := loadProfile(c)
	if err != nil {
		return err
	}

	if c.NArg() != 0 {
		return cli.NewExitError("Usage: \"prp parse\"", 1)
	}

	profile := configData.Profiles[*profileName]
	ctx := context.Background()
	client, err := getGithubClient(ctx, &profile.Token, &profile.APIURL)
	if err != nil {
		return err
	}

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return err
	}

	outputs := getBasePrData(ctx, client, user, &profile)

	sort.Sort(outputs)

	filteredOutputs := filterOutputs(outputs, c.String("owner"), c.StringSlice("repo"))

	getExtraData(ctx, client, user, c.Bool("need-rebase"), filteredOutputs)

	return printResults(filteredOutputs, c.Bool("verbose"), c.App.Writer)
}

func filterOutputs(outputs []*prInfo, owner string, repos []string) []*prInfo {
	filteredOutputs := make([]*prInfo, 0, len(outputs))
	for _, output := range outputs {
		if owner != "" && output.Owner != owner {
			continue
		}

		if len(repos) != 0 {
			matched := false
			for _, repoName := range repos {
				if repoName == fmt.Sprintf("%s/%s", output.RepoOwner, output.RepoName) {
					matched = true
				}
			}

			if !matched {
				continue
			}
		}

		filteredOutputs = append(filteredOutputs, output)
	}

	return filteredOutputs
}

func getBasePrData(ctx context.Context, client *github.Client, user *github.User, profile *config.PrpConfigProfile) sortablePrs {
	outputs := sortablePrs{}
	outputChannel := make(chan *prInfo)
	wg := sync.WaitGroup{}
	for _, repo := range profile.TrackedRepos {
		wg.Add(1)
		go func(repo config.PrpConfigRepo) {
			repoPrs, err := getRepoPullRequests(ctx, client, repo.Owner, repo.Name)
			if err != nil {
				wg.Done()
				return
			}

			for _, pr := range repoPrs {
				wg.Add(1)
				outputChannel <- &prInfo{
					RepoOwner:       repo.Owner,
					RepoName:        repo.Name,
					PullRequestID:   pr.GetNumber(),
					Title:           pr.GetTitle(),
					Owner:           pr.Head.User.GetLogin(),
					Branch:          pr.Head.GetRef(),
					TargetBranch:    pr.Base.GetRef(),
					HeadLabel:       pr.Head.GetLabel(),
					BaseLabel:       pr.Base.GetLabel(),
					SHA:             pr.Head.GetSHA(),
					BuildInfo:       map[string]bool{},
					NeedsMyApproval: user.GetLogin() != pr.Head.User.GetLogin(),
					IgnoredBuilds:   repo.IgnoredBuilds,
				}
			}
			wg.Done()
		}(repo)
	}

	go func() {
		for {
			outputs = append(outputs, <-outputChannel)
			wg.Done()
		}
	}()

	wg.Wait()
	return outputs
}

func getExtraData(ctx context.Context, client *github.Client, user *github.User, filterRebased bool, outputs []*prInfo) {
	wg := sync.WaitGroup{}
	for _, output := range outputs {
		wg.Add(1)
		go func(output *prInfo) {
			handleCommitComparision(ctx, client, output, filterRebased)
			wg.Done()
		}(output)

		wg.Add(1)
		go func(output *prInfo) {
			handleComments(ctx, client, user, output)
			wg.Done()
		}(output)

		wg.Add(1)
		go func(output *prInfo) {
			handleLabels(ctx, client, output)
			wg.Done()
		}(output)

		wg.Add(1)
		go func(output *prInfo) {
			handleStatuses(ctx, client, output)
			wg.Done()
		}(output)
	}

	wg.Wait()
}

// CompleteParse handles bash autocompletion for the 'parse' command
func CompleteParse(c *cli.Context) {
	lastParam := os.Args[len(os.Args)-2]
	if lastParam != "--user" && lastParam != "--repo" {
		for _, flag := range c.App.Command("parse").Flags {
			name := strings.Split(flag.GetName(), ",")[0]
			if !c.IsSet(name) {
				fmt.Fprintf(c.App.Writer, "--%s\n", name)
			}
		}
		return
	}

	configData, profileName, err := loadProfile(c)
	if err != nil {
		return
	}

	profile := configData.Profiles[*profileName]
	if lastParam == "--user" {
		handleUserCompletion(&profile, c.App.Writer)
		return
	}

	for _, repo := range profile.TrackedRepos {
		fmt.Fprintf(c.App.Writer, "%s/%s\n", repo.Owner, repo.Name)
	}
}

func handleUserCompletion(profile *config.PrpConfigProfile, writer io.Writer) {
	ctx := context.Background()
	client, err := getGithubClient(ctx, &profile.Token, &profile.APIURL)
	if err != nil {
		return
	}

	suggestionList := []string{}
	suggestionChan := make(chan string)
	wg := sync.WaitGroup{}
	for _, repo := range profile.TrackedRepos {
		wg.Add(1)
		go func(repo config.PrpConfigRepo) {
			repoPrs, err := getRepoPullRequests(ctx, client, repo.Owner, repo.Name)
			if err != nil {
				wg.Done()
				return
			}

			for _, pr := range repoPrs {
				wg.Add(1)
				suggestionChan <- pr.Head.User.GetLogin()
			}

			wg.Done()
		}(repo)
	}

	go func() {
		for {
			newSuggestion := <-suggestionChan
			suggestionList = append(suggestionList, newSuggestion)
			wg.Done()
		}
	}()

	wg.Wait()

	suggestionList = unique(suggestionList)
	sort.Strings(suggestionList)
	fmt.Fprintln(writer, strings.Join(suggestionList, "\n"))
}

func printResults(outputs []*prInfo, verbose bool, w io.Writer) error {
	tabW := tabwriter.NewWriter(w, 0, 0, 0, ' ', tabwriter.Debug)
	fmt.Fprintln(tabW, "Repo\tID\tTitle\tOwner\tBranch\tTarget\t+1\tUTD\tStatus\tReview\tLabels")
	for _, po := range outputs {
		if !po.Hidden {
			title := po.Title
			if !verbose {
				title = fmt.Sprintf("%.10s", title)
			}

			labels := strings.Join(po.Labels, ",")
			if !verbose {
				shortLabels := []string{}
				for _, label := range po.Labels {
					shortLabels = append(shortLabels, shortenLabel(label))
				}

				labels = strings.Join(shortLabels, ",")
			}

			fmt.Fprintf(
				tabW,
				"%s\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				po.RepoName,
				po.PullRequestID,
				title,
				po.Owner,
				po.Branch,
				po.TargetBranch,
				strconv.Itoa(po.Approvals),
				boolToString(po.Rebased),
				buildStatus(po.BuildInfo),
				boolToString(po.NeedsMyApproval),
				labels,
			)
		}
	}

	return tabW.Flush()
}

func buildStatus(contexts map[string]bool) string {
	keys := make([]string, 0, len(contexts))
	for key := range contexts {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	var status []string
	for _, key := range keys {
		if contexts[key] {
			status = append(status, "Y")
		} else {
			status = append(status, "N")
		}
	}

	return strings.Join(status, "/")
}

func boolToString(status bool) string {
	if status {
		return "Y"
	}

	return "N"
}

func getRepoPullRequests(ctx context.Context, client *github.Client, owner, name string) ([]*github.PullRequest, error) {
	opt := &github.PullRequestListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	allPrs := []*github.PullRequest{}
	for {
		pullRequests, resp, err := client.PullRequests.List(ctx, owner, name, opt)
		if err != nil {
			return nil, err
		}

		allPrs = append(allPrs, pullRequests...)
		if resp.NextPage == 0 {
			return allPrs, nil
		}

		opt.ListOptions.Page = resp.NextPage
	}
}

func handleComments(ctx context.Context, client *github.Client, user *github.User, output *prInfo) {
	//TODO Also look at the new review comments
	opt := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	allComments := []*github.IssueComment{}
	for {
		comments, resp, err := client.Issues.ListComments(ctx, output.RepoOwner, output.RepoName, output.PullRequestID, opt)
		if err != nil {
			return
		}

		allComments = append(allComments, comments...)
		if resp.NextPage == 0 {
			break
		}

		opt.ListOptions.Page = resp.NextPage
	}

	for _, comment := range allComments {
		if strings.Contains(comment.GetBody(), ":+1:") || strings.Contains(comment.GetBody(), ":thumbsup:") || strings.Contains(comment.GetBody(), "LGTM") {
			output.Approvals++
			if comment.User.GetLogin() == user.GetLogin() {
				output.NeedsMyApproval = false
			}
		}
	}
}

func handleCommitComparision(ctx context.Context, client *github.Client, output *prInfo, filterRebased bool) {
	commitComparison, _, err := client.Repositories.CompareCommits(ctx, output.RepoOwner, output.RepoName, output.HeadLabel, output.BaseLabel)
	if err != nil {
		return
	}

	output.Rebased = commitComparison.GetAheadBy() == 0
	if filterRebased && output.Rebased {
		output.Hidden = true
	}
}

func handleLabels(ctx context.Context, client *github.Client, output *prInfo) {
	labels, _, err := client.Issues.ListLabelsByIssue(ctx, output.RepoOwner, output.RepoName, output.PullRequestID, nil)
	if err != nil {
		return
	}

	for _, label := range labels {
		output.Labels = append(output.Labels, label.GetName())
	}
}

func handleStatuses(ctx context.Context, client *github.Client, output *prInfo) {
	statuses, _, err := client.Repositories.ListStatuses(ctx, output.RepoOwner, output.RepoName, output.SHA, nil)
	if err != nil {
		return
	}

statusFor:
	for _, status := range statuses {
		for _, ignoredBuild := range output.IgnoredBuilds {
			if ignoredBuild == status.GetContext() {
				continue statusFor
			}
		}

		if _, ok := output.BuildInfo[status.GetContext()]; !ok {
			output.BuildInfo[status.GetContext()] = false
		}

		if status.GetState() == "success" {
			output.BuildInfo[status.GetContext()] = true
		}
	}
}

func shortenLabel(label string) string {
	parts := strings.Split(label, "-")
	initials := []string{}
	for _, part := range parts {
		initials = append(initials, strings.ToUpper(string(part[0])))
	}

	return strings.Join(initials, "")
}
