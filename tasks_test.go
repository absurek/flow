package flow_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/absurek/flow"
	"golang.org/x/net/html"
)

func TestAddingATaskAppendsItToTheOpenGroup(t *testing.T) {
	app := startApp(t)
	postMeeting(t, app, "Planning", "2026-01-02T15:04", "Talked through the timeline.")
	loc := postProject(t, app, "Ledger")
	other := postProject(t, app, "Other")

	status, doc := getPage(t, app, loc)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if got := taskTexts(doc); len(got) != 0 {
		t.Fatalf("tasks = %#v, want none on a new Project", got)
	}
	form := addTaskForm(doc)
	if form == nil || strings.ToLower(attr(form, "method")) != "post" || attr(form, "action") != loc+"/tasks" {
		method, action := "", ""
		if form != nil {
			method, action = attr(form, "method"), attr(form, "action")
		}
		t.Fatalf("add form method=%q action=%q, want POST %s/tasks", method, action, loc)
	}
	addField := elementByName(form, "input", "text")
	if addField == nil || attr(addField, "type") != "text" {
		t.Fatal("adding a Task must take a single line of text")
	}
	if hasAttribute(doc, "data-citation") || strings.Contains(textContent(doc), "Planning") || strings.Contains(strings.ToLower(textContent(doc)), "citation") {
		t.Fatalf("project page text = %q, want no citation rule", textContent(doc))
	}
	if hasAttribute(doc, "draggable") || strings.Contains(strings.ToLower(textContent(doc)), "drag") {
		t.Fatal("there is no control for dragging Tasks")
	}

	postTask(t, app, loc, "Write the notes")
	_, page := getPage(t, app, loc)
	rows := taskRows(page)
	if len(rows) != 1 {
		t.Fatalf("task count = %d, want the first Task", len(rows))
	}
	if attr(rows[0], "data-task") == "" {
		t.Fatal("task row must expose data-task")
	}
	if got := taskText(rows[0]); got != "Write the notes" {
		t.Fatalf("task text = %q, want Write the notes", got)
	}
	if taskChecked(rows[0]) {
		t.Fatal("a new Task is open")
	}
	if checkbox := taskCheckbox(rows[0]); checkbox == nil {
		t.Fatal("the row must show a checkbox")
	}
	if strings.Contains(textContent(rows[0]), "2026") || strings.Contains(textContent(rows[0]), "Jan") {
		t.Fatalf("row text = %q, want no creation time", textContent(rows[0]))
	}
	add := elementByName(addTaskForm(page), "input", "text")
	divider := findElement(page, "hr")
	if add == nil || divider == nil || !elementPrecedes(page, rows[0], add) || !elementPrecedes(page, add, divider) {
		t.Fatal("the new Task sits at the bottom of the open group, above the divider")
	}
	if got := attr(add, "value"); got != "" {
		t.Fatalf("add field = %q, want it cleared after the Task is added", got)
	}
	if hasAttribute(page, "data-citation") || strings.Contains(textContent(page), "Planning") {
		t.Fatalf("project page text = %q, want no citation rule", textContent(page))
	}
	if hasAttribute(page, "draggable") || strings.Contains(strings.ToLower(textContent(page)), "drag") {
		t.Fatal("there is no control for dragging Tasks")
	}

	postTask(t, app, loc, "Ship it")
	_, page = getPage(t, app, loc)
	if got := taskTexts(page); !sameStrings(got, []string{"Write the notes", "Ship it"}) {
		t.Fatalf("tasks = %#v, want the newer open Task at the bottom", got)
	}
	_, otherPage := getPage(t, app, other)
	if got := taskTexts(otherPage); len(got) != 0 {
		t.Fatalf("other Project tasks = %#v, want none", got)
	}
}

func TestDoneTasksStayAboveOpenTasksByCreationTime(t *testing.T) {
	app := startApp(t)
	loc := postProject(t, app, "Ledger")
	postTask(t, app, loc, "Older")
	postTask(t, app, loc, "Newer")
	postTask(t, app, loc, "Latest")

	postTaskDone(t, app, loc, "Newer", true)
	postTaskDone(t, app, loc, "Older", true)
	_, page := getPage(t, app, loc)
	assertTasks(t, page, []taskState{
		{text: "Older", done: true},
		{text: "Newer", done: true},
		{text: "Latest", done: false},
	})

	postTaskDone(t, app, loc, "Latest", true)
	_, page = getPage(t, app, loc)
	assertTasks(t, page, []taskState{
		{text: "Older", done: true},
		{text: "Newer", done: true},
		{text: "Latest", done: true},
	})

	postTaskDone(t, app, loc, "Newer", false)
	_, page = getPage(t, app, loc)
	assertTasks(t, page, []taskState{
		{text: "Older", done: true},
		{text: "Latest", done: true},
		{text: "Newer", done: false},
	})

	postTask(t, app, loc, "Follow up")
	_, page = getPage(t, app, loc)
	assertTasks(t, page, []taskState{
		{text: "Older", done: true},
		{text: "Latest", done: true},
		{text: "Newer", done: false},
		{text: "Follow up", done: false},
	})

	postTaskDone(t, app, loc, "Older", false)
	_, page = getPage(t, app, loc)
	assertTasks(t, page, []taskState{
		{text: "Latest", done: true},
		{text: "Older", done: false},
		{text: "Newer", done: false},
		{text: "Follow up", done: false},
	})
}

func TestLeavingTheTaskFieldSavesItsText(t *testing.T) {
	app := startApp(t)
	loc := postProject(t, app, "Ledger")
	postTask(t, app, loc, "Older")
	postTask(t, app, loc, "Newer")

	_, page := getPage(t, app, loc)
	row := taskRowByText(page, "Older")
	id := attr(row, "data-task")
	form := taskForm(row, "text")
	if form == nil || strings.ToLower(attr(form, "method")) != "post" || attr(form, "action") != "/tasks/"+id {
		method, action := "", ""
		if form != nil {
			method, action = attr(form, "method"), attr(form, "action")
		}
		t.Fatalf("text form method=%q action=%q, want POST /tasks/%s", method, action, id)
	}
	if field := elementByName(form, "input", "text"); field == nil || attr(field, "type") != "text" {
		t.Fatal("a Task is a single line")
	}

	res := postFormNoRedirect(t, app, "/tasks/"+id, url.Values{"text": {"  Older revised  "}}, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST text status = %d, want 303", res.StatusCode)
	}
	if got := res.Header.Get("Location"); got != loc {
		t.Fatalf("location = %q, want %s", got, loc)
	}

	_, page = getPage(t, app, loc)
	assertTasks(t, page, []taskState{
		{text: "Older revised", done: false},
		{text: "Newer", done: false},
	})
	if got := attr(taskRowByText(page, "Older revised"), "data-task"); got != id {
		t.Fatalf("data-task = %q, want the same Task %s", got, id)
	}
	if strings.Contains(textContent(taskRowByText(page, "Older revised")), "2026") || strings.Contains(textContent(taskRowByText(page, "Older revised")), "Jan") {
		t.Fatal("editing text showed a creation time")
	}

	postTaskDone(t, app, loc, "Older revised", true)
	postTaskDone(t, app, loc, "Older revised", false)
	_, page = getPage(t, app, loc)
	assertTasks(t, page, []taskState{
		{text: "Older revised", done: false},
		{text: "Newer", done: false},
	})

	for _, blank := range []string{"", "   ", " \n "} {
		res = postFormNoRedirect(t, app, "/tasks/"+id, url.Values{"text": {blank}}, nil)
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("text %q status = %d, want 400", blank, res.StatusCode)
		}
		doc := parseResponse(t, res)
		if !strings.Contains(textContent(doc), "Task text is required.") {
			t.Fatalf("text %q page = %q, want an error", blank, textContent(doc))
		}
		assertTasks(t, elementByID(doc, "project-tasks"), []taskState{
			{text: "Older revised", done: false},
			{text: "Newer", done: false},
		})
	}

	_, page = getPage(t, app, loc)
	if strings.Contains(textContent(page), "Task text is required.") {
		t.Fatal("a later visit still shows the rejected text")
	}
	assertTasks(t, page, []taskState{
		{text: "Older revised", done: false},
		{text: "Newer", done: false},
	})

	res = postFormNoRedirect(t, app, "/tasks/"+id, url.Values{"text": {"alpha\r\nbeta"}}, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST line status = %d, want 303", res.StatusCode)
	}
	_, page = getPage(t, app, loc)
	assertTasks(t, page, []taskState{
		{text: "alpha beta", done: false},
		{text: "Newer", done: false},
	})
	if got := attr(taskRowByText(page, "alpha beta"), "data-task"); got != id {
		t.Fatalf("data-task = %q, want the same Task %s", got, id)
	}

	res = postFormNoRedirect(t, app, loc+"/tasks", url.Values{"text": {" \n "}}, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("blank add status = %d, want 400", res.StatusCode)
	}
	doc := parseResponse(t, res)
	if !strings.Contains(textContent(doc), "Task text is required.") {
		t.Fatalf("blank add text = %q, want an error", textContent(doc))
	}
	assertTasks(t, elementByID(doc, "project-tasks"), []taskState{
		{text: "alpha beta", done: false},
		{text: "Newer", done: false},
	})
}

func TestDeletingATaskUpdatesTheListAndCounts(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	other := postProject(t, app, "Other")
	postTask(t, app, ledger, "Done work")
	postTask(t, app, ledger, "Open work")
	postTask(t, app, ledger, "Also open")
	postTaskDone(t, app, ledger, "Done work", true)
	postTask(t, app, other, "Elsewhere")

	_, page := getPage(t, app, ledger)
	row := taskRowByText(page, "Done work")
	id := attr(row, "data-task")
	form := taskForm(row, "delete")
	if form == nil || strings.ToLower(attr(form, "method")) != "post" || attr(form, "action") != "/tasks/"+id+"/delete" {
		method, action := "", ""
		if form != nil {
			method, action = attr(form, "method"), attr(form, "action")
		}
		t.Fatalf("delete form method=%q action=%q, want POST /tasks/%s/delete", method, action, id)
	}
	if strings.TrimSpace(textContent(elementByName(form, "button", ""))) != "Delete" {
		t.Fatalf("delete control = %q, want Delete", textContent(elementByName(form, "button", "")))
	}
	if hrefs := linkHrefs(row); len(hrefs) != 0 {
		t.Fatalf("task row links = %v, want no confirmation page", hrefs)
	}

	status, confirm := getPage(t, app, "/tasks/"+id+"/delete")
	if status == http.StatusOK {
		t.Fatalf("GET delete status = %d, want no confirmation page", status)
	}
	if strings.Contains(textContent(confirm), "Done work") || strings.Contains(strings.ToLower(textContent(confirm)), "are you sure") {
		t.Fatalf("confirmation text = %q, want no task confirmation", textContent(confirm))
	}

	_, list := getPage(t, app, "/projects")
	assertProjectCounts(t, list, "Ledger", "1 done", "2 open")
	assertProjectCounts(t, list, "Other", "0 done", "1 open")

	res := postFormNoRedirect(t, app, "/tasks/"+id+"/delete", nil, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want 303", res.StatusCode)
	}
	if got := res.Header.Get("Location"); got != ledger {
		t.Fatalf("location = %q, want %s", got, ledger)
	}
	_, page = getPage(t, app, ledger)
	assertTasks(t, page, []taskState{
		{text: "Open work", done: false},
		{text: "Also open", done: false},
	})

	_, list = getPage(t, app, "/projects")
	assertProjectCounts(t, list, "Ledger", "0 done", "2 open")
	assertProjectCounts(t, list, "Other", "0 done", "1 open")

	openID := attr(taskRowByText(page, "Open work"), "data-task")
	res = postFormNoRedirect(t, app, "/tasks/"+openID+"/delete", nil, nil)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != ledger {
		t.Fatalf("delete open status=%d location=%q", res.StatusCode, res.Header.Get("Location"))
	}
	_, page = getPage(t, app, ledger)
	assertTasks(t, page, []taskState{{text: "Also open", done: false}})
	_, list = getPage(t, app, "/projects")
	assertProjectCounts(t, list, "Ledger", "0 done", "1 open")

	lastID := attr(taskRowByText(page, "Also open"), "data-task")
	res = postFormNoRedirect(t, app, "/tasks/"+lastID+"/delete", nil, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("delete last status = %d, want 303", res.StatusCode)
	}
	_, page = getPage(t, app, ledger)
	if got := taskTexts(page); len(got) != 0 {
		t.Fatalf("tasks = %#v, want none", got)
	}
	if form := addTaskForm(page); form == nil || attr(form, "action") != ledger+"/tasks" {
		t.Fatal("a Project with no Tasks can still add one")
	}
	_, otherPage := getPage(t, app, other)
	assertTasks(t, otherPage, []taskState{{text: "Elsewhere", done: false}})
	_, list = getPage(t, app, "/projects")
	assertProjectCounts(t, list, "Ledger", "0 done", "0 open")
	assertProjectCounts(t, list, "Other", "0 done", "1 open")
}

func TestMissingTaskIsNotFound(t *testing.T) {
	app := startApp(t)
	loc := postProject(t, app, "Ledger")
	postTask(t, app, loc, "Kept")
	_, page := getPage(t, app, loc)
	id := attr(taskRowByText(page, "Kept"), "data-task")

	res := postFormNoRedirect(t, app, "/tasks/"+id+"/delete", nil, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want 303", res.StatusCode)
	}

	for _, path := range []string{
		"/tasks/" + id,
		"/tasks/" + id + "/delete",
		"/tasks/999999",
		"/tasks/999999/delete",
		"/tasks/nope",
		"/tasks/0",
		"/tasks/-1",
	} {
		res = postFormNoRedirect(t, app, path, url.Values{"text": {"Ghost"}, "done": {"1"}}, nil)
		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("POST %s status = %d, want 404", path, res.StatusCode)
		}
		doc := parseResponse(t, res)
		if !strings.Contains(textContent(doc), "Task not found.") {
			t.Fatalf("POST %s text = %q, want Task not found.", path, textContent(doc))
		}
		if got := res.Header.Get("Location"); got != "" {
			t.Fatalf("POST %s location = %q, want no redirect", path, got)
		}
	}

	_, page = getPage(t, app, loc)
	if got := taskTexts(page); len(got) != 0 {
		t.Fatalf("tasks = %#v, want the missing Task to stay gone", got)
	}
}

func TestTaskTextIsShownAsText(t *testing.T) {
	app := startApp(t)
	loc := postProject(t, app, "Ledger")
	text := "<script>alert(1)</script> & <b>nope</b>"
	postTask(t, app, loc, text)

	_, page := getPage(t, app, loc)
	if got := taskTexts(page); !sameStrings(got, []string{text}) {
		t.Fatalf("tasks = %#v, want the Task text as text", got)
	}
	if countElements(page, "script") != 1 || countElements(page, "b") != 0 {
		t.Fatal("task text was parsed as HTML")
	}

	id := attr(taskRowByText(page, text), "data-task")
	edited := "hello </textarea><script>alert(1)</script> \"quoted\""
	res := postFormNoRedirect(t, app, "/tasks/"+id, url.Values{"text": {edited}}, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST text status = %d, want 303", res.StatusCode)
	}
	_, page = getPage(t, app, loc)
	if got := taskTexts(page); !sameStrings(got, []string{edited}) {
		t.Fatalf("tasks = %#v, want the edited text as text", got)
	}
	if countElements(page, "script") != 1 || countElements(page, "textarea") != 1 {
		t.Fatal("edited task text was parsed as HTML")
	}
}

func TestTaskChangesUpdateInPlace(t *testing.T) {
	app := startApp(t)
	loc := postProject(t, app, "Ledger")
	_, page := getPage(t, app, loc)
	if elementByID(page, "project-tasks") == nil {
		t.Fatal("project page must have a task region")
	}
	add := addTaskForm(page)
	if attr(add, "hx-post") != loc+"/tasks" || attr(add, "hx-target") != "#project-tasks" || attr(add, "hx-swap") != "outerHTML" {
		t.Fatalf("add form hx-post=%q hx-target=%q hx-swap=%q, want an in-place add", attr(add, "hx-post"), attr(add, "hx-target"), attr(add, "hx-swap"))
	}

	hx := http.Header{"HX-Request": {"true"}}
	added := readPost(t, app, loc+"/tasks", url.Values{"text": {"Write the notes"}}, hx)
	if added.Status != http.StatusOK || strings.Contains(added.Body, "<html") {
		t.Fatalf("HTMX add status=%d body=%q, want the task region only", added.Status, added.Body)
	}
	doc := parseHTML(t, added.Body)
	if elementByID(doc, "project-tasks") == nil {
		t.Fatal("HTMX add did not return the task region")
	}
	assertTasks(t, elementByID(doc, "project-tasks"), []taskState{{text: "Write the notes", done: false}})
	if attr(addTaskForm(doc), "hx-post") != loc+"/tasks" || attr(addTaskForm(doc), "hx-target") != "#project-tasks" {
		t.Fatal("replaced task region lost its add form")
	}

	row := taskRowByText(doc, "Write the notes")
	id := attr(row, "data-task")
	textForm := taskForm(row, "text")
	if attr(textForm, "hx-post") != "/tasks/"+id || !strings.Contains(attr(textForm, "hx-trigger"), "blur") || attr(textForm, "hx-target") != "#project-tasks" || attr(textForm, "hx-swap") != "outerHTML" {
		t.Fatalf("text form hx-post=%q hx-trigger=%q hx-target=%q hx-swap=%q, want blur to replace the task region", attr(textForm, "hx-post"), attr(textForm, "hx-trigger"), attr(textForm, "hx-target"), attr(textForm, "hx-swap"))
	}
	doneForm := taskForm(row, "checkbox")
	if attr(doneForm, "hx-post") != "/tasks/"+id || !strings.Contains(attr(doneForm, "hx-trigger"), "change") || attr(doneForm, "hx-target") != "#project-tasks" || attr(doneForm, "hx-swap") != "outerHTML" {
		t.Fatalf("done form hx-post=%q hx-trigger=%q hx-target=%q hx-swap=%q, want change to replace the task region", attr(doneForm, "hx-post"), attr(doneForm, "hx-trigger"), attr(doneForm, "hx-target"), attr(doneForm, "hx-swap"))
	}
	deleteForm := taskForm(row, "delete")
	if attr(deleteForm, "hx-post") != "/tasks/"+id+"/delete" || attr(deleteForm, "hx-target") != "#project-tasks" || attr(deleteForm, "hx-swap") != "outerHTML" {
		t.Fatalf("delete form hx-post=%q hx-target=%q hx-swap=%q, want delete to replace the task region", attr(deleteForm, "hx-post"), attr(deleteForm, "hx-target"), attr(deleteForm, "hx-swap"))
	}

	added = readPost(t, app, loc+"/tasks", url.Values{"text": {"Ship it"}}, hx)
	doc = parseHTML(t, added.Body)
	assertTasks(t, elementByID(doc, "project-tasks"), []taskState{
		{text: "Write the notes", done: false},
		{text: "Ship it", done: false},
	})

	edited := readPost(t, app, "/tasks/"+id, url.Values{"text": {"Notes revised"}}, hx)
	if edited.Status != http.StatusOK || strings.Contains(edited.Body, "<html") {
		t.Fatalf("HTMX edit status=%d body=%q, want the task region only", edited.Status, edited.Body)
	}
	doc = parseHTML(t, edited.Body)
	assertTasks(t, elementByID(doc, "project-tasks"), []taskState{
		{text: "Notes revised", done: false},
		{text: "Ship it", done: false},
	})

	blank := readPost(t, app, "/tasks/"+id, url.Values{"text": {"  "}}, hx)
	if blank.Status != http.StatusOK || strings.Contains(blank.Body, "<html") {
		t.Fatalf("HTMX blank edit status=%d body=%q, want the task region only", blank.Status, blank.Body)
	}
	doc = parseHTML(t, blank.Body)
	if !strings.Contains(textContent(doc), "Task text is required.") {
		t.Fatalf("HTMX blank edit text = %q, want an error", textContent(doc))
	}
	assertTasks(t, elementByID(doc, "project-tasks"), []taskState{
		{text: "Notes revised", done: false},
		{text: "Ship it", done: false},
	})

	shipID := attr(taskRowByText(doc, "Ship it"), "data-task")
	done := readPost(t, app, "/tasks/"+shipID, url.Values{"done": {"1", "0"}}, hx)
	if done.Status != http.StatusOK || strings.Contains(done.Body, "<html") {
		t.Fatalf("HTMX done status=%d body=%q, want the task region only", done.Status, done.Body)
	}
	doc = parseHTML(t, done.Body)
	assertTasks(t, elementByID(doc, "project-tasks"), []taskState{
		{text: "Ship it", done: true},
		{text: "Notes revised", done: false},
	})

	reopened := readPost(t, app, "/tasks/"+shipID, url.Values{"done": {"0"}}, hx)
	doc = parseHTML(t, reopened.Body)
	assertTasks(t, elementByID(doc, "project-tasks"), []taskState{
		{text: "Notes revised", done: false},
		{text: "Ship it", done: false},
	})

	deleted := readPost(t, app, "/tasks/"+shipID+"/delete", nil, hx)
	if deleted.Status != http.StatusOK || strings.Contains(deleted.Body, "<html") {
		t.Fatalf("HTMX delete status=%d body=%q, want the task region only", deleted.Status, deleted.Body)
	}
	doc = parseHTML(t, deleted.Body)
	assertTasks(t, elementByID(doc, "project-tasks"), []taskState{{text: "Notes revised", done: false}})
	if attr(taskForm(taskRowByText(doc, "Notes revised"), "delete"), "hx-post") != "/tasks/"+id+"/delete" {
		t.Fatal("replaced task region lost its delete form")
	}

	_, page = getPage(t, app, loc)
	assertTasks(t, page, []taskState{{text: "Notes revised", done: false}})
	if strings.Contains(textContent(page), "Task text is required.") {
		t.Fatal("the rejected text is still showing")
	}
}

func TestCheckingATaskUpdatesEveryListOfThatProject(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")
	postTask(t, app, ledger, "Ship it")
	postTask(t, app, ledger, "Leave open")
	postMeetingProjects(t, app, "Earlier", "2026-01-02T15:04", "first notes", ledger)
	postMeetingProjects(t, app, "Later", "2026-02-02T15:04", "second notes", ledger)

	_, feed := getPage(t, app, "/")
	earlier := meetingCard(feed, "Earlier")
	later := meetingCard(feed, "Later")
	earlierForm := taskForm(taskRowByText(earlier, "Ship it"), "checkbox")
	laterForm := taskForm(taskRowByText(later, "Ship it"), "checkbox")
	earlierTarget := strings.TrimPrefix(attr(earlierForm, "hx-target"), "#")
	laterTarget := strings.TrimPrefix(attr(laterForm, "hx-target"), "#")
	if earlierTarget == "" || laterTarget == "" || earlierTarget == laterTarget {
		t.Fatalf("targets = %q and %q, want two Meeting lists", earlierTarget, laterTarget)
	}

	toggled := readPost(t, app, attr(earlierForm, "action"), url.Values{"done": {"1"}}, http.Header{
		"HX-Request": {"true"},
		"HX-Target":  {earlierTarget},
	})
	if toggled.Status != http.StatusOK || strings.Contains(toggled.Body, "<html") {
		t.Fatalf("toggle status=%d body=%q, want the task lists", toggled.Status, toggled.Body)
	}
	doc := parseHTML(t, toggled.Body)
	earlierList := elementByID(doc, earlierTarget)
	laterList := elementByID(doc, laterTarget)
	if earlierList == nil || laterList == nil {
		t.Fatal("the response must include both Meeting lists")
	}
	if attr(earlierList, "hx-swap-oob") != "" {
		t.Fatal("the targeted list is the primary fragment")
	}
	if attr(laterList, "hx-swap-oob") != "outerHTML" {
		t.Fatalf("other list swap = %q, want an out-of-band outerHTML swap", attr(laterList, "hx-swap-oob"))
	}
	doneSplit := []taskState{
		{text: "Ship it", done: true},
		{text: "Leave open", done: false},
	}
	assertTasks(t, earlierList, doneSplit)
	assertTasks(t, laterList, doneSplit)
	if hasAttribute(earlierList, "data-citation") || hasAttribute(laterList, "data-citation") {
		t.Fatal("Meeting lists omit the citation rule")
	}

	projectList := elementByID(doc, "project-tasks")
	if projectList == nil || attr(projectList, "hx-swap-oob") != "outerHTML" {
		t.Fatal("the Project list must swap out of band")
	}
	assertTasks(t, projectList, doneSplit)
	if !hasAttribute(projectList, "data-citation") {
		t.Fatal("the Project list includes the citation rule")
	}

	composerList := elementByID(doc, "composer-tasks-"+projectIDString(ledger))
	if composerList == nil || attr(composerList, "hx-swap-oob") != "outerHTML" {
		t.Fatal("the composer list must swap out of band")
	}
	assertTasks(t, composerList, doneSplit)
	if hasAttribute(composerList, "data-citation") {
		t.Fatal("the composer list omits the citation rule")
	}

	res := postFormNoRedirect(t, app, attr(earlierForm, "action"), url.Values{"done": {"0"}}, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want a normal request to redirect", res.StatusCode)
	}
	status, page := getPage(t, app, res.Header.Get("Location"))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want the resulting page", status)
	}
	assertTasks(t, page, []taskState{
		{text: "Ship it", done: false},
		{text: "Leave open", done: false},
	})
}

func assertProjectCounts(t *testing.T, doc *html.Node, name, done, open string) {
	t.Helper()

	item := listItemText(doc, name)
	if !strings.Contains(item, done) || !strings.Contains(item, open) {
		t.Fatalf("%s list text = %q, want %s and %s", name, item, done, open)
	}
}

func listItemText(doc *html.Node, name string) string {
	var found string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "li" && strings.Contains(textContent(n), name) {
			found = textContent(n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return found
}

type taskState struct {
	text string
	done bool
}

func assertTasks(t *testing.T, doc *html.Node, want []taskState) {
	t.Helper()

	rows := taskRows(doc)
	if len(rows) != len(want) {
		t.Fatalf("tasks = %#v, want %d", taskTexts(doc), len(want))
	}
	for i, row := range rows {
		if got := taskText(row); got != want[i].text || taskChecked(row) != want[i].done {
			t.Fatalf("task %d = %q done=%v, want %q done=%v", i, got, taskChecked(row), want[i].text, want[i].done)
		}
	}
}

func postTaskDone(t *testing.T, app *flow.App, loc, text string, done bool) {
	t.Helper()

	_, doc := getPage(t, app, loc)
	row := taskRowByText(doc, text)
	if row == nil {
		t.Fatalf("no Task %q", text)
	}
	form := taskForm(row, "checkbox")
	if form == nil || strings.ToLower(attr(form, "method")) != "post" || attr(form, "action") != "/tasks/"+attr(row, "data-task") {
		method, action := "", ""
		if form != nil {
			method, action = attr(form, "method"), attr(form, "action")
		}
		t.Fatalf("done form method=%q action=%q, want POST /tasks/%s", method, action, attr(row, "data-task"))
	}

	value := "0"
	if done {
		value = "1"
	}
	res := postFormNoRedirect(t, app, attr(form, "action"), url.Values{"done": {value}}, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST done status = %d, want 303", res.StatusCode)
	}
	if got := res.Header.Get("Location"); got != loc {
		t.Fatalf("location = %q, want %s", got, loc)
	}
}

func taskRowByText(doc *html.Node, text string) *html.Node {
	for _, row := range taskRows(doc) {
		if taskText(row) == text {
			return row
		}
	}

	return nil
}

func taskForm(row *html.Node, control string) *html.Node {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil || n == nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "form" {
			if control == "checkbox" && taskCheckbox(n) != nil {
				found = n
				return
			}
			if control == "text" && elementByName(n, "input", "text") != nil {
				found = n
				return
			}
			if control == "delete" && elementByName(n, "button", "") != nil {
				found = n
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(row)

	return found
}

func postTask(t *testing.T, app *flow.App, loc, text string) {
	t.Helper()

	res := postFormNoRedirect(t, app, loc+"/tasks", url.Values{"text": {text}}, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST task status = %d, want 303", res.StatusCode)
	}
	if got := res.Header.Get("Location"); got != loc {
		t.Fatalf("location = %q, want %s", got, loc)
	}
}

func taskRows(n *html.Node) []*html.Node {
	var rows []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && nodeHasAttr(n, "data-task") {
			rows = append(rows, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	return rows
}

func taskTexts(n *html.Node) []string {
	rows := taskRows(n)
	texts := make([]string, 0, len(rows))
	for _, row := range rows {
		texts = append(texts, taskText(row))
	}

	return texts
}

func taskText(row *html.Node) string {
	return attr(elementByName(row, "input", "text"), "value")
}

func taskCheckbox(row *html.Node) *html.Node {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "input" && attr(n, "type") == "checkbox" {
			found = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(row)

	return found
}

func taskChecked(row *html.Node) bool {
	return nodeHasAttr(taskCheckbox(row), "checked")
}

func addTaskForm(n *html.Node) *html.Node {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "form" && strings.HasSuffix(attr(n, "action"), "/tasks") {
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

func nodeHasAttr(n *html.Node, key string) bool {
	if n == nil {
		return false
	}
	for _, a := range n.Attr {
		if a.Key == key {
			return true
		}
	}

	return false
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}
