package agent

import "testing"

func TestExtractLikelyOutputPathFromUserMessage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"ok first create a new file call new-tab.txt", "new-tab.txt"},
		{"save as out.md please", "out.md"},
		{"please implement HelloWorld in core/sample/main.go", "core/sample/main.go"},
		{"no filename here", ""},
	}
	for _, tc := range cases {
		if got := extractLikelyOutputPathFromUserMessage(tc.in); got != tc.want {
			t.Errorf("extractLikelyOutputPathFromUserMessage(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsUserRequestingFileWrite(t *testing.T) {
	t.Parallel()
	if !isUserRequestingFileWrite("please create a file for the tab") {
		t.Fatal("expected create file intent")
	}
	if !isUserRequestingFileWrite("put the tab in the same directory") {
		t.Fatal("expected put tab intent")
	}
	if !isUserRequestingFileWrite("blank screen can you please fix it") {
		t.Fatal("expected fix intent")
	}
	if isUserRequestingFileWrite("what is a C major chord") {
		t.Fatal("should not match generic question")
	}
	if !isUserRequestingFileWrite("store that artical in a markdown file") {
		t.Fatal("expected store that file export intent")
	}
}

func TestUserRequestsFileExport(t *testing.T) {
	t.Parallel()
	if !userRequestsFileExport("store that artical in a markdown file") {
		t.Fatal("expected file export")
	}
	if !userRequestsFileExport("please create that file new-artical-test.md") {
		t.Fatal("expected create that file")
	}
	if userRequestsFileExport("write me a linkedin article") {
		t.Fatal("content delivery alone should not be file export")
	}
}

func TestUserWantsCreateOperation(t *testing.T) {
	t.Parallel()
	if !userWantsCreateOperation("create a new file foo.txt") {
		t.Fatal("expected create")
	}
	if !userWantsCreateOperation("new file for the tab") {
		t.Fatal("expected new file")
	}
	if userWantsCreateOperation("edit server.go only") {
		t.Fatal("should not be create")
	}
}

func TestExtractAnyCodeFenceContent(t *testing.T) {
	t.Parallel()
	in := "Here is the tab:\n```\ne|---\nB|---\n```\nDone."
	got := extractAnyCodeFenceContent(in)
	if got != "e|---\nB|---" {
		t.Fatalf("got %q", got)
	}
}
