package bettermail

import (
	"bytes"
	"fmt"
	"html"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/github"

	"golang.org/x/net/context"

	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

func safeFormattedDate(date string) string {
	// Insert zero-width spaces every few characters so that Apple Data
	// Detectors and Gmail's calendar event dection don't pick up on these
	// dates.
	var buffer bytes.Buffer
	dateLength := len(date)
	for i := 0; i < dateLength; i += 2 {
		if i == dateLength-1 {
			buffer.WriteString(date[i : i+1])
		} else {
			buffer.WriteString(date[i : i+2])
			if date[i] != ' ' && date[i+1] != ' ' && i < dateLength-2 {
				buffer.WriteString("\u200b")
			}
		}
	}
	return buffer.String()
}

type DisplayCommitFileType int

const (
	CommitFileAdded DisplayCommitFileType = iota
	CommitFileRemoved
	CommitFileModified
)

func (t DisplayCommitFileType) Style() string {
	var style string
	if t == CommitFileAdded {
		style = "added"
	} else if t == CommitFileRemoved {
		style = "removed"
	} else if t == CommitFileModified {
		style = "modified"
	} else {
		style = "unknown"
	}
	return fmt.Sprintf("commit.files.file.type.%s", style)
}

func (t DisplayCommitFileType) Letter() string {
	if t == CommitFileAdded {
		return "+"
	}
	if t == CommitFileRemoved {
		return "-"
	}
	if t == CommitFileModified {
		return "•"
	}
	return "?"
}

type DisplayCommitFile struct {
	Path string
	Type DisplayCommitFileType
	URL  string
}

type DisplayCommitFileByPath []DisplayCommitFile

func (a DisplayCommitFileByPath) Len() int           { return len(a) }
func (a DisplayCommitFileByPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a DisplayCommitFileByPath) Less(i, j int) bool { return a[i].Path < a[j].Path }

type DisplayCommiter struct {
	Login     string
	Name      string
	AvatarURL string
}

type DisplayCommit struct {
	SHA         string
	ShortSHA    string
	URL         string
	Title       string
	MessageHTML string
	Date        time.Time
	Commiter    DisplayCommiter
	Files       []DisplayCommitFile
}

const (
	DisplayDateFormat     = "3:04pm"
	DisplayDateFullFormat = "Monday January 2 3:04pm"
)

func getTitleAndMessageFromCommitMessage(message string) (string, string) {
	messagePieces := strings.SplitN(message, "\n", 2)
	title := messagePieces[0]
	message = ""
	if len(messagePieces) == 2 {
		message = messagePieces[1]
	}
	// Mimic title turncation done by the GitHub web UI
	if len(title) > 80 {
		titleTail := "…" + title[80:]
		if len(message) > 0 {
			message = titleTail + "\n" + message
		} else {
			message = titleTail
		}
		title = title[:80] + "…"
	}
	return title, message
}

func renderMessageMarkdown(message string, repo *WebHookRepository, c context.Context) string {
	// The Markdown endpoint does not escape <, >, etc. so we need to do it
	// ourselves.
	messageHtml := html.EscapeString(message)
	client := github.NewClient(urlfetch.Client(c))
	messageHtmlRendered, _, err := client.Markdown(messageHtml, &github.MarkdownOptions{
		Mode:    "gfm",
		Context: *repo.FullName,
	})
	if err != nil {
		log.Warningf(c, "Could not do markdown rendering, got error %s", err)
		messageHtml = fmt.Sprintf("<div style=\"%s\">%s</div>",
			getStyle("commit.message.block"), messageHtml)
	} else {
		// Use our link style
		messageHtmlRendered = strings.Replace(
			messageHtmlRendered,
			"<a ",
			fmt.Sprintf("<a style=\"%s\" ", getStyle("link")),
			-1)
		// Respect whitespace within blocks...
		messageHtmlRendered = strings.Replace(
			messageHtmlRendered,
			"<p>",
			fmt.Sprintf("<p style=\"%s\">", getStyle("commit.message.block")),
			-1)
		messageHtmlRendered = strings.Replace(
			messageHtmlRendered,
			"<li>",
			fmt.Sprintf("<li style=\"%s\">", getStyle("commit.message.block")),
			-1)
		// ...but avoid doubling of newlines.
		messageHtmlRendered = strings.Replace(
			messageHtmlRendered,
			"<br>\n",
			"<br>",
			-1)
		messageHtml = messageHtmlRendered
	}
	return messageHtml
}

func newDisplayCommit(commit *WebHookCommit, sender *github.User, repo *WebHookRepository, location *time.Location, c context.Context) DisplayCommit {
	title, message := getTitleAndMessageFromCommitMessage(*commit.Message)
	messageHtml := ""
	if len(message) > 0 {
		messageHtml = renderMessageMarkdown(message, repo, c)
	}

	files := make([]DisplayCommitFile, 0)
	for _, path := range commit.Added {
		files = append(files, DisplayCommitFile{Path: path, Type: CommitFileAdded})
	}
	for _, path := range commit.Removed {
		files = append(files, DisplayCommitFile{Path: path, Type: CommitFileRemoved})
	}
	for _, path := range commit.Modified {
		files = append(files, DisplayCommitFile{Path: path, Type: CommitFileModified})
	}
	sort.Sort(DisplayCommitFileByPath(files))
	for i := range files {
		files[i].URL = fmt.Sprintf("%s#diff-%d", *commit.URL, i)
	}

	commiter := DisplayCommiter{
		Login:     *commit.Author.Username,
		Name:      *commit.Author.Name,
		AvatarURL: fmt.Sprintf("https://github.com/identicons/%s.png", *commit.Author.Username),
	}
	if sender.Login != nil && commiter.Login == *sender.Login && sender.AvatarURL != nil {
		commiter.AvatarURL = *sender.AvatarURL
	}

	return DisplayCommit{
		SHA:         *commit.ID,
		ShortSHA:    (*commit.ID)[:7],
		URL:         *commit.URL,
		Title:       title,
		MessageHTML: messageHtml,
		Date:        commit.Timestamp.In(location),
		Commiter:    commiter,
		Files:       files,
	}
}

func (commit DisplayCommit) DisplayDate() string {
	return safeFormattedDate(commit.Date.Format(DisplayDateFormat))
}

func (commit DisplayCommit) DisplayDateTooltip() string {
	return commit.Date.Format(DisplayDateFullFormat)
}
