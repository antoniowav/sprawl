package sim

import "testing"

func TestUndoRedo(t *testing.T) {
	c := flat(10, 10)
	c.At(2, 0).Terrain = Trees
	start := c.Funds
	_, e := c.ApplyEdit(ToolRoad, LPath(Pt{0, 0}, Pt{5, 0}, false), false)
	if e == nil || len(e.Changes) != 6 {
		t.Fatalf("edit %+v", e)
	}
	c.Undo(e)
	if c.Funds != start || c.At(0, 0).Kind != Empty || c.At(2, 0).Terrain != Trees {
		t.Fatal("undo didn't restore tiles and money")
	}
	if err := c.Redo(e); err != nil || c.At(3, 0).Kind != Road || c.Funds != start-6*CostRoad {
		t.Fatalf("redo: %v", err)
	}
	c.Undo(e)
	c.Apply(ToolZoneR, []Pt{{1, 0}}, false)
	if c.Redo(e) == nil {
		t.Error("redo over a changed map")
	}
}

func TestUndoBuilding(t *testing.T) {
	c := flat(10, 10)
	_, e := c.ApplyEdit(ToolPlant, []Pt{{2, 2}}, false)
	c.Undo(e)
	for _, p := range RectPts(Pt{2, 2}, Pt{4, 4}) {
		if tl := c.At(p.X, p.Y); tl.Kind != Empty || tl.Anchor != -1 {
			t.Fatal("building not removed by undo")
		}
	}
	if c.Funds != StartingFunds {
		t.Error("plant not refunded")
	}
	if _, e := c.ApplyEdit(ToolPlant, []Pt{{9, 9}}, false); e != nil {
		t.Error("failed action produced an edit")
	}
}
