package schema

import "testing"

func TestParseObjectRequired(t *testing.T) {
	_, err := ParseObject([]byte(`{"options":[]}`), ObjectSpec{Required: []string{"question"}})
	if err == nil {
		t.Fatal("expected missing question error")
	}
	ve, ok := err.(*ValidationError)
	if !ok || ve.Path != "question" {
		t.Fatalf("got %v", err)
	}
	obj, err := ParseObject([]byte(`{"question":"pick one"}`), ObjectSpec{Required: []string{"question"}})
	if err != nil || obj["question"] != "pick one" {
		t.Fatalf("got %v err=%v", obj, err)
	}
}

func TestParseObjectEmptyString(t *testing.T) {
	_, err := ParseObject([]byte(`{"question":"  "}`), ObjectSpec{Required: []string{"question"}})
	if err == nil {
		t.Fatal("expected empty string error")
	}
}

func TestParseInto(t *testing.T) {
	type args struct {
		Question string `json:"question"`
	}
	out, err := ParseInto[args]([]byte(`{"question":"hi"}`), ObjectSpec{Required: []string{"question"}})
	if err != nil || out.Question != "hi" {
		t.Fatalf("got %+v err=%v", out, err)
	}
}
