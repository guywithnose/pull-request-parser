package command

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/guywithnose/pull-request-parser/config"
	"github.com/urfave/cli"
)

func sortRepoNames(profile *config.Profile) []string {
	repoNames := []string{}
	for _, repo := range profile.TrackedRepos {
		repoNames = append(repoNames, fmt.Sprintf("%s/%s", repo.Owner, repo.Name))
	}

	sort.Strings(repoNames)
	return repoNames
}

func unique(in []string) []string {
	set := make(map[string]bool, len(in))
	for _, value := range in {
		set[value] = true
	}

	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}

	return out
}

func shortenLabel(label string) string {
	parts := strings.Split(label, "-")
	initials := []string{}
	for _, part := range parts {
		initials = append(initials, strings.ToUpper(string(part[0])))
	}

	return strings.Join(initials, "")
}

func printResults(prs <-chan *pullRequest, verbose bool, w io.Writer) error {
	tabW := tabwriter.NewWriter(w, 0, 0, 0, ' ', tabwriter.Debug)
	fmt.Fprintln(tabW, "Repo\tID\tTitle\tOwner\tBranch\tTarget\t+1\tUTD\tStatus\tReview\tLabels")
	for pr := range prs {
		title := pr.Title
		if !verbose {
			title = fmt.Sprintf("%.10s", title)
		}

		labels := strings.Join(pr.Labels, ",")
		if !verbose {
			shortLabels := []string{}
			for _, label := range pr.Labels {
				shortLabels = append(shortLabels, shortenLabel(label))
			}

			labels = strings.Join(shortLabels, ",")
		}

		fmt.Fprintf(
			tabW,
			"%s\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			pr.Repo.Name,
			pr.PullRequestID,
			title,
			pr.Owner,
			pr.Branch,
			pr.TargetBranch,
			strconv.Itoa(pr.Approvals),
			boolToString(pr.Rebased),
			buildStatus(pr.BuildInfo),
			boolToString(pr.NeedsMyApproval),
			labels,
		)
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

func filterPullRequestsByRepo(prs <-chan *pullRequest, owner string, repos []string) <-chan *pullRequest {
	filteredPullRequests := make(chan *pullRequest, 10)
	go func() {
		for pr := range prs {
			if pr.matchesRepoFilter(owner, repos) {
				filteredPullRequests <- pr
			}
		}

		close(filteredPullRequests)
	}()
	return filteredPullRequests
}

func stringSliceContains(needle string, haystack []string) bool {
	for _, straw := range haystack {
		if needle == straw {
			return true
		}
	}

	return false
}

func checkPath(localPath string) error {
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return cli.NewExitError(fmt.Sprintf("Path does not exist: %s", localPath), 1)
	}

	if _, err := os.Stat(fmt.Sprintf("%s/.git", localPath)); os.IsNotExist(err) {
		return cli.NewExitError(fmt.Sprintf("Path is not a git repo: %s", localPath), 1)
	}

	return nil
}
