package command

import (
	"io"
	"sync"

	"github.com/google/go-github/github"
	"github.com/guywithnose/pull-request-parser/config"
)

type prParser struct {
	client  *github.Client
	user    *github.User
	profile *config.Profile
}

func newParser(client *github.Client, user *github.User, profile *config.Profile) *prParser {
	return &prParser{
		client:  client,
		user:    user,
		profile: profile,
	}
}

func (parser prParser) getBasePullRequestData(errorWriter io.Writer) <-chan *pullRequest {
	prs := make(chan *pullRequest, 10)
	go func() {
		wg := sync.WaitGroup{}
		for _, repo := range parser.profile.TrackedRepos {
			wg.Add(1)
			go func(repo config.Repo) {
				parser.getRepositoryPullRequests(repo, prs, errorWriter)
				wg.Done()
			}(repo)
		}

		wg.Wait()
		close(prs)
	}()
	return prs
}

func (parser prParser) getRepositoryPullRequests(repo config.Repo, prs chan<- *pullRequest, errorWriter io.Writer) {
	repoPrs := getRepoPullRequestsAndReportErrors(parser.client, repo.Owner, repo.Name, errorWriter)

	for pr := range repoPrs {
		prs <- &pullRequest{
			client:          parser.client,
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
			NeedsMyApproval: parser.user.GetLogin() != pr.Head.User.GetLogin(),
			IgnoredBuilds:   repo.IgnoredBuilds,
		}
	}
}

func (parser prParser) parsePullRequests(prs <-chan *pullRequest, owner string, repos []string, needsRebase bool) <-chan *pullRequest {
	prs = filterPullRequestsByRepo(prs, owner, repos)
	prs = parser.getAdditionalData(prs)
	if needsRebase {
		prs = parser.filterRebased(prs)
	}

	return prs
}

func (parser prParser) getAdditionalData(prs <-chan *pullRequest) <-chan *pullRequest {
	results := make(chan *pullRequest, 10)
	go func() {
		wg := sync.WaitGroup{}
		for pr := range prs {
			wg.Add(1)
			go func(pr *pullRequest) {
				pr.getAdditionalData(parser.user)
				results <- pr
				wg.Done()
			}(pr)
		}

		wg.Wait()
		close(results)
	}()

	return results
}

func (parser prParser) filterRebased(prs <-chan *pullRequest) <-chan *pullRequest {
	results := make(chan *pullRequest, 10)
	go func() {
		for pr := range prs {
			if !pr.Rebased {
				results <- pr
			}
		}

		close(results)
	}()

	return results
}
