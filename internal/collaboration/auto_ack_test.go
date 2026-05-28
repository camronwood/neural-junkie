package collaboration

import "testing"

func TestShouldAutoAckWorkspaceOnApprove(t *testing.T) {
	tests := []struct {
		name string
		c    *Collaboration
		want bool
	}{
		{
			name: "sandbox with repo",
			c:    &Collaboration{SourceRepoPath: "/tmp/repo", ExecutionMode: ExecutionModeSandbox},
			want: true,
		},
		{
			name: "worktree",
			c:    &Collaboration{SourceRepoPath: "/tmp/repo", ExecutionMode: ExecutionModeWorktree},
			want: false,
		},
		{
			name: "no repo",
			c:    &Collaboration{ExecutionMode: ExecutionModeSandbox},
			want: false,
		},
		{
			name: "nil",
			c:    nil,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldAutoAckWorkspaceOnApprove(tt.c); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}
