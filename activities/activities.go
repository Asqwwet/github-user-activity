package activities

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var httpClient = &http.Client{
	Timeout: 15 * time.Second,
}

const (
	githubAPIBaseURL = "https://api.github.com"
)

type GitHubActivity struct {
	Type      string    `json:"type"`
	Repo      Repo      `json:"repo"`
	CreatedAt time.Time `json:"created_at"`
	Payload   Payload   `json:"payload"`
}

type Repo struct {
	Name string `json:"name"`
}

type Payload struct {
	Action  string `json:"action"`
	Ref     string `json:"ref"`
	RefType string `json:"ref_type"`
	Size    *int   `json:"size"`

	/*
		Why we use in here pointer cause if it send empty json where size are
		not given then size = 0 is automatically if not pointer, if pointer then
		can be checked by nil
	*/
	Commits     []Commit `json:"commits"`
	PullRequest struct {
		Number int `json:"number"`
	} `json:"pull_request"`
	Issue struct {
		Number int `json:"number"`
	} `json:"issue"`
}

type Commit struct {
	Message string `json:"message"`
}

type githubErrorResponse struct {
	Message string `json:"message"`
}

func GetUserActivities(username string) ([]GitHubActivity, error) {
	username = strings.TrimSpace(username)

	if username == "" {
		return nil, errors.New("username cannot be empty")
	}

	endpoint := fmt.Sprintf(
		"%s/users/%s/events/public?per_page=100",
		githubAPIBaseURL,
		/*
			не передаем с помощью pathescape потомучто это уже url  адресс
		*/
		url.PathEscape(username),
	)

	request, err := http.NewRequest(http.MethodGet, endpoint, nil)

	if err != nil {
		return nil, fmt.Errorf("create Github request: %w", err)
	}

	// request.header.set(key, value) формат

	request.Header.Set("Accept", "application/vnd.github+json")

	// задаем в запросе доп информацию что нам нужно формат в json-e

	request.Header.Set("User-Agent", "github-activity-project")

	// All API requests must include a valid User-Agent header.
	// need to know who making request

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request GitHub activity: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("GitHub user %q not found", username)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status code %d", response.StatusCode)
	}

	var events []GitHubActivity
	if err := json.NewDecoder(response.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("decode GitHub activity: %w", err)
	}

	return events, nil
}

func DisplayActivity(username string, events []GitHubActivity) error {
	if len(events) == 0 {
		return fmt.Errorf("no recent public activity found for %q", username)
	}

	fmt.Printf("%s's recent public activities\n", username)

	for _, event := range events {
		date := "unknown date"
		if !event.CreatedAt.IsZero() {
			date = event.CreatedAt.Local().Format("2006-01-02 15:04")
		}
		message := describeEvent(event)
		fmt.Printf("%s   -%s\n", date, message)
	}

	return nil

}

func describeEvent(event GitHubActivity) string {
	switch event.Type {
	case "PushEvent":
		count, ok := pushCommitCount(event.Payload)
		if ok {
			return fmt.Sprintf("Pushed %d commit(s) to %s", count, event.Repo.Name)
		}
		return fmt.Sprintf("Pushed commits to %s", event.Repo.Name)
	case "IssuesEvent":
		if event.Payload.Issue.Number > 0 {
			return fmt.Sprintf("%s issue #%d in %s", titleCase(event.Payload.Action), event.Payload.Issue.Number, event.Repo.Name)
		}
		return fmt.Sprintf("%s an issue in %s", titleCase(event.Payload.Action), event.Repo.Name)

	case "IssueCommentEvent":
		return fmt.Sprintf("%s an issue comment in %s", titleCase(event.Payload.Action), event.Repo.Name)

	case "PullRequestEvent":
		if event.Payload.PullRequest.Number > 0 {
			return fmt.Sprintf("%s pull request #%d in %s", titleCase(event.Payload.Action), event.Payload.PullRequest.Number, event.Repo.Name)
		}
		return fmt.Sprintf("%s a pull request in %s", titleCase(event.Payload.Action), event.Repo.Name)

	case "WatchEvent":
		return fmt.Sprintf("Starred %s", event.Repo.Name)

	case "ForkEvent":
		return fmt.Sprintf("Forked %s", event.Repo.Name)

	case "CreateEvent":
		if event.Payload.RefType != "" {
			return fmt.Sprintf("Created %s in %s", event.Payload.RefType, event.Repo.Name)
		}

		return fmt.Sprintf("Created something in %s", event.Repo.Name)

	case "DeleteEvent":
		if event.Payload.RefType != "" {
			return fmt.Sprintf("Deleted %s in %s", event.Payload.RefType, event.Repo.Name)
		}
		return fmt.Sprintf("Deleted something in %s", event.Repo.Name)

	case "ReleaseEvent":
		return fmt.Sprintf("%s a release in %s", titleCase(event.Payload.Action), event.Repo.Name)

	default:
		return fmt.Sprintf("%s in %s", event.Type, event.Repo.Name)
	}

}

/*
so this counts push commit count
1) checking payload.Size
2) sometimes it can give array of commits

		And why bool because if 0 commits natural, and 0 automaticall commit in order
		to differ
*/

func pushCommitCount(payload Payload) (int, bool) {
	if payload.Size != nil {
		return *payload.Size, true
	}

	if len(payload.Commits) > 0 {
		return len(payload.Commits), true
	}

	return 0, false
}

func titleCase(action string) string {
	action = strings.TrimSpace(action)
	if action == "" {
		return "Updated"
	}

	return strings.ToUpper(action[:1]) + action[1:]
}
