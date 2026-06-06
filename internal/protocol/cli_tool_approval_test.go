package protocol

import "testing"

func TestShouldAutoApproveCLIToolCall(t *testing.T) {
	cases := []struct {
		mode     string
		tool     string
		input    map[string]interface{}
		wantAuto bool
	}{
		{"yolo", "run_shell_command", map[string]interface{}{"command": "rm -rf /"}, true},
		{"auto_edit", "read_file", map[string]interface{}{"path": "main.go"}, true},
		{"auto_edit", "run_shell_command", map[string]interface{}{"command": "cat README.md"}, true},
		{"auto_edit", "shell", map[string]interface{}{"command": "grep schema internal/"}, true},
		{"auto_edit", "run_shell_command", map[string]interface{}{"command": "npm install"}, false},
		{"auto_edit", "run_shell_command", map[string]interface{}{"command": "rm -rf node_modules"}, false},
		{"auto_apply_edits", "read_file", map[string]interface{}{"path": "main.go"}, true},
		{"auto_apply_edits", "grep", map[string]interface{}{"pattern": "foo"}, true},
		{"auto_apply_edits", "run_shell_command", map[string]interface{}{"command": "npm install"}, false},
		{"interactive", "read_file", map[string]interface{}{"path": "main.go"}, false},
		{"interactive", "run_shell_command", map[string]interface{}{"command": "cat README.md"}, false},
	}
	for _, tc := range cases {
		name := tc.mode + "/" + tc.tool
		t.Run(name, func(t *testing.T) {
			got := ShouldAutoApproveCLIToolCall(tc.mode, tc.tool, tc.input)
			if got != tc.wantAuto {
				t.Fatalf("ShouldAutoApproveCLIToolCall(%q, %q, %#v) = %v, want %v",
					tc.mode, tc.tool, tc.input, got, tc.wantAuto)
			}
		})
	}
}

func TestIsCLIShellToolName(t *testing.T) {
	if !IsCLIShellToolName("run_shell_command") {
		t.Fatal("expected run_shell_command")
	}
	if !IsCLIShellToolName("shell") {
		t.Fatal("expected shell")
	}
	if IsCLIShellToolName("read_file") {
		t.Fatal("read_file is not a shell tool")
	}
}
