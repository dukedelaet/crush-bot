package crush

import "testing"

func TestAtLeast(t *testing.T) {
	if !AtLeast("v0.91.2", "0.91.2") {
		t.Fatal("eq")
	}
	if !AtLeast("v0.92.0", "0.91.2") {
		t.Fatal("newer")
	}
	if AtLeast("v0.90.0", "0.91.2") {
		t.Fatal("older")
	}
}

func TestParseVersion(t *testing.T) {
	maj, min, patch, ok := ParseVersion("crush version v0.91.2")
	if !ok || maj != 0 || min != 91 || patch != 2 {
		t.Fatalf("%d %d %d %v", maj, min, patch, ok)
	}
}
