package app

import (
	"testing"
)

func TestSummarizeSessions(t *testing.T) {
	views := []SessionView{
		{Provider: ProviderClaude, Status: StatusRunning, Project: "Alpha", Cost: 1.0},
		{Provider: ProviderClaude, Status: StatusWaiting, Project: "Alpha", Cost: 0.5},
		{Provider: ProviderCodex, Status: StatusApproval, Project: "Beta", Cost: 0.0},
	}
	rows := summarizeSessions(views, "project")
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Total+rows[1].Total != 3 {
		t.Fatalf("expected total 3, got %d", rows[0].Total+rows[1].Total)
	}
}
