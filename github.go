package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// https://docs.github.com/en/graphql/reference/objects#contributionscollection
type GithubAPIGQLResponse struct {
	Data struct {
		User struct {
			Name                    string `json:"name"`
			ContributionsCollection struct {
				TotalCommitContributions                int `json:"totalCommitContributions"`
				TotalIssueContributions                 int `json:"totalIssueContributions"`
				TotalPullRequestContributions           int `json:"totalPullRequestContributions"`
				TotalPullRequestReviewContributions     int `json:"totalPullRequestReviewContributions"`
				TotalRepositoriesWithContributedCommits int `json:"totalRepositoriesWithContributedCommits"`
				ContributionCalendar                    struct {
					TotalContributions int `json:"totalContributions"`
					Weeks              []struct {
						ContributionDays []struct {
							ContributionCount int    `json:"contributionCount"`
							Date              string `json:"date"`
							Weekday           int    `json:"weekday"`
						} `json:"contributionDays"`
					} `json:"weeks"`
				} `json:"contributionCalendar"`
			} `json:"contributionsCollection"`
		} `json:"user"`
	} `json:"data"`
}

func readGitHubTestFile() *GithubAPIGQLResponse {
	file, _ := ioutil.ReadFile("./test/github-response.json")
	var resp GithubAPIGQLResponse
	_ = json.Unmarshal([]byte(file), &resp)
	return &resp
}

type GitHub struct {
	client http.Client
	token  string
}

func NewGitHub(token string) *GitHub {
	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	return &GitHub{
		client: http.Client{
			Timeout:   time.Second * 10,
			Transport: netTransport,
		},
		token: token,
	}
}

func (gh *GitHub) gqlRequest() *GithubAPIGQLResponse {
	query := fmt.Sprintf(`
		query {
			user(login: "%s") {
				name contributionsCollection (from: "%s" , to: "%s"){
					totalCommitContributions
					totalIssueContributions

					totalPullRequestContributions
					totalPullRequestReviewContributions

					totalRepositoriesWithContributedCommits
					contributionCalendar {
						totalContributions weeks {
							contributionDays {
								contributionCount date weekday
							}
						}
					}
				}
			}
		}`, "noelruault", time.Now().AddDate(-1, 0, 0).Format(time.RFC3339), time.Now().Format(time.RFC3339))

	graphQLRequest := struct {
		Query     string `json:"query"`
		Variables string `json:"variables"`
	}{Query: query}

	gqlMarshalled, err := json.Marshal(graphQLRequest)
	if err != nil {
		panic(err)
	}

	ss := string(gqlMarshalled)
	req, err := http.NewRequest(http.MethodPost, "https://api.github.com/graphql", strings.NewReader(ss))
	if err != nil {
		log.Fatal(err)
	}

	decodeToken, _ := base64.StdEncoding.DecodeString(gh.token)
	req.Header.Add("Authorization", "Bearer "+string(decodeToken))
	req.Header.Add("Content-Type", "application/json")

	res, err := gh.client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	var ghresp GithubAPIGQLResponse
	json.NewDecoder(res.Body).Decode(&ghresp)
	return &ghresp
}

func getGitHubMonthlyContributions(contributions *GithubAPIGQLResponse) map[string]int {
	contributionData := make(map[string]int, 0)

	weeks := contributions.Data.User.ContributionsCollection.ContributionCalendar.Weeks
	for i := range weeks {
		for j := range weeks[i].ContributionDays {
			contributionData[weeks[i].ContributionDays[j].Date] += weeks[i].ContributionDays[j].ContributionCount
		}
	}

	return contributionData
}
