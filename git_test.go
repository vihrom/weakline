package main

import (
	"testing"
)

func TestParseBranchLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		expBranch string
		expAhead  int
		expBehind int
	}{
		{
			name:      "Simple branch",
			line:      "## main",
			expBranch: "main",
		},
		{
			name:      "Branch with remote",
			line:      "## feature/ui...origin/feature/ui",
			expBranch: "feature/ui",
		},
		{
			name:      "Ahead only",
			line:      "## main...origin/main [ahead 3]",
			expBranch: "main",
			expAhead:  3,
		},
		{
			name:      "Behind only",
			line:      "## main...origin/main [behind 5]",
			expBranch: "main",
			expBehind: 5,
		},
		{
			name:      "Ahead and Behind",
			line:      "## master...origin/master [ahead 1, behind 2]",
			expBranch: "master",
			expAhead:  1,
			expBehind: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var st Status
			parseBranchLine([]byte(tt.line), &st)

			if st.Branch != tt.expBranch {
				t.Errorf("expected branch %q, got %q", tt.expBranch, st.Branch)
			}
			if st.Ahead != tt.expAhead {
				t.Errorf("expected ahead %d, got %d", tt.expAhead, st.Ahead)
			}
			if st.Behind != tt.expBehind {
				t.Errorf("expected behind %d, got %d", tt.expBehind, st.Behind)
			}
		})
	}
}
