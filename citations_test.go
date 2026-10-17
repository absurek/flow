package flow_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/absurek/flow"
	"golang.org/x/net/html"
)

func TestReferencedProjectWithNoTasksShowsTheCitationRule(t *testing.T) {
	app := startApp(t, func(cfg *flow.Config) {
		cfg.Location = time.FixedZone("UTC+2", 2*60*60)
	})
	ledger := postProject(t, app, "Ledger")
	other := postProject(t, app, "Other")

	res := postFormNoRedirect(t, app, "/meetings", url.Values{
		"title":       {"Planning"},
		"happened_at": {"2026-01-02T15:04"},
		"body":        {"talked about Ledger"},
		"project":     {projectIDString(ledger), projectIDString(ledger)},
	}, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST meeting status = %d, want 303", res.StatusCode)
	}

	_, feed := getPage(t, app, "/")
	meetingURL := linkHref(feed, "Planning")
	if meetingURL == "" {
		t.Fatal("the feed did not link the Meeting")
	}

	_, page := getPage(t, app, ledger)
	entries := citationEntries(page)
	if len(entries) != 1 {
		t.Fatalf("citations = %d, want one citation for the referencing Meeting", len(entries))
	}
	entry := entries[0]
	if got := strings.Join(strings.Fields(textContent(entry)), " "); got != "2 Jan 2026, 15:04 Planning" {
		t.Fatalf("citation text = %q, want the formatted time and the Meeting title", got)
	}
	if got := linkHref(entry, "Planning"); got != meetingURL {
		t.Fatalf("citation link = %q, want %s", got, meetingURL)
	}
	if attr(entry, "data-citation") == "" {
		t.Fatal("citation entry must expose data-citation")
	}
	if strings.Contains(textContent(entry), "talked about Ledger") {
		t.Fatal("the citation includes the Meeting body")
	}
	if len(taskRows(page)) != 0 {
		t.Fatal("a Project with no Tasks should not list one")
	}

	divider := findElement(page, "hr")
	notes := elementByName(page, "textarea", "notes")
	if divider == nil || notes == nil || !elementPrecedes(page, entry, divider) || !elementPrecedes(page, divider, notes) {
		t.Fatal("the citation rule sits on the task list, above the divider")
	}
	if countElements(page, "hr") != 1 {
		t.Fatalf("divider count = %d, want one separator above notes", countElements(page, "hr"))
	}
	if strings.TrimSpace(textContent(divider)) != "" {
		t.Fatalf("divider text = %q, want only a separator", textContent(divider))
	}
	if strings.Contains(textContent(notes), "Planning") || hasAttribute(notes, "data-citation") {
		t.Fatal("the divider above notes lists the Meeting")
	}

	_, otherPage := getPage(t, app, other)
	if hasAttribute(otherPage, "data-citation") || strings.Contains(textContent(otherPage), "Planning") {
		t.Fatalf("other Project text = %q, want no citation rule", textContent(otherPage))
	}

	status, meeting := getPage(t, app, meetingURL)
	if status != http.StatusOK {
		t.Fatalf("citation status = %d, want 200", status)
	}
	if hasAttribute(meeting, "data-citation") {
		t.Fatal("the Meeting shows the citation rule")
	}
	if !strings.Contains(textContent(meeting), "Ledger") {
		t.Fatalf("Meeting text = %q, want the referenced Project", textContent(meeting))
	}
	if got := taskTexts(meeting); len(got) != 0 {
		t.Fatalf("tasks = %#v, want that Project's Tasks, which are none yet", got)
	}
}

func TestCitationRuleSitsBetweenDoneAndOpenTasks(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postTask(t, app, ledger, "Open one")
	postTask(t, app, ledger, "Open two")
	postMeetingProjects(t, app, "Planning", "2026-01-02T15:04", "the discussion", ledger)

	_, page := getPage(t, app, ledger)
	if got := taskCitationOrder(page); !sameStrings(got, []string{
		"citation:Planning",
		"task:Open one",
		"task:Open two",
	}) {
		t.Fatalf("order = %#v, want the rule above the open Tasks", got)
	}
	assertTasks(t, page, []taskState{
		{text: "Open one", done: false},
		{text: "Open two", done: false},
	})

	postTaskDone(t, app, ledger, "Open one", true)
	postTaskDone(t, app, ledger, "Open two", true)
	_, page = getPage(t, app, ledger)
	if got := taskCitationOrder(page); !sameStrings(got, []string{
		"task:Open one",
		"task:Open two",
		"citation:Planning",
	}) {
		t.Fatalf("order = %#v, want the rule below the done Tasks", got)
	}
	assertTasks(t, page, []taskState{
		{text: "Open one", done: true},
		{text: "Open two", done: true},
	})

	postTaskDone(t, app, ledger, "Open two", false)
	_, page = getPage(t, app, ledger)
	if got := taskCitationOrder(page); !sameStrings(got, []string{
		"task:Open one",
		"citation:Planning",
		"task:Open two",
	}) {
		t.Fatalf("order = %#v, want the rule between the done and open Tasks", got)
	}
	assertTasks(t, page, []taskState{
		{text: "Open one", done: true},
		{text: "Open two", done: false},
	})
	if attr(taskRowByText(page, "Open one"), "data-task") == "" || attr(taskRowByText(page, "Open two"), "data-task") == "" {
		t.Fatal("task rows must expose data-task")
	}

	add := addTaskForm(page)
	citation := citationEntries(page)
	if len(citation) != 1 || add == nil || !elementPrecedes(page, citation[0], add) || !elementPrecedes(page, taskRowByText(page, "Open two"), add) {
		t.Fatal("the open Task and the citation rule sit above the new-Task field")
	}
}

func TestCitationsListEveryReferencingMeetingNewestFirst(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postTask(t, app, ledger, "Ship it")
	postTask(t, app, ledger, "Follow up")
	other := postProject(t, app, "Other")
	postTask(t, app, other, "Elsewhere")

	postMeetingProjects(t, app, "Earlier talk", "2026-01-02T15:04", "first discussion", ledger, ledger)
	_, page := getPage(t, app, ledger)
	if got := citationSummaries(page); !sameStrings(got, []string{"2 Jan 2026, 15:04 Earlier talk"}) {
		t.Fatalf("citations = %#v, want the one referencing Meeting", got)
	}

	postMeetingProjects(t, app, "Same time", "2026-03-03T09:00", "tied earlier", ledger)
	_, page = getPage(t, app, ledger)
	if got := citationSummaries(page); !sameStrings(got, []string{
		"3 Mar 2026, 09:00 Same time",
		"2 Jan 2026, 15:04 Earlier talk",
	}) {
		t.Fatalf("citations = %#v, want the second Meeting added, newest happened-at first", got)
	}

	postMeetingProjects(t, app, "Also same", "2026-03-03T09:00", "tied later", ledger)
	postMeetingProjects(t, app, "Newest", "2026-04-04T18:30", "latest discussion", ledger)
	postMeetingProjects(t, app, "Other talk", "2026-05-05T10:00", "not ledger", other)
	postMeetingProjects(t, app, "Both", "2026-02-02T12:00", "both projects", ledger, other)

	_, page = getPage(t, app, ledger)
	if got := citationSummaries(page); !sameStrings(got, []string{
		"4 Apr 2026, 18:30 Newest",
		"3 Mar 2026, 09:00 Also same",
		"3 Mar 2026, 09:00 Same time",
		"2 Feb 2026, 12:00 Both",
		"2 Jan 2026, 15:04 Earlier talk",
	}) {
		t.Fatalf("citations = %#v, want happened-at order, with the later-created Meeting first on a tie", got)
	}
	seen := map[string]bool{}
	for _, entry := range citationEntries(page) {
		id := attr(entry, "data-citation")
		title := citationTitle(entry)
		if id == "" || seen[id] {
			t.Fatalf("data-citation = %q, want each referencing Meeting once", id)
		}
		seen[id] = true
		if got := linkHref(entry, title); got != "/meetings/"+id {
			t.Fatalf("citation %q link = %q, want /meetings/%s", title, got, id)
		}
	}
	if strings.Contains(textContent(page), "Other talk") {
		t.Fatal("a Meeting that does not reference the Project is cited")
	}

	divider := findElement(page, "hr")
	notes := elementByName(page, "textarea", "notes")
	if divider == nil || notes == nil || countElements(page, "hr") != 1 || strings.TrimSpace(textContent(divider)) != "" {
		t.Fatal("the divider above notes must stay a single empty separator")
	}
	for _, entry := range citationEntries(page) {
		if !elementPrecedes(page, entry, divider) || !elementPrecedes(page, divider, notes) {
			t.Fatal("citations belong above the divider, not in the notes separator")
		}
	}
	if strings.Contains(textContent(notes), "Newest") || strings.Contains(textContent(notes), "Earlier talk") {
		t.Fatal("the notes field lists Meetings")
	}

	_, meeting := getPage(t, app, linkHref(citationEntries(page)[0], "Newest"))
	if hasAttribute(meeting, "data-citation") {
		t.Fatal("opening a citation shows the citation rule")
	}
	if got := taskTexts(projectSection(meeting, ledger)); !sameStrings(got, []string{"Ship it", "Follow up"}) {
		t.Fatalf("tasks = %#v, want that Project's Tasks", got)
	}
	if strings.Contains(textContent(meeting), "Elsewhere") {
		t.Fatal("opening the citation shows another Project's Tasks")
	}

	_, both := getPage(t, app, linkHref(page, "Both"))
	if got := taskTexts(projectSection(both, ledger)); !sameStrings(got, []string{"Ship it", "Follow up"}) {
		t.Fatalf("tasks = %#v, want Ledger's Tasks on the shared Meeting", got)
	}
}

func TestFinishingATaskMovesItAboveTheCitationRule(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postTask(t, app, ledger, "Already done")
	postTaskDone(t, app, ledger, "Already done", true)
	postTask(t, app, ledger, "Still open")
	other := postProject(t, app, "Other")
	postTask(t, app, other, "Leave me")
	postMeetingProjects(t, app, "Planning", "2026-01-02T15:04", "before the finish", ledger, other)

	_, feed := getPage(t, app, "/")
	meetingURL := linkHref(feed, "Planning")
	_, page := getPage(t, app, ledger)
	if got := taskCitationOrder(page); !sameStrings(got, []string{
		"task:Already done",
		"citation:Planning",
		"task:Still open",
	}) {
		t.Fatalf("order = %#v, want open work below the rule", got)
	}

	postTask(t, app, ledger, "Fresh")
	_, page = getPage(t, app, ledger)
	if got := taskCitationOrder(page); !sameStrings(got, []string{
		"task:Already done",
		"citation:Planning",
		"task:Still open",
		"task:Fresh",
	}) {
		t.Fatalf("order = %#v, want the new Task in the open group below the rule", got)
	}
	if taskChecked(taskRowByText(page, "Fresh")) {
		t.Fatal("the first Task added after a citation is done")
	}
	if attr(taskRowByText(page, "Fresh"), "data-task") == "" {
		t.Fatal("task row must expose data-task")
	}

	row := taskRowByText(page, "Still open")
	toggled := readPost(t, app, "/tasks/"+attr(row, "data-task"), url.Values{"done": {"1"}}, http.Header{
		"HX-Request": {"true"},
		"HX-Target":  {"project-tasks"},
	})
	if toggled.Status != http.StatusOK || strings.Contains(toggled.Body, "<html") {
		t.Fatalf("toggle status=%d body=%q, want the Project task list", toggled.Status, toggled.Body)
	}
	fragment := parseHTML(t, toggled.Body)
	if got := taskCitationOrder(elementByID(fragment, "project-tasks")); !sameStrings(got, []string{
		"task:Already done",
		"task:Still open",
		"citation:Planning",
		"task:Fresh",
	}) {
		t.Fatalf("order = %#v, want the finished Task above the rule", got)
	}

	_, page = getPage(t, app, ledger)
	if got := taskCitationOrder(page); !sameStrings(got, []string{
		"task:Already done",
		"task:Still open",
		"citation:Planning",
		"task:Fresh",
	}) {
		t.Fatalf("order = %#v, want the finished Task above the rule", got)
	}
	assertTasks(t, page, []taskState{
		{text: "Already done", done: true},
		{text: "Still open", done: true},
		{text: "Fresh", done: false},
	})

	_, feed = getPage(t, app, "/")
	_, meeting := getPage(t, app, meetingURL)
	for _, doc := range []*html.Node{meetingCard(feed, "Planning"), meeting} {
		if hasAttribute(doc, "data-citation") {
			t.Fatal("an embedded Task list shows the citation rule")
		}
		section := projectSection(doc, ledger)
		assertTasks(t, section, []taskState{
			{text: "Already done", done: true},
			{text: "Still open", done: true},
			{text: "Fresh", done: false},
		})
		if !elementPrecedes(section, taskRowByText(section, "Still open"), taskRowByText(section, "Fresh")) {
			t.Fatal("the finished Task stays below an open Task on the embedded list")
		}
		assertTasks(t, projectSection(doc, other), []taskState{{text: "Leave me", done: false}})
	}

	card := meetingCard(meeting, "Planning")
	fresh := taskRowByText(projectSection(card, ledger), "Fresh")
	target := strings.TrimPrefix(attr(taskForm(fresh, "checkbox"), "hx-target"), "#")
	embedded := readPost(t, app, "/tasks/"+attr(fresh, "data-task"), url.Values{"done": {"1"}}, http.Header{
		"HX-Request": {"true"},
		"HX-Target":  {target},
	})
	if embedded.Status != http.StatusOK {
		t.Fatalf("embedded toggle status = %d, want 200", embedded.Status)
	}
	embeddedDoc := parseHTML(t, embedded.Body)
	embeddedList := elementByID(embeddedDoc, target)
	if hasAttribute(embeddedList, "data-citation") {
		t.Fatal("finishing a Task on the Meeting renders the citation rule")
	}
	assertTasks(t, embeddedList, []taskState{
		{text: "Already done", done: true},
		{text: "Still open", done: true},
		{text: "Fresh", done: true},
	})

	_, page = getPage(t, app, ledger)
	if got := taskCitationOrder(page); !sameStrings(got, []string{
		"task:Already done",
		"task:Still open",
		"task:Fresh",
		"citation:Planning",
	}) {
		t.Fatalf("order = %#v, want every finished Task above the rule", got)
	}
}

func citationSummaries(doc *html.Node) []string {
	entries := citationEntries(doc)
	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		got = append(got, strings.Join(strings.Fields(textContent(entry)), " "))
	}

	return got
}

func postMeetingProjects(t *testing.T, app *flow.App, title, happenedAt, body string, projects ...string) {
	t.Helper()

	form := url.Values{
		"title":       {title},
		"happened_at": {happenedAt},
		"body":        {body},
	}
	for _, project := range projects {
		form.Add("project", projectIDString(project))
	}
	res := postFormNoRedirect(t, app, "/meetings", form, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST meeting %q status = %d, want 303", title, res.StatusCode)
	}
}

func taskCitationOrder(doc *html.Node) []string {
	var order []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && nodeHasAttr(n, "data-task") {
			order = append(order, "task:"+taskText(n))
			return
		}
		if n.Type == html.ElementNode && nodeHasAttr(n, "data-citation") {
			order = append(order, "citation:"+citationTitle(n))
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return order
}

func citationTitle(entry *html.Node) string {
	var title string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if title != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			title = strings.TrimSpace(textContent(n))
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(entry)

	return title
}

func citationEntries(n *html.Node) []*html.Node {
	var entries []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && nodeHasAttr(n, "data-citation") {
			entries = append(entries, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	return entries
}
