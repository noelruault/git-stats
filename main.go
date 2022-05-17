package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

const (
	dateFormat = "2006-01-02"
	yyyyMM     = "2006-01"
)

var TEST = false

func buildLineChart(providers *RepositoryProviders, since time.Time) {
	var githubResponse *GithubAPIGQLResponse
	if TEST {
		githubResponse = readGitHubTestFile()
	} else {
		githubResponse = providers.github.gqlRequest()
	}

	var allStats ContributionStats
	var gls *GitlabStats
	var gitlabContributions map[string]int
	if TEST {
		gitlabContributions = readGitlabTestFile()
		gls = &GitlabStats{}
	} else {
		gitlabContributions = providers.gitlab.GetAllContributions()
		gls = providers.gitlab.GetOtherStats(since)
	}

	allStats = MergeContributionStats(githubResponse, gls)
	allStats.GithubTotalContributions = githubResponse.Data.User.ContributionsCollection.ContributionCalendar.TotalContributions
	allStats.GitlabTotalContributions = gls.TotalContributions

	// merge gitlab and github contributions
	githubContributions := getGitHubMonthlyContributions(githubResponse)

	// add github map
	var combinedContributions = make(map[string]int)
	for k, v := range githubContributions {
		combinedContributions[k] += v
	}

	// add gitlab map
	for k, v := range gitlabContributions {
		combinedContributions[k] += v
	}

	xAxis := getXAxis(combinedContributions)
	toLine(xAxis, true,
		GitContribution{Name: "Gitlab", Color: "#FC6D26", Type: "solid", Data: gitlabContributions},
		GitContribution{Name: "Github", Color: "#00000", Type: "solid", Data: githubContributions},
	)

	allStats.Print()
}

// TODO Refactor getXAxis
func getXAxis(contributions map[string]int) (xAxis []string) {
	// sort map by keys
	keys := make([]string, 0, len(contributions))
	for k := range contributions {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	previousDate, _ := time.Parse(dateFormat, keys[0])
	for i := range keys {
		currentDate, _ := time.Parse(dateFormat, keys[i])
		if previousDate.Month() == currentDate.Month() {
			continue
		}

		// add missing months. Use previous date and increment since the current date is found
		if currentDate.Month() != previousDate.AddDate(0, 1, 0).Month() {
			incrementedDate := previousDate

			for i := 0; currentDate.Month() != incrementedDate.AddDate(0, 1, 0).Month(); i++ {
				incrementedDate = previousDate.AddDate(0, i, 0)
				xAxis = append(xAxis, incrementedDate.Format(yyyyMM))
			}

			previousDate = currentDate
			continue
		}

		// if the month is different
		xAxis = append(xAxis, previousDate.Format(yyyyMM))
		previousDate = currentDate
	}

	xAxis = append(xAxis, previousDate.Format(yyyyMM))
	return xAxis
}

type GitContribution struct {
	Name string
	// Color of line in hexadecimal
	Color string
	// Type of line，options: "solid", "dashed", "dotted". default "solid"
	Type string
	Data map[string]int
}

func toLine(xAxis []string, combine bool, contributions ...GitContribution) {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			PageTitle: "Contribution graph",
			// Theme:           "", // types.ThemePurplePassion
			BackgroundColor: "transparent",
		}),
	)
	line.SetXAxis(xAxis)

	// initialize slice based on xAxis
	contributionData := make([]opts.LineData, 0)
	for i := range xAxis {
		contributionData = append(contributionData,
			opts.LineData{
				Name:   xAxis[i], // TODO: check if is a valid date?
				Value:  nil,
				Symbol: "none",
			},
		)
	}

	// add up contributions to append another that holds a combination of all of them
	if combine {
		var combinedContributions = make(map[string]int)
		for i := range contributions {
			for k, v := range contributions[i].Data {
				combinedContributions[k] += v
			}
		}

		// add combined contribution
		contributions = append(contributions,
			GitContribution{
				Name:  "Combined",
				Color: "#02CB84",
				Type:  "dotted",
				Data:  combinedContributions,
			})
	}

	for i := range contributions {
		line.AddSeries(
			contributions[i].Name,
			getContributionData(xAxis, contributions[i].Data),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: contributions[i].Color,
				Type:  contributions[i].Type,
			}),
		).SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{Smooth: true}),
			// charts.WithAreaStyleOpts(opts.AreaStyle{Opacity: 1}),
		)
		log.Prefix()
	}

	f, err := os.Create("out/charts/lines.html")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	line.Render(io.MultiWriter(f))
}

func getContributionData(xAxis []string, contribution map[string]int) []opts.LineData {
	// initialize slice based on xAxis
	contributionData := make([]opts.LineData, 0)
	for i := range xAxis {
		contributionData = append(contributionData,
			opts.LineData{
				Name:   xAxis[i], // TODO: check if is a valid date?
				Value:  0,
				Symbol: "none",
			},
		)
	}

	var contributionCounter int
	daysContributions := contribution

	// sort map by keys
	keys := make([]string, 0, len(daysContributions))
	for k := range daysContributions {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	previousDate, _ := time.Parse(dateFormat, keys[0])
	for _, k := range keys {
		currentDate, _ := time.Parse(dateFormat, k)
		if previousDate.Month() != currentDate.Month() {
			for i := range contributionData {
				if contributionData[i].Name == previousDate.Format(yyyyMM) {
					contributionData[i].Value = contributionCounter
					continue
				}
			}

			// reset
			contributionCounter = 0
			previousDate = currentDate
		}

		contributionCounter += daysContributions[k]
	}

	for i := range contributionData {
		if contributionData[i].Name == previousDate.Format(yyyyMM) {
			contributionData[i].Value = contributionCounter
			continue
		}
	}

	return contributionData
}

type ContributionStats struct {
	GithubTotalCommitContributions                int
	GithubTotalPullRequestContributions           int
	GithubTotalPullRequestReviewContributions     int
	GithubTotalRepositoriesWithContributedCommits int
	GithubTotalContributions                      int

	GitlabTotalPushedCommits      int
	GitlabSuccessfulMergeRequests int
	GitlabComments                int
	GitlabTotalContributions      int
}

func MergeContributionStats(gh *GithubAPIGQLResponse, gl *GitlabStats) ContributionStats {
	return ContributionStats{
		// GitHub
		GithubTotalCommitContributions:                gh.Data.User.ContributionsCollection.TotalCommitContributions,
		GithubTotalPullRequestContributions:           gh.Data.User.ContributionsCollection.TotalPullRequestContributions,
		GithubTotalPullRequestReviewContributions:     gh.Data.User.ContributionsCollection.TotalPullRequestReviewContributions,
		GithubTotalRepositoriesWithContributedCommits: gh.Data.User.ContributionsCollection.TotalRepositoriesWithContributedCommits,

		// GitLab
		GitlabTotalPushedCommits:      gl.TotalPushedCommits,
		GitlabSuccessfulMergeRequests: gl.SuccessfulMergeRequests,
		GitlabComments:                gl.Comments,
	}
}

func (s *ContributionStats) Print() {
	var msg string

	msg += "\n--- GitHub ---"
	msg += "\nContributions on GitHub: " + fmt.Sprint(s.GithubTotalContributions)
	msg += "\n--- GitHub: Other stats ---"
	msg += "\nTotal merge requests: " + fmt.Sprint(s.GithubTotalPullRequestContributions)
	msg += "\nTotal commit contributions: " + fmt.Sprint(s.GithubTotalCommitContributions)
	msg += "\nTotal merge request reviews: " + fmt.Sprint(s.GithubTotalPullRequestReviewContributions)
	msg += "\nTotal repositories contributed: " + fmt.Sprint(s.GithubTotalRepositoriesWithContributedCommits)

	msg += "\n--- Gitlab ---"
	msg += "\nContributions on Gitlab: " + fmt.Sprint(s.GitlabTotalContributions)
	msg += "\n--- Gitlab: Other stats ---"
	msg += "\nTotal merge requests: " + fmt.Sprint(s.GitlabSuccessfulMergeRequests)
	msg += "\nTotal submitted comments: " + fmt.Sprint(s.GitlabComments)
	msg += "\nTotal commit contributions: " + fmt.Sprint(s.GitlabTotalPushedCommits)

	log.Print(msg)
}

type RepositoryProviders struct {
	github *GitHub
	gitlab *GitLab
}

func main() {
	test := flag.Bool("test", false, "Mocked responses")

	oneYearAgo := time.Now().AddDate(-1, 0, 0).Format(dateFormat)
	since := flag.String("since", oneYearAgo, "Since date with format 2006-12-21. Defaults to one year ago from now.")
	gitlabUser := flag.String("gitlab.user", "", "GitLab user")
	gitlabToken := flag.String("gitlab.token", "", "GitLab auth token")
	githubToken := flag.String("github.token", "", "GitHub auth token")
	flag.Parse()

	TEST = *test
	log.Printf("GIT-STATS started: test=%t, since %s", TEST, *since)

	var providers RepositoryProviders
	providers.github = NewGitHub(base64.StdEncoding.EncodeToString([]byte(*githubToken)))
	providers.gitlab = NewGitlab(base64.StdEncoding.EncodeToString([]byte(*gitlabToken)))
	providers.gitlab.user = *gitlabUser

	sinceTime, _ := time.Parse(dateFormat, *since)

	buildLineChart(&providers, sinceTime)
}
