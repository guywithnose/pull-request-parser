package command

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/guywithnose/pull-request-parser/config"
	"golang.org/x/oauth2"
)

type prInfo struct {
	Repo            *config.PrpConfigRepo
	PullRequestID   int
	Title           string
	Owner           string
	Branch          string
	TargetBranch    string
	HeadLabel       string
	BaseLabel       string
	SHA             string
	BaseSSHURL      string
	HeadSSHURL      string
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
	if s[i].Repo.Name == s[j].Repo.Name {
		return s[i].PullRequestID < s[j].PullRequestID
	}

	return s[i].Repo.Name < s[j].Repo.Name
}

func (s sortablePrs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func getGithubClient(ctx context.Context, token, apiURL *string, useCache bool) (*github.Client, error) {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: *token})
	tokenClient := oauth2.NewClient(ctx, tokenSource)

	if useCache {
		cache := diskcache.New(fmt.Sprintf("%s/prpCache", os.TempDir()))
		transport := httpcache.NewTransport(cache)
		transport.Transport = tokenClient.Transport
		tokenClient.Transport = transport
	}

	client := github.NewClient(tokenClient)
	if apiURL != nil && *apiURL != "" {
		url, err := url.Parse(*apiURL)
		if err != nil {
			return nil, err
		}

		client.BaseURL = url
	}

	return client, nil
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
		comments, resp, err := client.Issues.ListComments(ctx, output.Repo.Owner, output.Repo.Name, output.PullRequestID, opt)
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
	commitComparison, _, err := client.Repositories.CompareCommits(ctx, output.Repo.Owner, output.Repo.Name, output.HeadLabel, output.BaseLabel)
	if err != nil {
		return
	}

	output.Rebased = commitComparison.GetAheadBy() == 0
	if filterRebased && output.Rebased {
		output.Hidden = true
	}
}

func handleLabels(ctx context.Context, client *github.Client, output *prInfo) {
	labels, _, err := client.Issues.ListLabelsByIssue(ctx, output.Repo.Owner, output.Repo.Name, output.PullRequestID, nil)
	if err != nil {
		return
	}

	for _, label := range labels {
		output.Labels = append(output.Labels, label.GetName())
	}
}

func handleStatuses(ctx context.Context, client *github.Client, output *prInfo) {
	statuses, _, err := client.Repositories.ListStatuses(ctx, output.Repo.Owner, output.Repo.Name, output.SHA, nil)
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
					Repo:            &repo,
					PullRequestID:   pr.GetNumber(),
					Title:           pr.GetTitle(),
					Owner:           pr.Head.User.GetLogin(),
					Branch:          pr.Head.GetRef(),
					TargetBranch:    pr.Base.GetRef(),
					HeadLabel:       pr.Head.GetLabel(),
					BaseLabel:       pr.Base.GetLabel(),
					SHA:             pr.Head.GetSHA(),
					BaseSSHURL:      pr.Base.Repo.GetSSHURL(),
					HeadSSHURL:      pr.Head.Repo.GetSSHURL(),
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

func getAllRepos(ctx context.Context, client *github.Client, login string) []*github.Repository {
	allRepos := make([]*github.Repository, 0, 30)
	opt := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		repositories, resp, err := client.Repositories.List(ctx, login, opt)
		if err != nil {
			return []*github.Repository{}
		}

		allRepos = append(allRepos, repositories...)
		if resp.NextPage == 0 {
			return allRepos
		}

		opt.ListOptions.Page = resp.NextPage
	}
}
