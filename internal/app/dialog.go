package app

import (
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/antoniowav/sprawl/input"
	"github.com/antoniowav/sprawl/render"
	"github.com/antoniowav/sprawl/sim"
)

type dialogKind int

const (
	dlgSaveAs dialogKind = iota
	dlgOpen
	dlgConfirm
)

// dialog is a modal box: a name field, a list of saves, or a question
// with buttons.
type dialog struct {
	kind    dialogKind
	title   string
	text    []string
	input   string
	list    []saveInfo
	sel     int
	buttons []button
	then    func() // runs after a successful save-as (e.g. quit)
}

// button is a choice in a confirm dialog, picked by its key or Enter on
// the highlighted one.
type button struct {
	key   rune
	label string
	do    func()
}

func (a *App) openDialog(d *dialog) {
	a.dlg, a.mode, a.dirty = d, modeDialog, true
	a.paint, a.visual, a.drag = false, false, false
}

func (a *App) closeDialog() { a.dlg, a.mode, a.dirty = nil, modeNormal, true }

// --- the file actions ---

// quickSave saves to the current save name, or asks for one the first time.
func (a *App) quickSave(then func()) {
	if a.saveName == "" {
		a.saveAs(then)
		return
	}
	if a.save(a.saveName) && then != nil {
		then()
	}
}

func (a *App) saveAs(then func()) {
	name := a.saveName
	if name == "" {
		name = saveName(a.city.Name)
	}
	a.openDialog(&dialog{kind: dlgSaveAs, title: "save as", input: name, then: then,
		text: []string{"Name for this save:"}})
}

func (a *App) openSaves() {
	saves := listSaveInfo()
	if len(saves) == 0 {
		a.flash(sim.Info, "no saves yet · Ctrl+S saves this city")
		return
	}
	a.openDialog(&dialog{kind: dlgOpen, title: "open", list: saves})
}

// guard runs act, first offering to save if there are unsaved changes.
func (a *App) guard(question string, act func()) {
	if !a.unsaved {
		act()
		return
	}
	a.openDialog(&dialog{kind: dlgConfirm, title: "unsaved changes",
		text: []string{question, "Unsaved progress will be lost."},
		buttons: []button{
			{'s', "Save", func() { a.closeDialog(); a.quickSave(act) }},
			{'d', "Don't save", func() { a.closeDialog(); act() }},
			{0, "Cancel", a.closeDialog},
		}})
}

func (a *App) requestQuit() { a.guard("Quit "+a.city.Name+"?", func() { a.quit = true }) }

func (a *App) requestNew() { a.guard("Start a new city?", a.startNewCityForm) }

// --- input ---

func (a *App) updateDialog() {
	d := a.dlg
	esc := inpututil.IsKeyJustPressed(ebiten.KeyEscape)
	enter := inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter)
	chars := a.poll.Chars()
	if len(chars) > 0 || esc || enter {
		a.dirty = true
	}
	switch d.kind {
	case dlgSaveAs:
		for _, r := range chars {
			if strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_ ", r) && len(d.input) < 32 {
				d.input += string(r)
			}
		}
		if input.Repeated(ebiten.KeyBackspace) && d.input != "" {
			d.input, a.dirty = d.input[:len(d.input)-1], true
		}
		switch {
		case esc:
			a.closeDialog()
		case enter && strings.TrimSpace(d.input) != "":
			a.closeDialog()
			if a.save(d.input) && d.then != nil {
				d.then()
			}
		}
	case dlgOpen:
		n := len(d.list)
		switch {
		case input.Repeated(ebiten.KeyArrowDown) || hasRune(chars, 'j'):
			d.sel, a.dirty = (d.sel+1)%n, true
		case input.Repeated(ebiten.KeyArrowUp) || hasRune(chars, 'k'):
			d.sel, a.dirty = (d.sel+n-1)%n, true
		case inpututil.IsKeyJustPressed(ebiten.KeyDelete):
			a.confirmDelete(d.list[d.sel].name)
		case esc:
			a.closeDialog()
		case enter:
			name := d.list[d.sel].name
			a.closeDialog()
			a.guard("Open "+name+"?", func() { a.load(name) })
		}
		if click, ok := a.dialogClick(); ok && click < n {
			if click == d.sel {
				name := d.list[d.sel].name
				a.closeDialog()
				a.guard("Open "+name+"?", func() { a.load(name) })
			} else {
				d.sel, a.dirty = click, true
			}
		}
	case dlgConfirm:
		for _, r := range chars {
			for _, b := range d.buttons {
				if b.key != 0 && (r == b.key || r == b.key-32) {
					b.do()
					return
				}
			}
		}
		switch {
		case input.Repeated(ebiten.KeyArrowRight) || input.Repeated(ebiten.KeyTab):
			d.sel, a.dirty = (d.sel+1)%len(d.buttons), true
		case input.Repeated(ebiten.KeyArrowLeft):
			d.sel, a.dirty = (d.sel+len(d.buttons)-1)%len(d.buttons), true
		case esc:
			d.buttons[len(d.buttons)-1].do()
		case enter:
			d.buttons[d.sel].do()
		}
		if click, ok := a.dialogClick(); ok && click < len(d.buttons) {
			d.buttons[click].do()
		}
	}
}

func (a *App) confirmDelete(name string) {
	a.openDialog(&dialog{kind: dlgConfirm, title: "delete save",
		text: []string{"Delete the save " + name + "?", "This can't be undone."},
		buttons: []button{
			{'d', "Delete", func() {
				os.Remove(savePath(name))
				a.closeDialog()
				a.flash(sim.Info, "deleted %s", name)
				a.openSaves()
			}},
			{0, "Cancel", func() { a.closeDialog(); a.openSaves() }},
		}})
}

// dialogClick returns the index of the list row or button under a left
// click, using the hit boxes the HUD recorded when it drew the dialog.
func (a *App) dialogClick() (int, bool) {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return 0, false
	}
	x, y := ebiten.CursorPosition()
	for i, r := range a.hud.Hits {
		if x >= r.Min.X && x < r.Max.X && y >= r.Min.Y && y < r.Max.Y {
			return i, true
		}
	}
	return 0, false
}

func hasRune(rs []rune, r rune) bool {
	for _, c := range rs {
		if c == r {
			return true
		}
	}
	return false
}

// view converts the dialog for the HUD.
func (d *dialog) view() *render.Dialog {
	v := &render.Dialog{Title: d.title, Text: d.text, Sel: d.sel}
	switch d.kind {
	case dlgSaveAs:
		v.Input = &d.input
		v.Hint = "Enter save · Esc cancel"
	case dlgOpen:
		for _, s := range d.list {
			v.List = append(v.List, [2]string{s.name, s.when})
		}
		v.Hint = "↑↓ choose · Enter open · Del delete · Esc cancel"
	case dlgConfirm:
		for _, b := range d.buttons {
			l := b.label
			if b.key != 0 {
				l = "[" + strings.ToUpper(string(b.key)) + "] " + l
			} else {
				l = "[Esc] " + l
			}
			v.Buttons = append(v.Buttons, l)
		}
	}
	return v
}
