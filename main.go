package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

const dateFormat = "2006-01-02"

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
	contributionMap := getGitHubMonthlyContributions(githubResponse)

	for k, v := range gitlabContributions {
		if _, ok := gitlabContributions[k]; ok {
			contributionMap[k] = gitlabContributions[k] + v
		}
	}

	toLine(&contributionMap)

	htmlToImage()
	allStats.Print()
}

func toLine(contributions *map[string]int) {
	line := charts.NewLine()

	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			PageTitle: "Contribution graph",
			// Theme:           types.ThemePurplePassion,
			BackgroundColor: "transparent",
		}),
	)

	daysContributions := *contributions
	// sort map by keys
	keys := make([]string, 0, len(daysContributions))
	for k := range daysContributions {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var contributionCounter int
	contributionData := make([]opts.LineData, 0)
	flagDate, _ := time.Parse(dateFormat, keys[0])

	for _, k := range keys {
		dayDate, _ := time.Parse(dateFormat, k)
		if flagDate.Month() != dayDate.Month() {
			contributionData = append(contributionData,
				opts.LineData{
					Name:   flagDate.Month().String(),
					Value:  contributionCounter,
					Symbol: "none",
				},
			)

			// add missing months
			if dayDate.Month() != flagDate.AddDate(0, 1, 0).Month() {
				flag2Date := flagDate

				for i := 1; dayDate.Month() != flag2Date.AddDate(0, 1, 0).Month(); i++ {
					flag2Date = flagDate.AddDate(0, i, 0)
					contributionData = append(contributionData,
						opts.LineData{
							Name:   flag2Date.Month().String(),
							Value:  0,
							Symbol: "none",
						})
				}
			}

			// reset
			contributionCounter = 0
			flagDate = dayDate
		}

		contributionCounter += daysContributions[k]
	}

	contributionData = append(contributionData,
		opts.LineData{
			Name:   flagDate.Month().String(),
			Value:  contributionCounter,
			Symbol: "none",
		},
	)

	var xAxis []string
	for i := range contributionData {
		xAxis = append(xAxis, contributionData[i].Name)
	}
	// Put data into instance
	line.SetXAxis(xAxis).
		AddSeries("Category A", contributionData).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{Smooth: true}),
			charts.WithAreaStyleOpts(opts.AreaStyle{Opacity: 1}),
		)

	f, err := os.Create("out/charts/lines.html")
	if err != nil {
		panic(err)
	}
	line.Render(io.MultiWriter(f))
}

func htmlToImage() {
	command := "npx node-html-to-image-cli out/charts/lines.html out/images/lines.png"
	parts := strings.Fields(command)
	data, err := exec.Command(parts[0], parts[1:]...).Output()
	if err != nil {
		panic(err)
	}

	log.Print(string(data))
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
	log.Printf("STARTED: \ntest=%t", TEST)

	var providers RepositoryProviders
	providers.github = NewGitHub(base64.StdEncoding.EncodeToString([]byte(*githubToken)))
	providers.gitlab = NewGitlab(base64.StdEncoding.EncodeToString([]byte(*gitlabToken)))
	providers.gitlab.user = *gitlabUser

	sinceTime, _ := time.Parse(dateFormat, *since)

	buildLineChart(&providers, sinceTime)
}
