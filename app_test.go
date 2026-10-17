package flow_test

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/absurek/flow"
	"golang.org/x/net/html"
)

func TestPortFlagListensOnLoopback(t *testing.T) {
	cfg, err := flow.ParseConfig([]string{"-port", "0"}, func(string) string { return "" })
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	cfg.DBPath = filepath.Join(t.TempDir(), "flow.db")

	app, err := flow.Listen(cfg)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		if err := app.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	})

	if !strings.HasPrefix(app.URL(), "http://127.0.0.1:") {
		t.Fatalf("url = %q, want loopback", app.URL())
	}

	res, err := http.Get(app.URL() + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
}

func TestAddrFlagRefusesNonLoopback(t *testing.T) {
	_, err := flow.ParseConfig([]string{"-addr", "0.0.0.0:8080"}, func(string) string { return "" })
	if err == nil {
		t.Fatal("expected -addr 0.0.0.0 to be refused")
	}
	if !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("error = %q, want it to include loopback", err)
	}
}

func TestPortFlagWinsOverEnvironment(t *testing.T) {
	cfg, err := flow.ParseConfig([]string{"-port", "9090"}, func(key string) string {
		if key == "PORT" {
			return "9091"
		}

		return ""
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.Addr != "127.0.0.1:9090" {
		t.Fatalf("addr = %q, want flag port 9090 on loopback", cfg.Addr)
	}
}

func TestPortEnvironmentOverridesPortAndStaysOnLoopback(t *testing.T) {
	cfg, err := flow.ParseConfig(nil, func(key string) string {
		if key == "PORT" {
			return "9091"
		}

		return ""
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.Addr != "127.0.0.1:9091" {
		t.Fatalf("addr = %q, want 127.0.0.1:9091", cfg.Addr)
	}
}

func TestPortFlagOverridesPortAndStaysOnLoopback(t *testing.T) {
	cfg, err := flow.ParseConfig([]string{"-port", "9090"}, func(string) string { return "" })
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.Addr != "127.0.0.1:9090" {
		t.Fatalf("addr = %q, want 127.0.0.1:9090", cfg.Addr)
	}
}

func TestDefaultListenAddressIsLoopback8080(t *testing.T) {
	cfg, err := flow.ParseConfig(nil, func(string) string { return "" })
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.Addr != "127.0.0.1:8080" {
		t.Fatalf("addr = %q, want 127.0.0.1:8080", cfg.Addr)
	}
}

func TestRefusesNonLoopbackBind(t *testing.T) {
	_, err := flow.Listen(flow.Config{
		Addr:   "0.0.0.0:0",
		DBPath: filepath.Join(t.TempDir(), "flow.db"),
	})
	if err == nil {
		t.Fatal("expected non-loopback address to be refused")
	}
	if !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("error = %q, want it to include loopback", err)
	}
}

func TestMeetingsRemainAfterRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flow.db")
	app := startApp(t, func(cfg *flow.Config) {
		cfg.DBPath = path
	})
	postMeeting(t, app, "Remembered", "2026-01-02T15:04", "still here")
	if err := app.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	restarted := startApp(t, func(cfg *flow.Config) {
		cfg.DBPath = path
	})
	_, doc := getPage(t, restarted, "/")
	text := textContent(doc)
	if !strings.Contains(text, "Remembered") || !strings.Contains(text, "2 Jan 2026, 15:04") || !strings.Contains(text, "still here") {
		t.Fatalf("after restart text = %q, want the Meeting", text)
	}

	status, page := getPage(t, restarted, linkHref(doc, "Remembered"))
	if status != http.StatusOK || !strings.Contains(textContent(page), "still here") {
		t.Fatalf("Meeting page after restart status=%d text=%q", status, textContent(page))
	}
}

func TestAbandonedComposerSavesNoMeeting(t *testing.T) {
	app := startApp(t)

	for range 2 {
		status, doc := getPage(t, app, "/")
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200", status)
		}
		assertNoMeetings(t, doc)
	}

	if _, err := http.PostForm(app.URL()+"/", url.Values{
		"title":       {"Nope"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"draft"},
	}); err != nil {
		t.Fatalf("POST /: %v", err)
	}
	if _, err := http.Get(app.URL() + "/projects"); err != nil {
		t.Fatalf("GET /projects: %v", err)
	}

	_, doc := getPage(t, app, "/")
	text := textContent(doc)
	if strings.Contains(text, "Nope") || strings.Contains(text, "draft") {
		t.Fatalf("abandoned composer saved a Meeting: %s", text)
	}
	assertNoMeetings(t, doc)

	postMeeting(t, app, "Kept", "2026-01-02T15:04", "posted")
	_, doc = getPage(t, app, "/")
	if !strings.Contains(textContent(doc), "Kept") || !strings.Contains(textContent(doc), "posted") {
		t.Fatal("POST /meetings did not create a Meeting")
	}
}

func TestTitleAndBodyAreShownAsText(t *testing.T) {
	app := startApp(t)
	postMeeting(t, app, "<script>alert(1)</script>", "2026-01-02T15:04", "hello <b>nope</b> & co")

	_, doc := getPage(t, app, "/")
	text := textContent(doc)
	if !strings.Contains(text, "<script>alert(1)</script>") || !strings.Contains(text, "hello <b>nope</b> & co") {
		t.Fatalf("feed text = %q, want the title and body as text", text)
	}

	_, page := getPage(t, app, linkHref(doc, "<script>alert(1)</script>"))
	pageText := textContent(page)
	if !strings.Contains(pageText, "<script>alert(1)</script>") || !strings.Contains(pageText, "hello <b>nope</b> & co") {
		t.Fatalf("Meeting page text = %q, want the title and body as text", pageText)
	}
}

func TestBodyLineBreaksArePreserved(t *testing.T) {
	app := startApp(t)
	postMeeting(t, app, "Wrapped", "2026-01-02T15:04", "alpha\nbeta")

	_, doc := getPage(t, app, "/")
	if !strings.Contains(textContent(doc), "alpha\nbeta") {
		t.Fatalf("feed text = %q, want a line break between alpha and beta", textContent(doc))
	}

	_, page := getPage(t, app, linkHref(doc, "Wrapped"))
	if !strings.Contains(textContent(page), "alpha\nbeta") {
		t.Fatalf("Meeting page text = %q, want the line break", textContent(page))
	}
}

func TestSuccessfulPostClearsTheComposer(t *testing.T) {
	app := startApp(t)
	postMeeting(t, app, "Planning", "2024-04-04T04:04", "Notes from the room")

	_, doc := getPage(t, app, "/")
	if got := attr(elementByName(doc, "input", "title"), "value"); got != "" {
		t.Fatalf("title = %q, want an empty composer", got)
	}
	if got := attr(elementByName(doc, "input", "happened_at"), "value"); got != "2026-01-02T15:04" {
		t.Fatalf("happened-at = %q, want the current local time", got)
	}
	if strings.TrimSpace(textContent(elementByName(doc, "textarea", "body"))) != "" {
		t.Fatalf("body = %q, want an empty composer", textContent(elementByName(doc, "textarea", "body")))
	}

	text := textContent(doc)
	for _, want := range []string{"Planning", "4 Apr 2024, 04:04", "Notes from the room"} {
		if !strings.Contains(text, want) {
			t.Fatalf("feed text = %q, want %s", text, want)
		}
	}
}

func TestBlankTitleIsRejectedAndRedisplayed(t *testing.T) {
	app := startApp(t)
	postMeeting(t, app, "Kept", "2026-01-01T08:00", "stays on the feed")

	res := postMeetingNoRedirect(t, app, "", "2026-04-04T12:30", "should not be saved")
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
	doc := parseResponse(t, res)
	text := textContent(doc)
	if !strings.Contains(text, "Title is required.") {
		t.Fatalf("text = %q, want a title error", text)
	}
	if got := attr(elementByName(doc, "input", "happened_at"), "value"); got != "2026-04-04T12:30" {
		t.Fatalf("happened-at = %q, want the submitted value", got)
	}
	if !strings.Contains(text, "should not be saved") {
		t.Fatalf("text = %q, want the submitted body", text)
	}
	if !strings.Contains(text, "Kept") {
		t.Fatalf("text = %q, want the existing Meeting to stay", text)
	}

	_, feed := getPage(t, app, "/")
	feedText := textContent(feed)
	if strings.Contains(feedText, "should not be saved") {
		t.Fatal("rejected Post created a Meeting")
	}
	if !strings.Contains(feedText, "Kept") {
		t.Fatal("existing Meeting disappeared")
	}
}

func TestWhitespaceTitleIsRejected(t *testing.T) {
	app := startApp(t)

	res := postMeetingNoRedirect(t, app, "   ", "2026-04-04T12:30", "ghost body")
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
	doc := parseResponse(t, res)
	if !strings.Contains(textContent(doc), "Title is required.") {
		t.Fatalf("text = %q, want a title error", textContent(doc))
	}
	if got := attr(elementByName(doc, "input", "title"), "value"); got != "   " {
		t.Fatalf("title = %q, want the submitted spaces", got)
	}

	_, feed := getPage(t, app, "/")
	if strings.Contains(textContent(feed), "ghost body") {
		t.Fatal("whitespace title created a Meeting")
	}
}

func TestInvalidHappenedAtIsRejectedAndRedisplayed(t *testing.T) {
	app := startApp(t)

	res := postMeetingNoRedirect(t, app, "Bad time", "yesterday", "notes")
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
	doc := parseResponse(t, res)
	if !strings.Contains(textContent(doc), "Happened-at is invalid.") {
		t.Fatalf("text = %q, want an invalid happened-at error", textContent(doc))
	}
	if got := attr(elementByName(doc, "input", "title"), "value"); got != "Bad time" {
		t.Fatalf("title = %q, want the submitted title", got)
	}
	if got := attr(elementByName(doc, "input", "happened_at"), "value"); got != "yesterday" {
		t.Fatalf("happened-at = %q, want the submitted value", got)
	}
	if !strings.Contains(textContent(doc), "notes") {
		t.Fatal("submitted body was not redisplayed")
	}

	_, feed := getPage(t, app, "/")
	if strings.Contains(textContent(feed), "Bad time") || strings.Contains(textContent(feed), "notes") {
		t.Fatal("invalid happened-at created a Meeting")
	}
}

func TestMissingHappenedAtIsRejectedAndRedisplayed(t *testing.T) {
	app := startApp(t)

	res := postMeetingNoRedirect(t, app, "Orphan title", "", "notes I typed")
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
	doc := parseResponse(t, res)
	text := textContent(doc)
	if !strings.Contains(text, "Happened-at is required.") {
		t.Fatalf("text = %q, want a happened-at error", text)
	}
	if got := attr(elementByName(doc, "input", "title"), "value"); got != "Orphan title" {
		t.Fatalf("title = %q, want the submitted title", got)
	}
	if got := attr(elementByName(doc, "input", "happened_at"), "value"); got != "" {
		t.Fatalf("happened-at = %q, want the submitted empty value", got)
	}
	if !strings.Contains(text, "notes I typed") {
		t.Fatalf("text = %q, want the submitted body", text)
	}

	_, feed := getPage(t, app, "/")
	if strings.Contains(textContent(feed), "Orphan title") || strings.Contains(textContent(feed), "notes I typed") {
		t.Fatal("rejected Post created a Meeting")
	}
}

func TestMeetingPostsWithoutProjects(t *testing.T) {
	app := startApp(t)

	_, composer := getPage(t, app, "/")
	if elementByName(composer, "select", "projects") != nil || elementByName(composer, "input", "project") != nil {
		t.Fatal("a Meeting can be posted with no Projects, but the composer asked for one")
	}

	postMeeting(t, app, "Solo", "2026-01-02T15:04", "no project")
	_, doc := getPage(t, app, "/")
	if !strings.Contains(textContent(doc), "Solo") {
		t.Fatalf("feed text = %q, want the Meeting posted with no Projects", textContent(doc))
	}
}

func TestTitlesNeedNotBeUnique(t *testing.T) {
	app := startApp(t)
	postMeeting(t, app, "Standup", "2026-01-03T09:00", "first standup")
	postMeeting(t, app, "Standup", "2026-01-04T09:00", "second standup")

	_, doc := getPage(t, app, "/")
	text := textContent(doc)
	if strings.Count(text, "Standup") < 2 || !strings.Contains(text, "first standup") || !strings.Contains(text, "second standup") {
		t.Fatalf("feed text = %q, want both Meetings even though the title is shared", text)
	}
}

func TestEmptyBodyIsAllowed(t *testing.T) {
	app := startApp(t)
	postMeeting(t, app, "Just a title", "2026-01-02T15:04", "")

	_, doc := getPage(t, app, "/")
	if !strings.Contains(textContent(doc), "Just a title") || !strings.Contains(textContent(doc), "2 Jan 2026, 15:04") {
		t.Fatalf("feed text = %q, want the title and time with an empty body", textContent(doc))
	}

	status, page := getPage(t, app, linkHref(doc, "Just a title"))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if !strings.Contains(textContent(page), "Just a title") || !strings.Contains(textContent(page), "2 Jan 2026, 15:04") {
		t.Fatalf("Meeting page text = %q, want the title and time", textContent(page))
	}
}

func TestBackdatedMeetingLandsInHappenedAtOrder(t *testing.T) {
	app := startApp(t)
	postMeeting(t, app, "Later", "2026-03-01T10:00", "the newest")
	postMeeting(t, app, "Earlier", "2020-01-01T09:00", "a past meeting")
	postMeeting(t, app, "Middle", "2026-02-01T10:00", "backdated between them")

	_, doc := getPage(t, app, "/")
	text := textContent(doc)
	later := strings.Index(text, "Later")
	middle := strings.Index(text, "Middle")
	earlier := strings.Index(text, "Earlier")
	if later < 0 || middle < 0 || earlier < 0 || later > middle || middle > earlier {
		t.Fatalf("feed order = %q, want Later then Middle then Earlier", text)
	}
	for _, when := range []string{"1 Mar 2026, 10:00", "1 Feb 2026, 10:00", "1 Jan 2020, 09:00"} {
		if !strings.Contains(text, when) {
			t.Fatalf("feed text = %q, want %s", text, when)
		}
	}
}

func TestSameHappenedAtShowsLaterCreatedMeetingFirst(t *testing.T) {
	app := startApp(t)
	postMeeting(t, app, "First posted", "2026-01-02T15:04", "created earlier")
	postMeeting(t, app, "Second posted", "2026-01-02T15:04", "created later")

	_, doc := getPage(t, app, "/")
	text := textContent(doc)
	second := strings.Index(text, "Second posted")
	first := strings.Index(text, "First posted")
	if second < 0 || first < 0 || second > first {
		t.Fatalf("feed order = %q, want Second posted before First posted", text)
	}
}

func TestMissingMeetingIsNotFound(t *testing.T) {
	app := startApp(t)

	for _, path := range []string{"/meetings/999999", "/meetings/not-a-meeting"} {
		status, doc := getPage(t, app, path)
		if status != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404", path, status)
		}
		if !strings.Contains(textContent(doc), "Meeting not found.") {
			t.Fatalf("%s text = %q, want Meeting not found.", path, textContent(doc))
		}
	}
}

func TestMeetingPageShowsTheSameCard(t *testing.T) {
	app := startApp(t)
	postMeetingNoRedirect(t, app, "Planning", "2026-01-02T15:04", "Talked through the timeline.")

	_, feed := getPage(t, app, "/")
	href := linkHref(feed, "Planning")
	if !strings.HasPrefix(href, "/meetings/") {
		t.Fatalf("Meeting link = %q", href)
	}

	status, doc := getPage(t, app, href)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	text := textContent(doc)
	if !strings.Contains(text, "Planning") || !strings.Contains(text, "2 Jan 2026, 15:04") || !strings.Contains(text, "Talked through the timeline.") {
		t.Fatalf("Meeting page text = %q, want the same title, time, and body", text)
	}
}

func TestPostedMeetingShowsTitleTimeAndBody(t *testing.T) {
	app := startApp(t)

	res := postMeetingNoRedirect(t, app, "Planning", "2026-01-02T15:04", "Talked through the timeline.")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", res.StatusCode)
	}
	if loc := res.Header.Get("Location"); loc != "/" {
		t.Fatalf("location = %q, want /", loc)
	}

	status, doc := getPage(t, app, "/")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	text := textContent(doc)
	if !strings.Contains(text, "Planning") {
		t.Fatalf("feed text = %q, want the title", text)
	}
	if !strings.Contains(text, "2 Jan 2026, 15:04") {
		t.Fatalf("feed text = %q, want happened-at 2 Jan 2026, 15:04", text)
	}
	if !strings.Contains(text, "Talked through the timeline.") {
		t.Fatalf("feed text = %q, want the full body", text)
	}
	href := linkHref(doc, "Planning")
	if !strings.HasPrefix(href, "/meetings/") {
		t.Fatalf("Meeting link = %q, want /meetings/{id}", href)
	}
	if !nodePrecedes(doc, "form", "Planning") {
		t.Fatal("composer should stay above the Meeting")
	}
}

func TestDefaultDatabaseFileIsGitignored(t *testing.T) {
	cfg, err := flow.ParseConfig(nil, func(string) string { return "" })
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.DBPath != "flow.db" {
		t.Fatalf("database path = %q, want flow.db", cfg.DBPath)
	}

	data, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "flow.db" {
			return
		}
	}
	t.Fatal(".gitignore does not list flow.db")
}

func TestRelativeDatabaseFileIsCreatedInTheWorkingTree(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	app := startApp(t, func(cfg *flow.Config) {
		cfg.DBPath = "flow.db"
	})
	if _, err := os.Stat(filepath.Join(dir, "flow.db")); err != nil {
		t.Fatalf("working tree database was not created: %v", err)
	}

	postMeeting(t, app, "Local", "2026-01-02T15:04", "on disk")
	_, doc := getPage(t, app, "/")
	if !strings.Contains(textContent(doc), "Local") {
		t.Fatal("Meeting was not stored in the working tree database")
	}
}

func TestDatabaseFileIsCreatedOnStartupWithoutSeedData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flow.db")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("temp database already exists: %v", err)
	}

	app := startApp(t, func(cfg *flow.Config) {
		cfg.DBPath = path
	})

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("database file was not created on startup: %v", err)
	}

	status, doc := getPage(t, app, "/")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	for _, href := range linkHrefs(doc) {
		if strings.HasPrefix(href, "/meetings/") {
			t.Fatalf("new database seeded a Meeting at %s", href)
		}
	}
}

func TestHappenedAtDefaultUsesTheServerLocation(t *testing.T) {
	app := startApp(t, func(cfg *flow.Config) {
		cfg.Now = func() time.Time {
			return time.Date(2026, time.January, 2, 13, 4, 0, 0, time.UTC)
		}
		cfg.Location = time.FixedZone("UTC+2", 2*60*60)
	})

	_, doc := getPage(t, app, "/")
	got := attr(elementByName(doc, "input", "happened_at"), "value")
	if got != "2026-01-02T15:04" {
		t.Fatalf("happened-at default = %q, want 2026-01-02T15:04 in UTC+2", got)
	}
}

func TestHappenedAtDefaultsToCurrentLocalTime(t *testing.T) {
	app := startApp(t)

	_, doc := getPage(t, app, "/")
	got := attr(elementByName(doc, "input", "happened_at"), "value")
	if got != "2026-01-02T15:04" {
		t.Fatalf("happened-at default = %q, want 2026-01-02T15:04", got)
	}
}

func TestEmptyFeedShowsComposerAndNoMeetings(t *testing.T) {
	app := startApp(t)

	status, doc := getPage(t, app, "/")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}

	form := findElement(doc, "form")
	if form == nil {
		t.Fatal("expected a composer form and no Meeting cards")
	}
	if strings.ToLower(attr(form, "method")) != "post" || attr(form, "action") != "/meetings" {
		t.Fatalf("composer method=%q action=%q, want POST /meetings", attr(form, "method"), attr(form, "action"))
	}
	if elementByName(doc, "input", "title") == nil || elementByName(doc, "input", "happened_at") == nil || elementByName(doc, "textarea", "body") == nil {
		t.Fatal("composer must take a title, a happened-at, and a body")
	}

	text := textContent(doc)
	for _, label := range []string{"Title", "Happened at", "Body"} {
		if !strings.Contains(text, label) {
			t.Fatalf("visible text = %q, want label %q", text, label)
		}
	}
	for _, href := range linkHrefs(doc) {
		if strings.HasPrefix(href, "/meetings/") {
			t.Fatalf("empty feed linked a Meeting at %s", href)
		}
	}
}

func TestPagesLinkTheirStylesheet(t *testing.T) {
	app := startApp(t)

	status, doc := getPage(t, app, "/")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	href := stylesheetHref(doc)
	if href != "/styles.css" {
		t.Fatalf("stylesheet href = %q, want /styles.css", href)
	}

	res, err := http.Get(app.URL() + href)
	if err != nil {
		t.Fatalf("GET stylesheet: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("stylesheet status = %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "text/css") {
		t.Fatalf("content-type = %q, want text/css", ct)
	}
}

func TestProjectsPageListsNoProjects(t *testing.T) {
	app := startApp(t)

	status, doc := getPage(t, app, "/projects")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if href := linkHref(doc, "Meetings"); href != "/" {
		t.Fatalf("Meetings link = %q, want /", href)
	}
	if href := linkHref(doc, "Projects"); href != "/projects" {
		t.Fatalf("Projects link = %q, want /projects", href)
	}
	if !strings.Contains(textContent(doc), "No Projects.") {
		t.Fatalf("visible text = %q, want an empty Projects list", textContent(doc))
	}
	for _, href := range linkHrefs(doc) {
		if strings.HasPrefix(href, "/projects/") {
			t.Fatalf("empty Projects list linked %s", href)
		}
	}
}

func TestNavLinksMeetingsAndProjects(t *testing.T) {
	app := startApp(t)

	status, doc := getPage(t, app, "/")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if href := linkHref(doc, "Meetings"); href != "/" {
		t.Fatalf("Meetings link = %q, want /", href)
	}
	if href := linkHref(doc, "Projects"); href != "/projects" {
		t.Fatalf("Projects link = %q, want /projects", href)
	}
}

func TestFeedOpensOnLocalhostWithoutLogin(t *testing.T) {
	app := startApp(t)

	res, err := http.Get(app.URL() + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if !strings.Contains(string(body), "Meetings") {
		t.Fatalf("body = %q, want Meetings visible without login", body)
	}
	if res.Request.URL.Path != "/" {
		t.Fatalf("final path = %q, want / (no login redirect)", res.Request.URL.Path)
	}
}

func startApp(t *testing.T, override ...func(*flow.Config)) *flow.App {
	t.Helper()

	cfg := flow.Config{
		Addr:     "127.0.0.1:0",
		DBPath:   filepath.Join(t.TempDir(), "flow.db"),
		Now:      func() time.Time { return time.Date(2026, time.January, 2, 15, 4, 0, 0, time.UTC) },
		Location: time.UTC,
	}
	for _, fn := range override {
		fn(&cfg)
	}

	app, err := flow.Listen(cfg)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		if err := app.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	})

	return app
}

func postMeeting(t *testing.T, app *flow.App, title, happenedAt, body string) {
	t.Helper()

	res := postMeetingNoRedirect(t, app, title, happenedAt, body)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST %q status = %d, want 303", title, res.StatusCode)
	}
}

func postMeetingNoRedirect(t *testing.T, app *flow.App, title, happenedAt, body string) *http.Response {
	t.Helper()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	form := url.Values{}
	form.Set("title", title)
	form.Set("happened_at", happenedAt)
	form.Set("body", body)

	res, err := client.PostForm(app.URL()+"/meetings", form)
	if err != nil {
		t.Fatalf("POST /meetings: %v", err)
	}
	t.Cleanup(func() { res.Body.Close() })

	return res
}

func nodePrecedes(doc *html.Node, tag, text string) bool {
	seenTag := false
	precedes := false
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if precedes {
			return
		}
		if n.Type == html.ElementNode && n.Data == tag {
			seenTag = true
		}
		if n.Type == html.TextNode && strings.Contains(n.Data, text) && seenTag {
			precedes = true
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return precedes
}

func assertNoMeetings(t *testing.T, doc *html.Node) {
	t.Helper()

	for _, href := range linkHrefs(doc) {
		if strings.HasPrefix(href, "/meetings/") {
			t.Fatalf("feed linked a Meeting at %s", href)
		}
	}
}

func parseResponse(t *testing.T, res *http.Response) *html.Node {
	t.Helper()

	doc, err := html.Parse(res.Body)
	if err != nil {
		t.Fatalf("parse response: %v", err)
	}

	return doc
}

func getPage(t *testing.T, app *flow.App, path string) (int, *html.Node) {
	t.Helper()

	res, err := http.Get(app.URL() + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer res.Body.Close()

	doc, err := html.Parse(res.Body)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	return res.StatusCode, doc
}

func findElement(n *html.Node, element string) *html.Node {
	return elementByName(n, element, "")
}

func elementByName(n *html.Node, element, name string) *html.Node {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == element && (name == "" || attr(n, "name") == name) {
			found = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	return found
}

func stylesheetHref(n *html.Node) string {
	var href string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if href != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "link" && attr(n, "rel") == "stylesheet" {
			href = attr(n, "href")
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	return href
}

func linkHrefs(n *html.Node) []string {
	var hrefs []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			if href := attr(n, "href"); href != "" {
				hrefs = append(hrefs, href)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	return hrefs
}

func linkHref(n *html.Node, text string) string {
	var href string
	var found bool
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" && strings.TrimSpace(textContent(n)) == text {
			href = attr(n, "href")
			found = true
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	return href
}

func textContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	return b.String()
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}

	return ""
}
