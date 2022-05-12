package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// commented, pushed, accepted, closed, merged...
type Contribution struct {
	Date  time.Time
	Value int
}

func readGitlabTestFile() map[string]int {
	response := make(map[string]int, 0)
	file, _ := ioutil.ReadFile("./test/gitlab-response.json")
	_ = json.Unmarshal([]byte(file), &response)
	return response
}

type GitLab struct {
	client    http.Client
	urlValues url.Values
	user      string
	token     string
}

func NewGitlab(token string) *GitLab {
	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	return &GitLab{
		client: http.Client{
			Timeout:   time.Second * 10,
			Transport: netTransport,
		},
		token: token,
	}
}

func (gl *GitLab) GetAllContributions() map[string]int {
	url := "https://gitlab.com/users/noelruault/calendar.json"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	res, err := gl.client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	response := make(map[string]int, 0)
	_ = json.Unmarshal(body, &response)
	return response
}

func (gl *GitLab) GetEvents() map[string]int {
	url := "https://gitlab.com/api/v4/users/" + gl.user + "/events"

	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	decodeToken, _ := base64.StdEncoding.DecodeString(gl.token)
	req.Header.Add("Authorization", "Bearer "+string(decodeToken))

	res, err := gl.client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	response := make(map[string]int, 0)
	_ = json.Unmarshal(body, &response)
	return response
}

type GitlabStats struct {
	// TotalPushedCommits to (or deleted commits from) a repository, individually or in bulk.
	TotalPushedCommits      int
	SuccessfulMergeRequests int
	// Comments on any noteable record.
	Comments           int
	TotalContributions int
}

// TODO:
// - Equivalent to "totalRepositoriesWithContributedCommits" https://docs.gitlab.com/ee/api/projects.html
// - Equivalent to "totalIssueContributions" https://docs.gitlab.com/ee/api/issues.html
func (gl *GitLab) GetOtherStats(since time.Time) *GitlabStats {

	var gls GitlabStats
	sinceDate := since.Format(dateFormat)

	gl.urlValues = url.Values{
		"action": []string{"pushed"},
		"after":  []string{sinceDate},
	}
	events, err := gl.GetEventsHead()
	if err != nil {
		return nil
	}
	gls.TotalPushedCommits = events["X-Total"]

	gl.urlValues = url.Values{
		"action": []string{"merged"},
		"after":  []string{sinceDate},
	}
	events, err = gl.GetEventsHead()
	if err != nil {
		return nil
	}
	gls.SuccessfulMergeRequests = events["X-Total"]

	gl.urlValues = url.Values{
		"action": []string{"commented"},
		"after":  []string{sinceDate},
	}
	events, err = gl.GetEventsHead()
	if err != nil {
		return nil
	}
	gls.Comments = events["X-Total"]

	gl.urlValues = url.Values{
		"after": []string{sinceDate},
	}
	events, err = gl.GetEventsHead()
	if err != nil {
		return nil
	}
	gls.TotalContributions = events["X-Total"]

	return &gls
}

func (gl *GitLab) GetEventsHead() (map[string]int, error) {
	uri := url.URL{
		Scheme:   "https",
		Path:     "gitlab.com/api/v4/users/" + gl.user + "/events",
		RawQuery: gl.urlValues.Encode(),
	}

	req, err := http.NewRequest(http.MethodHead, uri.String(), nil)
	if err != nil {
		log.Fatal(err)
	}

	decodeToken, _ := base64.StdEncoding.DecodeString(gl.token)
	req.Header.Add("Authorization", "Bearer "+string(decodeToken))

	res, err := gl.client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.Header["X-Total"] == nil {
		return nil, fmt.Errorf("X-Total key not found")
	}
	total, _ := strconv.Atoi(res.Header["X-Total"][0])
	return map[string]int{
		"X-Total": total,
	}, nil
}
