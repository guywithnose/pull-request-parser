package command

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"golang.org/x/oauth2"
)

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

func getRepoPullRequests(client *github.Client, owner, name string) (<-chan *github.PullRequest, <-chan error) {
	opt := &github.PullRequestListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	allPrs := make(chan *github.PullRequest, 100)
	errors := make(chan error, 1)
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

func getRepoPullRequestsAndReportErrors(client *github.Client, owner, name string, errWriter io.Writer) <-chan *github.PullRequest {
	repoPrs, errors := getRepoPullRequests(client, owner, name)
	go func() {
		for {
			err := <-errors
			if err == nil {
				return
			}

			fmt.Fprintln(errWriter, err)
		}
	}()

	return repoPrs
}
