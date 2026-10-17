package flow

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS meetings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    happened_at INTEGER NOT NULL,
    body TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE,
    notes TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS projects_name_unique ON projects(name);

CREATE TABLE IF NOT EXISTS tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    text TEXT NOT NULL,
    done INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL
);
`

func openDB(path string) (*sql.DB, error) {
	if path == "" {
		path = "flow.db"
	}

	dsn, err := sqliteDSN(path)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open database: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("configure database: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return db, nil
}

func sqliteDSN(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("database path: %w", err)
	}

	u := url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	if !strings.HasPrefix(u.Path, "/") {
		u.Path = "/" + u.Path
	}

	return u.String(), nil
}

type meeting struct {
	ID         int64
	Title      string
	HappenedAt time.Time
	Body       string
	CreatedAt  time.Time
}

func (a *App) listMeetings(ctx context.Context) ([]meeting, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT id, title, happened_at, body, created_at
		FROM meetings
		ORDER BY happened_at DESC, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var meetings []meeting
	for rows.Next() {
		var (
			item     meeting
			happened int64
			created  int64
		)
		if err := rows.Scan(&item.ID, &item.Title, &happened, &item.Body, &created); err != nil {
			return nil, err
		}

		item.HappenedAt = time.Unix(0, happened).UTC()
		item.CreatedAt = time.Unix(0, created).UTC()
		meetings = append(meetings, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return meetings, nil
}

func (a *App) meetingByID(ctx context.Context, id int64) (meeting, error) {
	var (
		item     meeting
		happened int64
		created  int64
	)
	err := a.db.QueryRowContext(ctx, `
		SELECT id, title, happened_at, body, created_at
		FROM meetings
		WHERE id = ?`, id).Scan(&item.ID, &item.Title, &happened, &item.Body, &created)
	if err != nil {
		return meeting{}, err
	}

	item.HappenedAt = time.Unix(0, happened).UTC()
	item.CreatedAt = time.Unix(0, created).UTC()

	return item, nil
}

type project struct {
	ID        int64
	Name      string
	Notes     string
	CreatedAt time.Time
	Done      int
	Open      int
}

func (a *App) listProjects(ctx context.Context) ([]project, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT p.id, p.name, p.notes, p.created_at,
			COALESCE(SUM(CASE WHEN t.done != 0 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN t.id IS NOT NULL AND t.done = 0 THEN 1 ELSE 0 END), 0)
		FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id
		GROUP BY p.id
		ORDER BY p.created_at DESC, p.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []project
	for rows.Next() {
		var (
			item    project
			created int64
		)
		if err := rows.Scan(&item.ID, &item.Name, &item.Notes, &created, &item.Done, &item.Open); err != nil {
			return nil, err
		}

		item.CreatedAt = time.Unix(0, created).UTC()
		projects = append(projects, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return projects, nil
}

func (a *App) projectByID(ctx context.Context, id int64) (project, error) {
	var (
		item    project
		created int64
	)
	err := a.db.QueryRowContext(ctx, `
		SELECT id, name, notes, created_at
		FROM projects
		WHERE id = ?`, id).Scan(&item.ID, &item.Name, &item.Notes, &created)
	if err != nil {
		return project{}, err
	}

	item.CreatedAt = time.Unix(0, created).UTC()

	return item, nil
}

func (a *App) projectNameTaken(ctx context.Context, name string, exceptID int64) (bool, error) {
	var id int64
	err := a.db.QueryRowContext(ctx, `
		SELECT id FROM projects
		WHERE name = ? AND id <> ?`, name, exceptID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (a *App) updateProjectName(ctx context.Context, id int64, name string) error {
	_, err := a.db.ExecContext(ctx, `
		UPDATE projects SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return fmt.Errorf("update project name: %w", err)
	}

	return nil
}

func (a *App) deleteProject(ctx context.Context, id int64) error {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE project_id = ?`, id); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}

	return nil
}

func (a *App) updateProjectNotes(ctx context.Context, id int64, notes string) error {
	_, err := a.db.ExecContext(ctx, `
		UPDATE projects SET notes = ? WHERE id = ?`, notes, id)
	if err != nil {
		return fmt.Errorf("update project notes: %w", err)
	}

	return nil
}

func (a *App) insertProject(ctx context.Context, name string) (int64, error) {
	res, err := a.db.ExecContext(ctx, `
		INSERT INTO projects (name, notes, created_at)
		VALUES (?, '', ?)`,
		name,
		time.Now().UnixNano(),
	)
	if err != nil {
		return 0, fmt.Errorf("insert project: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("insert project: %w", err)
	}

	return id, nil
}

type task struct {
	ID        int64
	ProjectID int64
	Text      string
	Done      bool
}

func (a *App) listTasks(ctx context.Context, projectID int64) ([]task, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT id, text, done
		FROM tasks
		WHERE project_id = ?
		ORDER BY done DESC, created_at ASC, id ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []task
	for rows.Next() {
		var (
			item task
			done int
		)
		if err := rows.Scan(&item.ID, &item.Text, &done); err != nil {
			return nil, err
		}

		item.Done = done != 0
		item.ProjectID = projectID
		tasks = append(tasks, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (a *App) taskByID(ctx context.Context, id int64) (task, error) {
	var (
		item task
		done int
	)
	err := a.db.QueryRowContext(ctx, `
		SELECT id, project_id, text, done
		FROM tasks
		WHERE id = ?`, id).Scan(&item.ID, &item.ProjectID, &item.Text, &done)
	if err != nil {
		return task{}, err
	}

	item.Done = done != 0

	return item, nil
}

func (a *App) deleteTask(ctx context.Context, id int64) error {
	_, err := a.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	return nil
}

func (a *App) updateTaskText(ctx context.Context, id int64, text string) error {
	_, err := a.db.ExecContext(ctx, `UPDATE tasks SET text = ? WHERE id = ?`, text, id)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	return nil
}

func (a *App) updateTaskDone(ctx context.Context, id int64, done bool) error {
	flag := 0
	if done {
		flag = 1
	}

	_, err := a.db.ExecContext(ctx, `UPDATE tasks SET done = ? WHERE id = ?`, flag, id)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	return nil
}

func (a *App) insertTask(ctx context.Context, projectID int64, text string) error {
	_, err := a.db.ExecContext(ctx, `
		INSERT INTO tasks (project_id, text, done, created_at)
		VALUES (?, ?, 0, ?)`,
		projectID,
		text,
		time.Now().UnixNano(),
	)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}

	return nil
}

func (a *App) insertMeeting(ctx context.Context, title string, happened time.Time, body string) error {
	_, err := a.db.ExecContext(ctx, `
		INSERT INTO meetings (title, happened_at, body, created_at)
		VALUES (?, ?, ?, ?)`,
		title,
		happened.UTC().UnixNano(),
		body,
		time.Now().UnixNano(),
	)
	if err != nil {
		return fmt.Errorf("insert meeting: %w", err)
	}

	return nil
}
