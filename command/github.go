package command

import (
	"context"
	"fmt"
	"io"
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
	Hidden          bool
	NeedsMyApproval bool
	BuildInfo       map[string]bool
	Labels          []string
	IgnoredBuilds   []string
}

func getGithubClient(token, apiURL *string, useCache bool) (*github.Client, error) {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: *token})
	tokenClient := oauth2.NewClient(context.Background(), tokenSource)

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

func getRepoPullRequests(client *github.Client, owner, name string) (chan *github.PullRequest, chan error) {
	opt := &github.PullRequestListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	allPrs := make(chan *github.PullRequest)
	errors := make(chan error)
	go func() {
		for {
			pullRequests, resp, err := client.PullRequests.List(context.Background(), owner, name, opt)
			if err != nil {
				errors <- err
				close(errors)
				close(allPrs)
				return
			}

			for _, pr := range pullRequests {
				allPrs <- pr
			}

			if resp.NextPage == 0 {
				close(allPrs)
				close(errors)
				return
			}

			opt.ListOptions.Page = resp.NextPage
		}
	}()

	return allPrs, errors
}

func handleComments(client *github.Client, user *github.User, output *prInfo) {
	//TODO Also look at the new review comments
	opt := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	allComments := []*github.IssueComment{}
	for {
		comments, resp, err := client.Issues.ListComments(context.Background(), output.Repo.Owner, output.Repo.Name, output.PullRequestID, opt)
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

func handleCommitComparision(client *github.Client, output *prInfo, filterRebased bool) {
	commitComparison, _, err := client.Repositories.CompareCommits(context.Background(), output.Repo.Owner, output.Repo.Name, output.HeadLabel, output.BaseLabel)
	if err != nil {
		return
	}

	output.Rebased = commitComparison.GetAheadBy() == 0
	if filterRebased && output.Rebased {
		output.Hidden = true
	}
}

func handleLabels(client *github.Client, output *prInfo) {
	labels, _, err := client.Issues.ListLabelsByIssue(context.Background(), output.Repo.Owner, output.Repo.Name, output.PullRequestID, nil)
	if err != nil {
		return
	}

	for _, label := range labels {
		output.Labels = append(output.Labels, label.GetName())
	}
}

func handleStatuses(client *github.Client, output *prInfo) {
	statuses, _, err := client.Repositories.ListStatuses(context.Background(), output.Repo.Owner, output.Repo.Name, output.SHA, nil)
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

func getBasePrData(client *github.Client, user *github.User, profile *config.PrpConfigProfile, errorWriter io.Writer) chan *prInfo {
	outputChannel := make(chan *prInfo, 10)
	go func() {
		wg := sync.WaitGroup{}
		for _, repo := range profile.TrackedRepos {
			wg.Add(1)
			go func(repo config.PrpConfigRepo) {
				repoPrs, errors := getRepoPullRequests(client, repo.Owner, repo.Name)
				go func() {
					for {
						err := <-errors
						if err == nil {
							return
						}

						fmt.Fprintln(errorWriter, err)
					}
				}()

				for pr := range repoPrs {
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

		wg.Wait()
		close(outputChannel)
	}()
	return outputChannel
}

func getAllRepos(client *github.Client, login string) []*github.Repository {
	allRepos := make([]*github.Repository, 0, 30)
	opt := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		repositories, resp, err := client.Repositories.List(context.Background(), login, opt)
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
