package tui

import (
	"reflect"
	"testing"
)

func TestExtractWorkItemIDsFromCommitMessage(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    []string
	}{
		{
			name:    "single id",
			message: "feat: add button #12345",
			want:    []string{"12345"},
		},
		{
			name:    "multiple ids",
			message: "fix: resolve race #100 #200 #300",
			want:    []string{"100", "200", "300"},
		},
		{
			name:    "deduplicate ids preserving order",
			message: "chore: cleanup #42 and follow-up #99 and #42",
			want:    []string{"42", "99"},
		},
		{
			name:    "multiline message",
			message: "feat: payment flow\n\nImplements checkout #555\nRefactor taxes #777",
			want:    []string{"555", "777"},
		},
		{
			name:    "ignore non numeric tags",
			message: "docs: update readme #abc #12a",
			want:    nil,
		},
		{
			name:    "no ids",
			message: "refactor: rename variables",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractWorkItemIDsFromCommitMessage(tt.message)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("extractWorkItemIDsFromCommitMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}
