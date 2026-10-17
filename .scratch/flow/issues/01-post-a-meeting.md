# 01: Post a Meeting onto the local feed

**Spec:** `.scratch/flow/spec.md`

**What to build:** On localhost, with no login, the user gets a dark Meetings feed of their own Meetings, newest happened-at first, with the later-created Meeting first when times tie. Each card shows the title, the absolute local time (`2 Jan 2026, 15:04`), and the full body, and each Meeting has its own page with the same card. The composer posts a title, a happened-at (defaulting to now; past or future allowed), and an optional body. A rejected Post leaves no Meeting and redisplays what was typed. A successful Post clears the composer and places the Meeting in order. Leaving or refreshing before Post saves nothing. The same Meetings are there after a restart, and the server refuses non-loopback connections. The nav reaches Meetings and an empty Projects list.

**Blocked by:** None (can start immediately)

**Status:** done

- [x] The app opens on localhost with no login. The process listens on the loopback interface only and refuses a non-loopback connection. The default port is 8080. An environment variable or flag can override the port. The bind address stays on loopback.
- [x] The record is one SQLite file in the working tree, created on startup if missing, with no seed data, and listed in gitignore. The schema is created at startup. Meetings are still there after a restart.
- [x] The layout is dark and plain. The nav's Meetings link goes to `/`. The Projects link goes to `/projects`, which shows an empty Projects list.
- [x] The feed is one page of every Meeting, ordered by happened-at descending, then Meeting creation time descending. An empty feed shows the composer and no Meeting cards.
- [x] Each feed card shows the title, the happened-at, and the full body. Happened-at is an absolute time in the machine's timezone, formatted like `2 Jan 2026, 15:04`. The Meeting page shows the same card and the same time format. A missing Meeting id responds as not found.
- [x] The composer at the top of the feed takes a title, a happened-at, and a body. Happened-at defaults to the server's current local time. Past and future timestamps are valid. An empty body is allowed. Titles are not unique. A Meeting can be posted with no Projects.
- [x] A blank title or a missing happened-at is rejected. The rejected Post creates no Meeting and redisplays the composer with an error and the submitted fields. A successful Post clears the composer, redirects to `/`, and shows the new Meeting in happened-at order, including a backdated Meeting in the middle of the feed.
- [x] Line breaks in the body are preserved. Titles and body text that look like HTML are shown as text.
- [x] Refreshing or leaving the composer before Post saves no Meeting.
- [x] Behavior is checked through the HTTP app against a temporary SQLite file, following redirects and asserting on visible text and link targets. Domain names in code and tests are Meeting, Project, Task, and Reference. Code follows the repo's Go coding standards and `gofmt`.

## Comments

Resolved on integration branch `feat/flow`.
