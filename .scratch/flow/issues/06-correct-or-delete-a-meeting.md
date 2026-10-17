# 06: Correct or delete a Meeting

**Spec:** `.scratch/flow/spec.md`

**What to build:** The user edits a posted Meeting's title, happened-at, body, and References and saves that set together. The Task lists on a posted Meeting stay visible and live outside edit mode, so ticking a Task does not require Save. Save commits the title, happened-at, body, and Reference set. Cancel restores the saved title, happened-at, body, and Reference set. Task changes made while a list was on screen remain after Cancel. Adding a Project during edit shows its Tasks immediately, and that Reference appears on the Project's citation rule only after Save. Removing a Reference and saving drops that Meeting from the citation rule. Cancel after removing a Reference puts that Reference back. Removing the last Reference removes the rule. Changing happened-at re-sorts the feed and the citation rule. Changing the title changes the citation's link text. The body can be cleared and saved. A rejected Save leaves the stored Meeting unchanged and redisplays the edit with an error and what was typed. Delete uses a confirmation page that names the Meeting. Leaving it without confirming leaves the Meeting unchanged. Confirming removes the Meeting from the feed, makes its page gone, drops it from every citation rule, and leaves the Projects, their Tasks, and their notes.

**Blocked by:** 05 Show the citation rule on the Project

**Status:** done

- [x] The user can edit a posted Meeting's title, happened-at, body, and References and save them together. Save redirects back to that Meeting. The body can be cleared and saved.
- [x] A blank title or a missing happened-at on Save is rejected. The stored Meeting stays unchanged, and the edit redisplays with an error and the submitted fields and pending References.
- [x] Cancel restores the saved title, happened-at, body, and Reference set. Task adds, text edits, deletes, and toggles made while a list was on screen remain after Cancel.
- [x] The Task lists stay visible and live outside edit mode. Ticking a Task does not require Save.
- [x] Adding a Project during edit shows its Tasks immediately. That Reference appears on the Project's citation rule only after Save. Cancel does not cite the Meeting for that Project.
- [x] Removing a Reference and saving drops that Meeting from the Project's citation rule and removes the rule when it was the last Reference. Cancel after removing a Reference puts that Reference back.
- [x] Reference order follows add order at Save. Removing a Reference and adding it again puts that Project at the end. A Meeting still references a given Project at most once.
- [x] Changing happened-at re-sorts the feed and the citation rule. Changing the title changes the citation's link text.
- [x] The delete confirmation page names the Meeting. Leaving it without confirming leaves the Meeting unchanged. Confirming removes the Meeting from the feed, makes its page gone, removes it from every citation rule, leaves the Projects' Tasks and notes, and redirects to the feed.
- [x] Behavior is checked through the HTTP app. Cancel of a Meeting edit returns the Meeting fields and keeps the Task edits. Deleting a Meeting leaves the Project and removes the citation.

## Comments

Resolved on integration branch `feat/flow`.
