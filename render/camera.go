package render

import "math"

// Camera tracks the view centre in world pixels (unscaled). The draw offset
// is always rounded to whole screen pixels, so tiles never shimmer.
type Camera struct {
	X, Y   float64 // current centre
	TX, TY float64 // target centre
	Zoom   int     // integer scale, 1..4
}

// Ease moves toward the target. It returns true while still moving.
func (c *Camera) Ease() bool {
	dx, dy := c.TX-c.X, c.TY-c.Y
	if dx == 0 && dy == 0 {
		return false
	}
	// Stop once within half a screen pixel.
	eps := 0.5 / float64(c.Zoom)
	if math.Abs(dx) < eps && math.Abs(dy) < eps {
		c.X, c.Y = c.TX, c.TY
		return true
	}
	c.X += dx * 0.25
	c.Y += dy * 0.25
	return true
}

// Jump sets both current and target.
func (c *Camera) Jump(x, y float64) { c.X, c.Y, c.TX, c.TY = x, y, x, y }

// Origin is the screen position of world pixel (0, 0) for a view of w×h.
func (c *Camera) Origin(w, h int) (int, int) {
	z := float64(c.Zoom)
	return w/2 - int(math.Round(c.X*z)), h/2 - int(math.Round(c.Y*z))
}

// TileAt converts a screen position to tile coordinates.
func (c *Camera) TileAt(sx, sy, w, h int) (int, int) {
	ox, oy := c.Origin(w, h)
	ts := TileSize * c.Zoom
	return floorDiv(sx-ox, ts), floorDiv(sy-oy, ts)
}

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}
