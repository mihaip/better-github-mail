package bettermail

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	log_ "log"
	"net/http"
	"strings"
	"time"

	"github.com/mailgun/mailgun-go"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

var templates map[string]*Template

type MailgunConfig struct {
	Domain    string
	APIKey    string
	PublicKey string
	Recipient string
}

var config MailgunConfig

func init() {
	initConfig()
	templates = loadTemplates()

	http.HandleFunc("/hook", hookHandler)
	http.HandleFunc("/hook-test-harness", hookTestHarnessHandler)
	http.HandleFunc("/test-mail-send", testMailSendHandler)
	http.HandleFunc("/_ah/bounce", bounceHandler)
	http.HandleFunc("/test-email-thread", testEmailThreadHandler)
}

func initConfig() {
	path := "config/mailgun"
	if appengine.IsDevAppServer() {
		path = path + "-dev"
	}
	path += ".json"
	configBytes, err := ioutil.ReadFile(path)
	if err != nil {
		log_.Panicf("Could not read config from %s: %s", path, err.Error())
	}
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		log_.Panicf("Could not parse config %s: %s", configBytes, err.Error())
	}
}

type EmailThread struct {
	CommitSHA string `datastore:",noindex"`
	Subject   string `datastore:",noindex"`
	MessageID string `datastore:",noindex"`
}

func createThread(sha string, subject string, messageId string, c context.Context) {
	thread := EmailThread{
		CommitSHA: sha,
		Subject:   subject,
		MessageID: messageId,
	}
	key := datastore.NewKey(c, "EmailThread", sha, 0, nil)
	_, err := datastore.Put(c, key, &thread)
	if err != nil {
		log.Errorf(c, "Error creating thread: %s", err)
	} else {
		log.Infof(c, "Created thread: %v", thread)
	}
}

func getEmailThreadForCommit(sha string, c context.Context) *EmailThread {
	thread := new(EmailThread)
	key := datastore.NewKey(c, "EmailThread", sha, 0, nil)
	err := datastore.Get(c, key, thread)
	if err != nil {
		log.Infof(c, "No thread found for SHA = %s", sha)
		return nil
	}
	return thread
}

func hookHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	eventType := r.Header.Get("X-Github-Event")
	email, commits, err := handlePayload(eventType, r.Body, c)
	if err != nil {
		log.Errorf(c, "Error %s handling %s payload", err, eventType)
		http.Error(w, "Error handling payload", http.StatusInternalServerError)
		return
	}
	if email == nil {
		fmt.Fprint(w, "Unhandled event type: %s", eventType)
		log.Warningf(c, "Unhandled event type: %s", eventType)
		return
	}
	msg, id, err := sendEmail(email, c)
	if commits != nil {
		for _, commit := range commits {
			createThread(commit.SHA, email.Subject, id, c)
		}
	}
	if err != nil {
		log.Errorf(c, "Could not send mail: %s %s", err, msg)
		http.Error(w, "Could not send mail", http.StatusInternalServerError)
		return
	}
	log.Infof(c, "Sent message id=%s", id)
	fmt.Fprint(w, "OK")
}

type Email struct {
	SenderName string
	SenderUserName string
	Subject string
	HTMLBody string
	Headers map [string]string
}

func sendEmail(email* Email, c context.Context) (msg string, id string, err error) {
	httpc := urlfetch.Client(c)
	mg := mailgun.NewMailgun(
		config.Domain,
		config.APIKey,
		config.PublicKey,
	)
	mg.SetClient(httpc)
	sender := fmt.Sprintf("%s <%s@%s>", email.SenderName, email.SenderUserName, config.Domain)
	message := mg.NewMessage(
		sender,
		email.Subject,
		email.HTMLBody,
		config.Recipient,
	)
	message.SetHtml(email.HTMLBody)
	for header, value := range email.Headers {
		message.AddHeader(header, value)
	}
	msg, id, err = mg.Send(message)
	if err != nil {
		log.Errorf(c, "Failed to send message: %v, ID %v, %+v", err, id, msg)
	} else {
		log.Infof(c, "Sent message: %s", id)
	}
	return msg, id, err
}

func handlePayload(eventType string, payloadReader io.Reader, c context.Context) (*Email, []DisplayCommit, error) {
	decoder := json.NewDecoder(payloadReader)
	if eventType == "push" {
		var payload PushPayload
		err := decoder.Decode(&payload)
		if err != nil {
			return nil, nil, err
		}
		return handlePushPayload(payload, c)
	} else if eventType == "commit_comment" {
		var payload CommitCommentPayload
		err := decoder.Decode(&payload)
		if err != nil {
			return nil, nil, err
		}
		email, err := handleCommitCommentPayload(payload, c)
		return email, nil, err
	}
	return nil, nil, nil
}

func handlePushPayload(payload PushPayload, c context.Context) (*Email, []DisplayCommit, error) {
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
		return nil, nil, err
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

	subjectCommit := displayCommits[0]
	subject := fmt.Sprintf("[%s] %s: %s", *payload.Repo.FullName, subjectCommit.ShortSHA, subjectCommit.Title)

	message := &Email{
		SenderName: senderName,
		SenderUserName: senderUserName,
		Subject:  subject,
		HTMLBody: mailHtml.String(),
	}
	return message, displayCommits, nil
}

func handleCommitCommentPayload(payload CommitCommentPayload, c context.Context) (*Email, error) {
	// TODO: allow location to be customized
	location, _ := time.LoadLocation("America/Los_Angeles")
	updatedDate := payload.Comment.UpdatedAt.In(location)

	commitSHA := *payload.Comment.CommitID
	commitShortSHA := commitSHA[:7]
	commitURL := *payload.Repo.HTMLURL + "/commit/" + commitSHA

	body := *payload.Comment.Body
	if len(body) > 0 {
		body = renderMessageMarkdown(body, payload.Repo, c)
	}

	var data = map[string]interface{}{
		"Payload":            payload,
		"Comment":            payload.Comment,
		"Sender":             payload.Sender,
		"Repo":               payload.Repo,
		"ShortSHA":           commitShortSHA,
		"Body":               body,
		"CommitURL":          commitURL,
		"UpdatedDisplayDate": safeFormattedDate(updatedDate.Format(DisplayDateFormat)),
	}

	var mailHtml bytes.Buffer
	if err := templates["commit-comment"].Execute(&mailHtml, data); err != nil {
		return nil, err
	}

	senderUserName := *payload.Sender.Login
	senderName := senderUserName

	thread := getEmailThreadForCommit(commitSHA, c)
	subject := fmt.Sprintf("[%s] %s", *payload.Repo.FullName, commitShortSHA)
	messageId := ""
	if thread != nil {
		subject = thread.Subject
		messageId = thread.MessageID
	}
	// We don't control the message ID, but hopefully subject-basic threading
	// wil work.
	subject = "Re: " + subject

	message := &Email{
		SenderName: senderName,
		SenderUserName: senderUserName,
		Subject: subject,
		HTMLBody: mailHtml.String(),
		Headers: map [string]string {
			"In-Reply-To": messageId,
		},
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

		message, _, err := handlePayload(eventType, strings.NewReader(payload), c)
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
		log.Warningf(c, "Bounce: %s", string(b))
	} else {
		log.Warningf(c, "Bounce: <unreadable body>")
	}
}

func testEmailThreadHandler(w http.ResponseWriter, r *http.Request) {
	if !appengine.IsDevAppServer() {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}
	values := r.URL.Query()
	sha, ok := values["sha"]
	if !ok || len(sha) < 1 {
		http.Error(w, "Need to specify sha param", http.StatusInternalServerError)
		return
	}
	c := appengine.NewContext(r)
	thread := getEmailThreadForCommit(sha[0], c)
	if thread == nil {
		http.Error(w, "No thread found", http.StatusInternalServerError)
	}
	fmt.Fprintf(w, "Subject: %s\n", thread.Subject)
	fmt.Fprintf(w, "MessageID: %s\n", thread.MessageID)
}

func testMailSendHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		templates["test-mail-send"].Execute(w, nil)
		return
	}
	if r.Method == "POST" {
		c := appengine.NewContext(r)
		email := &Email{
			SenderName:   r.FormValue("sender"),
			SenderUserName:   r.FormValue("sender"),
			Subject:  r.FormValue("subject"),
			HTMLBody: r.FormValue("html_body"),
		}
		_, id, err := sendEmail(email, c)
		var data = map[string]interface{}{
			"Message": email,
			"SendErr": err,
			"Id": id,
		}
		templates["test-mail-send"].Execute(w, data)
		return
	}
	http.Error(w, "", http.StatusMethodNotAllowed)
}
