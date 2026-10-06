package sim

import (
	"bytes"
	"compress/gzip"
	"reflect"
	"strings"
	"testing"
)

func TestSaveRoundTripAndDeterminism(t *testing.T) {
	c := New(64, 64, 9)
	c.Funds = 1e6
	c.Apply(ToolRoad, RectPts(Pt{10, 10}, Pt{40, 10}), false)
	c.Apply(ToolZoneR, RectPts(Pt{10, 11}, Pt{25, 11}), false)
	c.Apply(ToolZoneI, RectPts(Pt{26, 11}, Pt{40, 11}), false)
	c.Apply(ToolPlant, []Pt{{10, 12}}, false)
	c.Apply(ToolSchool, []Pt{{20, 13}}, false)
	c.TakeLoan()
	for i := 0; i < 333; i++ { // mid-day, mid-month on purpose
		c.Tick()
	}
	var buf bytes.Buffer
	if err := c.Save(&buf); err != nil {
		t.Fatal(err)
	}
	size := buf.Len()
	d, err := Load(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c.Tiles, d.Tiles) || c.Funds != d.Funds || c.Day != d.Day || !reflect.DeepEqual(c.Loan, d.Loan) {
		t.Fatal("state differs after load")
	}
	for i := 0; i < 1000; i++ {
		c.Tick()
		d.Tick()
	}
	if !reflect.DeepEqual(c.Tiles, d.Tiles) || c.Funds != d.Funds || !reflect.DeepEqual(c.Log, d.Log) {
		t.Fatal("loaded city diverged from the original")
	}
	if size > 200_000 {
		t.Errorf("save is %d bytes", size)
	}
}

func TestLoadRejects(t *testing.T) {
	if _, err := Load(strings.NewReader("hello")); err == nil {
		t.Error("garbage accepted")
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Write([]byte(`{"Version": 99}`))
	zw.Close()
	if _, err := Load(&buf); err == nil || !strings.Contains(err.Error(), "version 99") {
		t.Errorf("future version: %v", err)
	}
	buf.Reset()
	zw = gzip.NewWriter(&buf)
	zw.Write([]byte(`{"Version": 1, "W": 4, "H": 4, "Terrain": ""}`))
	zw.Close()
	if _, err := Load(&buf); err == nil {
		t.Error("truncated save accepted")
	}
}
