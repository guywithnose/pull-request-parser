package command

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/guywithnose/pull-request-parser/config"
)

func sortRepoNames(profile *config.PrpConfigProfile) []string {
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

func printResults(outputs <-chan *prInfo, verbose bool, w io.Writer) error {
	tabW := tabwriter.NewWriter(w, 0, 0, 0, ' ', tabwriter.Debug)
	fmt.Fprintln(tabW, "Repo\tID\tTitle\tOwner\tBranch\tTarget\t+1\tUTD\tStatus\tReview\tLabels")
	for po := range outputs {
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
				po.Repo.Name,
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

func filterOutput(output *prInfo, owner string, repos []string) bool {
	if owner != "" && output.Owner != owner {
		return false
	}

	if len(repos) != 0 {
		for _, repoName := range repos {
			if repoName == fmt.Sprintf("%s/%s", output.Repo.Owner, output.Repo.Name) {
				return true
			}
		}

		return false
	}

	return true
}

func stringSliceContains(needle string, haystack []string) bool {
	for _, straw := range haystack {
		if needle == straw {
			return true
		}
	}

	return false
}
