package bettermail

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "sort"
    "strings"
    "time"

    "appengine"
    "appengine/mail"

    "github.com/google/go-github/github"
)

var templates map[string]*Template

func init() {
    templates = loadTemplates()

    http.HandleFunc("/hook", handler)
}

type PushPayload struct {
	After      *string         `json:"after,omitempty"`
	Before     *string         `json:"before,omitempty"`
	Commits    []WebHookCommit `json:"commits,omitempty"`
	Compare    *string         `json:"compare,omitempty"`
	Created    *bool           `json:"created,omitempty"`
	Deleted    *bool           `json:"deleted,omitempty"`
	Forced     *bool           `json:"forced,omitempty"`
	HeadCommit *WebHookCommit  `json:"head_commit,omitempty"`
	Pusher     *github.User           `json:"pusher,omitempty"`
	Ref        *string         `json:"ref,omitempty"`
	Repo       *WebHookRepository     `json:"repository,omitempty"`
}

// WebHookCommit represents the commit variant we receive from GitHub in a
// WebHookPayload.
type WebHookCommit struct {
	Added     []string       `json:"added,omitempty"`
	Author    *github.WebHookAuthor `json:"author,omitempty"`
	Committer *github.WebHookAuthor `json:"committer,omitempty"`
	Distinct  *bool          `json:"distinct,omitempty"`
	URL        *string        `json:"url,omitempty"`
	ID        *string        `json:"id,omitempty"`
	Message   *string        `json:"message,omitempty"`
	Modified  []string       `json:"modified,omitempty"`
	Removed   []string       `json:"removed,omitempty"`
	Timestamp *time.Time     `json:"timestamp,omitempty"`
}

type WebHookRepository struct {
	ID               *int             `json:"id,omitempty"`
	Owner            *github.User            `json:"owner,omitempty"`
	Name             *string          `json:"name,omitempty"`
	FullName         *string          `json:"full_name,omitempty"`
	Description      *string          `json:"description,omitempty"`
	Homepage         *string          `json:"homepage,omitempty"`
	DefaultBranch    *string          `json:"default_branch,omitempty"`
	MasterBranch     *string          `json:"master_branch,omitempty"`
	CreatedAt        *github.Timestamp       `json:"created_at,omitempty"`
	PushedAt         *github.Timestamp       `json:"pushed_at,omitempty"`
	UpdatedAt        *github.Timestamp       `json:"updated_at,omitempty"`
	HTMLURL          *string          `json:"html_url,omitempty"`
	CloneURL         *string          `json:"clone_url,omitempty"`
	GitURL           *string          `json:"git_url,omitempty"`
	MirrorURL        *string          `json:"mirror_url,omitempty"`
	SSHURL           *string          `json:"ssh_url,omitempty"`
	SVNURL           *string          `json:"svn_url,omitempty"`
	Language         *string          `json:"language,omitempty"`
	Fork             *bool            `json:"fork"`
	ForksCount       *int             `json:"forks_count,omitempty"`
	NetworkCount     *int             `json:"network_count,omitempty"`
	OpenIssuesCount  *int             `json:"open_issues_count,omitempty"`
	StargazersCount  *int             `json:"stargazers_count,omitempty"`
	SubscribersCount *int             `json:"subscribers_count,omitempty"`
	WatchersCount    *int             `json:"watchers_count,omitempty"`
	Size             *int             `json:"size,omitempty"`
	AutoInit         *bool            `json:"auto_init,omitempty"`
	Parent           *github.Repository      `json:"parent,omitempty"`
	Source           *github.Repository      `json:"source,omitempty"`
	Organization     *string          `json:"organization,omitempty"`
	Permissions      *map[string]bool `json:"permissions,omitempty"`

	// Only provided when using RepositoriesService.Get while in preview
	License *github.License `json:"license,omitempty"`

	// Additional mutable fields when creating and editing a repository
	Private      *bool `json:"private"`
	HasIssues    *bool `json:"has_issues"`
	HasWiki      *bool `json:"has_wiki"`
	HasDownloads *bool `json:"has_downloads"`
	// Creating an organization repository. Required for non-owners.
	TeamID *int `json:"team_id"`

	// API URLs
	URL              *string `json:"url,omitempty"`
	ArchiveURL       *string `json:"archive_url,omitempty"`
	AssigneesURL     *string `json:"assignees_url,omitempty"`
	BlobsURL         *string `json:"blobs_url,omitempty"`
	BranchesURL      *string `json:"branches_url,omitempty"`
	CollaboratorsURL *string `json:"collaborators_url,omitempty"`
	CommentsURL      *string `json:"comments_url,omitempty"`
	CommitsURL       *string `json:"commits_url,omitempty"`
	CompareURL       *string `json:"compare_url,omitempty"`
	ContentsURL      *string `json:"contents_url,omitempty"`
	ContributorsURL  *string `json:"contributors_url,omitempty"`
	DownloadsURL     *string `json:"downloads_url,omitempty"`
	EventsURL        *string `json:"events_url,omitempty"`
	ForksURL         *string `json:"forks_url,omitempty"`
	GitCommitsURL    *string `json:"git_commits_url,omitempty"`
	GitRefsURL       *string `json:"git_refs_url,omitempty"`
	GitTagsURL       *string `json:"git_tags_url,omitempty"`
	HooksURL         *string `json:"hooks_url,omitempty"`
	IssueCommentURL  *string `json:"issue_comment_url,omitempty"`
	IssueEventsURL   *string `json:"issue_events_url,omitempty"`
	IssuesURL        *string `json:"issues_url,omitempty"`
	KeysURL          *string `json:"keys_url,omitempty"`
	LabelsURL        *string `json:"labels_url,omitempty"`
	LanguagesURL     *string `json:"languages_url,omitempty"`
	MergesURL        *string `json:"merges_url,omitempty"`
	MilestonesURL    *string `json:"milestones_url,omitempty"`
	NotificationsURL *string `json:"notifications_url,omitempty"`
	PullsURL         *string `json:"pulls_url,omitempty"`
	ReleasesURL      *string `json:"releases_url,omitempty"`
	StargazersURL    *string `json:"stargazers_url,omitempty"`
	StatusesURL      *string `json:"statuses_url,omitempty"`
	SubscribersURL   *string `json:"subscribers_url,omitempty"`
	SubscriptionURL  *string `json:"subscription_url,omitempty"`
	TagsURL          *string `json:"tags_url,omitempty"`
	TreesURL         *string `json:"trees_url,omitempty"`
	TeamsURL         *string `json:"teams_url,omitempty"`
}

type DisplayCommitFileType int

const (
	CommitFileAdded DisplayCommitFileType = iota
	CommitFileRemoved
	CommitFileModified
)

type DisplayCommitFile struct {
    Path string
    Type DisplayCommitFileType
    URL string
}

type DisplayCommitFileByPath []DisplayCommitFile

func (a DisplayCommitFileByPath) Len() int           { return len(a) }
func (a DisplayCommitFileByPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a DisplayCommitFileByPath) Less(i, j int) bool { return a[i].Path < a[j].Path }

type DisplayCommit struct {
	DisplaySHA string
	URL        string
	Title      string
	Message    string
	Date   time.Time
	Files   []DisplayCommitFile
}

const (
	CommitDisplayDateFormat      = "3:04pm"
	CommitDisplayDateFullFormat  = "Monday January 2 3:04pm"
)

func newDisplayCommit(commit *WebHookCommit, location *time.Location) DisplayCommit {
	messagePieces := strings.SplitN(*commit.Message, "\n", 2)
	title := messagePieces[0]
	message := ""
	if len(messagePieces) == 2 {
		message = messagePieces[1]
	}

	files := make([]DisplayCommitFile, 0)
	for _, path := range(commit.Added) {
	    files = append(files, DisplayCommitFile{Path: path, Type: CommitFileAdded})
	}
	for _, path := range(commit.Removed) {
	    files = append(files, DisplayCommitFile{Path: path, Type: CommitFileRemoved})
	}
	for _, path := range(commit.Modified) {
	    files = append(files, DisplayCommitFile{Path: path, Type: CommitFileModified})
	}
	sort.Sort(DisplayCommitFileByPath(files))
	for i := range(files) {
	    files[i].URL = fmt.Sprintf("%s#diff-%d", *commit.URL, i)
	}

	return DisplayCommit{
		DisplaySHA: (*commit.ID)[:7],
		URL:        *commit.URL,
		Title:      title,
		Message:    message,
		Date:   commit.Timestamp.In(location),
		Files: files,
	}
}

func (commit DisplayCommit) DisplayDate() string {
	return commit.Date.Format(CommitDisplayDateFormat)
}

func (commit DisplayCommit) DisplayDateTooltip() string {
    return commit.Date.Format(CommitDisplayDateFullFormat)
}

func handler(w http.ResponseWriter, r *http.Request) {
    c := appengine.NewContext(r)
    decoder := json.NewDecoder(r.Body)
    eventType := r.Header.Get("X-Github-Event")
    if eventType == "push" {
        var payload PushPayload
        err := decoder.Decode(&payload)
        if err != nil {
            c.Errorf("Error %s parsing push payload", err)
            http.Error(w, "Invalid push payload", http.StatusBadRequest)
            return
        }
        sent, err := handlePushPayload(payload, c)
        if err != nil {
            c.Errorf("Error %s handling push payload", err)
            http.Error(w, "Could not handle push payload", http.StatusInternalServerError)
            return
        }
        if !sent {
            c.Errorf("Could not send mail")
            http.Error(w, "Could not send mail", http.StatusInternalServerError)
            return
        }
        fmt.Fprint(w, "OK")
        return
    }
    fmt.Fprint(w, "Unhandled event type: %s", eventType)
    c.Warningf("Unhandled event type: %s", eventType)
}

func handlePushPayload(payload PushPayload, c appengine.Context) (bool, error) {
    // TODO: allow location to be customized
    location, _ := time.LoadLocation("America/Los_Angeles")

    displayCommits := make([]DisplayCommit, 0)
    for i := range payload.Commits {
        displayCommits = append(displayCommits, newDisplayCommit(&payload.Commits[i], location))
    }
	var data = map[string]interface{}{
		"Payload": payload,
		"Commits": displayCommits,
	}
	var mailHtml bytes.Buffer
	if err := templates["push"].Execute(&mailHtml, data); err != nil {
		return false, err
	}

    senderUserName := *payload.Pusher.Name
    senderName := senderUserName
    // We don't have the display name in the pusher, but usually it's one of the
    // commiters, so get it from there (without having to do any extra API
    // requests)
    for _, commit := range payload.Commits {
        if *commit.Author.Username == senderUserName {
            senderName = *commit.Author.Name
            break
        }
        if *commit.Committer.Username == senderUserName {
            senderName = *commit.Committer.Name
            break
        }
    }

    sender := fmt.Sprintf("%s <%s@better-github-mail.appspot.com>", senderName, senderUserName)
    displayHeadCommit := newDisplayCommit(payload.HeadCommit, location)
	subject := fmt.Sprintf("[%s] %s: %s", *payload.Repo.FullName, displayHeadCommit.DisplaySHA, displayHeadCommit.Title)

	message := &mail.Message{
		Sender:   sender,
		To:       []string{"mihai@quip.com"},
		Subject:  subject,
		HTMLBody: mailHtml.String(),
	}
	err := mail.Send(c, message)
	return true, err
}
