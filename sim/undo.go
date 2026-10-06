package sim

import "strconv"

// Edit records what an applied tool changed, so it can be undone.
type Edit struct {
	Tool    Tool
	Changes []TileChange
	Cost    float64
}

// TileChange is one tile before and after an edit (persistent fields only).
type TileChange struct {
	I             int
	Before, After Tile
}

// persistent copies the fields a player edit can change.
func persistent(t Tile) Tile {
	return Tile{Terrain: t.Terrain, Kind: t.Kind, Level: t.Level, Variant: t.Variant,
		Line: t.Line, Pipe: t.Pipe, Anchor: t.Anchor}
}

func setPersistent(dst *Tile, src Tile) {
	dst.Terrain, dst.Kind, dst.Level, dst.Variant = src.Terrain, src.Kind, src.Level, src.Variant
	dst.Line, dst.Pipe, dst.Anchor = src.Line, src.Pipe, src.Anchor
}

// ApplyEdit is Apply that also returns an Edit for undo (nil on failure).
func (c *City) ApplyEdit(t Tool, pts []Pt, pipesOnly bool) (Plan, *Edit) {
	p := c.PlanTool(t, pts, pipesOnly)
	if p.Err != "" {
		return p, nil
	}
	before := make([]Tile, len(p.Tiles))
	for i, pt := range p.Tiles {
		before[i] = persistent(*c.At(pt.X, pt.Y))
	}
	p = c.Apply(t, pts, pipesOnly)
	e := &Edit{Tool: t, Cost: p.Cost}
	for i, pt := range p.Tiles {
		idx := pt.Y*c.W + pt.X
		e.Changes = append(e.Changes, TileChange{idx, before[i], persistent(c.Tiles[idx])})
	}
	return p, e
}

// Undo reverts an edit and refunds its cost. Tiles that have changed since
// (a lot that grew, say) are still reverted: the player asked for it.
func (c *City) Undo(e *Edit) {
	for _, ch := range e.Changes {
		setPersistent(&c.Tiles[ch.I], ch.Before)
	}
	c.Funds += e.Cost
	c.lvDirty = true
}

// Redo re-applies an undone edit if the tiles are still as Undo left them
// and the money is there.
func (c *City) Redo(e *Edit) error {
	if e.Tool != ToolBulldoze && e.Cost > c.Funds {
		return errNoFunds(e.Cost)
	}
	for _, ch := range e.Changes {
		if persistent(c.Tiles[ch.I]) != ch.Before {
			return errChanged
		}
	}
	for _, ch := range e.Changes {
		setPersistent(&c.Tiles[ch.I], ch.After)
	}
	c.Funds -= e.Cost
	c.lvDirty = true
	return nil
}

type simError string

func (e simError) Error() string { return string(e) }

const errChanged = simError("can't redo: the map changed since")

func errNoFunds(cost float64) error {
	return simError("not enough funds to redo ($" + strconv.Itoa(int(cost)) + " needed)")
}
