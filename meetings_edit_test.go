package flow_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/absurek/flow"
	"golang.org/x/net/html"
)

func TestSaveUpdatesTheMeetingAndCanClearTheBody(t *testing.T) {
	app := startApp(t)
	postMeeting(t, app, "Planning", "2026-01-02T15:04", "original body")

	_, feed := getPage(t, app, "/")
	meetingURL := linkHref(feed, "Planning")
	_, page := getPage(t, app, meetingURL)
	editURL := linkHref(page, "Edit")
	if editURL == "" {
		t.Fatal("the Meeting page should open the edit")
	}

	status, edit := getPage(t, app, editURL)
	if status != http.StatusOK {
		t.Fatalf("edit status = %d, want 200", status)
	}
	if got := attr(elementByName(edit, "input", "title"), "value"); got != "Planning" {
		t.Fatalf("title = %q, want the saved title", got)
	}
	if got := attr(elementByName(edit, "input", "happened_at"), "value"); got != "2026-01-02T15:04" {
		t.Fatalf("happened-at = %q, want the saved time", got)
	}
	if got := strings.TrimSpace(textContent(elementByName(edit, "textarea", "body"))); got != "original body" {
		t.Fatalf("body = %q, want the saved body", got)
	}

	form := formContaining(edit, elementByName(edit, "input", "title"))
	if form == nil || strings.ToLower(attr(form, "method")) != "post" || attr(form, "action") != meetingURL {
		method, action := "", ""
		if form != nil {
			method, action = attr(form, "method"), attr(form, "action")
		}
		t.Fatalf("save form method=%q action=%q, want POST %s", method, action, meetingURL)
	}
	if strings.TrimSpace(textContent(submitControl(edit, "Save"))) == "" {
		t.Fatal("the edit should offer Save")
	}

	res := postFormNoRedirect(t, app, meetingURL, url.Values{
		"title":       {"Planning revised"},
		"happened_at": {"2026-06-06T18:30"},
		"body":        {""},
	}, nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != meetingURL {
		t.Fatalf("save status = %d location = %q, want 303 %s", res.StatusCode, res.Header.Get("Location"), meetingURL)
	}

	_, page = getPage(t, app, meetingURL)
	text := textContent(page)
	if linkHref(page, "Planning revised") != meetingURL || !strings.Contains(text, "6 Jun 2026, 18:30") {
		t.Fatalf("Meeting text = %q, want the saved title and time", text)
	}
	if strings.Contains(text, "original body") || linkHref(page, "Planning") != "" {
		t.Fatalf("Meeting text = %q, want the body cleared and the old title gone", text)
	}

	_, feed = getPage(t, app, "/")
	feedText := textContent(feed)
	if linkHref(feed, "Planning revised") != meetingURL || !strings.Contains(feedText, "6 Jun 2026, 18:30") || strings.Contains(feedText, "original body") {
		t.Fatalf("feed text = %q, want the saved Meeting with an empty body", feedText)
	}
	if linkHref(feed, "Planning") != "" {
		t.Fatal("the old title is still on the feed")
	}
}

func TestRejectedSaveLeavesTheMeetingAndRedisplaysTheEdit(t *testing.T) {
	app := startApp(t)
	postMeeting(t, app, "Planning", "2026-01-02T15:04", "original body")
	_, feed := getPage(t, app, "/")
	meetingURL := linkHref(feed, "Planning")

	res := postFormNoRedirect(t, app, meetingURL, url.Values{
		"title":       {""},
		"happened_at": {"2026-04-04T12:30"},
		"body":        {"should not be saved"},
	}, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("blank title status = %d, want 400", res.StatusCode)
	}
	rejected := parseResponse(t, res)
	if !strings.Contains(textContent(rejected), "Title is required.") {
		t.Fatalf("text = %q, want a title error", textContent(rejected))
	}
	if got := attr(elementByName(rejected, "input", "title"), "value"); got != "" {
		t.Fatalf("title = %q, want the submitted empty title", got)
	}
	if got := attr(elementByName(rejected, "input", "happened_at"), "value"); got != "2026-04-04T12:30" {
		t.Fatalf("happened-at = %q, want the submitted value", got)
	}
	if !strings.Contains(textContent(elementByName(rejected, "textarea", "body")), "should not be saved") {
		t.Fatal("rejected Save dropped the body")
	}
	assertMeetingUnchanged(t, app, meetingURL)

	res = postFormNoRedirect(t, app, meetingURL, url.Values{
		"title":       {"   "},
		"happened_at": {"2026-04-04T12:30"},
		"body":        {"spaces"},
	}, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("whitespace title status = %d, want 400", res.StatusCode)
	}
	rejected = parseResponse(t, res)
	if !strings.Contains(textContent(rejected), "Title is required.") {
		t.Fatalf("text = %q, want a title error", textContent(rejected))
	}
	if got := attr(elementByName(rejected, "input", "title"), "value"); got != "   " {
		t.Fatalf("title = %q, want the submitted spaces", got)
	}
	assertMeetingUnchanged(t, app, meetingURL)

	res = postFormNoRedirect(t, app, meetingURL, url.Values{
		"title":       {"Renamed"},
		"happened_at": {""},
		"body":        {"notes I typed"},
	}, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing happened-at status = %d, want 400", res.StatusCode)
	}
	rejected = parseResponse(t, res)
	if !strings.Contains(textContent(rejected), "Happened-at is required.") {
		t.Fatalf("text = %q, want a happened-at error", textContent(rejected))
	}
	if got := attr(elementByName(rejected, "input", "title"), "value"); got != "Renamed" {
		t.Fatalf("title = %q, want the submitted title", got)
	}
	if got := attr(elementByName(rejected, "input", "happened_at"), "value"); got != "" {
		t.Fatalf("happened-at = %q, want the submitted empty value", got)
	}
	if !strings.Contains(textContent(elementByName(rejected, "textarea", "body")), "notes I typed") {
		t.Fatal("rejected Save dropped the body")
	}
	assertMeetingUnchanged(t, app, meetingURL)

	res = postFormNoRedirect(t, app, meetingURL, url.Values{
		"title":       {"Bad time"},
		"happened_at": {"yesterday"},
		"body":        {"notes"},
	}, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid happened-at status = %d, want 400", res.StatusCode)
	}
	rejected = parseResponse(t, res)
	if !strings.Contains(textContent(rejected), "Happened-at is invalid.") {
		t.Fatalf("text = %q, want an invalid happened-at error", textContent(rejected))
	}
	if got := attr(elementByName(rejected, "input", "title"), "value"); got != "Bad time" {
		t.Fatalf("title = %q, want the submitted title", got)
	}
	if got := attr(elementByName(rejected, "input", "happened_at"), "value"); got != "yesterday" {
		t.Fatalf("happened-at = %q, want the submitted value", got)
	}
	assertMeetingUnchanged(t, app, meetingURL)
}

func TestTaskListsStayLiveOutsideEdit(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postTask(t, app, ledger, "Ship it")
	postMeetingProjects(t, app, "Planning", "2026-01-02T15:04", "original body", ledger)

	_, feed := getPage(t, app, "/")
	meetingURL := linkHref(feed, "Planning")
	_, page := getPage(t, app, meetingURL)
	if linkHref(page, "Edit") == "" {
		t.Fatal("the Meeting page should still open the edit")
	}
	row := taskRowByText(page, "Ship it")
	if row == nil || attr(row, "data-task") == "" {
		t.Fatal("the Meeting shows its Tasks outside the edit")
	}
	done := taskForm(row, "checkbox")
	if done == nil || attr(done, "action") == meetingURL || !strings.HasPrefix(attr(done, "action"), "/tasks/") {
		t.Fatalf("done form action = %q, want a Task route, not Save", attr(done, "action"))
	}

	res := postFormNoRedirect(t, app, attr(done, "action"), url.Values{"done": {"1"}}, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("tick status = %d, want 303 without Save", res.StatusCode)
	}
	_, page = getPage(t, app, meetingURL)
	assertTasks(t, page, []taskState{{text: "Ship it", done: true}})
	if linkHref(page, "Planning") != meetingURL || !strings.Contains(textContent(page), "original body") {
		t.Fatal("ticking a Task changed the Meeting")
	}
	_, project := getPage(t, app, ledger)
	assertTasks(t, project, []taskState{{text: "Ship it", done: true}})
}

func TestEditShowsTheSavedReferences(t *testing.T) {
	app := startApp(t)
	alpha := postProject(t, app, "Alpha")
	postTask(t, app, alpha, "Alpha task")
	beta := postProject(t, app, "Beta")
	postProject(t, app, "Gamma")
	postMeetingProjects(t, app, "Planning", "2026-01-02T15:04", "both", beta, alpha)

	_, feed := getPage(t, app, "/")
	meetingURL := linkHref(feed, "Planning")
	_, page := getPage(t, app, meetingURL)
	_, edit := getPage(t, app, linkHref(page, "Edit"))
	if got := pendingProjectHrefs(edit); !sameStrings(got, []string{beta, alpha}) {
		t.Fatalf("pending = %#v, want Beta then Alpha", got)
	}
	if got := hiddenProjectIDs(edit); !sameStrings(got, []string{projectIDString(beta), projectIDString(alpha)}) {
		t.Fatalf("project fields = %#v, want Beta then Alpha", got)
	}
	assertTasks(t, edit, []taskState{{text: "Alpha task", done: false}})
	if addTaskFormFor(edit, beta) == nil {
		t.Fatal("a saved Reference with no Tasks still shows an add-task field")
	}
	if got := pickerNames(edit); !sameStrings(got, []string{"Gamma"}) {
		t.Fatalf("picker = %#v, want Gamma, the Project not already referenced", got)
	}
	if hasAttribute(edit, "data-citation") || elementByName(edit, "textarea", "notes") != nil {
		t.Fatal("the edit shows the citation rule or notes")
	}

	res := postFormNoRedirect(t, app, meetingURL, composerValues(edit, url.Values{
		"title": {"Planning"},
		"body":  {"both"},
	}), nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != meetingURL {
		t.Fatalf("save status = %d location = %q, want 303 %s", res.StatusCode, res.Header.Get("Location"), meetingURL)
	}
	_, page = getPage(t, app, meetingURL)
	if got := projectHrefsIn(meetingCard(page, "Planning")); !sameStrings(got, []string{beta, alpha}) {
		t.Fatalf("references = %#v, want Beta then Alpha", got)
	}
}

func TestAddingAProjectDuringEditCitesItOnlyAfterSave(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postTask(t, app, ledger, "Ship it")
	other := postProject(t, app, "Other")
	postTask(t, app, other, "Other task")
	postMeetingProjects(t, app, "Planning", "2026-01-02T15:04", "just ledger", ledger)

	_, feed := getPage(t, app, "/")
	meetingURL := linkHref(feed, "Planning")
	_, page := getPage(t, app, meetingURL)
	_, edit := getPage(t, app, linkHref(page, "Edit"))
	add := pickerButton(edit, "Other")
	if add == nil || attr(add, "formaction") == "" || attr(add, "formaction") == "/projects" {
		t.Fatalf("picker action = %q, want the edit's own action", attr(add, "formaction"))
	}

	res := postFormNoRedirect(t, app, attr(add, "formaction"), composerValues(edit, url.Values{
		"add": {attr(add, "value")},
	}), nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("add status = %d, want to stay on the edit", res.StatusCode)
	}
	edit = parseResponse(t, res)
	if got := pendingProjectHrefs(edit); !sameStrings(got, []string{ledger, other}) {
		t.Fatalf("pending = %#v, want Ledger then Other", got)
	}
	assertTasks(t, edit, []taskState{
		{text: "Ship it", done: false},
		{text: "Other task", done: false},
	})
	if got := pickerNames(edit); len(got) != 0 {
		t.Fatalf("picker = %#v, want Other omitted once chosen", got)
	}
	if got := attr(elementByName(edit, "input", "title"), "value"); got != "Planning" {
		t.Fatalf("title = %q, want the edit to keep it", got)
	}

	_, otherPage := getPage(t, app, other)
	if hasAttribute(otherPage, "data-citation") || strings.Contains(textContent(otherPage), "Planning") {
		t.Fatal("adding a Project during edit cited the Meeting before Save")
	}
	_, page = getPage(t, app, meetingURL)
	if got := projectHrefsIn(meetingCard(page, "Planning")); !sameStrings(got, []string{ledger}) {
		t.Fatalf("references = %#v, want only the saved Ledger", got)
	}

	res = postFormNoRedirect(t, app, meetingURL, composerValues(edit, nil), nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != meetingURL {
		t.Fatalf("save status = %d location = %q, want 303 %s", res.StatusCode, res.Header.Get("Location"), meetingURL)
	}
	_, page = getPage(t, app, meetingURL)
	if got := projectHrefsIn(meetingCard(page, "Planning")); !sameStrings(got, []string{ledger, other}) {
		t.Fatalf("references = %#v, want Ledger then Other after Save", got)
	}
	_, otherPage = getPage(t, app, other)
	if got := citationSummaries(otherPage); !sameStrings(got, []string{"2 Jan 2026, 15:04 Planning"}) {
		t.Fatalf("citations = %#v, want the Meeting after Save", got)
	}
	if got := linkHref(citationEntries(otherPage)[0], "Planning"); got != meetingURL {
		t.Fatalf("citation link = %q, want %s", got, meetingURL)
	}
}

func TestCreatingAProjectDuringEditStaysOnTheEdit(t *testing.T) {
	app := startApp(t)
	postMeeting(t, app, "Planning", "2026-01-02T15:04", "notes")
	_, feed := getPage(t, app, "/")
	meetingURL := linkHref(feed, "Planning")
	_, page := getPage(t, app, meetingURL)
	_, edit := getPage(t, app, linkHref(page, "Edit"))
	if elementByName(edit, "input", "name") == nil {
		t.Fatal("the edit picker must create a Project by name")
	}
	create := submitControl(edit, "Create Project")
	if create == nil || attr(create, "formaction") == "" || attr(create, "formaction") == "/projects" {
		t.Fatalf("create action = %q, want the edit's own action", attr(create, "formaction"))
	}

	res := postFormNoRedirect(t, app, attr(create, "formaction"), composerValues(edit, url.Values{
		"name":   {" Ledger "},
		"create": {"1"},
	}), nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("create status = %d, want to stay on the edit", res.StatusCode)
	}
	edit = parseResponse(t, res)
	if got := pendingProjectNames(edit); !sameStrings(got, []string{"Ledger"}) {
		t.Fatalf("pending = %#v, want the trimmed Ledger", got)
	}
	pending := pendingProjectHrefs(edit)
	if len(pending) != 1 || addTaskFormFor(edit, pending[0]) == nil {
		t.Fatal("creating a Project during edit should show its Tasks immediately")
	}
	if got := attr(elementByName(edit, "input", "name"), "value"); got != "" {
		t.Fatalf("name = %q, want the field cleared after create", got)
	}
	_, list := getPage(t, app, "/projects")
	if linkHref(list, "Ledger") != pending[0] {
		t.Fatal("the Project was not saved before the Meeting")
	}
	_, ledger := getPage(t, app, pending[0])
	if hasAttribute(ledger, "data-citation") || strings.Contains(textContent(ledger), "Planning") {
		t.Fatal("creating a Project during edit cited the Meeting before Save")
	}

	res = postFormNoRedirect(t, app, meetingURL, composerValues(edit, nil), nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != meetingURL {
		t.Fatalf("save status = %d location = %q, want 303 %s", res.StatusCode, res.Header.Get("Location"), meetingURL)
	}
	_, page = getPage(t, app, meetingURL)
	if got := projectHrefsIn(meetingCard(page, "Planning")); !sameStrings(got, pending) {
		t.Fatalf("references = %#v, want Ledger after Save", got)
	}
	_, ledger = getPage(t, app, pending[0])
	if got := citationSummaries(ledger); !sameStrings(got, []string{"2 Jan 2026, 15:04 Planning"}) {
		t.Fatalf("citations = %#v, want the Meeting after Save", got)
	}
}

func TestCancelAfterAddingAProjectDoesNotCiteIt(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	other := postProject(t, app, "Other")
	postTask(t, app, other, "Other task")
	postMeetingProjects(t, app, "Planning", "2026-01-02T15:04", "just ledger", ledger)

	_, feed := getPage(t, app, "/")
	meetingURL := linkHref(feed, "Planning")
	edit := openMeetingEdit(t, app, meetingURL)
	add := pickerButton(edit, "Other")
	res := postFormNoRedirect(t, app, attr(add, "formaction"), composerValues(edit, url.Values{
		"add": {attr(add, "value")},
	}), nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("add status = %d, want to stay on the edit", res.StatusCode)
	}
	edit = parseResponse(t, res)
	if got := linkHref(edit, "Cancel"); got != meetingURL {
		t.Fatalf("Cancel = %q, want %s", got, meetingURL)
	}

	_, page := getPage(t, app, meetingURL)
	if got := projectHrefsIn(meetingCard(page, "Planning")); !sameStrings(got, []string{ledger}) {
		t.Fatalf("references = %#v, want Cancel to drop the uncommitted Project", got)
	}
	_, otherPage := getPage(t, app, other)
	if hasAttribute(otherPage, "data-citation") || strings.Contains(textContent(otherPage), "Planning") {
		t.Fatal("Cancel cited the Meeting for a Project that was not saved")
	}
	_, ledgerPage := getPage(t, app, ledger)
	if got := citationSummaries(ledgerPage); !sameStrings(got, []string{"2 Jan 2026, 15:04 Planning"}) {
		t.Fatalf("citations = %#v, want the saved Reference kept", got)
	}
}

func TestRemovingAReferenceDropsTheCitation(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postTask(t, app, ledger, "Ship it")
	postProjectNotes(t, app, ledger, "private notes")
	other := postProject(t, app, "Other")
	postTask(t, app, other, "Stay")
	postMeetingProjects(t, app, "Also", "2026-02-02T09:00", "other only", other)
	postMeetingProjects(t, app, "Planning", "2026-01-02T15:04", "both", ledger, other)

	_, feed := getPage(t, app, "/")
	meetingURL := linkHref(feed, "Planning")
	edit := openMeetingEdit(t, app, meetingURL)
	remove := removeButton(edit, projectIDString(ledger))
	if remove == nil {
		t.Fatal("a saved Reference can be removed")
	}
	res := postFormNoRedirect(t, app, attr(remove, "formaction"), composerValues(edit, url.Values{
		"remove": {attr(remove, "value")},
	}), nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("remove status = %d, want to stay on the edit", res.StatusCode)
	}
	edit = parseResponse(t, res)
	if got := pendingProjectHrefs(edit); !sameStrings(got, []string{other}) {
		t.Fatalf("pending = %#v, want only Other", got)
	}

	res = postFormNoRedirect(t, app, meetingURL, composerValues(edit, nil), nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != meetingURL {
		t.Fatalf("save status = %d location = %q, want 303 %s", res.StatusCode, res.Header.Get("Location"), meetingURL)
	}

	_, ledgerPage := getPage(t, app, ledger)
	if hasAttribute(ledgerPage, "data-citation") || strings.Contains(textContent(ledgerPage), "Planning") {
		t.Fatal("removing the last Reference left the citation rule")
	}
	assertTasks(t, ledgerPage, []taskState{{text: "Ship it", done: false}})
	if got := textContent(elementByName(ledgerPage, "textarea", "notes")); got != "private notes" {
		t.Fatalf("notes = %q, want them kept", got)
	}
	_, otherPage := getPage(t, app, other)
	if got := citationSummaries(otherPage); !sameStrings(got, []string{
		"2 Feb 2026, 09:00 Also",
		"2 Jan 2026, 15:04 Planning",
	}) {
		t.Fatalf("citations = %#v, want Planning kept where it still references Other", got)
	}
	_, page := getPage(t, app, meetingURL)
	if got := projectHrefsIn(meetingCard(page, "Planning")); !sameStrings(got, []string{other}) {
		t.Fatalf("references = %#v, want only Other", got)
	}

	edit = openMeetingEdit(t, app, meetingURL)
	remove = removeButton(edit, projectIDString(other))
	res = postFormNoRedirect(t, app, attr(remove, "formaction"), composerValues(edit, url.Values{
		"remove": {attr(remove, "value")},
	}), nil)
	edit = parseResponse(t, res)
	res = postFormNoRedirect(t, app, meetingURL, composerValues(edit, nil), nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("save status = %d, want 303", res.StatusCode)
	}
	_, otherPage = getPage(t, app, other)
	if got := citationSummaries(otherPage); !sameStrings(got, []string{"2 Feb 2026, 09:00 Also"}) {
		t.Fatalf("citations = %#v, want Planning dropped and Also kept", got)
	}
	_, page = getPage(t, app, meetingURL)
	if got := projectHrefsIn(meetingCard(page, "Planning")); len(got) != 0 {
		t.Fatalf("references = %#v, want none", got)
	}
	if !strings.Contains(textContent(page), "both") {
		t.Fatal("removing References cleared the Meeting body")
	}
}

func TestCancelAfterRemovingAReferencePutsItBack(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postMeetingProjects(t, app, "Planning", "2026-01-02T15:04", "original body", ledger)

	_, feed := getPage(t, app, "/")
	meetingURL := linkHref(feed, "Planning")
	edit := openMeetingEdit(t, app, meetingURL)
	remove := removeButton(edit, projectIDString(ledger))
	res := postFormNoRedirect(t, app, attr(remove, "formaction"), composerValues(edit, url.Values{
		"title":       {"Draft"},
		"happened_at": {"2026-08-08T08:08"},
		"body":        {"draft body"},
		"remove":      {attr(remove, "value")},
	}), nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("remove status = %d, want to stay on the edit", res.StatusCode)
	}
	edit = parseResponse(t, res)
	if len(pendingProjectHrefs(edit)) != 0 {
		t.Fatal("the edit kept a Reference it removed")
	}

	_, page := getPage(t, app, linkHref(edit, "Cancel"))
	text := textContent(page)
	if linkHref(page, "Planning") != meetingURL || !strings.Contains(text, "2 Jan 2026, 15:04") || !strings.Contains(text, "original body") {
		t.Fatalf("Meeting text = %q, want Cancel to restore the saved fields", text)
	}
	if strings.Contains(text, "Draft") || strings.Contains(text, "draft body") || strings.Contains(text, "8 Aug 2026, 08:08") {
		t.Fatalf("Meeting text = %q, want the uncommitted edit gone", text)
	}
	if got := projectHrefsIn(meetingCard(page, "Planning")); !sameStrings(got, []string{ledger}) {
		t.Fatalf("references = %#v, want the removed Reference back", got)
	}
	_, ledgerPage := getPage(t, app, ledger)
	if got := citationSummaries(ledgerPage); !sameStrings(got, []string{"2 Jan 2026, 15:04 Planning"}) {
		t.Fatalf("citations = %#v, want the Meeting still cited", got)
	}
}

func TestSavedReferenceOrderFollowsAddOrder(t *testing.T) {
	app := startApp(t)
	alpha := postProject(t, app, "Alpha")
	beta := postProject(t, app, "Beta")
	gamma := postProject(t, app, "Gamma")
	postMeetingProjects(t, app, "Planning", "2026-01-02T15:04", "ordered", gamma, alpha)

	_, feed := getPage(t, app, "/")
	meetingURL := linkHref(feed, "Planning")
	edit := openMeetingEdit(t, app, meetingURL)
	remove := removeButton(edit, projectIDString(alpha))
	res := postFormNoRedirect(t, app, attr(remove, "formaction"), composerValues(edit, url.Values{
		"remove": {attr(remove, "value")},
	}), nil)
	edit = parseResponse(t, res)
	res = postFormNoRedirect(t, app, attr(pickerButton(edit, "Beta"), "formaction"), composerValues(edit, url.Values{
		"add": {attr(pickerButton(edit, "Beta"), "value")},
	}), nil)
	edit = parseResponse(t, res)
	res = postFormNoRedirect(t, app, attr(pickerButton(edit, "Alpha"), "formaction"), composerValues(edit, url.Values{
		"add": {attr(pickerButton(edit, "Alpha"), "value")},
	}), nil)
	edit = parseResponse(t, res)
	if got := pendingProjectNames(edit); !sameStrings(got, []string{"Gamma", "Beta", "Alpha"}) {
		t.Fatalf("pending = %#v, want Alpha back at the end", got)
	}
	if got := pickerNames(edit); len(got) != 0 {
		t.Fatalf("picker = %#v, want each Project only once", got)
	}

	res = postFormNoRedirect(t, app, meetingURL, composerValues(edit, nil), nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != meetingURL {
		t.Fatalf("save status = %d location = %q, want 303 %s", res.StatusCode, res.Header.Get("Location"), meetingURL)
	}
	_, page := getPage(t, app, meetingURL)
	if got := projectNamesIn(meetingCard(page, "Planning")); !sameStrings(got, []string{"Gamma", "Beta", "Alpha"}) {
		t.Fatalf("references = %#v, want add order at Save", got)
	}

	res = postFormNoRedirect(t, app, meetingURL, url.Values{
		"title":       {"Planning"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"ordered"},
		"project":     {projectIDString(gamma), projectIDString(alpha), projectIDString(alpha), projectIDString(beta)},
	}, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("save status = %d, want 303", res.StatusCode)
	}
	_, page = getPage(t, app, meetingURL)
	if got := projectNamesIn(meetingCard(page, "Planning")); !sameStrings(got, []string{"Gamma", "Alpha", "Beta"}) {
		t.Fatalf("references = %#v, want each Project once, in the submitted order", got)
	}
}

func TestChangingTheMeetingResortsTheFeedAndCitation(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postMeetingProjects(t, app, "First", "2026-03-01T10:00", "created earlier", ledger)
	postMeetingProjects(t, app, "Second", "2026-03-01T10:00", "created later", ledger)

	_, feed := getPage(t, app, "/")
	firstURL := linkHref(feed, "First")
	edit := openMeetingEdit(t, app, firstURL)
	res := postFormNoRedirect(t, app, firstURL, composerValues(edit, url.Values{
		"title":       {"First moved"},
		"happened_at": {"2026-04-04T09:00"},
	}), nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != firstURL {
		t.Fatalf("save status = %d location = %q, want 303 %s", res.StatusCode, res.Header.Get("Location"), firstURL)
	}

	_, feed = getPage(t, app, "/")
	feedText := textContent(feed)
	moved := strings.Index(feedText, "First moved")
	second := strings.Index(feedText, "Second")
	if moved < 0 || second < 0 || moved > second {
		t.Fatalf("feed order = %q, want First moved before Second", feedText)
	}
	_, ledgerPage := getPage(t, app, ledger)
	if got := citationSummaries(ledgerPage); !sameStrings(got, []string{
		"4 Apr 2026, 09:00 First moved",
		"1 Mar 2026, 10:00 Second",
	}) {
		t.Fatalf("citations = %#v, want happened-at order and the new title", got)
	}
	entry := citationEntries(ledgerPage)[0]
	if got := linkHref(entry, "First moved"); got != firstURL {
		t.Fatalf("citation link = %q, want %s", got, firstURL)
	}

	edit = openMeetingEdit(t, app, firstURL)
	res = postFormNoRedirect(t, app, firstURL, composerValues(edit, url.Values{
		"happened_at": {"2026-03-01T10:00"},
	}), nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("save status = %d, want 303", res.StatusCode)
	}
	_, feed = getPage(t, app, "/")
	feedText = textContent(feed)
	moved = strings.Index(feedText, "First moved")
	second = strings.Index(feedText, "Second")
	if moved < 0 || second < 0 || second > moved {
		t.Fatalf("feed order = %q, want the later-created Meeting first when happened-at ties", feedText)
	}
	_, ledgerPage = getPage(t, app, ledger)
	if got := citationSummaries(ledgerPage); !sameStrings(got, []string{
		"1 Mar 2026, 10:00 Second",
		"1 Mar 2026, 10:00 First moved",
	}) {
		t.Fatalf("citations = %#v, want the same order as the feed", got)
	}
}

func TestTaskEditsDuringMeetingEditRemainAfterCancel(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postTask(t, app, ledger, "Keep")
	postTask(t, app, ledger, "Drop")
	postTask(t, app, ledger, "Toggle")
	postMeetingProjects(t, app, "Planning", "2026-01-02T15:04", "original body", ledger)

	_, feed := getPage(t, app, "/")
	meetingURL := linkHref(feed, "Planning")
	_, page := getPage(t, app, meetingURL)
	editURL := linkHref(page, "Edit")

	edit := openMeetingEdit(t, app, meetingURL)
	add := addTaskFormFor(edit, ledger)
	res := postFormNoRedirect(t, app, attr(add, "action"), url.Values{"text": {"Added"}}, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("add status = %d, want the Task saved immediately", res.StatusCode)
	}

	edit = openMeetingEdit(t, app, meetingURL)
	keep := taskForm(taskRowByText(edit, "Keep"), "text")
	res = postFormNoRedirect(t, app, attr(keep, "action"), url.Values{"text": {"Keep revised"}}, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("edit status = %d, want the Task text saved immediately", res.StatusCode)
	}

	edit = openMeetingEdit(t, app, meetingURL)
	toggle := taskForm(taskRowByText(edit, "Toggle"), "checkbox")
	if toggle == nil || attr(toggle, "action") == meetingURL {
		t.Fatal("ticking a Task on the edit must not require Save")
	}
	res = postFormNoRedirect(t, app, attr(toggle, "action"), url.Values{"done": {"1"}}, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("toggle status = %d, want the Task saved immediately", res.StatusCode)
	}

	edit = openMeetingEdit(t, app, meetingURL)
	drop := taskRowByText(edit, "Drop")
	deleteForm := taskForm(drop, "delete")
	if deleteForm == nil || strings.TrimSpace(textContent(elementByName(deleteForm, "button", ""))) != "Delete" {
		t.Fatal("deleting a Task on the edit has no confirmation step")
	}
	res = postFormNoRedirect(t, app, attr(deleteForm, "action"), nil, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want the Task deleted immediately", res.StatusCode)
	}

	edit = openMeetingEdit(t, app, meetingURL)
	if linkHref(edit, "Cancel") != meetingURL || editURL == "" {
		t.Fatal("Cancel should leave the edit without saving it")
	}
	_, page = getPage(t, app, meetingURL)
	text := textContent(page)
	if linkHref(page, "Planning") != meetingURL || !strings.Contains(text, "2 Jan 2026, 15:04") || !strings.Contains(text, "original body") {
		t.Fatalf("Meeting text = %q, want the saved fields after Cancel", text)
	}
	if got := projectHrefsIn(meetingCard(page, "Planning")); !sameStrings(got, []string{ledger}) {
		t.Fatalf("references = %#v, want Ledger", got)
	}
	want := []taskState{
		{text: "Toggle", done: true},
		{text: "Keep revised", done: false},
		{text: "Added", done: false},
	}
	assertTasks(t, meetingCard(page, "Planning"), want)
	_, ledgerPage := getPage(t, app, ledger)
	assertTasks(t, ledgerPage, want)
	if got := citationSummaries(ledgerPage); !sameStrings(got, []string{"2 Jan 2026, 15:04 Planning"}) {
		t.Fatalf("citations = %#v, want the saved Reference", got)
	}
}

func TestRejectedSaveRedisplaysPendingReferences(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postTask(t, app, ledger, "Ship it")
	other := postProject(t, app, "Other")
	postMeetingProjects(t, app, "Planning", "2026-01-02T15:04", "original body", ledger)

	_, feed := getPage(t, app, "/")
	meetingURL := linkHref(feed, "Planning")
	edit := openMeetingEdit(t, app, meetingURL)
	add := pickerButton(edit, "Other")
	res := postFormNoRedirect(t, app, attr(add, "formaction"), composerValues(edit, url.Values{
		"add": {attr(add, "value")},
	}), nil)
	edit = parseResponse(t, res)

	res = postFormNoRedirect(t, app, meetingURL, composerValues(edit, url.Values{
		"title":       {""},
		"happened_at": {"2026-05-05T08:00"},
		"body":        {"not saved"},
	}), nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
	rejected := parseResponse(t, res)
	if !strings.Contains(textContent(rejected), "Title is required.") {
		t.Fatalf("text = %q, want a title error", textContent(rejected))
	}
	if got := attr(elementByName(rejected, "input", "happened_at"), "value"); got != "2026-05-05T08:00" {
		t.Fatalf("happened-at = %q, want the submitted value", got)
	}
	if !strings.Contains(textContent(elementByName(rejected, "textarea", "body")), "not saved") {
		t.Fatal("rejected Save dropped the body")
	}
	if got := pendingProjectHrefs(rejected); !sameStrings(got, []string{ledger, other}) {
		t.Fatalf("pending = %#v, want Ledger then Other", got)
	}
	assertTasks(t, rejected, []taskState{{text: "Ship it", done: false}})
	if addTaskFormFor(rejected, other) == nil {
		t.Fatal("rejected Save dropped the pending Project's Tasks")
	}

	_, page := getPage(t, app, meetingURL)
	text := textContent(page)
	if linkHref(page, "Planning") != meetingURL || !strings.Contains(text, "original body") || strings.Contains(text, "not saved") {
		t.Fatalf("Meeting text = %q, want the stored Meeting unchanged", text)
	}
	if got := projectHrefsIn(meetingCard(page, "Planning")); !sameStrings(got, []string{ledger}) {
		t.Fatalf("references = %#v, want only Ledger", got)
	}
	_, otherPage := getPage(t, app, other)
	if hasAttribute(otherPage, "data-citation") || strings.Contains(textContent(otherPage), "Planning") {
		t.Fatal("a rejected Save cited the pending Project")
	}
}

func TestDeleteMeetingRequiresConfirmation(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postTask(t, app, ledger, "Ship it")
	postProjectNotes(t, app, ledger, "private notes")
	other := postProject(t, app, "Other")
	postTask(t, app, other, "Stay")
	postMeetingProjects(t, app, "Also", "2026-02-02T09:00", "still cited", ledger)
	postMeetingProjects(t, app, "Planning", "2026-01-02T15:04", "keep the projects", ledger, other)

	_, feed := getPage(t, app, "/")
	meetingURL := linkHref(feed, "Planning")
	alsoURL := linkHref(feed, "Also")
	_, page := getPage(t, app, meetingURL)
	if got := linkHref(page, "Delete"); got != meetingURL+"/delete" {
		t.Fatalf("Delete link = %q, want %s", got, meetingURL+"/delete")
	}

	status, confirm := getPage(t, app, meetingURL+"/delete")
	if status != http.StatusOK {
		t.Fatalf("confirmation status = %d, want 200", status)
	}
	if !strings.Contains(textContent(confirm), "Planning") {
		t.Fatalf("confirmation text = %q, want the Meeting named", textContent(confirm))
	}
	form := elementByName(confirm, "form", "")
	if form == nil || strings.ToLower(attr(form, "method")) != "post" || attr(form, "action") != meetingURL+"/delete" {
		method, action := "", ""
		if form != nil {
			method, action = attr(form, "method"), attr(form, "action")
		}
		t.Fatalf("confirmation form method=%q action=%q, want POST %s", method, action, meetingURL+"/delete")
	}
	if linkHref(confirm, "Cancel") != meetingURL {
		t.Fatalf("Cancel = %q, want to leave the Meeting unchanged", linkHref(confirm, "Cancel"))
	}

	_, page = getPage(t, app, meetingURL)
	if !strings.Contains(textContent(page), "keep the projects") || linkHref(page, "Planning") != meetingURL {
		t.Fatal("leaving the confirmation deleted the Meeting")
	}
	_, feed = getPage(t, app, "/")
	if linkHref(feed, "Planning") != meetingURL || linkHref(feed, "Also") != alsoURL {
		t.Fatal("leaving the confirmation changed the feed")
	}
	_, ledgerPage := getPage(t, app, ledger)
	if got := citationSummaries(ledgerPage); !sameStrings(got, []string{
		"2 Feb 2026, 09:00 Also",
		"2 Jan 2026, 15:04 Planning",
	}) {
		t.Fatalf("citations = %#v, want both Meetings before delete", got)
	}

	res := postFormNoRedirect(t, app, meetingURL+"/delete", nil, nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/" {
		t.Fatalf("delete status = %d location = %q, want 303 /", res.StatusCode, res.Header.Get("Location"))
	}

	_, feed = getPage(t, app, "/")
	feedText := textContent(feed)
	if strings.Contains(feedText, "Planning") || strings.Contains(feedText, "keep the projects") || linkHref(feed, "Also") != alsoURL {
		t.Fatalf("feed text = %q, want Planning gone and Also kept", feedText)
	}
	status, missing := getPage(t, app, meetingURL)
	if status != http.StatusNotFound || !strings.Contains(textContent(missing), "Meeting not found.") {
		t.Fatalf("deleted Meeting status=%d text=%q", status, textContent(missing))
	}
	for _, path := range []string{meetingURL + "/edit", meetingURL + "/delete"} {
		status, missing = getPage(t, app, path)
		if status != http.StatusNotFound || !strings.Contains(textContent(missing), "Meeting not found.") {
			t.Fatalf("%s status=%d text=%q, want the Meeting gone", path, status, textContent(missing))
		}
	}

	_, ledgerPage = getPage(t, app, ledger)
	if got := citationSummaries(ledgerPage); !sameStrings(got, []string{"2 Feb 2026, 09:00 Also"}) {
		t.Fatalf("citations = %#v, want Planning removed and Also kept", got)
	}
	assertTasks(t, ledgerPage, []taskState{{text: "Ship it", done: false}})
	if got := textContent(elementByName(ledgerPage, "textarea", "notes")); got != "private notes" {
		t.Fatalf("notes = %q, want them kept", got)
	}
	_, otherPage := getPage(t, app, other)
	if hasAttribute(otherPage, "data-citation") || strings.Contains(textContent(otherPage), "Planning") {
		t.Fatal("deleting the last referencing Meeting left the citation rule")
	}
	assertTasks(t, otherPage, []taskState{{text: "Stay", done: false}})
	_, list := getPage(t, app, "/projects")
	if linkHref(list, "Ledger") != ledger || linkHref(list, "Other") != other {
		t.Fatal("deleting a Meeting removed a Project")
	}
}

func openMeetingEdit(t *testing.T, app *flow.App, meetingURL string) *html.Node {
	t.Helper()

	_, page := getPage(t, app, meetingURL)
	editURL := linkHref(page, "Edit")
	status, edit := getPage(t, app, editURL)
	if status != http.StatusOK {
		t.Fatalf("edit status = %d, want 200", status)
	}

	return edit
}

func TestCancelReturnsTheSavedMeeting(t *testing.T) {
	app := startApp(t)
	postMeeting(t, app, "Planning", "2026-01-02T15:04", "original body")
	_, feed := getPage(t, app, "/")
	meetingURL := linkHref(feed, "Planning")
	_, page := getPage(t, app, meetingURL)
	_, edit := getPage(t, app, linkHref(page, "Edit"))

	if got := linkHref(edit, "Cancel"); got != meetingURL {
		t.Fatalf("Cancel = %q, want %s", got, meetingURL)
	}

	res := postFormNoRedirect(t, app, meetingURL, url.Values{
		"title":       {"Draft title"},
		"happened_at": {""},
		"body":        {"draft body"},
	}, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("rejected save status = %d, want 400", res.StatusCode)
	}
	rejected := parseResponse(t, res)
	if got := linkHref(rejected, "Cancel"); got != meetingURL {
		t.Fatalf("Cancel after a rejected Save = %q, want %s", got, meetingURL)
	}

	_, page = getPage(t, app, linkHref(rejected, "Cancel"))
	text := textContent(page)
	if linkHref(page, "Planning") != meetingURL || !strings.Contains(text, "2 Jan 2026, 15:04") || !strings.Contains(text, "original body") {
		t.Fatalf("Meeting text = %q, want Cancel to restore the saved Meeting", text)
	}
	if strings.Contains(text, "Draft title") || strings.Contains(text, "draft body") {
		t.Fatalf("Meeting text = %q, want the uncommitted edit gone", text)
	}
}

func assertMeetingUnchanged(t *testing.T, app *flow.App, meetingURL string) {
	t.Helper()

	_, page := getPage(t, app, meetingURL)
	text := textContent(page)
	if linkHref(page, "Planning") != meetingURL || !strings.Contains(text, "2 Jan 2026, 15:04") || !strings.Contains(text, "original body") {
		t.Fatalf("Meeting text = %q, want the stored Meeting unchanged", text)
	}
	if strings.Contains(text, "should not be saved") || strings.Contains(text, "Renamed") || strings.Contains(text, "notes I typed") || strings.Contains(text, "spaces") {
		t.Fatalf("Meeting text = %q, want a rejected Save to leave it unchanged", text)
	}
}

func formContaining(doc, control *html.Node) *html.Node {
	if control == nil {
		return nil
	}
	for n := control.Parent; n != nil; n = n.Parent {
		if n.Type == html.ElementNode && n.Data == "form" {
			return n
		}
	}
	if id := attr(control, "form"); id != "" {
		return elementByID(doc, id)
	}

	return nil
}

func submitControl(doc *html.Node, label string) *html.Node {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "button" && strings.TrimSpace(textContent(n)) == label {
			found = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return found
}
