package bettermail

import (
	"time"

	"github.com/google/go-github/github"
)

type PushPayload struct {
	After      *string            `json:"after,omitempty"`
	Before     *string            `json:"before,omitempty"`
	Commits    []WebHookCommit    `json:"commits,omitempty"`
	Compare    *string            `json:"compare,omitempty"`
	Created    *bool              `json:"created,omitempty"`
	Deleted    *bool              `json:"deleted,omitempty"`
	Forced     *bool              `json:"forced,omitempty"`
	HeadCommit *WebHookCommit     `json:"head_commit,omitempty"`
	Pusher     *github.User       `json:"pusher,omitempty"`
	Sender     *github.User       `json:"sender,omitempty"`
	Ref        *string            `json:"ref,omitempty"`
	Repo       *WebHookRepository `json:"repository,omitempty"`
}

type CommitCommentPayload struct {
	Action     *string               `json:"action,omitempty"`
	Comment    *WebHookCommitComment `json:"comment,omitempty"`
	Repo       *WebHookRepository    `json:"repository,omitempty"`
	Sender     *github.User          `json:"sender,omitempty"`
}

// WebHookCommit represents the commit variant we receive from GitHub in a
// WebHookPayload.
type WebHookCommit struct {
	Added     []string              `json:"added,omitempty"`
	Author    *github.WebHookAuthor `json:"author,omitempty"`
	Committer *github.WebHookAuthor `json:"committer,omitempty"`
	Distinct  *bool                 `json:"distinct,omitempty"`
	URL       *string               `json:"url,omitempty"`
	ID        *string               `json:"id,omitempty"`
	Message   *string               `json:"message,omitempty"`
	Modified  []string              `json:"modified,omitempty"`
	Removed   []string              `json:"removed,omitempty"`
	Timestamp *time.Time            `json:"timestamp,omitempty"`
}

type WebHookCommitComment struct {
	ID        *int                  `json:"id,omitempty"`
	User      *github.User          `json:"user,omitempty"`
	URL       *string               `json:"url,omitempty"`
	HTML_URL  *string               `json:"html_url,omitempty"`
	CommitID  *string               `json:"commit_id,omitempty"`
	Body      *string               `json:"body,omitempty"`
	CreatedAt *time.Time            `json:"created_at,omitempty"`
	UpdatedAt *time.Time            `json:"updated_at,omitempty"`
	Position  *int                  `json:"position,omitempty"`
	Line      *int                  `json:"line,omitempty"`
	Path      *string               `json:"path,omitempty"`
}

type WebHookRepository struct {
	ID               *int               `json:"id,omitempty"`
	Owner            *github.User       `json:"owner,omitempty"`
	Name             *string            `json:"name,omitempty"`
	FullName         *string            `json:"full_name,omitempty"`
	Description      *string            `json:"description,omitempty"`
	Homepage         *string            `json:"homepage,omitempty"`
	DefaultBranch    *string            `json:"default_branch,omitempty"`
	MasterBranch     *string            `json:"master_branch,omitempty"`
	CreatedAt        *github.Timestamp  `json:"created_at,omitempty"`
	PushedAt         *github.Timestamp  `json:"pushed_at,omitempty"`
	UpdatedAt        *github.Timestamp  `json:"updated_at,omitempty"`
	HTMLURL          *string            `json:"html_url,omitempty"`
	CloneURL         *string            `json:"clone_url,omitempty"`
	GitURL           *string            `json:"git_url,omitempty"`
	MirrorURL        *string            `json:"mirror_url,omitempty"`
	SSHURL           *string            `json:"ssh_url,omitempty"`
	SVNURL           *string            `json:"svn_url,omitempty"`
	Language         *string            `json:"language,omitempty"`
	Fork             *bool              `json:"fork"`
	ForksCount       *int               `json:"forks_count,omitempty"`
	NetworkCount     *int               `json:"network_count,omitempty"`
	OpenIssuesCount  *int               `json:"open_issues_count,omitempty"`
	StargazersCount  *int               `json:"stargazers_count,omitempty"`
	SubscribersCount *int               `json:"subscribers_count,omitempty"`
	WatchersCount    *int               `json:"watchers_count,omitempty"`
	Size             *int               `json:"size,omitempty"`
	AutoInit         *bool              `json:"auto_init,omitempty"`
	Parent           *github.Repository `json:"parent,omitempty"`
	Source           *github.Repository `json:"source,omitempty"`
	Organization     *string            `json:"organization,omitempty"`
	Permissions      *map[string]bool   `json:"permissions,omitempty"`

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

// Represents the payload received from the /commits API call 
type ApiCommit struct {
	Commit           *WebHookCommit    `json:"commit,omitempty"`
	HTML_URL         *string           `json:"html_url,omitempty"`
}
