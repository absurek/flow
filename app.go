package flow

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed static/app.css
var stylesheet []byte

//go:embed static/htmx.min.js
var htmxJS []byte

//go:embed templates/*.html
var templateFiles embed.FS

const (
	happenedAtInput   = "2006-01-02T15:04"
	happenedAtDisplay = "2 Jan 2006, 15:04"
)

// Config is the startup configuration for the local app.
type Config struct {
	Addr     string
	DBPath   string
	Now      func() time.Time
	Location *time.Location
}

// App is the running local server.
type App struct {
	url    string
	server *http.Server
	db     *sql.DB
	tmpl   *template.Template
	now    func() time.Time
	loc    *time.Location
}

type composer struct {
	Title      string
	HappenedAt string
	Body       string
	Name       string
	Projects   []linkedProject
	Picker     []projectView
}

type linkedProject struct {
	ID        int64
	Name      string
	Tasks     []taskView
	TaskError string
	TasksID   string
}

type meetingView struct {
	ID         int64
	Title      string
	HappenedAt string
	Body       string
	Projects   []linkedProject
}

type feedPage struct {
	Title    string
	Errors   []string
	Composer composer
	Meetings []meetingView
}

type projectView struct {
	ID   int64
	Name string
	Done int
	Open int
}

type projectsPage struct {
	Title    string
	Name     string
	Errors   []string
	Projects []projectView
}

type taskView struct {
	ID   int64
	Text string
	Done bool
}

type projectPage struct {
	Title     string
	ID        int64
	Name      string
	Notes     string
	Errors    []string
	Tasks     []taskView
	TaskError string
	TasksID   string
}

type meetingPage struct {
	Title   string
	Meeting meetingView
}

type messagePage struct {
	Title   string
	Message string
}

// Listen binds a loopback address and serves the app.
func Listen(cfg Config) (*App, error) {
	addr, err := requireLoopback(cfg.Addr)
	if err != nil {
		return nil, err
	}

	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	loc := cfg.Location
	if loc == nil {
		loc = time.Local
	}

	db, err := openDB(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.ParseFS(templateFiles, "templates/*.html")
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}

	app := &App{
		db:   db,
		tmpl: tmpl,
		now:  now,
		loc:  loc,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", app.feed)
	mux.HandleFunc("POST /meetings", app.postMeeting)
	mux.HandleFunc("POST /composer/projects", app.postComposerProjects)
	mux.HandleFunc("GET /meetings/{id}", app.meetingPage)
	mux.HandleFunc("GET /projects", app.projects)
	mux.HandleFunc("POST /projects", app.postProject)
	mux.HandleFunc("GET /projects/{id}", app.projectPage)
	mux.HandleFunc("POST /projects/{id}/name", app.postProjectName)
	mux.HandleFunc("POST /projects/{id}/notes", app.postProjectNotes)
	mux.HandleFunc("POST /projects/{id}/tasks", app.postProjectTask)
	mux.HandleFunc("POST /tasks/{id}", app.postTask)
	mux.HandleFunc("POST /tasks/{id}/delete", app.postTaskDelete)
	mux.HandleFunc("GET /projects/{id}/delete", app.projectDeletePage)
	mux.HandleFunc("POST /projects/{id}/delete", app.postProjectDelete)
	mux.HandleFunc("GET /styles.css", serveStyles)
	mux.HandleFunc("GET /htmx.js", serveHTMX)

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		_ = server.Serve(ln)
	}()

	app.url = "http://" + ln.Addr().String()
	app.server = server

	return app, nil
}

func serveStyles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write(stylesheet)
}

func serveHTMX(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	_, _ = w.Write(htmxJS)
}

func htmxRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func (a *App) feed(w http.ResponseWriter, r *http.Request) {
	a.renderFeed(w, r, http.StatusOK, composer{
		HappenedAt: a.now().In(a.loc).Format(happenedAtInput),
	}, nil)
}

func (a *App) postMeeting(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "could not read the form", http.StatusBadRequest)
		return
	}

	title := r.PostForm.Get("title")
	happenedRaw := r.PostForm.Get("happened_at")
	body := r.PostForm.Get("body")
	ids, err := a.keptProjectIDs(r.Context(), formProjectIDs(r.PostForm["project"]))
	if err != nil {
		http.Error(w, "could not load Projects", http.StatusInternalServerError)
		return
	}
	projects, err := a.linkedProjects(r.Context(), ids, composerTasksID)
	if err != nil {
		http.Error(w, "could not load Projects", http.StatusInternalServerError)
		return
	}

	var errs []string
	if strings.TrimSpace(title) == "" {
		errs = append(errs, "Title is required.")
	}

	happenedAt := strings.TrimSpace(happenedRaw)
	happened, err := time.ParseInLocation(happenedAtInput, happenedAt, a.loc)
	if happenedAt == "" {
		errs = append(errs, "Happened-at is required.")
	} else if err != nil {
		errs = append(errs, "Happened-at is invalid.")
	}

	if len(errs) > 0 {
		a.renderFeed(w, r, http.StatusBadRequest, composer{
			Title:      title,
			HappenedAt: happenedRaw,
			Body:       body,
			Name:       r.PostForm.Get("name"),
			Projects:   projects,
		}, errs)
		return
	}

	if err := a.insertMeeting(r.Context(), strings.TrimSpace(title), happened, body, ids); err != nil {
		http.Error(w, "could not save the Meeting", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) postComposerProjects(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "could not read the form", http.StatusBadRequest)
		return
	}

	ids, err := a.keptProjectIDs(r.Context(), formProjectIDs(r.PostForm["project"]))
	if err != nil {
		http.Error(w, "could not load Projects", http.StatusInternalServerError)
		return
	}
	if remove := formProjectIDs([]string{r.PostForm.Get("remove")}); len(remove) == 1 {
		ids = withoutID(ids, remove[0])
	}
	addIDs, err := a.keptProjectIDs(r.Context(), formProjectIDs([]string{r.PostForm.Get("add")}))
	if err != nil {
		http.Error(w, "could not load Projects", http.StatusInternalServerError)
		return
	}
	for _, id := range addIDs {
		if !containsID(ids, id) {
			ids = append(ids, id)
		}
	}

	typedName := r.PostForm.Get("name")
	var errs []string
	status := http.StatusOK
	if _, ok := r.PostForm["create"]; ok {
		trimmed := strings.TrimSpace(typedName)
		if trimmed == "" {
			errs = append(errs, "Name is required.")
			status = http.StatusBadRequest
		} else {
			taken, err := a.projectNameTaken(r.Context(), trimmed, 0)
			if err != nil {
				http.Error(w, "could not save the Project", http.StatusInternalServerError)
				return
			}
			if taken {
				errs = append(errs, "A Project with that name already exists.")
				status = http.StatusBadRequest
			} else {
				id, err := a.insertProject(r.Context(), trimmed)
				if err != nil {
					http.Error(w, "could not save the Project", http.StatusInternalServerError)
					return
				}

				ids = append(ids, id)
				typedName = ""
			}
		}
	}

	projects, err := a.linkedProjects(r.Context(), ids, composerTasksID)
	if err != nil {
		http.Error(w, "could not load Projects", http.StatusInternalServerError)
		return
	}

	a.renderFeed(w, r, status, composer{
		Title:      r.PostForm.Get("title"),
		HappenedAt: r.PostForm.Get("happened_at"),
		Body:       r.PostForm.Get("body"),
		Name:       typedName,
		Projects:   projects,
	}, errs)
}

func composerTasksID(id int64) string {
	return fmt.Sprintf("composer-tasks-%d", id)
}

func formProjectIDs(values []string) []int64 {
	var ids []int64
	for _, value := range values {
		id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil || id <= 0 {
			continue
		}

		ids = append(ids, id)
	}

	return ids
}

func withoutID(ids []int64, drop int64) []int64 {
	kept := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id != drop {
			kept = append(kept, id)
		}
	}

	return kept
}

func containsID(ids []int64, id int64) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}

	return false
}

func (a *App) meetingPage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		a.notFound(w, "Meeting not found.")
		return
	}

	item, err := a.meetingByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		a.notFound(w, "Meeting not found.")
		return
	}
	if err != nil {
		http.Error(w, "could not load the Meeting", http.StatusInternalServerError)
		return
	}

	views, err := a.meetingViews(r.Context(), []meeting{item})
	if err != nil {
		http.Error(w, "could not load the Meeting", http.StatusInternalServerError)
		return
	}
	view := views[0]
	a.render(w, http.StatusOK, "meeting", meetingPage{
		Title:   view.Title,
		Meeting: view,
	})
}

func (a *App) notFound(w http.ResponseWriter, message string) {
	a.render(w, http.StatusNotFound, "notfound", messagePage{
		Title:   "Not found",
		Message: message,
	})
}

func (a *App) projects(w http.ResponseWriter, r *http.Request) {
	a.renderProjects(w, r, http.StatusOK, "", nil)
}

func (a *App) renderProjects(w http.ResponseWriter, r *http.Request, status int, name string, errs []string) {
	projects, err := a.listProjects(r.Context())
	if err != nil {
		http.Error(w, "could not load Projects", http.StatusInternalServerError)
		return
	}

	views := make([]projectView, 0, len(projects))
	for _, item := range projects {
		views = append(views, projectView{
			ID:   item.ID,
			Name: item.Name,
			Done: item.Done,
			Open: item.Open,
		})
	}

	a.render(w, status, "projects", projectsPage{
		Title:    "Projects",
		Name:     name,
		Errors:   errs,
		Projects: views,
	})
}

func (a *App) postProject(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "could not read the form", http.StatusBadRequest)
		return
	}

	typed := r.PostForm.Get("name")
	name := strings.TrimSpace(typed)
	if name == "" {
		a.renderProjects(w, r, http.StatusBadRequest, typed, []string{"Name is required."})
		return
	}

	taken, err := a.projectNameTaken(r.Context(), name, 0)
	if err != nil {
		http.Error(w, "could not save the Project", http.StatusInternalServerError)
		return
	}
	if taken {
		a.renderProjects(w, r, http.StatusBadRequest, typed, []string{"A Project with that name already exists."})
		return
	}

	id, err := a.insertProject(r.Context(), name)
	if err != nil {
		http.Error(w, "could not save the Project", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/projects/%d", id), http.StatusSeeOther)
}

func (a *App) projectPage(w http.ResponseWriter, r *http.Request) {
	item, ok := a.loadProject(w, r)
	if !ok {
		return
	}

	a.renderProject(w, r, http.StatusOK, item, nil, "")
}

func (a *App) postProjectName(w http.ResponseWriter, r *http.Request) {
	item, ok := a.loadProject(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "could not read the form", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.PostForm.Get("name"))
	if name == "" {
		a.finishProjectName(w, r, item, []string{"Name is required."})
		return
	}

	taken, err := a.projectNameTaken(r.Context(), name, item.ID)
	if err != nil {
		http.Error(w, "could not rename the Project", http.StatusInternalServerError)
		return
	}
	if taken {
		a.finishProjectName(w, r, item, []string{"A Project with that name already exists."})
		return
	}

	if err := a.updateProjectName(r.Context(), item.ID, name); err != nil {
		http.Error(w, "could not rename the Project", http.StatusInternalServerError)
		return
	}

	item.Name = name
	a.finishProjectName(w, r, item, nil)
}

func (a *App) finishProjectName(w http.ResponseWriter, r *http.Request, item project, errs []string) {
	if htmxRequest(r) {
		a.render(w, http.StatusOK, "project-name", projectPage{
			Title:  item.Name,
			ID:     item.ID,
			Name:   item.Name,
			Notes:  item.Notes,
			Errors: errs,
		})
		return
	}
	if len(errs) > 0 {
		a.renderProject(w, r, http.StatusBadRequest, item, errs, "")
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/projects/%d", item.ID), http.StatusSeeOther)
}

func (a *App) postProjectNotes(w http.ResponseWriter, r *http.Request) {
	item, ok := a.loadProject(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "could not read the form", http.StatusBadRequest)
		return
	}

	notes := r.PostForm.Get("notes")
	if err := a.updateProjectNotes(r.Context(), item.ID, notes); err != nil {
		http.Error(w, "could not save the notes", http.StatusInternalServerError)
		return
	}

	item.Notes = notes
	if htmxRequest(r) {
		a.render(w, http.StatusOK, "project-notes", projectPage{
			Title: item.Name,
			ID:    item.ID,
			Name:  item.Name,
			Notes: item.Notes,
		})
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/projects/%d", item.ID), http.StatusSeeOther)
}

func (a *App) projectDeletePage(w http.ResponseWriter, r *http.Request) {
	item, ok := a.loadProject(w, r)
	if !ok {
		return
	}

	a.render(w, http.StatusOK, "project-delete", projectPage{
		Title: "Delete " + item.Name,
		ID:    item.ID,
		Name:  item.Name,
	})
}

func (a *App) postProjectDelete(w http.ResponseWriter, r *http.Request) {
	item, ok := a.loadProject(w, r)
	if !ok {
		return
	}

	if err := a.deleteProject(r.Context(), item.ID); err != nil {
		http.Error(w, "could not delete the Project", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

func (a *App) loadProject(w http.ResponseWriter, r *http.Request) (project, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		a.notFound(w, "Project not found.")
		return project{}, false
	}

	item, err := a.projectByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		a.notFound(w, "Project not found.")
		return project{}, false
	}
	if err != nil {
		http.Error(w, "could not load the Project", http.StatusInternalServerError)
		return project{}, false
	}

	return item, true
}

func (a *App) postTaskDelete(w http.ResponseWriter, r *http.Request) {
	item, ok := a.loadTask(w, r)
	if !ok {
		return
	}

	if err := a.deleteTask(r.Context(), item.ID); err != nil {
		http.Error(w, "could not delete the Task", http.StatusInternalServerError)
		return
	}

	a.finishTasks(w, r, item.ProjectID, "")
}

func (a *App) postTask(w http.ResponseWriter, r *http.Request) {
	item, ok := a.loadTask(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "could not read the form", http.StatusBadRequest)
		return
	}

	if _, exists := r.PostForm["text"]; exists {
		text := singleLine(r.PostForm.Get("text"))
		if text == "" {
			a.finishTasks(w, r, item.ProjectID, "Task text is required.")
			return
		}
		if err := a.updateTaskText(r.Context(), item.ID, text); err != nil {
			http.Error(w, "could not save the Task", http.StatusInternalServerError)
			return
		}
	}
	if _, exists := r.PostForm["done"]; exists {
		if err := a.updateTaskDone(r.Context(), item.ID, taskDoneValue(r.PostForm["done"])); err != nil {
			http.Error(w, "could not save the Task", http.StatusInternalServerError)
			return
		}
	}

	a.finishTasks(w, r, item.ProjectID, "")
}

func (a *App) loadTask(w http.ResponseWriter, r *http.Request) (task, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		a.notFound(w, "Task not found.")
		return task{}, false
	}

	item, err := a.taskByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		a.notFound(w, "Task not found.")
		return task{}, false
	}
	if err != nil {
		http.Error(w, "could not load the Task", http.StatusInternalServerError)
		return task{}, false
	}

	return item, true
}

func taskDoneValue(values []string) bool {
	for _, value := range values {
		if value == "1" || value == "on" || value == "true" {
			return true
		}
	}

	return false
}

func singleLine(text string) string {
	text = strings.ReplaceAll(text, "\r\n", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")

	return strings.TrimSpace(text)
}

func (a *App) postProjectTask(w http.ResponseWriter, r *http.Request) {
	item, ok := a.loadProject(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "could not read the form", http.StatusBadRequest)
		return
	}

	text := singleLine(r.PostForm.Get("text"))
	if text == "" {
		a.finishTasks(w, r, item.ID, "Task text is required.")
		return
	}
	if err := a.insertTask(r.Context(), item.ID, text); err != nil {
		http.Error(w, "could not save the Task", http.StatusInternalServerError)
		return
	}

	a.finishTasks(w, r, item.ID, "")
}

func (a *App) finishTasks(w http.ResponseWriter, r *http.Request, projectID int64, taskErr string) {
	item, err := a.projectByID(r.Context(), projectID)
	if errors.Is(err, sql.ErrNoRows) {
		a.notFound(w, "Project not found.")
		return
	}
	if err != nil {
		http.Error(w, "could not load the Project", http.StatusInternalServerError)
		return
	}

	if htmxRequest(r) {
		a.renderProjectTemplate(w, r, http.StatusOK, "project-tasks", item, nil, taskErr, tasksTarget(r))
		return
	}
	if taskErr != "" {
		a.renderProjectTemplate(w, r, http.StatusBadRequest, "project", item, nil, taskErr, "project-tasks")
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/projects/%d", item.ID), http.StatusSeeOther)
}

func tasksTarget(r *http.Request) string {
	target := r.Header.Get("HX-Target")
	if target == "project-tasks" || composerTaskTarget(target) || meetingTaskTarget(target) {
		return target
	}

	return "project-tasks"
}

func composerTaskTarget(target string) bool {
	rest, ok := strings.CutPrefix(target, "composer-tasks-")

	return ok && decimalID(rest)
}

func meetingTaskTarget(target string) bool {
	rest, ok := strings.CutPrefix(target, "meeting-")
	if !ok {
		return false
	}
	meetingID, projectID, ok := strings.Cut(rest, "-tasks-")

	return ok && decimalID(meetingID) && decimalID(projectID)
}

func decimalID(value string) bool {
	if value == "" {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	return true
}

func (a *App) renderProject(w http.ResponseWriter, r *http.Request, status int, item project, errs []string, taskErr string) {
	a.renderProjectTemplate(w, r, status, "project", item, errs, taskErr, "project-tasks")
}

func (a *App) renderProjectTemplate(w http.ResponseWriter, r *http.Request, status int, name string, item project, errs []string, taskErr string, tasksID string) {
	tasks, err := a.listTasks(r.Context(), item.ID)
	if err != nil {
		http.Error(w, "could not load Tasks", http.StatusInternalServerError)
		return
	}

	a.render(w, status, name, projectPage{
		Title:     item.Name,
		ID:        item.ID,
		Name:      item.Name,
		Notes:     item.Notes,
		Errors:    errs,
		Tasks:     taskViews(tasks),
		TaskError: taskErr,
		TasksID:   tasksID,
	})
}

func (a *App) renderFeed(w http.ResponseWriter, r *http.Request, status int, fields composer, errs []string) {
	meetings, err := a.listMeetings(r.Context())
	if err != nil {
		http.Error(w, "could not load Meetings", http.StatusInternalServerError)
		return
	}

	projects, err := a.listProjects(r.Context())
	if err != nil {
		http.Error(w, "could not load Projects", http.StatusInternalServerError)
		return
	}
	fields.Picker = pickerProjects(projects, fields.Projects)
	views, err := a.meetingViews(r.Context(), meetings)
	if err != nil {
		http.Error(w, "could not load Meetings", http.StatusInternalServerError)
		return
	}

	a.render(w, status, "feed", feedPage{
		Title:    "Meetings",
		Errors:   errs,
		Composer: fields,
		Meetings: views,
	})
}

func (a *App) linkedProjects(ctx context.Context, ids []int64, tasksID func(int64) string) ([]linkedProject, error) {
	linked := make([]linkedProject, 0, len(ids))
	for _, id := range ids {
		item, err := a.projectByID(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}

		tasks, err := a.listTasks(ctx, item.ID)
		if err != nil {
			return nil, err
		}

		linked = append(linked, linkedProject{
			ID:      item.ID,
			Name:    item.Name,
			Tasks:   taskViews(tasks),
			TasksID: tasksID(item.ID),
		})
	}

	return linked, nil
}

func taskViews(tasks []task) []taskView {
	views := make([]taskView, 0, len(tasks))
	for _, item := range tasks {
		views = append(views, taskView{
			ID:   item.ID,
			Text: item.Text,
			Done: item.Done,
		})
	}

	return views
}

func pickerProjects(projects []project, chosen []linkedProject) []projectView {
	skip := make(map[int64]struct{}, len(chosen))
	for _, item := range chosen {
		skip[item.ID] = struct{}{}
	}

	views := make([]projectView, 0, len(projects))
	for _, item := range projects {
		if _, ok := skip[item.ID]; ok {
			continue
		}

		views = append(views, projectView{ID: item.ID, Name: item.Name})
	}
	sort.Slice(views, func(i, j int) bool {
		left := strings.ToLower(views[i].Name)
		right := strings.ToLower(views[j].Name)
		if left == right {
			return views[i].ID < views[j].ID
		}

		return left < right
	})

	return views
}

func (a *App) meetingViews(ctx context.Context, meetings []meeting) ([]meetingView, error) {
	views := make([]meetingView, 0, len(meetings))
	for _, item := range meetings {
		ids, err := a.referenceProjectIDs(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		projects, err := a.linkedProjects(ctx, ids, func(projectID int64) string {
			return fmt.Sprintf("meeting-%d-tasks-%d", item.ID, projectID)
		})
		if err != nil {
			return nil, err
		}

		views = append(views, meetingView{
			ID:         item.ID,
			Title:      item.Title,
			HappenedAt: item.HappenedAt.In(a.loc).Format(happenedAtDisplay),
			Body:       item.Body,
			Projects:   projects,
		})
	}

	return views, nil
}

func (a *App) render(w http.ResponseWriter, status int, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = a.tmpl.ExecuteTemplate(w, name, data)
}

func requireLoopback(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("invalid listen address %q: %w", addr, err)
	}
	if port == "" {
		return "", fmt.Errorf("invalid listen address %q: missing port", addr)
	}
	if host == "localhost" {
		return addr, nil
	}

	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return "", fmt.Errorf("refusing non-loopback address %s", host)
	}

	return addr, nil
}

// URL is the base URL of the running server.
func (a *App) URL() string {
	return a.url
}

// Close stops the server.
func (a *App) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := a.server.Shutdown(ctx)
	if dbErr := a.db.Close(); err == nil {
		err = dbErr
	}
	if err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	return nil
}
