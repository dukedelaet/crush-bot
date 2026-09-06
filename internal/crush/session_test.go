package crush

import "testing"

func TestParseTranscript(t *testing.T) {
	raw := []byte(`{"messages":[
		{"role":"user","parts":[{"type":"text","text":"hi"}]},
		{"role":"assistant","parts":[{"type":"reasoning"},{"type":"text","text":"hello there"}]},
		{"role":"assistant","parts":[{"type":"tool_use","name":"message_bot"}]}
	]}`)
	lines, err := parseTranscript(raw, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 {
		t.Fatalf("got %d %+v", len(lines), lines)
	}
	if lines[0].Role != "user" || lines[0].Text != "hi" {
		t.Fatalf("%+v", lines[0])
	}
	if lines[1].Text != "hello there" {
		t.Fatalf("%+v", lines[1])
	}
	if lines[2].Text != "[message_bot]" {
		t.Fatalf("%+v", lines[2])
	}
}

func TestParseTranscriptMaxMsgs(t *testing.T) {
	raw := []byte(`{"messages":[
		{"role":"user","parts":[{"type":"text","text":"one"}]},
		{"role":"user","parts":[{"type":"text","text":"two"}]},
		{"role":"user","parts":[{"type":"text","text":"three"}]}
	]}`)
	lines, err := parseTranscript(raw, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0].Text != "two" || lines[1].Text != "three" {
		t.Fatalf("%+v", lines)
	}
}
