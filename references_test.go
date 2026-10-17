package flow_test

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/absurek/flow"
	"golang.org/x/net/html"
)

func TestComposerPickerListsUnchosenProjectsAlphabetically(t *testing.T) {
	app := startApp(t)
	gamma := postProject(t, app, "Gamma")
	alpha := postProject(t, app, "alpha")
	beta := postProject(t, app, "Beta")

	_, doc := getPage(t, app, "/")
	if got := pickerNames(doc); !sameStrings(got, []string{"alpha", "Beta", "Gamma"}) {
		t.Fatalf("picker = %#v, want alpha, Beta, Gamma ignoring case", got)
	}
	if got := pendingProjectHrefs(doc); len(got) != 0 {
		t.Fatalf("pending = %#v, want no Projects chosen yet", got)
	}

	betaButton := pickerButton(doc, "Beta")
	if betaButton == nil || attr(betaButton, "value") == "" {
		t.Fatal("picker must offer Beta")
	}
	doc = postComposer(t, app, url.Values{
		"title":       {"Standup"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"talked about Beta"},
		"add":         {attr(betaButton, "value")},
	})
	if got := pickerNames(doc); !sameStrings(got, []string{"alpha", "Gamma"}) {
		t.Fatalf("picker = %#v, want chosen Projects omitted", got)
	}
	if got := pendingProjectHrefs(doc); !sameStrings(got, []string{beta}) {
		t.Fatalf("pending = %#v, want %s", got, beta)
	}
	if got := pendingProjectNames(doc); !sameStrings(got, []string{"Beta"}) {
		t.Fatalf("pending names = %#v, want Beta", got)
	}
	if got := attr(elementByName(doc, "input", "title"), "value"); got != "Standup" {
		t.Fatalf("title = %q, want the composer to keep Standup", got)
	}
	if got := attr(elementByName(doc, "input", "happened_at"), "value"); got != "2026-01-02T15:04" {
		t.Fatalf("happened-at = %q, want the composer to keep it", got)
	}
	if got := strings.TrimSpace(textContent(elementByName(doc, "textarea", "body"))); got != "talked about Beta" {
		t.Fatalf("body = %q, want the composer to keep it", got)
	}

	_, feed := getPage(t, app, "/")
	if strings.Contains(textContent(feed), "Standup") || strings.Contains(textContent(feed), "talked about Beta") {
		t.Fatal("picking a Project posted a Meeting")
	}
	if got := pickerNames(feed); !sameStrings(got, []string{"alpha", "Beta", "Gamma"}) {
		t.Fatalf("picker after leaving = %#v, want every Project again", got)
	}
	_, _ = alpha, gamma
}

func TestPickerRejectsBlankAndDuplicateProjectNames(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	_, doc := getPage(t, app, "/")
	doc = postComposer(t, app, url.Values{
		"title":       {"Planning"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"still writing"},
		"add":         {attr(pickerButton(doc, "Ledger"), "value")},
	})

	for _, name := range []string{"", "   "} {
		res := postFormNoRedirect(t, app, "/composer/projects", composerValues(doc, url.Values{
			"name":   {name},
			"create": {"1"},
		}), nil)
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("name %q status = %d, want 400", name, res.StatusCode)
		}
		rejected := parseResponse(t, res)
		if !strings.Contains(textContent(rejected), "Name is required.") {
			t.Fatalf("name %q text = %q, want a name error", name, textContent(rejected))
		}
		if got := attr(elementByName(rejected, "input", "name"), "value"); got != name {
			t.Fatalf("name field = %q, want the submitted %q", got, name)
		}
		if got := pendingProjectHrefs(rejected); !sameStrings(got, []string{ledger}) {
			t.Fatalf("pending = %#v, want Ledger kept", got)
		}
		if got := attr(elementByName(rejected, "input", "title"), "value"); got != "Planning" {
			t.Fatalf("title = %q, want the composer kept", got)
		}
	}

	for _, name := range []string{"ledger", " LEDGER ", "Ledger "} {
		res := postFormNoRedirect(t, app, "/composer/projects", composerValues(doc, url.Values{
			"name":   {name},
			"create": {"1"},
		}), nil)
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("name %q status = %d, want 400", name, res.StatusCode)
		}
		rejected := parseResponse(t, res)
		if !strings.Contains(textContent(rejected), "A Project with that name already exists.") {
			t.Fatalf("name %q text = %q, want a duplicate error", name, textContent(rejected))
		}
		if got := attr(elementByName(rejected, "input", "name"), "value"); got != name {
			t.Fatalf("name field = %q, want the submitted %q", got, name)
		}
		if got := pendingProjectHrefs(rejected); !sameStrings(got, []string{ledger}) {
			t.Fatalf("pending = %#v, want only Ledger", got)
		}
	}

	_, list := getPage(t, app, "/projects")
	if links := projectPageLinks(list); len(links) != 1 || links[0] != ledger {
		t.Fatalf("links = %v, want only Ledger", links)
	}
	_, feed := getPage(t, app, "/")
	if strings.Contains(textContent(feed), "Planning") || strings.Contains(textContent(feed), "still writing") {
		t.Fatal("a rejected Project create posted a Meeting")
	}
}

func TestCreatingAProjectFromThePickerSavesItAndShowsItsTasks(t *testing.T) {
	app := startApp(t)

	_, doc := getPage(t, app, "/")
	if elementByName(doc, "input", "name") == nil {
		t.Fatal("the picker must create a Project by name")
	}
	if pickerButton(doc, "Create Project") != nil {
		t.Fatal("Create Project is not an existing Project in the picker")
	}

	doc = postComposer(t, app, url.Values{
		"title":       {"Planning"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"not posted yet"},
		"name":        {" Ledger "},
		"create":      {"1"},
	})
	if got := pendingProjectNames(doc); !sameStrings(got, []string{"Ledger"}) {
		t.Fatalf("pending names = %#v, want the trimmed Ledger", got)
	}
	if got := pickerNames(doc); len(got) != 0 {
		t.Fatalf("picker = %#v, want the new Project omitted", got)
	}
	pending := pendingProjectHrefs(doc)
	if len(pending) != 1 {
		t.Fatalf("pending = %#v, want the new Project", pending)
	}
	form := addTaskForm(doc)
	if form == nil || strings.ToLower(attr(form, "method")) != "post" || attr(form, "action") != pending[0]+"/tasks" {
		method, action := "", ""
		if form != nil {
			method, action = attr(form, "method"), attr(form, "action")
		}
		t.Fatalf("add form method=%q action=%q, want POST %s/tasks", method, action, pending[0])
	}
	if got := taskTexts(doc); len(got) != 0 {
		t.Fatalf("tasks = %#v, want a new Project with no Tasks", got)
	}
	if got := attr(elementByName(doc, "input", "name"), "value"); got != "" {
		t.Fatalf("new Project field = %q, want it cleared after create", got)
	}

	_, list := getPage(t, app, "/projects")
	if got := linkHref(list, "Ledger"); got != pending[0] {
		t.Fatalf("Projects list link = %q, want %s saved before Post", got, pending[0])
	}
	_, page := getPage(t, app, pending[0])
	if got := attr(elementByName(page, "input", "name"), "value"); got != "Ledger" {
		t.Fatalf("name = %q, want Ledger", got)
	}
	if got := taskTexts(page); len(got) != 0 {
		t.Fatalf("tasks = %#v, want none", got)
	}

	_, feed := getPage(t, app, "/")
	text := textContent(feed)
	if strings.Contains(text, "Planning") || strings.Contains(text, "not posted yet") {
		t.Fatal("leaving the composer created a Meeting")
	}
	if got := pickerNames(feed); !sameStrings(got, []string{"Ledger"}) {
		t.Fatalf("picker after leaving = %#v, want the saved Project", got)
	}
	if got := pendingProjectHrefs(feed); len(got) != 0 {
		t.Fatalf("pending after leaving = %#v, want no Reference", got)
	}
}

func TestPickedProjectShowsItsCurrentTasksBeforePost(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postTask(t, app, ledger, "Older")
	postTask(t, app, ledger, "Newer")
	postTaskDone(t, app, ledger, "Older", true)
	other := postProject(t, app, "Other")
	postTask(t, app, other, "Elsewhere")

	_, doc := getPage(t, app, "/")
	button := pickerButton(doc, "Ledger")
	if button == nil {
		t.Fatal("picker must offer Ledger")
	}
	doc = postComposer(t, app, url.Values{
		"title":       {"Planning"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"during the meeting"},
		"add":         {attr(button, "value")},
	})
	if got := pendingProjectHrefs(doc); !sameStrings(got, []string{ledger}) {
		t.Fatalf("pending = %#v, want %s", got, ledger)
	}
	assertTasks(t, doc, []taskState{
		{text: "Older", done: true},
		{text: "Newer", done: false},
	})
	if taskRowByText(doc, "Elsewhere") != nil {
		t.Fatal("the composer showed another Project's Task")
	}

	_, feed := getPage(t, app, "/")
	if strings.Contains(textContent(feed), "during the meeting") {
		t.Fatal("showing Tasks posted a Meeting")
	}
	assertNoMeetings(t, feed)
}

func TestComposerTaskEditsPersistWhenTheMeetingIsNotPosted(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postTask(t, app, ledger, "Keep")
	postTask(t, app, ledger, "Drop")

	_, doc := getPage(t, app, "/")
	button := pickerButton(doc, "Ledger")
	doc = postComposer(t, app, url.Values{
		"title":       {"Abandoned"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"left behind"},
		"add":         {attr(button, "value")},
	})
	add := addTaskForm(doc)
	if add == nil || attr(add, "action") != ledger+"/tasks" {
		t.Fatalf("add form action = %q, want %s/tasks", attr(add, "action"), ledger)
	}
	res := postFormNoRedirect(t, app, attr(add, "action"), url.Values{"text": {"From the composer"}}, nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != ledger {
		t.Fatalf("add status = %d location = %q, want 303 %s", res.StatusCode, res.Header.Get("Location"), ledger)
	}

	doc = postComposer(t, app, url.Values{
		"title":       {"Abandoned"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"left behind"},
		"project":     {attr(button, "value")},
	})
	assertTasks(t, doc, []taskState{
		{text: "Keep", done: false},
		{text: "Drop", done: false},
		{text: "From the composer", done: false},
	})

	keep := taskRowByText(doc, "Keep")
	keepForm := taskForm(keep, "text")
	res = postFormNoRedirect(t, app, attr(keepForm, "action"), url.Values{"text": {"Kept"}}, nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != ledger {
		t.Fatalf("edit status = %d location = %q, want 303 %s", res.StatusCode, res.Header.Get("Location"), ledger)
	}

	doc = postComposer(t, app, url.Values{
		"title":       {"Abandoned"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"left behind"},
		"project":     {attr(button, "value")},
	})
	added := taskRowByText(doc, "From the composer")
	doneForm := taskForm(added, "checkbox")
	if doneForm == nil {
		t.Fatal("the composer list must toggle a Task")
	}
	res = postFormNoRedirect(t, app, attr(doneForm, "action"), url.Values{"done": {"1"}}, nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != ledger {
		t.Fatalf("toggle status = %d location = %q, want 303 %s", res.StatusCode, res.Header.Get("Location"), ledger)
	}

	doc = postComposer(t, app, url.Values{
		"title":       {"Abandoned"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"left behind"},
		"project":     {attr(button, "value")},
	})
	drop := taskRowByText(doc, "Drop")
	deleteForm := taskForm(drop, "delete")
	if deleteForm == nil || strings.TrimSpace(textContent(elementByName(deleteForm, "button", ""))) != "Delete" {
		t.Fatal("deleting a Task has no confirmation step, only Delete")
	}
	if strings.Contains(attr(deleteForm, "action"), "/delete") && strings.Contains(attr(deleteForm, "action"), "/projects/") {
		t.Fatal("Task delete must not be a Project delete")
	}
	res = postFormNoRedirect(t, app, attr(deleteForm, "action"), nil, nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != ledger {
		t.Fatalf("delete status = %d location = %q, want 303 %s", res.StatusCode, res.Header.Get("Location"), ledger)
	}

	_, page := getPage(t, app, ledger)
	assertTasks(t, page, []taskState{
		{text: "From the composer", done: true},
		{text: "Kept", done: false},
	})

	_, feed := getPage(t, app, "/")
	if strings.Contains(textContent(feed), "Abandoned") || strings.Contains(textContent(feed), "left behind") {
		t.Fatal("leaving after Task edits created a Meeting")
	}
	assertNoMeetings(t, feed)
	if got := pendingProjectHrefs(feed); len(got) != 0 {
		t.Fatalf("pending after leaving = %#v, want no Reference", got)
	}
	_, list := getPage(t, app, "/projects")
	if linkHref(list, "Ledger") != ledger {
		t.Fatal("the Project disappeared with the abandoned composer")
	}
}

func TestMeetingOmitsCitationNotesAndRename(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postTask(t, app, ledger, "Ship it")
	postProjectNotes(t, app, ledger, "private notes")

	postFormNoRedirect(t, app, "/meetings", url.Values{
		"title":       {"Planning"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"just the discussion"},
		"project":     {projectIDString(ledger)},
	}, nil)
	_, feed := getPage(t, app, "/")
	assertMeetingHasNoProjectEditor(t, meetingCard(feed, "Planning"))
	if strings.Contains(textContent(meetingCard(feed, "Planning")), "private notes") {
		t.Fatal("the feed Meeting showed the Project notes")
	}

	_, page := getPage(t, app, linkHref(feed, "Planning"))
	assertMeetingHasNoProjectEditor(t, page)
	text := textContent(page)
	if strings.Contains(text, "private notes") || strings.Contains(strings.ToLower(text), "citation") {
		t.Fatalf("Meeting text = %q, want no notes and no citation rule", text)
	}
	if !strings.Contains(text, "Ledger") || !strings.Contains(text, "just the discussion") {
		t.Fatalf("Meeting text = %q, want the Project and the body", text)
	}
	if got := taskTexts(page); !sameStrings(got, []string{"Ship it"}) {
		t.Fatalf("tasks = %#v, want Ship it", got)
	}

	_, project := getPage(t, app, ledger)
	if got := textContent(elementByName(project, "textarea", "notes")); got != "private notes" {
		t.Fatalf("notes = %q, want them kept on the Project page", got)
	}
}

func assertMeetingHasNoProjectEditor(t *testing.T, doc *html.Node) {
	t.Helper()

	if doc == nil {
		t.Fatal("missing Meeting")
	}
	if hasAttribute(doc, "data-citation") {
		t.Fatal("the Meeting shows the citation rule")
	}
	if elementByName(doc, "textarea", "notes") != nil {
		t.Fatal("the Meeting shows the notes field")
	}
	if elementByName(doc, "input", "name") != nil {
		t.Fatal("the Meeting shows rename")
	}
}

func TestDeletingAProjectRemovesItsReferences(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postTask(t, app, ledger, "Ship it")
	postProjectNotes(t, app, ledger, "private notes")
	other := postProject(t, app, "Other")
	postTask(t, app, other, "Stay")

	postFormNoRedirect(t, app, "/meetings", url.Values{
		"title":       {"Planning"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"keep this body"},
		"project":     {projectIDString(ledger), projectIDString(other)},
	}, nil)

	_, confirm := getPage(t, app, ledger+"/delete")
	if !strings.Contains(textContent(confirm), "Ledger") {
		t.Fatal("delete confirmation must name the Project")
	}
	res := postFormNoRedirect(t, app, ledger+"/delete", nil, nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/projects" {
		t.Fatalf("delete status = %d location = %q, want 303 /projects", res.StatusCode, res.Header.Get("Location"))
	}

	_, list := getPage(t, app, "/projects")
	if strings.Contains(textContent(list), "Ledger") || linkHref(list, "Other") != other {
		t.Fatalf("list text = %q, want Ledger gone and Other kept", textContent(list))
	}
	status, missing := getPage(t, app, ledger)
	if status != http.StatusNotFound || !strings.Contains(textContent(missing), "Project not found.") {
		t.Fatalf("deleted Project status=%d text=%q", status, textContent(missing))
	}

	_, feed := getPage(t, app, "/")
	card := meetingCard(feed, "Planning")
	if !strings.Contains(textContent(card), "Planning") || !strings.Contains(textContent(card), "keep this body") || !strings.Contains(textContent(card), "2 Jan 2026, 15:04") {
		t.Fatalf("feed card = %q, want the Meeting title, time, and body", textContent(card))
	}
	if strings.Contains(textContent(card), "Ledger") || strings.Contains(textContent(card), "Ship it") || strings.Contains(textContent(card), "private notes") {
		t.Fatalf("feed card = %q, want the deleted checklist gone", textContent(card))
	}
	if got := projectHrefsIn(card); !sameStrings(got, []string{other}) {
		t.Fatalf("feed links = %#v, want only Other", got)
	}
	assertTasks(t, card, []taskState{{text: "Stay", done: false}})

	_, page := getPage(t, app, linkHref(feed, "Planning"))
	pageCard := meetingCard(page, "Planning")
	if strings.Contains(textContent(pageCard), "Ledger") || strings.Contains(textContent(pageCard), "Ship it") {
		t.Fatalf("Meeting text = %q, want the deleted checklist gone", textContent(pageCard))
	}
	if !strings.Contains(textContent(pageCard), "keep this body") {
		t.Fatal("the Meeting lost its body")
	}
	assertTasks(t, pageCard, []taskState{{text: "Stay", done: false}})
}

func TestRecreatedProjectStartsWithoutReferences(t *testing.T) {
	app := startApp(t)
	first := postProject(t, app, "Ledger")
	postTask(t, app, first, "Old task")
	postProjectNotes(t, app, first, "old notes")
	postFormNoRedirect(t, app, "/meetings", url.Values{
		"title":       {"Planning"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"the old discussion"},
		"project":     {projectIDString(first)},
	}, nil)
	res := postFormNoRedirect(t, app, first+"/delete", nil, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want 303", res.StatusCode)
	}

	second := postProject(t, app, " LEDGER ")
	if second == first {
		t.Fatal("a Project created after delete reused the deleted Project")
	}
	_, page := getPage(t, app, second)
	if got := attr(elementByName(page, "input", "name"), "value"); got != "LEDGER" {
		t.Fatalf("name = %q, want LEDGER", got)
	}
	if got := taskTexts(page); len(got) != 0 {
		t.Fatalf("tasks = %#v, want none", got)
	}
	if notes := strings.TrimSpace(textContent(elementByName(page, "textarea", "notes"))); notes != "" {
		t.Fatalf("notes = %q, want none", notes)
	}
	if hasAttribute(page, "data-citation") || strings.Contains(textContent(page), "Planning") {
		t.Fatal("the new Project kept a Reference to the old Meeting")
	}

	_, feed := getPage(t, app, "/")
	card := meetingCard(feed, "Planning")
	if strings.Contains(textContent(card), "LEDGER") || strings.Contains(textContent(card), "Old task") {
		t.Fatalf("Meeting text = %q, want no checklist for the new Project", textContent(card))
	}
	if got := projectHrefsIn(card); len(got) != 0 {
		t.Fatalf("Meeting links = %#v, want no Reference", got)
	}
	if !strings.Contains(textContent(card), "the old discussion") {
		t.Fatal("the Meeting lost its body")
	}
}

func TestReferencedNamesAndTaskTextAreShownAsText(t *testing.T) {
	app := startApp(t)
	name := "<script>alert(1)</script>"
	taskText := "hello <b>nope</b> & co"
	loc := postProject(t, app, name)
	postTask(t, app, loc, taskText)

	_, composer := getPage(t, app, "/")
	if !strings.Contains(textContent(composer), name) {
		t.Fatalf("picker text = %q, want the Project name as text", textContent(composer))
	}
	picked := postComposer(t, app, url.Values{
		"title":       {"Planning"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"body <i>stays</i>"},
		"add":         {projectIDString(loc)},
	})
	if !strings.Contains(textContent(picked), name) {
		t.Fatalf("composer text = %q, want the Project name as text", textContent(picked))
	}
	if taskRowByText(picked, taskText) == nil {
		t.Fatalf("composer tasks = %#v, want the Task as text", taskTexts(picked))
	}

	postFormNoRedirect(t, app, "/meetings", composerValues(picked, nil), nil)
	_, feed := getPage(t, app, "/")
	if !strings.Contains(textContent(feed), name) {
		t.Fatalf("feed text = %q, want the Project name as text", textContent(feed))
	}
	if taskRowByText(feed, taskText) == nil {
		t.Fatalf("feed tasks = %#v, want the Task as text", taskTexts(feed))
	}
	if countElements(feed, "script") != 1 {
		t.Fatal("the Project name was parsed as HTML")
	}
	_, page := getPage(t, app, linkHref(feed, "Planning"))
	if !strings.Contains(textContent(page), name) {
		t.Fatalf("Meeting text = %q, want the Project name as text", textContent(page))
	}
	if taskRowByText(page, taskText) == nil {
		t.Fatalf("Meeting tasks = %#v, want the Task as text", taskTexts(page))
	}
	if countElements(page, "script") != 1 || countElements(page, "b") != 0 {
		t.Fatal("the Meeting parsed the Project name or Task text as HTML")
	}
}

func TestTaskChangesOnTheMeetingAreTheProjectTasks(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postTask(t, app, ledger, "Older")
	postTask(t, app, ledger, "Newer")
	postTask(t, app, ledger, "Latest")
	other := postProject(t, app, "Other")
	postTask(t, app, other, "Leave me")
	postTask(t, app, other, "Also open")

	postFormNoRedirect(t, app, "/meetings", url.Values{
		"title":       {"Planning"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"two lists"},
		"project":     {projectIDString(ledger), projectIDString(other)},
	}, nil)
	_, feed := getPage(t, app, "/")
	meetingURL := linkHref(feed, "Planning")

	_, meeting := getPage(t, app, meetingURL)
	card := meetingCard(meeting, "Planning")
	latestID := attr(taskRowByText(card, "Latest"), "data-task")
	_, ledgerPage := getPage(t, app, ledger)
	if got := attr(taskRowByText(ledgerPage, "Latest"), "data-task"); got != latestID {
		t.Fatalf("data-task = %q, want the same Task %s", got, latestID)
	}

	add := addTaskFormFor(card, ledger)
	res := postFormNoRedirect(t, app, attr(add, "action"), url.Values{"text": {"From the meeting"}}, nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != ledger {
		t.Fatalf("add status = %d location = %q, want 303 %s", res.StatusCode, res.Header.Get("Location"), ledger)
	}
	_, meeting = getPage(t, app, meetingURL)
	_, ledgerPage = getPage(t, app, ledger)
	assertTasks(t, projectSection(meetingCard(meeting, "Planning"), ledger), []taskState{
		{text: "Older", done: false},
		{text: "Newer", done: false},
		{text: "Latest", done: false},
		{text: "From the meeting", done: false},
	})
	assertTasks(t, ledgerPage, []taskState{
		{text: "Older", done: false},
		{text: "Newer", done: false},
		{text: "Latest", done: false},
		{text: "From the meeting", done: false},
	})

	card = meetingCard(meeting, "Planning")
	edit := taskForm(taskRowByText(card, "Latest"), "text")
	res = postFormNoRedirect(t, app, attr(edit, "action"), url.Values{"text": {"Latest revised"}}, nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != ledger {
		t.Fatalf("edit status = %d location = %q, want 303 %s", res.StatusCode, res.Header.Get("Location"), ledger)
	}
	_, ledgerPage = getPage(t, app, ledger)
	if got := attr(taskRowByText(ledgerPage, "Latest revised"), "data-task"); got != latestID {
		t.Fatalf("data-task = %q, want the edited Task %s", got, latestID)
	}

	_, meeting = getPage(t, app, meetingURL)
	card = meetingCard(meeting, "Planning")
	newer := taskRowByText(card, "Newer")
	doneForm := taskForm(newer, "checkbox")
	target := strings.TrimPrefix(attr(doneForm, "hx-target"), "#")
	if target == "" || target == "project-tasks" {
		t.Fatalf("toggle target = %q, want this Project's list on the Meeting", target)
	}
	toggled := readPost(t, app, attr(doneForm, "action"), url.Values{"done": {"1"}}, http.Header{
		"HX-Request": {"true"},
		"HX-Target":  {target},
	})
	if toggled.Status != http.StatusOK || strings.Contains(toggled.Body, "<html") {
		t.Fatalf("toggle status=%d body=%q, want this Project's task list", toggled.Status, toggled.Body)
	}
	fragment := parseHTML(t, toggled.Body)
	if elementByID(fragment, target) == nil {
		t.Fatalf("toggled list id = %q, want %s", attr(fragment.FirstChild, "id"), target)
	}
	if strings.Contains(textContent(fragment), "Leave me") {
		t.Fatal("toggling Ledger changed Other's Tasks")
	}
	assertTasks(t, fragment, []taskState{
		{text: "Newer", done: true},
		{text: "Older", done: false},
		{text: "Latest revised", done: false},
		{text: "From the meeting", done: false},
	})

	_, meeting = getPage(t, app, meetingURL)
	card = meetingCard(meeting, "Planning")
	assertTasks(t, projectSection(card, other), []taskState{
		{text: "Leave me", done: false},
		{text: "Also open", done: false},
	})
	_, ledgerPage = getPage(t, app, ledger)
	assertTasks(t, ledgerPage, []taskState{
		{text: "Newer", done: true},
		{text: "Older", done: false},
		{text: "Latest revised", done: false},
		{text: "From the meeting", done: false},
	})

	reopen := taskForm(taskRowByText(card, "Newer"), "checkbox")
	res = postFormNoRedirect(t, app, attr(reopen, "action"), url.Values{"done": {"0"}}, nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != ledger {
		t.Fatalf("reopen status = %d location = %q, want 303 %s", res.StatusCode, res.Header.Get("Location"), ledger)
	}
	_, meeting = getPage(t, app, meetingURL)
	_, ledgerPage = getPage(t, app, ledger)
	reopened := []taskState{
		{text: "Older", done: false},
		{text: "Newer", done: false},
		{text: "Latest revised", done: false},
		{text: "From the meeting", done: false},
	}
	assertTasks(t, projectSection(meetingCard(meeting, "Planning"), ledger), reopened)
	assertTasks(t, ledgerPage, reopened)

	_, meeting = getPage(t, app, meetingURL)
	card = meetingCard(meeting, "Planning")
	drop := taskRowByText(card, "Also open")
	deleteForm := taskForm(drop, "delete")
	if deleteForm == nil || strings.TrimSpace(textContent(elementByName(deleteForm, "button", ""))) != "Delete" {
		t.Fatal("deleting a Task on the Meeting has no confirmation step")
	}
	for _, href := range linkHrefs(drop) {
		if strings.Contains(href, "delete") {
			t.Fatalf("Task delete link = %q, want no confirmation page", href)
		}
	}
	res = postFormNoRedirect(t, app, attr(deleteForm, "action"), nil, nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != other {
		t.Fatalf("delete status = %d location = %q, want 303 %s", res.StatusCode, res.Header.Get("Location"), other)
	}
	_, meeting = getPage(t, app, meetingURL)
	_, otherPage := getPage(t, app, other)
	remaining := []taskState{{text: "Leave me", done: false}}
	assertTasks(t, projectSection(meetingCard(meeting, "Planning"), other), remaining)
	assertTasks(t, otherPage, remaining)
	assertTasks(t, projectSection(meetingCard(meeting, "Planning"), ledger), reopened)
}

func projectSection(card *html.Node, projectPath string) *html.Node {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil || n == nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "section" {
			for _, href := range linkHrefs(n) {
				if href == projectPath {
					found = n
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(card)

	return found
}

func TestMeetingShowsEachProjectsTasksAndAnAddField(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postTask(t, app, ledger, "Older")
	postTask(t, app, ledger, "Newer")
	postTask(t, app, ledger, "Latest")
	postTaskDone(t, app, ledger, "Newer", true)
	postTaskDone(t, app, ledger, "Older", true)
	empty := postProject(t, app, "Empty")

	postFormNoRedirect(t, app, "/meetings", url.Values{
		"title":       {"Planning"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"live lists"},
		"project":     {projectIDString(ledger), projectIDString(empty)},
	}, nil)

	_, feed := getPage(t, app, "/")
	assertMeetingTasks(t, meetingCard(feed, "Planning"), ledger, empty)

	_, page := getPage(t, app, linkHref(feed, "Planning"))
	assertMeetingTasks(t, meetingCard(page, "Planning"), ledger, empty)
}

func assertMeetingTasks(t *testing.T, card *html.Node, ledger, empty string) {
	t.Helper()

	if card == nil {
		t.Fatal("missing Meeting card")
	}
	if got := projectHrefsIn(card); !sameStrings(got, []string{ledger, empty}) {
		t.Fatalf("links = %#v, want Ledger then Empty", got)
	}
	assertTasks(t, card, []taskState{
		{text: "Older", done: true},
		{text: "Newer", done: true},
		{text: "Latest", done: false},
	})
	ledgerForm := addTaskFormFor(card, ledger)
	emptyForm := addTaskFormFor(card, empty)
	if ledgerForm == nil || emptyForm == nil {
		t.Fatal("each referenced Project shows an add-task field, including one with no Tasks")
	}
	if attr(elementByName(ledgerForm, "input", "text"), "type") != "text" || attr(elementByName(emptyForm, "input", "text"), "type") != "text" {
		t.Fatal("adding a Task takes a single line of text")
	}
	if _, ok := projectID(empty); ok && taskRowByText(card, "Empty") != nil {
		t.Fatal("the empty Project's name is not a Task")
	}
}

func TestComposerPostSavesReferencesInAddOrder(t *testing.T) {
	app := startApp(t)
	alpha := postProject(t, app, "Alpha")
	postTask(t, app, alpha, "Alpha task")
	beta := postProject(t, app, "Beta")

	_, doc := getPage(t, app, "/")
	doc = postComposer(t, app, url.Values{
		"title":       {"Planning"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"beta then alpha"},
		"add":         {attr(pickerButton(doc, "Beta"), "value")},
	})
	doc = postComposer(t, app, composerValues(doc, url.Values{
		"add": {attr(pickerButton(doc, "Alpha"), "value")},
	}))

	res := postFormNoRedirect(t, app, "/meetings", composerValues(doc, nil), nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/" {
		t.Fatalf("status = %d location = %q, want 303 /", res.StatusCode, res.Header.Get("Location"))
	}

	_, feed := getPage(t, app, "/")
	card := meetingCard(feed, "Planning")
	if got := projectHrefsIn(card); !sameStrings(got, []string{beta, alpha}) {
		t.Fatalf("feed links = %#v, want Beta then Alpha", got)
	}
	assertTasks(t, card, []taskState{{text: "Alpha task", done: false}})
	if got := attr(elementByName(feed, "input", "title"), "value"); got != "" {
		t.Fatalf("title = %q, want the composer cleared", got)
	}
	if got := pendingProjectHrefs(feed); len(got) != 0 {
		t.Fatalf("pending = %#v, want the composer cleared", got)
	}
}

func TestReferencesRemainAfterRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flow.db")
	app := startApp(t, func(cfg *flow.Config) {
		cfg.DBPath = path
	})
	ledger := postProject(t, app, "Ledger")
	postTask(t, app, ledger, "Ship it")
	postFormNoRedirect(t, app, "/meetings", url.Values{
		"title":       {"Planning"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"still here"},
		"project":     {projectIDString(ledger)},
	}, nil)
	if err := app.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	restarted := startApp(t, func(cfg *flow.Config) {
		cfg.DBPath = path
	})
	_, feed := getPage(t, restarted, "/")
	card := meetingCard(feed, "Planning")
	if !strings.Contains(textContent(card), "still here") || !strings.Contains(textContent(card), "2 Jan 2026, 15:04") {
		t.Fatalf("feed card = %q, want the Meeting after restart", textContent(card))
	}
	if got := projectHrefsIn(card); !sameStrings(got, []string{ledger}) {
		t.Fatalf("feed links = %#v, want Ledger", got)
	}
	assertTasks(t, card, []taskState{{text: "Ship it", done: false}})
}

func TestPostedMeetingReferencesEachProjectOnce(t *testing.T) {
	app := startApp(t)
	alpha := postProject(t, app, "Alpha")
	beta := postProject(t, app, "Beta")
	gamma := postProject(t, app, "Gamma")

	res := postFormNoRedirect(t, app, "/meetings", url.Values{
		"title":       {"Planning"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"covered three"},
		"project":     {projectIDString(gamma), projectIDString(alpha), projectIDString(alpha), projectIDString(beta)},
	}, nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/" {
		t.Fatalf("status = %d location = %q, want 303 /", res.StatusCode, res.Header.Get("Location"))
	}

	_, feed := getPage(t, app, "/")
	card := meetingCard(feed, "Planning")
	if got := projectNamesIn(card); !sameStrings(got, []string{"Gamma", "Alpha", "Beta"}) {
		t.Fatalf("feed references = %#v, want Gamma, Alpha, Beta once each", got)
	}
	if got := projectHrefsIn(card); !sameStrings(got, []string{gamma, alpha, beta}) {
		t.Fatalf("feed links = %#v, want each Project page once", got)
	}

	_, page := getPage(t, app, linkHref(feed, "Planning"))
	pageCard := meetingCard(page, "Planning")
	if got := projectNamesIn(pageCard); !sameStrings(got, []string{"Gamma", "Alpha", "Beta"}) {
		t.Fatalf("Meeting references = %#v, want Gamma, Alpha, Beta once each", got)
	}
	if got := projectHrefsIn(pageCard); !sameStrings(got, []string{gamma, alpha, beta}) {
		t.Fatalf("Meeting links = %#v, want each Project page once", got)
	}
	if !strings.Contains(textContent(pageCard), "covered three") || !strings.Contains(textContent(pageCard), "2 Jan 2026, 15:04") {
		t.Fatalf("Meeting text = %q, want the title time and body kept", textContent(pageCard))
	}
}

func projectNamesIn(n *html.Node) []string {
	var names []string
	for _, link := range projectAnchors(n) {
		names = append(names, strings.TrimSpace(textContent(link)))
	}

	return names
}

func projectHrefsIn(n *html.Node) []string {
	var hrefs []string
	for _, link := range projectAnchors(n) {
		hrefs = append(hrefs, attr(link, "href"))
	}

	return hrefs
}

func TestRejectedPostRedisplaysPendingReferences(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postTask(t, app, ledger, "Draft")
	_, page := getPage(t, app, ledger)
	row := taskRowByText(page, "Draft")
	res := postFormNoRedirect(t, app, attr(taskForm(row, "text"), "action"), url.Values{"text": {"Revised"}}, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("edit status = %d, want 303", res.StatusCode)
	}

	_, doc := getPage(t, app, "/")
	doc = postComposer(t, app, url.Values{
		"title":       {"Planning"},
		"happened_at": {"2026-04-04T12:30"},
		"body":        {"should not be saved"},
		"add":         {attr(pickerButton(doc, "Ledger"), "value")},
	})
	doc = postComposer(t, app, composerValues(doc, url.Values{
		"name":   {" Notes "},
		"create": {"1"},
	}))
	if got := pendingProjectNames(doc); !sameStrings(got, []string{"Ledger", "Notes"}) {
		t.Fatalf("pending = %#v, want Ledger then Notes", got)
	}
	notes := pendingProjectHrefs(doc)[1]

	res = postFormNoRedirect(t, app, "/meetings", composerValues(doc, url.Values{
		"title": {""},
	}), nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
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
		t.Fatal("rejected Post dropped the body")
	}
	if got := pendingProjectNames(rejected); !sameStrings(got, []string{"Ledger", "Notes"}) {
		t.Fatalf("pending = %#v, want Ledger then Notes", got)
	}
	assertTasks(t, rejected, []taskState{{text: "Revised", done: false}})
	notesForm := addTaskFormFor(rejected, notes)
	if notesForm == nil {
		t.Fatal("rejected Post dropped the new Project's add-task field")
	}

	_, feed := getPage(t, app, "/")
	if strings.Contains(textContent(feed), "should not be saved") || strings.Contains(textContent(feed), "Planning") {
		t.Fatal("rejected Post created a Meeting")
	}
	assertNoMeetings(t, feed)
	_, list := getPage(t, app, "/projects")
	if linkHref(list, "Ledger") != ledger || linkHref(list, "Notes") != notes {
		t.Fatal("rejected Post lost a Project created from the picker")
	}
	_, page = getPage(t, app, ledger)
	assertTasks(t, page, []taskState{{text: "Revised", done: false}})
	_, notesPage := getPage(t, app, notes)
	if got := taskTexts(notesPage); len(got) != 0 {
		t.Fatalf("Notes tasks = %#v, want none", got)
	}
}

func addTaskFormFor(doc *html.Node, projectPath string) *html.Node {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "form" && attr(n, "action") == projectPath+"/tasks" {
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

func TestPendingReferencesStayInAddOrderAndCanBeRemoved(t *testing.T) {
	app := startApp(t)
	alpha := postProject(t, app, "Alpha")
	postTask(t, app, alpha, "Alpha task")
	beta := postProject(t, app, "Beta")
	gamma := postProject(t, app, "Gamma")

	_, doc := getPage(t, app, "/")
	doc = postComposer(t, app, url.Values{
		"title":       {"Planning"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"three projects"},
		"add":         {attr(pickerButton(doc, "Gamma"), "value")},
	})
	doc = postComposer(t, app, composerValues(doc, url.Values{
		"add": {attr(pickerButton(doc, "Alpha"), "value")},
	}))
	doc = postComposer(t, app, composerValues(doc, url.Values{
		"add": {attr(pickerButton(doc, "Beta"), "value")},
	}))
	if got := pendingProjectNames(doc); !sameStrings(got, []string{"Gamma", "Alpha", "Beta"}) {
		t.Fatalf("pending = %#v, want Gamma, Alpha, Beta in add order", got)
	}
	if got := pendingProjectHrefs(doc); !sameStrings(got, []string{gamma, alpha, beta}) {
		t.Fatalf("pending hrefs = %#v, want Gamma, Alpha, Beta", got)
	}
	if got := pickerNames(doc); len(got) != 0 {
		t.Fatalf("picker = %#v, want every Project already chosen", got)
	}

	remove := removeButton(doc, projectIDString(alpha))
	if remove == nil {
		t.Fatal("a pending Reference can be removed before Post")
	}
	doc = postComposer(t, app, composerValues(doc, url.Values{
		"remove": {attr(remove, "value")},
	}))
	if got := pendingProjectNames(doc); !sameStrings(got, []string{"Gamma", "Beta"}) {
		t.Fatalf("pending = %#v, want Gamma, Beta after removing Alpha", got)
	}
	if got := pickerNames(doc); !sameStrings(got, []string{"Alpha"}) {
		t.Fatalf("picker = %#v, want Alpha available again", got)
	}
	_, alphaPage := getPage(t, app, alpha)
	assertTasks(t, alphaPage, []taskState{{text: "Alpha task", done: false}})
	_, list := getPage(t, app, "/projects")
	if linkHref(list, "Alpha") != alpha {
		t.Fatal("removing a pending Reference removed the Project")
	}

	doc = postComposer(t, app, composerValues(doc, url.Values{
		"add": {attr(pickerButton(doc, "Alpha"), "value")},
	}))
	if got := pendingProjectNames(doc); !sameStrings(got, []string{"Gamma", "Beta", "Alpha"}) {
		t.Fatalf("pending = %#v, want Alpha at the end", got)
	}

	for pending := doc; len(pendingProjectHrefs(pending)) > 0; {
		button := removeButton(pending, projectIDString(pendingProjectHrefs(pending)[0]))
		pending = postComposer(t, app, composerValues(pending, url.Values{
			"remove": {attr(button, "value")},
		}))
		doc = pending
	}
	if got := pendingProjectHrefs(doc); len(got) != 0 {
		t.Fatalf("pending = %#v, want none", got)
	}
	postMeeting(t, app, "Solo", "2026-01-02T15:04", "no project")
	_, feed := getPage(t, app, "/")
	if !strings.Contains(textContent(feed), "Solo") {
		t.Fatal("a Meeting can be posted with no Projects")
	}
	assertNoProjectChecklist(t, meetingCard(feed, "Solo"))
}

func composerValues(doc *html.Node, extra url.Values) url.Values {
	form := url.Values{
		"title":       {attr(elementByName(doc, "input", "title"), "value")},
		"happened_at": {attr(elementByName(doc, "input", "happened_at"), "value")},
		"body":        {textContent(elementByName(doc, "textarea", "body"))},
	}
	for _, id := range hiddenProjectIDs(doc) {
		form.Add("project", id)
	}
	for key, values := range extra {
		form.Del(key)
		for _, value := range values {
			form.Add(key, value)
		}
	}

	return form
}

func hiddenProjectIDs(doc *html.Node) []string {
	var ids []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "input" && attr(n, "name") == "project" {
			ids = append(ids, attr(n, "value"))
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return ids
}

func removeButton(doc *html.Node, projectID string) *html.Node {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "button" && attr(n, "name") == "remove" && attr(n, "value") == projectID {
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

func projectIDString(path string) string {
	id, ok := projectID(path)
	if !ok {
		return ""
	}

	return strconv.FormatInt(id, 10)
}

func meetingCard(doc *html.Node, title string) *html.Node {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "article" && strings.Contains(textContent(n), title) {
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

func assertNoProjectChecklist(t *testing.T, card *html.Node) {
	t.Helper()

	if card == nil {
		t.Fatal("missing Meeting card")
	}
	if got := taskTexts(card); len(got) != 0 {
		t.Fatalf("tasks = %#v, want no checklist", got)
	}
	for _, href := range linkHrefs(card) {
		if _, ok := projectID(href); ok {
			t.Fatalf("Meeting linked a Project at %s", href)
		}
	}
}

func pickerNames(doc *html.Node) []string {
	var names []string
	for _, button := range pickerButtons(doc) {
		names = append(names, strings.TrimSpace(textContent(button)))
	}

	return names
}

func pickerButton(doc *html.Node, name string) *html.Node {
	for _, button := range pickerButtons(doc) {
		if strings.TrimSpace(textContent(button)) == name {
			return button
		}
	}

	return nil
}

func pickerButtons(doc *html.Node) []*html.Node {
	var buttons []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "button" && attr(n, "name") == "add" {
			buttons = append(buttons, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return buttons
}

func pendingProjectHrefs(doc *html.Node) []string {
	var hrefs []string
	for _, link := range pendingProjectLinks(doc) {
		hrefs = append(hrefs, attr(link, "href"))
	}

	return hrefs
}

func pendingProjectNames(doc *html.Node) []string {
	var names []string
	for _, link := range pendingProjectLinks(doc) {
		names = append(names, strings.TrimSpace(textContent(link)))
	}

	return names
}

func pendingProjectLinks(doc *html.Node) []*html.Node {
	var links []*html.Node
	for _, link := range projectAnchors(doc) {
		if inside(link, "article") {
			continue
		}
		links = append(links, link)
	}

	return links
}

func projectAnchors(doc *html.Node) []*html.Node {
	var links []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			if _, ok := projectID(attr(n, "href")); ok {
				links = append(links, n)
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return links
}

func inside(n *html.Node, tag string) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Data == tag {
			return true
		}
	}

	return false
}

func postComposer(t *testing.T, app *flow.App, form url.Values) *html.Node {
	t.Helper()

	res := postFormNoRedirect(t, app, "/composer/projects", form, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST /composer/projects status = %d, want 200", res.StatusCode)
	}

	return parseResponse(t, res)
}
