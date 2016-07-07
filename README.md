# Better GitHub Mail

Replacement web hook for GitHub's <a href="https://help.github.com/articles/receiving-email-notifications-for-pushes-to-a-repository/">built-in</a> push email notifications. Has the following advantages:

  * HTML-formatted emails, with links to invidiual file diffs
  * Use commiter display names without spoofing the email domain (which fails when using DMARC).

It's currently running at [http://better-github-mail.appspot.com/](http://better-github-mail.appspot.com/).

## Running Locally

  1. [Install the Go App Engine SDK](https://developers.google.com/appengine/downloads#Google_App_Engine_SDK_for_Go).
  2. Make sure that `PROTOCOL_BUFFERS_PYTHON_IMPLEMENTATION` is set to `python`.
  3. Install the Go library for the GitHub API via `go get github.com/google/go-github/github`
  4. Run: `dev_appserver.py --enable_sendmail=yes app`

The server will then be running at [http://localhost:8080/](http://localhost:8080/), with the hook registered on the `/hook` path. Using [ngrok](https://ngrok.com/) you can generate a publicly accessible URL to use in the repository's service hook settings.

You can also test things via the `/hook-test-harness` harness, which allows you to see the emails that would be generated via an event payload.

## Deploying to App Engine

```
goapp deploy app
```
