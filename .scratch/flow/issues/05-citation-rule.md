# 05: Show the citation rule on the Project

**Spec:** `.scratch/flow/spec.md`

**What to build:** When one or more Meetings reference a Project, the Project page shows a citation rule on the Task list. The rule sits between the done Tasks and the open Tasks when both groups have Tasks, above the open Tasks when none are done, below the done Tasks when all are done, and on the page when the Project has References and no Tasks. Every referencing Meeting appears once, newest happened-at first, with the later-created Meeting first when happened-at ties. Each citation shows the same absolute time format as the feed (`2 Jan 2026, 15:04`) and the Meeting title, and links to that Meeting's page, where that Project's Tasks are shown. A second Meeting that references the Project adds a second citation. A Task finished after a Meeting referenced the Project moves above the rule on the Project page and on every embedded list. The first Task on a Project that already has citations lands in the open group, below the rule. When no Meeting references the Project, the rule is absent. The divider above notes is only a separator.

**Blocked by:** 04 Reference a Project from a Meeting

**Status:** done

- [x] No citation rule is shown when no Meeting references the Project.
- [x] With References and both done and open Tasks, the rule sits between the two groups. With only open Tasks, the rule sits above them. With only done Tasks, the rule sits below them. With References and no Tasks, the rule is still shown.
- [x] Every referencing Meeting appears once, newest happened-at first, then later-created Meeting first. Each citation shows the formatted time and the Meeting title and links to that Meeting's page. Opening the citation shows that Project's Tasks.
- [x] A second Meeting that references the Project adds a second citation.
- [x] A Task finished after a Reference moves above the rule on the Project page and on every embedded list. The first Task on a Project that already has citations lands in the open group below the rule.
- [x] The divider above notes is only a separator and does not list Meetings.
- [x] Citation entries expose `data-citation`. Tests assert on those hooks, on `data-task`, and on the visible text, including the order of Tasks relative to the rule. They do not assert on CSS classes, stylesheet contents, or SQL.
- [x] Behavior is checked through the HTTP app against a temporary SQLite file.

## Comments

Resolved on integration branch `feat/flow`.
