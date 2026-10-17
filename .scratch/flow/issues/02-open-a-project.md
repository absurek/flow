# 02: Open a Project from the Projects list

**Spec:** `.scratch/flow/spec.md`

**What to build:** The user creates a Project by name and lands on its page. The list shows every Project, newest created first, with done and open counts. With no Tasks yet, both counts are zero. The page shows the name, a divider, and one notes field. The citation rule is absent, because no Meeting references the Project. Rename saves when the name field is left. Notes persist as they are edited, including when cleared. A blank or duplicate rename keeps the previous name and shows an error. The stored name is the trimmed name. Delete uses a confirmation page that names the Project. Leaving that page without confirming leaves the Project unchanged. Confirming removes the Project and its notes and returns to the Projects list. Renaming does not change the Project's place in the list. A later Project may reuse a deleted name.

**Blocked by:** 01 Post a Meeting onto the local feed

**Status:** done

- [x] The Projects list shows every Project, newest created first. Each row shows the name, the done count, and the open count. A Project with no Tasks shows zero done and zero open.
- [x] Creating a Project with a name redirects to its page. A blank name is rejected. A name that matches an existing Project, ignoring case and surrounding spaces, is rejected. A rejected create inserts no Project and stays on the form with an error.
- [x] The stored name is the trimmed name. Names are unique under case-insensitive comparison of the trimmed name.
- [x] The Project page shows the name, a divider, and one notes field. No citation rule is shown.
- [x] Leaving the name field renames the Project, and the new name appears on the Projects list. A blank or duplicate rename keeps the previous name, shows an error, and does not write the rejected value. Rename does not change list order.
- [x] Notes persist as the user edits them, including when cleared. Notes keep line breaks and render as text. Project names and notes that look like HTML are shown as text.
- [x] Rename and notes update the affected region in place. A normal request to the same action still ends on a page that shows the resulting state.
- [x] The delete confirmation page names the Project. Leaving it without confirming leaves the Project unchanged. Confirming removes the Project and its notes and redirects to the Projects list.
- [x] A missing Project id responds as not found.
- [x] A later Project with the same name as a deleted one is a different Project and starts with no notes.
- [x] Behavior is checked through the HTTP app against a temporary SQLite file, following redirects and asserting on visible text and link targets.

## Comments

Resolved on integration branch `feat/flow`.
