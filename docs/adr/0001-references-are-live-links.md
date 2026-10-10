---
status: accepted
---

# References are live links, not snapshots

A Reference records that a Meeting discussed a Project. It does not copy tasks or notes, and it does not freeze which tasks were done. The task list always shows the current done/open split, and the rule between those groups cites every Meeting that references the Project.

## Considered Options

- **Latest citation only.** Older meetings would vanish from the Project.
- **Per-meeting snapshots.** A task finished later would stay under the meeting where it was still open, which fights a list sorted by current state.

Checking a task updates every embed of that list. The app cannot reconstruct what was done at the time of a past meeting.
