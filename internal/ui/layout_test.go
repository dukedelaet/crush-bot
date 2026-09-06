package ui

import "testing"

func TestLayoutPinsHelpAndSidebar(t *testing.T) {
	sideW, crushW, bodyH := layout(120, 40)
	if bodyH != 39 {
		t.Fatalf("bodyH %d want 39 (one row for help)", bodyH)
	}
	if sideW+dividerW+crushW != 120 {
		t.Fatalf("widths %d+%d+%d != 120", sideW, dividerW, crushW)
	}
	if sideW < minSideW || sideW > maxSideW {
		t.Fatalf("sidebar %d", sideW)
	}
	if crushW < 10 {
		t.Fatalf("crush pane %d", crushW)
	}
}

func TestLayoutNarrow(t *testing.T) {
	sideW, crushW, bodyH := layout(40, 10)
	if bodyH != 9 {
		t.Fatalf("bodyH %d", bodyH)
	}
	if sideW+dividerW+crushW != 40 {
		t.Fatalf("widths %d+%d+%d", sideW, dividerW, crushW)
	}
}
