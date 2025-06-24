package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Event struct {
	Type string `json:"type"`
	Repo struct {
		Name string `json:"name"`
	} `json:"repo"`
	Payload json.RawMessage `json:"payload"`
}

type PushEventPayload struct {
	Size int `json:"size"`
}

type IssuesEventPayload struct {
	Action string `json:"action"`
}

type WatchEventPayload struct {
	Action string `json:"action"`
}

type ForkEventPayload struct {
	Forkee struct {
		HTMLURL string `json:"html_url"`
	} `json:"forkee"`
}

type CreateEventPayload struct {
	RefType string `json:"ref_type"`
}

type DeleteEventPayload struct {
	RefType string `json:"ref_type"`
}

type PullRequestEventPayload struct {
	Action string `json:"action"`
}

func Usage() {
	fmt.Println("Usage: ./github-activity <username>")
}

func GetUserEvent(username string) ([]Event, error) {
	url := "https://api.github.com/users/" + username + "/events"

	httpClient := &http.Client{}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "GitHubActivityCLI")
	req.Header.Set("Accepts", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("response error: %s", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusNotFound:
			return nil, fmt.Errorf("error: user %s not found", username)
		case http.StatusForbidden:
			return nil, fmt.Errorf("API rate limit exceeded")
		default:
			return nil, fmt.Errorf("error: API request failed with status code: %d", resp.StatusCode)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error with read: %s", err)
	}

	var events []Event
	if err := json.Unmarshal(body, &events); err != nil {
		return nil, fmt.Errorf("error with unmarshal: %s", err)
	}

	return events, nil
}

func formatEvent(event Event) string {
	repoName := event.Repo.Name
	switch event.Type {
	case "PushEvent":
		var payload PushEventPayload
		if err := json.Unmarshal(event.Payload, &payload); err == nil {
			return fmt.Sprintf("Pushed %d commits to %s", payload.Size, repoName)
		}
	case "IssuesEvent":
		var payload IssuesEventPayload
		if err := json.Unmarshal(event.Payload, &payload); err == nil && payload.Action == "opened" {
			return fmt.Sprintf("Opened a new issue in %s", repoName)
		}
	case "WatchEvent":
		var payload WatchEventPayload
		if err := json.Unmarshal(event.Payload, &payload); err == nil && payload.Action == "started" {
			return fmt.Sprintf("Starred %s", repoName)
		}
	case "ForkEvent":
		var payload ForkEventPayload
		if err := json.Unmarshal(event.Payload, &payload); err == nil {
			return fmt.Sprintf("Forked %s to %s", repoName, payload.Forkee.HTMLURL)
		}
	case "CreateEvent":
		var payload CreateEventPayload
		if err := json.Unmarshal(event.Payload, &payload); err == nil && payload.RefType == "repository" {
			return fmt.Sprintf("Created repository %s", repoName)
		}
	case "PullRequestEvent":
		var payload PullRequestEventPayload
		if err := json.Unmarshal(event.Payload, &payload); err == nil && payload.Action == "opened" {
			return fmt.Sprintf("Opened a pull request in %s", repoName)
		}
	case "DeleteEvent":
		var payload DeleteEventPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil && payload.RefType == "repository" {
			return fmt.Sprintf("Deleted repositury %s", repoName)
		}
	}
	return ""
}

func main() {
	if len(os.Args) != 2 {
		Usage()
		os.Exit(0)
	}

	username := os.Args[1]
	events, err := GetUserEvent(username)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	if len(events) == 0 {
		fmt.Printf("No recent activity found for %s\n", username)
		return
	}

	fmt.Printf("Last 30 activities for %s:\n", username)
	for i := 29; i >= 0; i-- {
		if msg := formatEvent(events[i]); msg != "" {
			fmt.Printf("- %s\n", msg)
		}
	}
}
