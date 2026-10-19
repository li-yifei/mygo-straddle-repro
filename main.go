// mygo-straddle-repro reproduces a macOS hit-testing issue with MyGo child
// windows on two displays with "Displays have separate Spaces" on.
//
// A window created with WindowOptions.Parent is attached with
// addChildWindow:ordered:NSWindowAbove. When such a window ends up (partly)
// on the other display, macOS draws it only on one display's Space while the
// window server still hit-tests its full frame, so clicks on another window
// under the invisible part are swallowed. The buttons open the same windows
// with and without Parent so the two can be compared. See README.md.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var boundary = flag.Int("boundary", 1512, "x of the boundary between the primary display and the display to its right (the primary display's width)")

type repro struct {
	parent *mygo.Window
	// wins are the windows opened from Parent, by title.
	wins  map[string]*mygo.Window
	order []string
}

func (r *repro) parentView(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(20).Gap(12).Children(func() {
		ui.Text(c, "MyGo child-window repro (Parent)").FontSize(20).Bold()
		ui.Text(c, "Needs two displays with \"Displays have separate Spaces\" on. See README.md for the steps. "+
			"A/A' and B/B' are identical except that A and B are created with WindowOptions.Parent = this window; "+
			"none uses a StateKey.").Wrap()
		ui.Column(c).Gap(8).Children(func() {
			if ui.Button(c.Key("open-a"), "Open child A (Parent set)").Clicked() {
				r.open("Child A", true, 980, 760, 0, 0, childView("Child A", true))
			}
			if ui.Button(c.Key("open-a2"), "Open child A' (no Parent)").Clicked() {
				r.open("Child A'", false, 980, 760, 0, 0, childView("Child A'", false))
			}
			if ui.Button(c.Key("open-b"), "Open window B (Parent set)").Clicked() {
				r.open("Window B", true, 760, 820, 0, 0, (&counter{name: "Window B"}).view)
			}
			if ui.Button(c.Key("open-b2"), "Open window B' (no Parent)").Clicked() {
				r.open("Window B'", false, 760, 820, 0, 0, (&counter{name: "Window B'"}).view)
			}
			// Secondary: force the straddling frame seen in the original
			// report, (boundary-330, -241 980x760), with Parent set.
			x := *boundary - 330
			if ui.Button(c.Key("open-a-straddle"), fmt.Sprintf("Open child A straddling (Parent set, x=%d, y=-241)", x)).Clicked() {
				r.open("Child A", true, 980, 760, x, -241, childView("Child A", true))
			}
		})
		for _, d := range mygo.Screen.Displays() {
			ui.Text(c.Key(fmt.Sprint("display-", d.ID)), fmt.Sprintf("display %d: %s primary=%v", d.ID, rect(d.Bounds), d.Primary)).FontSize(12).TextColor(t.TextMuted)
		}
		frameLine(c, "Parent", r.parent)
		for _, name := range r.order {
			frameLine(c, name, r.wins[name])
		}
	})
}

// frameLine shows a window's frame and how many displays it intersects.
func frameLine(c *ui.Context, name string, w *mygo.Window) {
	if w == nil || w.IsDestroyed() {
		return
	}
	b := w.Bounds()
	n := 0
	for _, d := range mygo.Screen.Displays() {
		if intersects(b, d.Bounds) {
			n++
		}
	}
	s := fmt.Sprintf("%s: %s", name, rect(b))
	if n > 1 {
		s += fmt.Sprintf("  STRADDLES %d displays", n)
	}
	ui.Text(c.Key("frame-"+name), s).Selectable()
}

// open (re)opens the window named title, a child of Parent when withParent.
// With x and y both zero MyGo centers it ([NSWindow center]).
func (r *repro) open(title string, withParent bool, width, height, x, y int, view func(*ui.Context)) {
	if old := r.wins[title]; old != nil && !old.IsDestroyed() {
		old.Close()
	}
	opts := mygo.WindowOptions{Title: title, Width: width, Height: height, X: x, Y: y, Content: ui.View(view)}
	if withParent {
		opts.Parent = r.parent
	}
	// NewWindow off the view's build; it hops to the main thread itself.
	go func() {
		w := mygo.NewWindow(opts)
		w.OnMove(r.parent.Invalidate)
		w.OnResize(r.parent.Invalidate)
		w.OnClosed(r.parent.Invalidate)
		mygo.RunOnMain(func() {
			if _, ok := r.wins[title]; !ok {
				r.order = append(r.order, title)
			}
			r.wins[title] = w
			r.parent.Invalidate()
		})
	}()
}

func childView(title string, withParent bool) func(*ui.Context) {
	return func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Gap(8).Children(func() {
			ui.Text(c, title).FontSize(20).Bold()
			ui.Text(c, fmt.Sprintf("980x760, Parent set: %v, no StateKey.", withParent)).Wrap()
		})
	}
}

// counter is window B / B': a dead click shows as a count that does not change.
type counter struct {
	name string
	n    int
}

func (b *counter) view(c *ui.Context) {
	ui.Column(c).Fill().Center().Gap(16).Children(func() {
		ui.Text(c, b.name).FontSize(20).Bold()
		ui.Textf(c, "count: %d", b.n).FontSize(40).Bold()
		if ui.PrimaryButton(c.Key("click"), "Click me").Clicked() {
			b.n++
		}
		ui.Text(c, "Put the button under the other window's overhang, then click.").Wrap()
	})
}

func rect(r mygo.Rectangle) string {
	return fmt.Sprintf("(%d,%d %dx%d)", r.X, r.Y, r.Width, r.Height)
}

func intersects(a, b mygo.Rectangle) bool {
	return a.X < b.X+b.Width && b.X < a.X+a.Width && a.Y < b.Y+b.Height && b.Y < a.Y+a.Height
}

func main() {
	flag.Parse()
	r := &repro{wins: map[string]*mygo.Window{}}
	mygo.App.WhenReady(func() {
		r.parent = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Parent",
			Width:   1512,
			Height:  915,
			Content: ui.View(r.parentView),
		})
		r.parent.OnMove(r.parent.Invalidate)
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
