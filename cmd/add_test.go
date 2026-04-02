package cmd

import (
	"testing"
)

func TestParseAddArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantName  string
		wantTrack string
		wantErr   bool
	}{
		{
			name:     "name only",
			args:     []string{"my-feature"},
			wantName: "my-feature",
		},
		{
			name:      "name with track",
			args:      []string{"review/fix-bug", "--track", "origin/fix-bug"},
			wantName:  "review/fix-bug",
			wantTrack: "origin/fix-bug",
		},
		{
			name:    "no args",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "track without value",
			args:    []string{"my-feature", "--track"},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"my-feature", "--unknown"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, track, err := parseAddArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			if track != tt.wantTrack {
				t.Errorf("track = %q, want %q", track, tt.wantTrack)
			}
		})
	}
}
