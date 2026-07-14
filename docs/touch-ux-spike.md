# Milestone 0 — Touch UX Spike

This spike validates CodeRoam's highest-risk assumption: CodeMirror 6 and xterm.js embedded in
Flutter WebViews can feel deliberately touch-first rather than merely functional.

## Required physical devices

- iPhone.
- iPad.
- Android phone.
- Android tablet.
- At least one tablet test with a physical keyboard and pointer.
- At least one split-screen/multitasking test.

## Editor test script

1. Place the cursor repeatedly near punctuation and line boundaries.
2. Long-press words and drag both selection handles.
3. Select and copy multiple lines.
4. Paste multiline code with indentation.
5. Enter Turkish characters and use an IME composition keyboard.
6. Open and close the software keyboard repeatedly.
7. Rotate and resize the app while a selection and completion popup are active.
8. Search a 10,000+ line fixture.
9. Use completion, diagnostics, and code actions by touch.
10. Switch focus between Flutter controls and WebView.
11. Repeat with keyboard and pointer.

## Terminal test script

1. Type rapidly with software and hardware keyboards.
2. Use the contextual developer key row.
3. Send Ctrl+C, Ctrl+D, Esc, Tab, arrows, and bracket characters.
4. Select and copy terminal output.
5. Stream rapid output while scrolling.
6. Resize with the keyboard open.
7. Switch full-screen and split layouts.
8. Simulate duplicated and delayed bridge messages.

## Deliverable

Record results in `docs/ux-spike-results.md` with:

- device and OS,
- pass/fail for each behavior,
- videos/screenshots stored outside the repository when sensitive,
- bridge or overlay changes required,
- go/no-go decision for the selected editor and terminal surfaces.
