# MyGo child window swallows clicks on another display (macOS, separate Spaces)

Minimal repro for `github.com/egoist/mygo` **v0.5.5**, native UI only (no web
view, no StateKey, no modal, no sheet).

## Environment

- macOS with two displays and System Settings → Desktop & Dock →
  **"Displays have separate Spaces" on**.
- Original report: display 1 (primary) = `(0,0 1512×945)`, display 2 =
  `(1512,−456 1920×1080)` (MyGo / CGWindow global coordinates: top-left of
  the primary display, y down).

## Build and run

```sh
go run .                     # or: go build -o straddle-repro . && ./straddle-repro
go run . --boundary 1512     # x where display 2 starts (primary width), for the forced-straddle button
```

The **Parent** window (1512×915) lists the displays and the live frame of
every window it opened, flagging any frame that **STRADDLES** two displays.

| Button | Window | `Parent` | Size |
|---|---|---|---|
| Open child A (Parent set) | Child A | Parent | 980×760 |
| Open child A' (no Parent) | Child A' | – | 980×760 |
| Open window B (Parent set) | Window B (click counter) | Parent | 760×820 |
| Open window B' (no Parent) | Window B' (click counter) | – | 760×820 |
| Open child A straddling | Child A at `(boundary−330, −241)` | Parent | 980×760 |

A/A' and B/B' are identical except for `WindowOptions.Parent`. A window
with `Parent` is attached with `addChildWindow:ordered:NSWindowAbove`
(`internal/darwin/window.go`); with `X == Y == 0` it is placed by
`[NSWindow center]`.

## Steps

### 1. Child window dragged to the other display (Parent set)

1. Move **Parent** onto display 2, near the boundary with display 1.
2. Click **Open child A (Parent set)**. Note its frame in Parent.
3. Drag **Child A** (by its title bar) fully or partly onto display 1.
   Note: does it disappear from display 1, stay drawn there, or snap back?
   Note its frame in Parent (STRADDLES?).
4. Click **Open window B' (no Parent)** (and repeat with **B (Parent set)**),
   drag it onto display 1 so its **Click me** button lies inside the area
   Child A covers (or covered, if it was not drawn there).
5. Click **Click me** inside that area, and B's title bar there.

   **Observed:** the count does not change and the title bar does not
   respond — the clicks go to Child A's frame although Child A is not drawn
   on display 1. Outside the overlap the clicks work.
6. Move **Child A** fully onto one display: clicks on B work immediately.

### 2. Same without Parent (control)

Repeat 1 with **Open child A' (no Parent)**. **Expected:** dragging A' to
display 1 moves it to that display's Space (drawn there or not, it no
longer captures clicks for an invisible region); B's counter keeps working.

### 3. Sibling ordering

1. Open **Child A (Parent set)**, then **Window B (Parent set)**; overlap
   them, then click B's title bar and body, and the reverse (A above B).
   Does the clicked sibling come to the front? Child windows keep their
   attach order, so the later-attached one may always stay above;
   `AXRaise` on the other one does nothing.
2. Repeat with A' / B' (no Parent) for comparison.

### Forced straddle (secondary)

**Open child A straddling** creates Child A (Parent set) at
`(boundary−330, −241 980×760)`, the frame from the original report, so it
straddles the boundary wherever Parent is. Then do steps 1.4–1.6.

## What to collect

- The frames from the Parent window (or `osascript -e 'tell application
  "System Events" to get {position, size} of every window of (first process
  whose name is "mygo-straddle-repro")'`).
- Which window the window server hit-tests at a point, with the bundled
  `hit.swift` (CGWindowListCopyWindowInfo, on-screen only, layer 0,
  front to back; window names need Screen Recording permission for the
  terminal):

  ```sh
  swift hit.swift 1300 300 1450 100     # pairs of global x y points in the overlap
  ```

  In the bug, the point inside B's button reports `… / Child A` while Child
  A is not visible on that display.
- Whether Child A is drawn on display 1 after the drag, and on which
  display's Space Mission Control shows it.
- Whether `AXRaise` (Accessibility Inspector → Actions → Raise) on B changes
  the order.
- The app's main thread is idle the whole time (no modal, no sheet).

## Suggested fix

- Do not let a child window live across, or on a different display than,
  its parent's Space: when the parent and child are on different displays
  (`NSWindow.screen` differ) or the child's frame intersects more than one
  `NSScreen`, either detach it (`removeChildWindow:`) or constrain it.
- Constrain every new or restored frame (`X/Y`, `[NSWindow center]`,
  StateKey restore, `SetBounds`) to a single `NSScreen.visibleFrame` —
  the one with the largest intersection, or the parent's screen.
  AppKit's `constrainFrameRect:toScreen:` only handles the vertical /
  menu-bar case and does not stop a horizontal straddle.
