# Flow

Status: ready-for-agent

## Problem Statement

The user wants one local place to write down meetings and to track the projects discussed in them. Meeting notes, project tasks, and the fact that a meeting discussed a project are easy to lose when they live in separate documents. The user wants a timeline of their own meetings, a list of projects with checkable tasks and notes, and a visible link from each project back to the meetings that discussed it.

## Solution

Flow is a single-user web app on localhost. The Meetings feed is a timeline of the user's own posts. Each Meeting can reference one or more Projects and shows those Projects' live tasks. The Projects area lists every Project and opens one page per Project: tasks split into done and open, a citation rule between those groups naming every Meeting that references the Project, and one notes field.

A Reference is a live link. It does not copy tasks or notes, and it does not freeze which tasks were done. Checking a task updates every place that Project's tasks appear. The app cannot reconstruct what was done at the time of a past Meeting (ADR-0001).

## User Stories

### Meetings feed

1. As the user, I want a Meetings feed of only my Meetings, so that the timeline is my own record.
2. As the user, I want every Meeting on that one page, so that I can scroll the whole timeline.
3. As the user, I want the feed ordered by happened-at, newest first, so that the latest meeting is at the top.
4. As the user, I want Meetings that share a happened-at ordered with the later-created Meeting first, so that the order is stable.
5. As the user, I want an empty feed to show the composer and no Meeting cards, so that I can write the first Meeting.
6. As the user, I want each feed card to show the title, the happened-at, and the full body, so that I can read a Meeting without opening another page.
7. As the user, I want happened-at shown as an absolute time in the machine's timezone, formatted like `2 Jan 2026, 15:04`, so that the time is unambiguous.
8. As the user, I want the same time format on the feed, the Meeting page, and the citation rule, so that a time means the same thing everywhere.
9. As the user, I want each Meeting to have its own page with the same card as the feed, so that a citation can link to it.
10. As the user, I want a nav link to Meetings and a nav link to Projects, so that I can move between the two areas.

### Posting a Meeting

11. As the user, I want a composer at the top of the feed, so that I can post a Meeting the way I post on a timeline.
12. As the user, I want to set a title, a happened-at, and a body, so that the Meeting records what happened and when.
13. As the user, I want happened-at to default to the current local time, so that a Meeting I am in is dated without typing.
14. As the user, I want to set happened-at in the past, so that I can log a Meeting after it happened.
15. As the user, I want to set happened-at in the future, so that a future time sorts to the top when it is the newest.
16. As the user, I want to post a Meeting with an empty body, so that a title and a time are enough.
17. As the user, I want a blank title rejected, so that every Meeting has a title.
18. As the user, I want a missing happened-at rejected, so that every Meeting has a time.
19. As the user, I want a rejected Post to create no Meeting, so that a failed submit stays off the feed.
20. As the user, I want a rejected Post to redisplay the composer with an error and what I typed, so that I can correct it.
21. As the user, I want a successful Post to clear the composer, so that I can write the next Meeting.
22. As the user, I want the new Meeting to appear in happened-at order, so that a backdated Meeting lands in the middle of the feed when that is its place.
23. As the user, I want line breaks in the body preserved, so that the text keeps its shape.
24. As the user, I want body text that looks like HTML shown as text, so that the page does not treat it as markup.
25. As the user, I want two Meetings to be allowed the same title, so that titles do not have to be unique.
26. As the user, I want to post a Meeting with no Projects, so that a Meeting can stand on its own.
27. As the user, I want refreshing or leaving the composer before Post to save no Meeting, so that an abandoned form is not a draft.

### References while composing

28. As the user, I want to pick a Project while composing, so that the Meeting can record that we discussed it.
29. As the user, I want the picker in alphabetical order, ignoring case, so that I can find a name.
30. As the user, I want the picker to hide Projects already chosen on this Meeting, so that I reference each Project once.
31. As the user, I want to create a Project by name from the picker, so that I can start a Project without leaving the composer.
32. As the user, I want that Project saved as soon as I create it, so that it exists even if I never Post.
33. As the user, I want the picked Project's current tasks shown on the composer before Post, so that I can see the work during the Meeting.
34. As the user, I want to add, edit, delete, and toggle tasks from that composer list, so that I can update the Project while writing.
35. As the user, I want those task changes kept if I leave without posting, so that Project work is not tied to the unposted Meeting.
36. As the user, I want leaving without posting to create no Reference, so that the Project does not cite a Meeting I did not keep.
37. As the user, I want to remove a pending Reference before Post, so that a mistaken pick is dropped.
38. As the user, I want removing that pending Reference to leave the Project and its tasks in place, so that dropping a link does not delete work.
39. As the user, I want several References on one Meeting, so that one conversation can cover more than one Project.
40. As the user, I want those lists in the order I added the References, so that the card follows the conversation.
41. As the user, I want a Project I remove and add again to land at the end, so that add-order stays the rule.

### Editing and deleting a Meeting

42. As the user, I want to edit a posted Meeting's title, happened-at, body, and References, and save them together, so that I can correct a post.
43. As the user, I want the task list on a posted Meeting to stay visible and live outside edit mode, so that ticking a task does not require Save.
44. As the user, I want Save to commit the title, happened-at, body, and Reference set, so that those fields change only when I commit.
45. As the user, I want Cancel to restore the saved title, happened-at, body, and Reference set, so that I can abandon an edit.
46. As the user, I want task changes made while a list was on screen to remain after Cancel, so that task edits are not part of the Meeting draft.
47. As the user, I want adding a Project during edit to show its tasks immediately, so that I can update them before I save.
48. As the user, I want that new Reference to appear on the Project only after Save, so that a cancelled edit does not cite the Meeting.
49. As the user, I want removing a Reference and saving to drop that Meeting from the Project's citation rule, so that the rule matches the saved link.
50. As the user, I want Cancel after removing a Reference to put that Reference back, so that an uncommitted removal is reversible.
51. As the user, I want changing happened-at to re-sort the feed and the citation rule, so that both follow the Meeting's time.
52. As the user, I want changing the title to change the citation's link text, so that the Project names the Meeting by its current title.
53. As the user, I want to clear a Meeting body and save, so that I can remove the text.
54. As the user, I want a confirmation step before a Meeting is deleted, so that one misclick does not remove it.
55. As the user, I want declining that confirmation to leave the Meeting unchanged, so that I can back out.
56. As the user, I want confirming deletion to remove the Meeting from the feed and to make its page gone, so that it cannot be opened.
57. As the user, I want the Project's tasks and notes to remain after the Meeting is deleted, so that deleting a Meeting does not delete the work.
58. As the user, I want the deleted Meeting removed from every citation rule, so that the rule cites only live Meetings.

### Tasks on a Meeting

59. As the user, I want each referenced Project's name on the Meeting to link to the Project page, so that I can open it.
60. As the user, I want the Meeting to show that Project's tasks, done first and open after, oldest created first within each group, so that the list matches the Project page.
61. As the user, I want an add-task field on each of those lists, so that I can append a task from the Meeting.
62. As the user, I want a task I add on the Meeting to appear on the Project page, so that there is one list.
63. As the user, I want a task text I edit on the Meeting to show on the Project page, so that the text has one current value.
64. As the user, I want a task I delete on the Meeting to disappear on the Project page, so that delete applies everywhere.
65. As the user, I want deleting a task to happen without a confirmation step, so that the list stays quick to edit.
66. As the user, I want marking a task done on the Meeting to move it into the done group on the Meeting and on the Project page, so that both views stay current.
67. As the user, I want marking a task open to return it to the open group at its original creation-time position, so that unchecking does not make it the newest task.
68. As the user, I want the Meeting list to omit the citation rule, the notes field, and rename, so that those belong to the Project page.
69. As the user, I want a referenced Project with no tasks to show its name and an add-task field, so that I can add the first task from the Meeting.
70. As the user, I want toggling a task on one Project to leave another Project's tasks on the same Meeting unchanged, so that each list is its own.

### Projects list

71. As the user, I want a Projects list of every Project, newest created first, so that the Project I just added is at the top.
72. As the user, I want each row to show the name, the done count, and the open count, so that I can see progress at a glance.
73. As the user, I want a Project with no tasks to show zero done and zero open, so that empty work is visible.
74. As the user, I want to create a Project with a name and land on its page, so that I can add tasks next.
75. As the user, I want a blank Project name rejected, so that every Project has a name.
76. As the user, I want a name that matches an existing Project, ignoring case and surrounding spaces, rejected, so that the picker is unambiguous.
77. As the user, I want a rejected create to leave no extra Project, so that a failed submit is clean.
78. As the user, I want to start a delete from the list, so that I can remove a Project without opening it.
79. As the user, I want renaming a Project to keep its place in the newest-first list, so that creation order does not change when the name changes.

### Project page

80. As the user, I want the Project page to show the name, the tasks, the citation rule, a divider, and one notes field, so that the page matches the agreed sketch.
81. As the user, I want to rename the Project by leaving the name field, so that the name saves without a separate button.
82. As the user, I want the new name on the Projects list and on every Meeting that references it, so that the name has one current value.
83. As the user, I want a blank or duplicate rename to keep the previous name and show an error, so that a bad edit does not wipe the name.
84. As the user, I want the stored name to be the trimmed name, so that surrounding spaces are not part of the identity.
85. As the user, I want to add a task on the Project page, so that I can grow the list there.
86. As the user, I want a new task at the bottom of the open group, so that open tasks read in the order I added them.
87. As the user, I want to edit a task's text by leaving the field, so that the text saves without a separate button.
88. As the user, I want a blank task-text edit to keep the previous text, so that a task cannot become empty.
89. As the user, I want a task to be a single line, so that the list stays a checklist.
90. As the user, I want to mark a task done and see it move into the done group, above the citation rule, so that done work sits together.
91. As the user, I want to mark a task open and see it return among the open tasks by creation time, so that state and age both drive the order.
92. As the user, I want an older done task to stay above a newer done task, so that creation order holds inside the done group.
93. As the user, I want an older open task to stay above a newer open task, so that creation order holds inside the open group.
94. As the user, I want no control for dragging tasks, so that I do not fight the fixed sort.
95. As the user, I want a task row to show the checkbox and the text, so that creation time stays off the row.
96. As the user, I want deleting a task on the Project page to remove it everywhere and to update the counts, so that the list and the index agree.
97. As the user, I want notes to persist as I edit them, including when I clear them, so that the field is a scratch pad without a save button.
98. As the user, I want notes to keep line breaks and to render as text, so that the scratch pad is plain text.
99. As the user, I want a confirmation step before a Project is deleted from its page, so that one misclick does not remove it.
100. As the user, I want declining that confirmation to leave the Project unchanged, so that I can back out.
101. As the user, I want confirming deletion to remove the Project and its tasks and notes, so that the work is gone.
102. As the user, I want Meetings that referenced it to keep their title and body and to lose that checklist, so that the Meeting text survives.
103. As the user, I want a later Project with the same name as a deleted one to start with no tasks, no notes, and no References, so that the new Project is a different one.

### Citation rule

104. As the user, I want no citation rule when no Meeting references the Project, so that the rule appears only for real links.
105. As the user, I want the rule between the done tasks and the open tasks when both groups have tasks, so that it marks the split.
106. As the user, I want the rule above the open tasks when none are done, so that the split is still visible.
107. As the user, I want the rule below the done tasks when all are done, so that the split is still visible.
108. As the user, I want the rule to show when the Project has References and no tasks, so that a discussion is recorded before any task exists.
109. As the user, I want every referencing Meeting on the rule, newest happened-at first, so that the latest discussion is first.
110. As the user, I want citations that share a happened-at ordered with the later-created Meeting first, so that the rule uses the same order as the feed.
111. As the user, I want each citation to show the formatted time and the Meeting title, linking to that Meeting's page, so that I can open the Meeting.
112. As the user, I want a Meeting that references the Project once to appear once, so that a duplicate pick cannot double the citation.
113. As the user, I want a second Meeting that references the Project to add a second citation, so that each discussion is listed.
114. As the user, I want a task I finish after a Meeting referenced the Project to move above the rule on the Project page and on every embedded list, so that the split follows current state.
115. As the user, I want the first task on a Project that already has citations to land in the open group below the rule, so that new work starts open.
116. As the user, I want removing the last Reference to remove the rule, so that an uncited Project has no rule.
117. As the user, I want the divider above notes to be only a separator, so that it is not a second list of Meetings.
118. As the user, I want opening a citation to land on the Meeting page, including that Project's tasks, so that the link returns to the discussion.

### Local use

119. As the user, I want to open the app on localhost with no login, so that starting it is enough.
120. As the user, I want the server to refuse non-loopback connections, so that the record stays on this machine.
121. As the user, I want a dark, plain layout, so that the pages are readable and quiet.
122. As the user, I want my Meetings, Projects, tasks, notes, and References to still be there after a restart, so that the SQLite file is the record.

## Implementation Decisions

- The app is one Go process. Pages are server-rendered HTML. Interactions use HTMX. The process serves its own stylesheet. There is no second frontend server and no component library. The look is a dark theme built with Tailwind utility classes: simple cards, readable type, no custom illustration.
- The process listens on the loopback interface only. The default port is 8080. An environment variable or flag can override the port. The default bind address stays on loopback.
- Persistence is one SQLite file in the working tree, created on startup if missing, and listed in gitignore. There is no seed data. Schema is created at startup.
- A Project has a stable id, a trimmed display name, notes, and a creation time. Names are unique under case-insensitive comparison of the trimmed name. Notes are plain text and may be empty. Creation time does not change on rename.
- A Task has a stable id, belongs to one Project, has single-line text, a done/open flag, and a creation time. Creation time is set once and does not change when the text or the flag changes. The app stores no completion time, due date, assignee, priority, or manual position. Sort is applied when rendering: done tasks first, then open tasks; within each group, oldest creation time first.
- A Meeting has a stable id, a title, a happened-at time, a body, and a creation time. Title and happened-at are required. The body is optional plain text. Titles are not unique. Creation time does not change on edit. Happened-at is stored as an absolute instant and displayed in the machine's local timezone as `2 Jan 2026, 15:04` (day, short month, year, 24-hour time). The default happened-at for a new composer is the server's current local time. Any timestamp, past or future, is valid.
- A Reference links one Meeting to one Project and stores a position. It stores no task text, no notes, and no copy of which tasks were done. A Meeting references a given Project at most once; the server enforces that even if a request repeats the pair. Position follows add order at the moment the Meeting is posted or saved. Removing a Reference and adding it again assigns a new end position. Deleting a Meeting deletes its References and leaves the Projects. Deleting a Project deletes its tasks, notes, and References and leaves the Meetings' titles and bodies.
- Feed order and citation order are happened-at descending, then Meeting creation time descending. The Projects index is Project creation time descending.
- The picker lists Projects this Meeting does not already reference, in case-insensitive alphabetical order, plus a way to create a Project by name. Projects already in the pending set are omitted too. Creating a Project from the picker persists the Project immediately and adds it to the pending Reference set.
- Pending References and pending Meeting fields live in the composer or the edit form until Post or Save. Task adds, text edits, deletes, and toggles on a visible list persist immediately to the Project, including when the Meeting is never posted or the edit is cancelled.
- Routes, as form posts and page gets:
  - `GET /` is the feed and the composer.
  - `POST /meetings` posts a Meeting, then redirects to `/`.
  - `GET /meetings/{id}` is the Meeting page.
  - `POST /meetings/{id}` saves an edit, then redirects to that Meeting.
  - `GET /meetings/{id}/delete` is the Meeting confirmation page. `POST /meetings/{id}/delete` deletes it, then redirects to `/`.
  - `GET /projects` is the index.
  - `POST /projects` creates a Project, then redirects to its page.
  - `GET /projects/{id}` is the Project page.
  - `POST /projects/{id}/name` renames it.
  - `POST /projects/{id}/notes` saves notes.
  - `GET /projects/{id}/delete` is the Project confirmation page. `POST /projects/{id}/delete` deletes it, then redirects to `/projects`.
  - `POST /projects/{id}/tasks` adds a task.
  - `POST /tasks/{id}` updates task text or the done/open flag.
  - `POST /tasks/{id}/delete` deletes a task.
- Post, Save, create-Project, and both deletes use Post/Redirect/Get. Rename, notes, and task add, edit, toggle, and delete are HTMX requests that replace the affected region in place. A normal request to those task and notes URLs still ends on a page that shows the resulting state.
- Confirmation is an HTML page that names the Meeting or Project and posts the real delete. It is not a browser-native dialog, so the HTTP seam can perform it. Task delete has no confirmation page.
- Validation failures for Post redisplay the composer with an error and the submitted fields and pending References, and they do not create a Meeting. Projects created from the picker and task edits already saved stay saved. A rename or task-text failure keeps the last stored value, shows an error, and does not write the rejected value. A duplicate or blank Project create stays on the form and does not insert a row.
- Plain text is escaped when rendered. The body, notes, titles, Project names, and task text are not parsed as markdown or HTML.
- The nav's Meetings link goes to `/`. The Projects link goes to `/projects`.
- A missing Meeting or Project id responds as not found.
- One local user is assumed. Concurrent writes use last write wins. There is no account, session, or CSRF story beyond ordinary same-origin form posts on localhost.
- Code follows the repo's Go coding standards and `gofmt`.
- Domain names in code and tests use the glossary: Meeting, Project, Task, Reference. Avoid calling a Project an issue, a todo, or a ticket, and avoid treating a Reference as a snapshot, a mention, a tag, or an attachment.

## Testing Decisions

- One seam: the HTTP app. Tests start the server against a temporary SQLite file and drive it with GET and POST the way the browser would, following redirects.
- Assert on the HTML a person would read: visible text, link targets, and the order of tasks relative to the citation rule. Task rows expose `data-task`. Citation entries expose `data-citation`. Tests use those hooks and the visible text. They do not assert on CSS classes, stylesheet contents, or SQL.
- A good test checks behavior through that HTTP seam. It does not open the database to prove a sort, and it does not call an internal repository or domain function. Handlers, templates, and SQL stay behind the seam. There is no second test suite.
- Prior art: the repo has no tests yet. These HTTP tests are the first suite.
- The suite should cover the behaviors a lower seam could get wrong: happened-at order and its creation-time tie-break; task grouping and creation order, including unchecking back into the open group; the citation rule's presence, placement, and order; a task finished after a Reference moving above the rule on the Project page and on the Meeting; abandoning the composer after task edits (Project and tasks remain, no Meeting and no Reference); Cancel of a Meeting edit (Meeting fields return, task edits remain); deleting a Project (Meeting text remains, checklist and citation go); deleting a Meeting (Project remains, citation goes); case-insensitive trimmed name uniqueness; a recreated name after delete being a new empty Project; plain-text escaping; confirmation required before Meeting and Project delete, and not before task delete.

## Out of Scope

- Accounts, passwords, multiple users, and sharing.
- Binding a non-loopback interface.
- Search, filters, archive, and pagination.
- Manual reorder of tasks or References.
- Due dates, assignees, priorities, and completion timestamps.
- Markdown, rich text, and file attachments.
- Export, import, and sync.
- A light theme or a theme switch.
- Reconstructing which tasks were done at the time of a past Meeting.
- A draft Meeting that survives refresh.

## Further Notes

- The glossary is the source for the words Meeting, Project, Task, and Reference. ADR-0001 is the source for the live-link decision. This spec is the source for the behavior built on top of them.
- The lower rule in the original sketch, above the notes field, is only a visual divider. It is not a citation.
- The database file is gitignored. The spec and later tickets under this feature directory are tracked with the repo.
