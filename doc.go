// https://stackoverflow.com/questions/18262288/finding-total-contributions-of-a-user-from-github-api
// https://github.com/MichaelCurrin/github-reporting-py/blob/c52b5c83d4f151b0fd85500d814e52df040409f2/ghgql/queries/contributions/calendar.gql

/*
GITHUB API Call

# example: https://stackoverflow.com/a/57131513
# date: https://docs.github.com/en/search-github/searching-on-github/searching-issues-and-pull-requests#search-by-when-an-issue-or-pull-request-was-created-or-last-updated
query {
     user(login: "noelruault") {
        #  https://docs.github.com/en/graphql/reference/objects#contributionscollection
        name contributionsCollection (from: "2021-12-01T00:00:00Z" , to: "2022-05-03T00:00:00Z"){ # Dates in ISO 8601
            totalCommitContributions # How many commits were made by the user in this time span.
            totalIssueContributions # How many issues the user opened.

            totalPullRequestContributions # How many pull requests the user opened.
            totalPullRequestReviewContributions # How many pull request reviews the user left.

            totalRepositoriesWithContributedCommits # How many different repositories the user committed to.
            contributionCalendar {
                totalContributions weeks {
                    contributionDays {
                        contributionCount date weekday
                    }
                }
            }
        }
    }
}
*/

package main
