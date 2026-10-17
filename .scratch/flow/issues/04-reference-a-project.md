# 04: Reference a Project from a Meeting

**Spec:** `.scratch/flow/spec.md`

**What to build:** While composing a Meeting, the user picks a Project or creates one by name. The picker lists Projects this Meeting does not already reference, in case-insensitive alphabetical order, and hides Projects already in the pending set. Creating a Project from the picker saves it immediately and adds it to the pending Reference set, even if the Meeting is never posted. The picked Project's current Tasks appear before Post. Add, edit, delete, and toggle on that list persist immediately and remain if the user never Posts. Leaving without Posting creates no Meeting and no Reference. The user can remove a pending Reference and leave the Project and its Tasks in place. Several References stay in the order they were added. A Project removed and added again goes to the end. Posting with no Projects is still allowed. A rejected Post still saves no Meeting and no Reference, and it redisplays the composer with the submitted fields and the pending References.

A posted Meeting shows each referenced Project's name, linking to the Project page, and that Project's live Tasks in the same grouping as the Project page, with an add-task field on each list. A Project with no Tasks still shows its name and an add-task field. Task changes on the Meeting are the same Tasks on the Project page. Toggling one Project leaves another Project's Tasks on the same Meeting unchanged. The Meeting omits the citation rule, the notes field, and rename.

Deleting a Project removes its Tasks, notes, and References. Meetings that referenced it keep their title and body and lose that checklist. A later Project with the same name as a deleted one starts with no Tasks, no notes, and no References. A Meeting references a given Project at most once.

**Blocked by:** 03 Check off Tasks on the Project page

**Status:** done

- [x] The composer picker lists Projects not already chosen, in alphabetical order ignoring case, and omits Projects already in the pending set.
- [x] Creating a Project by name from the picker saves that Project immediately, adds it to the pending Reference set, and shows its current Tasks. The Project remains if the user never Posts.
- [x] Adding, editing, deleting, and toggling Tasks on a visible composer list persists immediately. Those Task changes remain if the user leaves without Posting. Leaving without Posting creates no Meeting and no Reference.
- [x] The user can remove a pending Reference before Post. The Project and its Tasks stay. Several References on one Meeting appear in the order they were added. A Project removed and added again lands at the end. Posting with no Projects is allowed.
- [x] A rejected Post creates no Meeting and no Reference. It redisplays the composer with an error, the submitted fields, and the pending References. Projects created from the picker, and Task edits already saved, stay saved.
- [x] A posted Meeting references each chosen Project once. A repeated Meeting–Project pair is still a single Reference.
- [x] On the feed and the Meeting page, each referenced Project's name links to the Project page. The Meeting shows that Project's Tasks, done first and open after, oldest created first within each group, plus an add-task field. A referenced Project with no Tasks shows its name and an add-task field.
- [x] A Task added, edited, deleted, or toggled on the Meeting is the same Task on the Project page. Marking a Task done or open moves it between groups on both pages, and marking it open returns it to its original creation-time position. Deleting a Task has no confirmation step. Toggling a Task on one Project leaves another Project's Tasks on the same Meeting unchanged.
- [x] The Meeting omits the citation rule, the notes field, and rename.
- [x] Confirming deletion of a Project removes its Tasks, notes, and References and returns to the Projects list. Meetings that referenced it keep their title and body and lose that checklist.
- [x] A later Project with the same name as a deleted one starts with no Tasks, no notes, and no References.
- [x] Project names and Task text that look like HTML are shown as text.
- [x] Behavior is checked through the HTTP app. Abandoning the composer after Task edits leaves the Project and Tasks, and leaves no Meeting and no Reference.

## Comments

Resolved on integration branch `feat/flow`.
