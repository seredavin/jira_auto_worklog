package main

import (
	"fmt"
	"github.com/andygrunwald/go-jira"
	"github.com/jinzhu/now"
	"os"
	"time"
)

func main() {
	jiraClient, err := getJiraClient()

	issues := getInProgressIssues(err, jiraClient)

	var alreadyWorkedTime = 0

	for _, issue := range issues {
		worklog, _, err := jiraClient.Issue.GetWorklogs(issue.ID)
		if err != nil {
			continue
		}
		alreadyWorkedTime = getAlreadyWorkedTimeForIssue(worklog, alreadyWorkedTime)
	}

	var timeSpent = getTimeSpent(alreadyWorkedTime, len(issues))

	if timeSpent > 0 {
		for _, issue := range issues {
			j := new(jira.WorklogRecord)
			var t = jira.Time(time.Now())
			j.Started = &t
			j.TimeSpent = fmt.Sprintf("%dm", timeSpent)
			_, _, err := jiraClient.Issue.AddWorklogRecord(issue.ID, j)
			if err != nil {
				out := fmt.Sprintf("ERROR In issue %v", issue.Fields.Summary)
				fmt.Println(out)
				continue
			}
			out := fmt.Sprintf("In issue %v add %d minutes", issue.Fields.Summary, timeSpent)
			fmt.Println(out)
		}
	} else {
		fmt.Println("Time already filled")
	}

}

func getAlreadyWorkedTimeForIssue(worklog *jira.Worklog, alreadyWorkedTime int) int {
	for _, record := range worklog.Worklogs {
		start := now.BeginningOfDay()
		end := now.EndOfDay()
		startedInRecord := record.Started
		t := time.Time(*startedInRecord)
		if t.After(start) && t.Before(end) {
			alreadyWorkedTime += record.TimeSpentSeconds
		}
	}
	return alreadyWorkedTime
}

func getInProgressIssues(err error, jiraClient *jira.Client) []jira.Issue {
	jql := "assignee = currentUser() AND status = \"In Progress\" "
	fmt.Printf("Usecase: Running a JQL query '%s'\n", jql)

	issues, err := GetAllIssues(jiraClient, jql)
	if err != nil {
		panic(err)
	}
	return issues
}

func getJiraClient() (*jira.Client, error) {
	base := os.Args[1]
	tp := jira.BasicAuthTransport{
		Username: os.Args[2],
		Password: os.Args[3],
	}

	jiraClient, err := jira.NewClient(tp.Client(), base)
	if err != nil {
		panic(err)
	}
	return jiraClient, err
}

func getTimeSpent(alreadyWorkedTime int, inProgressIssuesCount int) int {
	return (480 - (alreadyWorkedTime / 60)) / inProgressIssuesCount
}

func GetAllIssues(client *jira.Client, searchString string) ([]jira.Issue, error) {
	last := 0
	var issues []jira.Issue
	for {
		opt := &jira.SearchOptions{
			MaxResults: 1000, // Max results can go up to 1000
			StartAt:    last,
		}

		chunk, resp, err := client.Issue.Search(searchString, opt)
		if err != nil {
			return nil, err
		}

		total := resp.Total
		if issues == nil {
			issues = make([]jira.Issue, 0, total)
		}
		issues = append(issues, chunk...)
		last = resp.StartAt + len(chunk)
		if last >= total {
			return issues, nil
		}
	}

}
