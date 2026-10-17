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
	"strconv"
	"strings"
	"time"
)

//go:embed static/app.css
var stylesheet []byte

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
}

type meetingView struct {
	ID         int64
	Title      string
	HappenedAt string
	Body       string
}

type feedPage struct {
	Title    string
	Errors   []string
	Composer composer
	Meetings []meetingView
}

type projectsPage struct {
	Title string
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
	mux.HandleFunc("GET /meetings/{id}", app.meetingPage)
	mux.HandleFunc("GET /projects", app.projects)
	mux.HandleFunc("GET /styles.css", serveStyles)

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
		}, errs)
		return
	}

	if err := a.insertMeeting(r.Context(), strings.TrimSpace(title), happened, body); err != nil {
		http.Error(w, "could not save the Meeting", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
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

	view := a.meetingViews([]meeting{item})[0]
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
	a.render(w, http.StatusOK, "projects", projectsPage{Title: "Projects"})
}

func (a *App) renderFeed(w http.ResponseWriter, r *http.Request, status int, fields composer, errs []string) {
	meetings, err := a.listMeetings(r.Context())
	if err != nil {
		http.Error(w, "could not load Meetings", http.StatusInternalServerError)
		return
	}

	a.render(w, status, "feed", feedPage{
		Title:    "Meetings",
		Errors:   errs,
		Composer: fields,
		Meetings: a.meetingViews(meetings),
	})
}

func (a *App) meetingViews(meetings []meeting) []meetingView {
	views := make([]meetingView, 0, len(meetings))
	for _, item := range meetings {
		views = append(views, meetingView{
			ID:         item.ID,
			Title:      item.Title,
			HappenedAt: item.HappenedAt.In(a.loc).Format(happenedAtDisplay),
			Body:       item.Body,
		})
	}

	return views
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
