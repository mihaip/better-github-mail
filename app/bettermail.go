package bettermail

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"appengine"
	"appengine/mail"
	"appengine/urlfetch"
)

var templates map[string]*Template

func init() {
	templates = loadTemplates()

	http.HandleFunc("/hook", hookHandler)
	http.HandleFunc("/hook-test-harness", hookTestHarnessHandler)
	http.HandleFunc("/test-mail-send", testMailSendHandler)
	http.HandleFunc("/_ah/bounce", bounceHandler)
}

func hookHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	eventType := r.Header.Get("X-Github-Event")
	message, err := handlePayload(eventType, r.Body, c)
	if err != nil {
		c.Errorf("Error %s handling %s payload", err, eventType)
		http.Error(w, "Error handling payload", http.StatusInternalServerError)
		return
	}
	if message == nil {
		fmt.Fprint(w, "Unhandled event type: %s", eventType)
		c.Warningf("Unhandled event type: %s", eventType)
		return
	}
	err = mail.Send(c, message)
	if err != nil {
		c.Errorf("Could not send mail: %s", err)
		http.Error(w, "Could not send mail", http.StatusInternalServerError)
		return
	}
	c.Infof("Sent mail to %s", message.To[0])
	fmt.Fprint(w, "OK")
}

func handlePayload(eventType string, payloadReader io.Reader, c appengine.Context) (*mail.Message, error) {
	decoder := json.NewDecoder(payloadReader)
	if eventType == "push" {
		var payload PushPayload
		err := decoder.Decode(&payload)
		if err != nil {
			return nil, err
		}
		return handlePushPayload(payload, c)
	} else if eventType == "commit-comment" {
		var payload CommitCommentPayload
		err := decoder.Decode(&payload)
		if err != nil {
			return nil, err
		}
		return handleCommitCommentPayload(payload, c)
	}
	return nil, nil
}

func handlePushPayload(payload PushPayload, c appengine.Context) (*mail.Message, error) {
	// TODO: allow location to be customized
	location, _ := time.LoadLocation("America/Los_Angeles")

	displayCommits := make([]DisplayCommit, 0)
	for i := range payload.Commits {
		displayCommits = append(displayCommits, newDisplayCommit(&payload.Commits[i], payload.Sender, payload.Repo, location, c))
	}
	branchName := (*payload.Ref)[11:]
	branchUrl := fmt.Sprintf("https://github.com/%s/tree/%s", *payload.Repo.FullName, branchName)
	pushedDate := payload.Repo.PushedAt.In(location)
	// Last link is a link so that the GitHub Gmail extension
	// (https://github.com/muan/github-gmail) will open the diff view.
	extensionUrl := displayCommits[0].URL
	if len(displayCommits) > 1 {
		extensionUrl = *payload.Compare
	}
	var data = map[string]interface{}{
		"Payload":                  payload,
		"Commits":                  displayCommits,
		"BranchName":               branchName,
		"BranchURL":                branchUrl,
		"PushedDisplayDate":        safeFormattedDate(pushedDate.Format(DisplayDateFormat)),
		"PushedDisplayDateTooltip": pushedDate.Format(DisplayDateFullFormat),
		"ExtensionURL":             extensionUrl,
	}
	var mailHtml bytes.Buffer
	if err := templates["push"].Execute(&mailHtml, data); err != nil {
		return nil, err
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

	sender := fmt.Sprintf("%s <%s@better-github-mail.appspotmail.com>", senderName, senderUserName)
	subjectCommit := displayCommits[0]
	subject := fmt.Sprintf("[%s] %s: %s", *payload.Repo.FullName, subjectCommit.ShortSHA, subjectCommit.Title)

	recipient := "eng+commits@quip.com"
	if appengine.IsDevAppServer() {
		recipient = "mihai@quip.com"
	}

	message := &mail.Message{
		Sender:   sender,
		To:       []string{recipient},
		Subject:  subject,
		HTMLBody: mailHtml.String(),
	}
	return message, nil
}

func fetchCommit(payload CommitCommentPayload, commit *ApiCommit, c appengine.Context) error {
	commitSHA := *payload.Comment.CommitID
	url := strings.Replace(*payload.Repo.CommitsURL, "{/sha}", "/" + commitSHA, 1)

	client := urlfetch.Client(c)
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(commit)
}

func handleCommitCommentPayload(payload CommitCommentPayload, c appengine.Context) (*mail.Message, error) {
	var commit ApiCommit
	title := ""
	err := fetchCommit(payload, &commit, c)
	if err != nil {
		c.Infof("Error fetching commit = %v\n", err)
	} else {
		title, _ = getTitleAndMessageFromCommitMessage(*commit.Commit.Message)
	}

	// TODO: allow location to be customized
	location, _ := time.LoadLocation("America/Los_Angeles")
	updatedDate := payload.Comment.UpdatedAt.In(location)

	commitShortSHA := (*payload.Comment.CommitID)[:7]

	var data = map[string]interface{}{
		"Payload":                  payload,
		"Comment":                  payload.Comment,
		"Sender":                   payload.Sender,
		"Repo":                     payload.Repo,
		"ShortSHA":                 commitShortSHA,
		"Body":                     *payload.Comment.Body,
		"CommitURL":                commit.HTML_URL,
		"UpdatedDisplayDate":       safeFormattedDate(updatedDate.Format(DisplayDateFormat)),
	}

	var mailHtml bytes.Buffer
	if err := templates["commit-comment"].Execute(&mailHtml, data); err != nil {
		return nil, err
	}

	senderUserName := *payload.Sender.Login
	senderName := senderUserName

	sender := fmt.Sprintf("%s <%s@better-github-mail.appspotmail.com>", senderName, senderUserName)
	subject := fmt.Sprintf("Re: [%s] %s: %s", *payload.Repo.FullName, commitShortSHA, title)
	c.Infof("subject = %s\n", subject)

	recipient := "eng+commits@quip.com"
	if appengine.IsDevAppServer() {
		recipient = "mihai@quip.com"
	}

	message := &mail.Message{
		Sender:   sender,
		To:       []string{recipient},
		Subject:  subject,
		HTMLBody: mailHtml.String(),
	}
	return message, nil
}

func hookTestHarnessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		templates["hook-test-harness"].Execute(w, nil)
		return
	}
	if r.Method == "POST" {
		eventType := r.FormValue("event_type")
		payload := r.FormValue("payload")
		c := appengine.NewContext(r)

		message, err := handlePayload(eventType, strings.NewReader(payload), c)
		var data = map[string]interface{}{
			"EventType":  eventType,
			"Payload":    payload,
			"Message":    message,
			"MessageErr": err,
		}
		templates["hook-test-harness"].Execute(w, data)
		return
	}
	http.Error(w, "", http.StatusMethodNotAllowed)
}

func bounceHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	if b, err := ioutil.ReadAll(r.Body); err == nil {
		c.Warningf("Bounce: %s", string(b))
	} else {
		c.Warningf("Bounce: <unreadable body>")
	}
}

func testMailSendHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		templates["test-mail-send"].Execute(w, nil)
		return
	}
	if r.Method == "POST" {
		c := appengine.NewContext(r)
		message := &mail.Message{
			Sender:   r.FormValue("sender"),
			To:       []string{"mihai@quip.com"},
			Subject:  r.FormValue("subject"),
			HTMLBody: r.FormValue("html_body"),
		}
		err := mail.Send(c, message)
		var data = map[string]interface{}{
			"Message": message,
			"SendErr": err,
		}
		templates["test-mail-send"].Execute(w, data)
		return
	}
	http.Error(w, "", http.StatusMethodNotAllowed)
}
