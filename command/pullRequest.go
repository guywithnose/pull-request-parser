package command

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/go-github/github"
	"github.com/guywithnose/pull-request-parser/config"
)

type pullRequest struct {
	client          *github.Client
	Repo            *config.Repo
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
	NeedsMyApproval bool
	BuildInfo       map[string]bool
	Labels          []string
	IgnoredBuilds   []string
	Color           string
}

func (pr *pullRequest) getApprovals(user *github.User) {
	approvingUsers := mergeUsers(pr.getApprovingUsersFromComments(), pr.getApprovingUsersFromReviews())
	for approvingUser := range approvingUsers {
		if approvingUser == user.GetLogin() {
			pr.NeedsMyApproval = false
		}
	}

	pr.Approvals = len(approvingUsers)
}

func (pr pullRequest) getApprovingUsersFromComments() map[string]bool {
	allComments := pr.getComments()
	approvingUsers := make(map[string]bool)
	for comment := range allComments {
		if strings.Contains(comment.GetBody(), ":+1:") || strings.Contains(comment.GetBody(), ":thumbsup:") || strings.Contains(comment.GetBody(), "LGTM") {
			approvingUsers[comment.User.GetLogin()] = true
		}
	}

	return approvingUsers
}

func (pr pullRequest) getApprovingUsersFromReviews() map[string]bool {
	allReviews := pr.getReviews()
	approvingUsers := make(map[string]bool)
	for review := range allReviews {
		if review.GetState() == "APPROVED" {
			approvingUsers[review.User.GetLogin()] = true
		}
	}

	return approvingUsers
}

func mergeUsers(users1, users2 map[string]bool) map[string]bool {
	for name := range users2 {
		users1[name] = true
	}

	return users1
}

func (pr pullRequest) getComments() <-chan *github.IssueComment {
	opt := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	allComments := make(chan *github.IssueComment)
	go func() {
		defer close(allComments)
		for {
			comments, resp, err := pr.client.Issues.ListComments(context.Background(), pr.Repo.Owner, pr.Repo.Name, pr.PullRequestID, opt)
			if err != nil {
				return
			}

			for _, comment := range comments {
				allComments <- comment
			}

			if resp.NextPage == 0 {
				return
			}

			opt.ListOptions.Page = resp.NextPage
		}
	}()
	return allComments
}

func (pr pullRequest) getReviews() <-chan *github.PullRequestReview {
	allReviews := make(chan *github.PullRequestReview)
	go func() {
		defer close(allReviews)
		reviews, _, err := pr.client.PullRequests.ListReviews(context.Background(), pr.Repo.Owner, pr.Repo.Name, pr.PullRequestID)
		if err != nil {
			return
		}

		for _, review := range reviews {
			allReviews <- review
		}
	}()
	return allReviews
}

func (pr *pullRequest) getLabels() {
	labels, _, err := pr.client.Issues.ListLabelsByIssue(context.Background(), pr.Repo.Owner, pr.Repo.Name, pr.PullRequestID, nil)
	if err != nil {
		return
	}

	for _, label := range labels {
		pr.Labels = append(pr.Labels, label.GetName())
	}
}

func (pr *pullRequest) getStatuses() {
	statuses, _, err := pr.client.Repositories.ListStatuses(context.Background(), pr.Repo.Owner, pr.Repo.Name, pr.SHA, nil)
	if err != nil {
		return
	}

	pr.parseStatuses(statuses)
}

func (pr *pullRequest) parseStatuses(statuses []*github.RepoStatus) {
	for _, status := range statuses {
		if pr.buildIsIgnored(status) {
			return
		}

		if _, ok := pr.BuildInfo[status.GetContext()]; !ok {
			pr.BuildInfo[status.GetContext()] = false
		}

		if status.GetState() == "success" {
			pr.BuildInfo[status.GetContext()] = true
		}
	}
}

func (pr pullRequest) buildIsIgnored(status *github.RepoStatus) bool {
	for _, ignoredBuild := range pr.IgnoredBuilds {
		if ignoredBuild == status.GetContext() {
			return true
		}
	}

	return false
}

func (pr *pullRequest) getCommitComparison() {
	commitComparison, _, err := pr.client.Repositories.CompareCommits(context.Background(), pr.Repo.Owner, pr.Repo.Name, pr.HeadLabel, pr.BaseLabel)
	if err != nil {
		return
	}

	pr.Rebased = commitComparison.GetAheadBy() == 0
}

func (pr *pullRequest) getAdditionalData(user *github.User) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		pr.getCommitComparison()
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		pr.getApprovals(user)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		pr.getLabels()
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		pr.getStatuses()
		wg.Done()
	}()

	wg.Wait()
}

func (pr pullRequest) checkLocalPath() error {
	if pr.Repo.LocalPath == "" {
		return fmt.Errorf("Path was not set for repo: %s/%s", pr.Repo.Owner, pr.Repo.Name)
	}

	return checkPath(pr.Repo.LocalPath)
}

const (
	red   = "<fg 1>"
	green = "<fg 2>"
)

func (pr *pullRequest) setColor(user *github.User) {
	if user.GetLogin() == pr.Owner {
		if !pr.Rebased {
			pr.Color = red
		} else {
			pr.Color = green
		}
	}
}

func (pr pullRequest) matchesRepoFilter(owner string, repos []string) bool {
	if owner != "" && pr.Owner != owner {
		return false
	}

	for _, repoName := range repos {
		if repoName == fmt.Sprintf("%s/%s", pr.Repo.Owner, pr.Repo.Name) {
			return true
		}
	}

	return len(repos) == 0
}
