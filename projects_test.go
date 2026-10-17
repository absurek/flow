package flow_test

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/absurek/flow"
	"golang.org/x/net/html"
)

func TestCreateProjectRedirectsToItsPage(t *testing.T) {
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

	form := findElement(doc, "form")
	if form == nil || strings.ToLower(attr(form, "method")) != "post" || attr(form, "action") != "/projects" {
		method, action := "", ""
		if form != nil {
			method, action = attr(form, "method"), attr(form, "action")
		}
		t.Fatalf("create form method=%q action=%q, want POST /projects", method, action)
	}
	if elementByName(doc, "input", "name") == nil {
		t.Fatal("create form must take a name")
	}

	res := postProjectNoRedirect(t, app, "Ledger")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", res.StatusCode)
	}
	loc := res.Header.Get("Location")
	if _, ok := projectID(loc); !ok {
		t.Fatalf("location = %q, want /projects/{id}", loc)
	}

	status, page := getPage(t, app, loc)
	if status != http.StatusOK {
		t.Fatalf("project page status = %d, want 200", status)
	}
	if href := linkHref(page, "Meetings"); href != "/" {
		t.Fatalf("Meetings link = %q, want /", href)
	}
	if href := linkHref(page, "Projects"); href != "/projects" {
		t.Fatalf("Projects link = %q, want /projects", href)
	}
	if got := attr(elementByName(page, "input", "name"), "value"); got != "Ledger" {
		t.Fatalf("name = %q, want Ledger", got)
	}
}

func TestProjectsListShowsNewestFirstWithZeroCounts(t *testing.T) {
	app := startApp(t)

	alpha := postProject(t, app, "Alpha")
	beta := postProject(t, app, "Beta")
	gamma := postProject(t, app, "Gamma")

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
	if strings.Contains(textContent(doc), "No Projects.") {
		t.Fatal("list with Projects still says there are none")
	}

	text := textContent(doc)
	gammaAt := strings.Index(text, "Gamma")
	betaAt := strings.Index(text, "Beta")
	alphaAt := strings.Index(text, "Alpha")
	if gammaAt < 0 || betaAt < 0 || alphaAt < 0 || gammaAt > betaAt || betaAt > alphaAt {
		t.Fatalf("list order = %q, want Gamma then Beta then Alpha", text)
	}
	if strings.Count(text, "0 done") != 3 || strings.Count(text, "0 open") != 3 {
		t.Fatalf("list text = %q, want each Project to show 0 done and 0 open", text)
	}
	if got := linkHref(doc, "Gamma"); got != gamma {
		t.Fatalf("Gamma link = %q, want %s", got, gamma)
	}
	if got := linkHref(doc, "Beta"); got != beta {
		t.Fatalf("Beta link = %q, want %s", got, beta)
	}
	if got := linkHref(doc, "Alpha"); got != alpha {
		t.Fatalf("Alpha link = %q, want %s", got, alpha)
	}
}

func TestBlankProjectNameIsRejected(t *testing.T) {
	app := startApp(t)
	kept := postProject(t, app, "Kept")

	for _, name := range []string{"", "   "} {
		res := postProjectNoRedirect(t, app, name)
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("name %q status = %d, want 400", name, res.StatusCode)
		}
		doc := parseResponse(t, res)
		if !strings.Contains(textContent(doc), "Name is required.") {
			t.Fatalf("name %q text = %q, want a name error", name, textContent(doc))
		}
		if got := attr(elementByName(doc, "input", "name"), "value"); got != name {
			t.Fatalf("name field = %q, want the submitted %q", got, name)
		}
		if links := projectPageLinks(doc); len(links) != 1 || links[0] != kept {
			t.Fatalf("name %q links = %v, want only %s", name, links, kept)
		}
	}

	_, doc := getPage(t, app, "/projects")
	if links := projectPageLinks(doc); len(links) != 1 || links[0] != kept {
		t.Fatalf("links = %v, want only %s after rejected creates", links, kept)
	}
	if strings.Contains(textContent(doc), "Name is required.") {
		t.Fatal("a later visit still shows the rejected create")
	}
}

func TestStoredProjectNameIsTrimmed(t *testing.T) {
	app := startApp(t)

	loc := postProject(t, app, "  Ledger  ")
	_, page := getPage(t, app, loc)
	if got := attr(elementByName(page, "input", "name"), "value"); got != "Ledger" {
		t.Fatalf("stored name = %q, want the trimmed name", got)
	}

	_, doc := getPage(t, app, "/projects")
	if got := linkHref(doc, "Ledger"); got != loc {
		t.Fatalf("list link for Ledger = %q, want %s", got, loc)
	}
	if strings.Contains(textContent(doc), "  Ledger  ") {
		t.Fatalf("list text = %q, want the trimmed name only", textContent(doc))
	}
}

func TestDuplicateProjectNameIsRejected(t *testing.T) {
	app := startApp(t)
	ledger := postProject(t, app, "Ledger")

	for _, name := range []string{"ledger", " LEDGER ", "Ledger "} {
		res := postProjectNoRedirect(t, app, name)
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("name %q status = %d, want 400", name, res.StatusCode)
		}
		doc := parseResponse(t, res)
		if !strings.Contains(textContent(doc), "A Project with that name already exists.") {
			t.Fatalf("name %q text = %q, want a duplicate error", name, textContent(doc))
		}
		if got := attr(elementByName(doc, "input", "name"), "value"); got != name {
			t.Fatalf("name field = %q, want the submitted %q", got, name)
		}
		if links := projectPageLinks(doc); len(links) != 1 || links[0] != ledger {
			t.Fatalf("name %q links = %v, want only %s", name, links, ledger)
		}
	}

	other := postProject(t, app, "Ledger Book")
	_, doc := getPage(t, app, "/projects")
	links := projectPageLinks(doc)
	if len(links) != 2 || links[0] != other || links[1] != ledger {
		t.Fatalf("links = %v, want %s then %s", links, other, ledger)
	}
	if strings.Contains(textContent(doc), "A Project with that name already exists.") {
		t.Fatal("a later visit still shows the rejected create")
	}
}

func TestProjectNamesAreUniqueUnderUnicodeCaseFold(t *testing.T) {
	app := startApp(t)
	omega := postProject(t, app, "Ωmega")
	other := postProject(t, app, "Other")

	res := postProjectNoRedirect(t, app, "ωmega")
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("create status = %d, want 400", res.StatusCode)
	}
	doc := parseResponse(t, res)
	if !strings.Contains(textContent(doc), "A Project with that name already exists.") {
		t.Fatalf("create text = %q, want a duplicate error", textContent(doc))
	}
	if got := attr(elementByName(doc, "input", "name"), "value"); got != "ωmega" {
		t.Fatalf("name field = %q, want the submitted name", got)
	}
	if links := projectPageLinks(doc); len(links) != 2 || links[0] != other || links[1] != omega {
		t.Fatalf("links = %v, want only the existing Projects", links)
	}

	res = postFormNoRedirect(t, app, other+"/name", url.Values{"name": {"ωmega"}}, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("rename status = %d, want 400", res.StatusCode)
	}
	doc = parseResponse(t, res)
	if !strings.Contains(textContent(doc), "A Project with that name already exists.") {
		t.Fatalf("rename text = %q, want a duplicate error", textContent(doc))
	}
	if got := attr(elementByName(doc, "input", "name"), "value"); got != "Other" {
		t.Fatalf("name field = %q, want the previous name", got)
	}

	_, page := getPage(t, app, other)
	if got := attr(elementByName(page, "input", "name"), "value"); got != "Other" {
		t.Fatalf("name = %q, want the rejected rename unwritten", got)
	}
	_, list := getPage(t, app, "/projects")
	if strings.Contains(textContent(list), "ωmega") {
		t.Fatalf("list text = %q, want Ωmega kept and ωmega rejected", textContent(list))
	}
}

func TestProjectPageShowsNameDividerAndNotes(t *testing.T) {
	app := startApp(t)
	postMeeting(t, app, "Planning", "2026-01-02T15:04", "Talked through the timeline.")
	loc := postProject(t, app, "Ledger")

	status, doc := getPage(t, app, loc)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if href := linkHref(doc, "Meetings"); href != "/" {
		t.Fatalf("Meetings link = %q, want /", href)
	}
	if href := linkHref(doc, "Projects"); href != "/projects" {
		t.Fatalf("Projects link = %q, want /projects", href)
	}

	name := elementByName(doc, "input", "name")
	notes := elementByName(doc, "textarea", "notes")
	divider := findElement(doc, "hr")
	if name == nil || notes == nil || divider == nil {
		t.Fatal("project page must show the name, a divider, and one notes field")
	}
	if !elementPrecedes(doc, name, divider) || !elementPrecedes(doc, divider, notes) {
		t.Fatal("the divider must sit between the name and the notes")
	}
	if countElements(doc, "textarea") != 1 {
		t.Fatalf("textarea count = %d, want one notes field", countElements(doc, "textarea"))
	}
	if countElements(doc, "hr") != 1 {
		t.Fatalf("divider count = %d, want one divider", countElements(doc, "hr"))
	}
	if len(taskRows(doc)) != 0 {
		t.Fatal("a Project with no Tasks should not list one")
	}
	if hasAttribute(doc, "data-citation") || strings.Contains(textContent(doc), "Planning") || strings.Contains(textContent(doc), "citation") {
		t.Fatalf("project page text = %q, want no citation rule", textContent(doc))
	}
	if strings.TrimSpace(textContent(notes)) != "" {
		t.Fatalf("notes = %q, want an empty notes field", textContent(notes))
	}
}

func TestProjectNotesPersistAsEdited(t *testing.T) {
	app := startApp(t)
	loc := postProject(t, app, "Ledger")

	postProjectNotes(t, app, loc, "alpha\nbeta")
	_, page := getPage(t, app, loc)
	if got := textContent(elementByName(page, "textarea", "notes")); got != "alpha\nbeta" {
		t.Fatalf("notes = %q, want the line break kept", got)
	}

	postProjectNotes(t, app, loc, "")
	_, page = getPage(t, app, loc)
	if got := textContent(elementByName(page, "textarea", "notes")); strings.TrimSpace(got) != "" {
		t.Fatalf("notes = %q, want them cleared", got)
	}

	postProjectNotes(t, app, loc, "back again")
	_, page = getPage(t, app, loc)
	if got := textContent(elementByName(page, "textarea", "notes")); got != "back again" {
		t.Fatalf("notes = %q, want the edited notes", got)
	}
}

func TestRenameKeepsTheProjectInPlace(t *testing.T) {
	app := startApp(t)
	alpha := postProject(t, app, "Alpha")
	beta := postProject(t, app, "Beta")

	postProjectName(t, app, alpha, "  Zebra  ")
	_, page := getPage(t, app, alpha)
	if got := attr(elementByName(page, "input", "name"), "value"); got != "Zebra" {
		t.Fatalf("name = %q, want the trimmed name", got)
	}

	_, doc := getPage(t, app, "/projects")
	text := textContent(doc)
	betaAt := strings.Index(text, "Beta")
	zebraAt := strings.Index(text, "Zebra")
	if betaAt < 0 || zebraAt < 0 || betaAt > zebraAt || strings.Contains(text, "Alpha") {
		t.Fatalf("list order = %q, want Beta then Zebra", text)
	}
	if got := linkHref(doc, "Zebra"); got != alpha {
		t.Fatalf("Zebra link = %q, want the same Project %s", got, alpha)
	}
	if got := linkHref(doc, "Beta"); got != beta {
		t.Fatalf("Beta link = %q, want %s", got, beta)
	}

	postProjectName(t, app, alpha, "ZEBRA")
	_, doc = getPage(t, app, "/projects")
	text = textContent(doc)
	betaAt = strings.Index(text, "Beta")
	zebraAt = strings.Index(text, "ZEBRA")
	if betaAt < 0 || zebraAt < 0 || betaAt > zebraAt || strings.Contains(text, "Zebra") {
		t.Fatalf("list order = %q, want Beta then ZEBRA", text)
	}
}

func TestRejectedRenameKeepsThePreviousName(t *testing.T) {
	app := startApp(t)
	alpha := postProject(t, app, "Alpha")
	beta := postProject(t, app, "Beta")

	cases := []struct {
		name  string
		error string
	}{
		{name: "", error: "Name is required."},
		{name: "   ", error: "Name is required."},
		{name: " alpha ", error: "A Project with that name already exists."},
		{name: "ALPHA", error: "A Project with that name already exists."},
	}
	for _, tc := range cases {
		res := postFormNoRedirect(t, app, beta+"/name", url.Values{"name": {tc.name}}, nil)
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("name %q status = %d, want 400", tc.name, res.StatusCode)
		}
		doc := parseResponse(t, res)
		if !strings.Contains(textContent(doc), tc.error) {
			t.Fatalf("name %q text = %q, want %s", tc.name, textContent(doc), tc.error)
		}
		if got := attr(elementByName(doc, "input", "name"), "value"); got != "Beta" {
			t.Fatalf("name %q field = %q, want the previous name", tc.name, got)
		}
	}

	_, page := getPage(t, app, beta)
	if got := attr(elementByName(page, "input", "name"), "value"); got != "Beta" {
		t.Fatalf("name = %q, want the previous name kept", got)
	}
	_, doc := getPage(t, app, "/projects")
	text := textContent(doc)
	betaAt := strings.Index(text, "Beta")
	alphaAt := strings.Index(text, "Alpha")
	if betaAt < 0 || alphaAt < 0 || betaAt > alphaAt || strings.Contains(text, "ALPHA") {
		t.Fatalf("list text = %q, want Beta then Alpha with the rejected name unwritten", text)
	}
	if links := projectPageLinks(doc); len(links) != 2 || links[0] != beta || links[1] != alpha {
		t.Fatalf("links = %v, want %s then %s", links, beta, alpha)
	}
}

func postProjectName(t *testing.T, app *flow.App, loc, name string) {
	t.Helper()

	res := postFormNoRedirect(t, app, loc+"/name", url.Values{"name": {name}}, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST name status = %d, want 303", res.StatusCode)
	}
	if got := res.Header.Get("Location"); got != loc {
		t.Fatalf("location = %q, want %s", got, loc)
	}
}

func TestDeleteProjectRequiresConfirmation(t *testing.T) {
	app := startApp(t)
	alpha := postProject(t, app, "Alpha")
	beta := postProject(t, app, "Beta")
	postProjectNotes(t, app, alpha, "alpha notes")
	postProjectNotes(t, app, beta, "beta notes")

	_, list := getPage(t, app, "/projects")
	if !containsHref(list, alpha+"/delete") {
		t.Fatal("the Projects list should link to delete confirmation")
	}
	_, page := getPage(t, app, alpha)
	if got := linkHref(page, "Delete"); got != alpha+"/delete" {
		t.Fatalf("Delete link = %q, want %s", got, alpha+"/delete")
	}

	status, confirm := getPage(t, app, alpha+"/delete")
	if status != http.StatusOK {
		t.Fatalf("confirmation status = %d, want 200", status)
	}
	if !strings.Contains(textContent(confirm), "Alpha") {
		t.Fatalf("confirmation text = %q, want the Project name", textContent(confirm))
	}
	form := findElement(confirm, "form")
	if form == nil || strings.ToLower(attr(form, "method")) != "post" || attr(form, "action") != alpha+"/delete" {
		method, action := "", ""
		if form != nil {
			method, action = attr(form, "method"), attr(form, "action")
		}
		t.Fatalf("confirmation form method=%q action=%q, want POST %s", method, action, alpha+"/delete")
	}
	if got := linkHref(confirm, "Cancel"); got != alpha {
		t.Fatalf("Cancel link = %q, want %s", got, alpha)
	}

	_, list = getPage(t, app, "/projects")
	if links := projectPageLinks(list); len(links) != 2 || links[0] != beta || links[1] != alpha {
		t.Fatalf("links after visiting confirmation = %v, want both Projects kept", links)
	}
	_, alphaPage := getPage(t, app, alpha)
	if got := textContent(elementByName(alphaPage, "textarea", "notes")); got != "alpha notes" {
		t.Fatalf("notes = %q, want them kept when delete is not confirmed", got)
	}

	res := postFormNoRedirect(t, app, alpha+"/delete", nil, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want 303", res.StatusCode)
	}
	if got := res.Header.Get("Location"); got != "/projects" {
		t.Fatalf("location = %q, want /projects", got)
	}

	_, list = getPage(t, app, "/projects")
	text := textContent(list)
	if strings.Contains(text, "Alpha") || !strings.Contains(text, "Beta") {
		t.Fatalf("list text = %q, want Beta kept and Alpha removed", text)
	}
	if links := projectPageLinks(list); len(links) != 1 || links[0] != beta {
		t.Fatalf("links = %v, want only %s", links, beta)
	}
	status, missing := getPage(t, app, alpha)
	if status != http.StatusNotFound || !strings.Contains(textContent(missing), "Project not found.") {
		t.Fatalf("deleted Project status=%d text=%q", status, textContent(missing))
	}
	_, betaPage := getPage(t, app, beta)
	if got := textContent(elementByName(betaPage, "textarea", "notes")); got != "beta notes" {
		t.Fatalf("notes = %q, want the other Project's notes kept", got)
	}
}

func TestRecreatedProjectStartsWithNoNotes(t *testing.T) {
	app := startApp(t)
	first := postProject(t, app, "Ledger")
	postProjectNotes(t, app, first, "secret\nnotes")

	res := postFormNoRedirect(t, app, first+"/delete", nil, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want 303", res.StatusCode)
	}

	second := postProject(t, app, " LEDGER ")
	if second == first {
		t.Fatal("a Project created after delete reused the deleted Project")
	}

	status, page := getPage(t, app, second)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if got := attr(elementByName(page, "input", "name"), "value"); got != "LEDGER" {
		t.Fatalf("name = %q, want the trimmed new name", got)
	}
	if got := textContent(elementByName(page, "textarea", "notes")); strings.TrimSpace(got) != "" {
		t.Fatalf("notes = %q, want a new Project with no notes", got)
	}
	if strings.Contains(textContent(page), "secret") {
		t.Fatal("the new Project kept the deleted Project's notes")
	}

	status, missing := getPage(t, app, first)
	if status != http.StatusNotFound {
		t.Fatalf("deleted Project status = %d, want 404", status)
	}
	if strings.Contains(textContent(missing), "secret") {
		t.Fatal("the deleted Project's notes are still visible")
	}

	_, doc := getPage(t, app, "/projects")
	if links := projectPageLinks(doc); len(links) != 1 || links[0] != second {
		t.Fatalf("links = %v, want only the new Project %s", links, second)
	}
	if !strings.Contains(textContent(doc), "LEDGER") || strings.Contains(textContent(doc), "secret") {
		t.Fatalf("list text = %q, want the new name and no old notes", textContent(doc))
	}
}

func TestMissingProjectIsNotFound(t *testing.T) {
	app := startApp(t)
	postProject(t, app, "Kept")

	for _, path := range []string{
		"/projects/999999",
		"/projects/not-a-project",
		"/projects/0",
		"/projects/-1",
		"/projects/999999/delete",
		"/projects/not-a-project/delete",
	} {
		status, doc := getPage(t, app, path)
		if status != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404", path, status)
		}
		if !strings.Contains(textContent(doc), "Project not found.") {
			t.Fatalf("%s text = %q, want Project not found.", path, textContent(doc))
		}
	}

	for _, path := range []string{
		"/projects/999999/name",
		"/projects/999999/notes",
		"/projects/999999/delete",
		"/projects/nope/name",
	} {
		res := postFormNoRedirect(t, app, path, url.Values{"name": {"Ghost"}, "notes": {"nope"}}, nil)
		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("POST %s status = %d, want 404", path, res.StatusCode)
		}
		doc := parseResponse(t, res)
		if !strings.Contains(textContent(doc), "Project not found.") {
			t.Fatalf("POST %s text = %q, want Project not found.", path, textContent(doc))
		}
		if loc := res.Header.Get("Location"); loc != "" {
			t.Fatalf("POST %s location = %q, want no redirect", path, loc)
		}
	}

	_, doc := getPage(t, app, "/projects")
	if links := projectPageLinks(doc); len(links) != 1 || !strings.Contains(textContent(doc), "Kept") {
		t.Fatalf("list text = %q, want the existing Project left in place", textContent(doc))
	}
}

func containsHref(doc *html.Node, href string) bool {
	for _, candidate := range linkHrefs(doc) {
		if candidate == href {
			return true
		}
	}

	return false
}

func TestRenameAndNotesUpdateInPlace(t *testing.T) {
	app := startApp(t)
	loc := postProject(t, app, "Ledger")
	_, page := getPage(t, app, loc)

	scriptSrc := scriptSrc(page)
	if scriptSrc != "/htmx.js" {
		t.Fatalf("script src = %q, want the vendored /htmx.js", scriptSrc)
	}
	for _, src := range scriptSrcs(page) {
		if strings.Contains(src, "://") || strings.HasPrefix(src, "//") {
			t.Fatalf("script src = %q, want HTMX served by the app", src)
		}
	}

	res, err := http.Get(app.URL() + scriptSrc)
	if err != nil {
		t.Fatalf("GET %s: %v", scriptSrc, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("htmx status = %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Fatalf("htmx content-type = %q, want javascript", ct)
	}
	library, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read htmx: %v", err)
	}
	if !strings.Contains(string(library), "htmx") {
		t.Fatal("vendored script does not look like HTMX")
	}

	nameForm := elementByID(page, "project-name")
	if nameForm == nil || attr(nameForm, "hx-post") != loc+"/name" || !strings.Contains(attr(nameForm, "hx-trigger"), "blur") || attr(nameForm, "hx-target") != "#project-name" || attr(nameForm, "hx-swap") != "outerHTML" {
		t.Fatalf("name region hx-post=%q hx-trigger=%q hx-target=%q hx-swap=%q, want blur to %s/name replacing #project-name", attr(nameForm, "hx-post"), attr(nameForm, "hx-trigger"), attr(nameForm, "hx-target"), attr(nameForm, "hx-swap"), loc)
	}
	notesForm := elementByID(page, "project-notes")
	if notesForm == nil || attr(notesForm, "hx-post") != loc+"/notes" || !strings.Contains(attr(notesForm, "hx-trigger"), "blur") || attr(notesForm, "hx-target") != "#project-notes" || attr(notesForm, "hx-swap") != "outerHTML" {
		t.Fatalf("notes region hx-post=%q hx-trigger=%q hx-target=%q hx-swap=%q, want blur to %s/notes replacing #project-notes", attr(notesForm, "hx-post"), attr(notesForm, "hx-trigger"), attr(notesForm, "hx-target"), attr(notesForm, "hx-swap"), loc)
	}

	hx := http.Header{"HX-Request": {"true"}}
	renamed := readPost(t, app, loc+"/name", url.Values{"name": {"Books"}}, hx)
	if renamed.Status != http.StatusOK || strings.Contains(renamed.Body, "<html") {
		t.Fatalf("HTMX rename status=%d body=%q, want the name region only", renamed.Status, renamed.Body)
	}
	renamedDoc := parseHTML(t, renamed.Body)
	if elementByID(renamedDoc, "project-name") == nil {
		t.Fatal("HTMX rename did not return the name region")
	}
	if got := attr(elementByName(renamedDoc, "input", "name"), "value"); got != "Books" {
		t.Fatalf("HTMX name = %q, want Books", got)
	}
	if attr(elementByID(renamedDoc, "project-name"), "hx-post") != loc+"/name" {
		t.Fatal("replaced name region lost its blur save")
	}

	_, doc := getPage(t, app, "/projects")
	if linkHref(doc, "Books") != loc {
		t.Fatalf("list link = %q, want the renamed Project", linkHref(doc, "Books"))
	}

	rejected := readPost(t, app, loc+"/name", url.Values{"name": {"  "}}, hx)
	if rejected.Status != http.StatusOK || strings.Contains(rejected.Body, "<html") {
		t.Fatalf("HTMX rejected rename status=%d body=%q, want the name region only", rejected.Status, rejected.Body)
	}
	rejectedDoc := parseHTML(t, rejected.Body)
	if !strings.Contains(textContent(rejectedDoc), "Name is required.") {
		t.Fatalf("HTMX rename text = %q, want an error in the name region", textContent(rejectedDoc))
	}
	if got := attr(elementByName(rejectedDoc, "input", "name"), "value"); got != "Books" {
		t.Fatalf("HTMX name = %q, want the previous name", got)
	}

	noted := readPost(t, app, loc+"/notes", url.Values{"notes": {"alpha\nbeta"}}, hx)
	if noted.Status != http.StatusOK || strings.Contains(noted.Body, "<html") {
		t.Fatalf("HTMX notes status=%d body=%q, want the notes region only", noted.Status, noted.Body)
	}
	notedDoc := parseHTML(t, noted.Body)
	if elementByID(notedDoc, "project-notes") == nil {
		t.Fatal("HTMX notes did not return the notes region")
	}
	if got := textContent(elementByName(notedDoc, "textarea", "notes")); got != "alpha\nbeta" {
		t.Fatalf("HTMX notes = %q, want the edited notes", got)
	}

	_, page = getPage(t, app, loc)
	if got := attr(elementByName(page, "input", "name"), "value"); got != "Books" {
		t.Fatalf("name = %q, want Books", got)
	}
	if got := textContent(elementByName(page, "textarea", "notes")); got != "alpha\nbeta" {
		t.Fatalf("notes = %q, want the edited notes", got)
	}
	if strings.Contains(textContent(page), "Name is required.") {
		t.Fatal("the rejected rename is still showing")
	}
}

type postedBody struct {
	Status int
	Body   string
}

func readPost(t *testing.T, app *flow.App, path string, form url.Values, headers http.Header) postedBody {
	t.Helper()

	res := postFormNoRedirect(t, app, path, form, headers)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return postedBody{Status: res.StatusCode, Body: string(body)}
}

func parseHTML(t *testing.T, body string) *html.Node {
	t.Helper()

	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}

	return doc
}

func elementByID(n *html.Node, id string) *html.Node {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && attr(n, "id") == id {
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

func scriptSrc(n *html.Node) string {
	srcs := scriptSrcs(n)
	if len(srcs) == 0 {
		return ""
	}

	return srcs[0]
}

func scriptSrcs(n *html.Node) []string {
	var srcs []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			if src := attr(n, "src"); src != "" {
				srcs = append(srcs, src)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	return srcs
}

func TestProjectNameAndNotesAreShownAsText(t *testing.T) {
	app := startApp(t)
	name := "<script>alert(1)</script>"
	notes := "hello <b>nope</b> & co\n</textarea><script>alert(1)</script>"

	loc := postProject(t, app, name)
	postProjectNotes(t, app, loc, notes)

	_, doc := getPage(t, app, "/projects")
	if !strings.Contains(textContent(doc), name) {
		t.Fatalf("list text = %q, want the name as text", textContent(doc))
	}
	if got := linkHref(doc, name); got != loc {
		t.Fatalf("name link = %q, want %s", got, loc)
	}

	_, page := getPage(t, app, loc)
	if got := attr(elementByName(page, "input", "name"), "value"); got != name {
		t.Fatalf("name = %q, want the name as text", got)
	}
	if got := textContent(elementByName(page, "textarea", "notes")); got != notes {
		t.Fatalf("notes = %q, want the notes as text", got)
	}

	_, confirm := getPage(t, app, loc+"/delete")
	if !strings.Contains(textContent(confirm), name) {
		t.Fatalf("confirmation text = %q, want the name as text", textContent(confirm))
	}
}

func postProjectNotes(t *testing.T, app *flow.App, loc, notes string) {
	t.Helper()

	res := postFormNoRedirect(t, app, loc+"/notes", url.Values{"notes": {notes}}, nil)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST notes status = %d, want 303", res.StatusCode)
	}
	if got := res.Header.Get("Location"); got != loc {
		t.Fatalf("location = %q, want %s", got, loc)
	}
}

func elementPrecedes(doc, earlier, later *html.Node) bool {
	seenEarlier := false
	precedes := false
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if precedes {
			return
		}
		if n == earlier {
			seenEarlier = true
		}
		if n == later && seenEarlier {
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

func countElements(n *html.Node, element string) int {
	count := 0
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == element {
			count++
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	return count
}

func hasAttribute(n *html.Node, key string) bool {
	found := false
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found {
			return
		}
		if n.Type == html.ElementNode {
			for _, a := range n.Attr {
				if a.Key == key {
					found = true
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	return found
}

func projectPageLinks(doc *html.Node) []string {
	var hrefs []string
	for _, href := range linkHrefs(doc) {
		if _, ok := projectID(href); ok {
			hrefs = append(hrefs, href)
		}
	}

	return hrefs
}

func postProject(t *testing.T, app *flow.App, name string) string {
	t.Helper()

	res := postProjectNoRedirect(t, app, name)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST project %q status = %d, want 303", name, res.StatusCode)
	}
	loc := res.Header.Get("Location")
	if _, ok := projectID(loc); !ok {
		t.Fatalf("location = %q, want /projects/{id}", loc)
	}

	return loc
}

func postProjectNoRedirect(t *testing.T, app *flow.App, name string) *http.Response {
	t.Helper()

	return postFormNoRedirect(t, app, "/projects", url.Values{"name": {name}}, nil)
}

func postFormNoRedirect(t *testing.T, app *flow.App, path string, form url.Values, headers http.Header) *http.Response {
	t.Helper()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest(http.MethodPost, app.URL()+path, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	t.Cleanup(func() { res.Body.Close() })

	return res
}

func projectID(path string) (int64, bool) {
	rest, ok := strings.CutPrefix(path, "/projects/")
	if !ok || rest == "" || strings.Contains(rest, "/") {
		return 0, false
	}

	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}

	return id, true
}
