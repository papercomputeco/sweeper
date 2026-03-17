package worker

import "testing"

func TestExtractDiff(t *testing.T) {
	response := "Here is the fix:\n\n```diff\n--- a/main.go\n+++ b/main.go\n@@ -1,3 +1,3 @@\n-old line\n+new line\n```\n\nDone."
	got := extractDiff(response)
	if got == "" {
		t.Fatal("expected diff, got empty")
	}
	if got != "--- a/main.go\n+++ b/main.go\n@@ -1,3 +1,3 @@\n-old line\n+new line" {
		t.Errorf("unexpected diff: %q", got)
	}
}

func TestExtractDiffNone(t *testing.T) {
	got := extractDiff("No diff here, just some text.")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestExtractDiffMultipleBlocks(t *testing.T) {
	response := "```diff\nfirst diff\n```\n\n```diff\nsecond diff\n```"
	got := extractDiff(response)
	if got != "first diff" {
		t.Errorf("expected first diff block, got %q", got)
	}
}
