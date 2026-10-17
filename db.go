package flow

import (
	"context"
	"database/sql"
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
