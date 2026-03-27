package tui

import "testing"

func TestCreatePRSourceSelectionHelpers(t *testing.T) {
	m := Model{
		createPRBranches:        []string{"feature/a", "feature/b"},
		createPRHasMoreBranches: true,
	}

	if got := m.createPRSourceOptionCount(); got != 3 {
		t.Fatalf("createPRSourceOptionCount() = %d, want 3", got)
	}

	m.createPRSelectedSource = 2
	if !m.isCreatePRLoadMoreSelected() {
		t.Fatal("expected load-more option to be selected")
	}

	if branch, ok := m.selectedCreatePRSourceBranch(); ok || branch != "" {
		t.Fatalf("selectedCreatePRSourceBranch() = (%q, %v), want (\"\", false)", branch, ok)
	}

	m.createPRSelectedSource = 1
	if m.isCreatePRLoadMoreSelected() {
		t.Fatal("did not expect load-more option to be selected")
	}

	branch, ok := m.selectedCreatePRSourceBranch()
	if !ok || branch != "feature/b" {
		t.Fatalf("selectedCreatePRSourceBranch() = (%q, %v), want (\"feature/b\", true)", branch, ok)
	}
}
