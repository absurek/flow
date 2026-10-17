# 03: Check off Tasks on the Project page

**Spec:** `.scratch/flow/spec.md`

**What to build:** The Project page shows that Project's Tasks. Done Tasks come first, then open Tasks. Within each group, the oldest created Task comes first. The user adds a Task, edits its text by leaving the field, marks it done or open, and deletes it without a confirmation step. A new Task lands at the bottom of the open group. Marking a Task open returns it to its original creation-time position in the open group. A blank task-text edit keeps the previous text. A Task is a single line. The row shows the checkbox and the text. There is no control for dragging Tasks. Deleting a Task updates the done and open counts on the Projects list. The citation rule stays absent until a Meeting references the Project.

**Blocked by:** 02 Open a Project from the Projects list

**Status:** done

- [x] The Project page lists that Project's Tasks, done first and open after, oldest created first within each group. An older done Task stays above a newer done Task. An older open Task stays above a newer open Task.
- [x] Adding a Task appends it to the open group. The user can add the first Task on a Project that has none.
- [x] Leaving the text field saves the Task text. A blank edit keeps the previous text. A Task is a single line. The row shows the checkbox and the text, and exposes `data-task`. Creation time is not shown on the row and does not change when the text or the done/open flag changes.
- [x] Marking a Task done moves it into the done group. Marking it open returns it to the open group at its original creation-time position.
- [x] Deleting a Task has no confirmation page. The Task disappears, and the done and open counts on the Projects list update.
- [x] There is no control for dragging Tasks.
- [x] Task text that looks like HTML is shown as text.
- [x] Add, edit, toggle, and delete update the affected region in place. A normal request to the same action still ends on a page that shows the resulting state.
- [x] Behavior is checked through the HTTP app against a temporary SQLite file, asserting on visible text and the order of Task rows. Tests do not open the database or call an internal repository to prove the sort.

## Comments

Resolved on integration branch `feat/flow`.
