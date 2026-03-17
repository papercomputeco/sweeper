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

func TestExtractDiffPlainBlock(t *testing.T) {
	// Model omits "diff" language tag but content starts with "diff " header.
	response := "Here:\n\n```\ndiff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n-old\n+new\n```"
	got := extractDiff(response)
	if got == "" {
		t.Fatal("expected diff from plain block, got empty")
	}
	if got != "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n-old\n+new" {
		t.Errorf("unexpected diff: %q", got)
	}
}

func TestExtractDiffPlainBlockDashDash(t *testing.T) {
	// Plain block starting with --- header.
	response := "```\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n-old\n+new\n```"
	got := extractDiff(response)
	if got == "" {
		t.Fatal("expected diff from --- block, got empty")
	}
}

func TestExtractDiffPrefersTaggedBlock(t *testing.T) {
	// When both tagged and plain blocks exist, prefer tagged.
	response := "```\n--- a/wrong.go\n+++ b/wrong.go\n```\n\n```diff\n--- a/right.go\n+++ b/right.go\n```"
	got := extractDiff(response)
	if got != "--- a/right.go\n+++ b/right.go" {
		t.Errorf("expected tagged block to win, got %q", got)
	}
}

func TestExtractDiffPlainBlockIgnoresNonDiff(t *testing.T) {
	// Plain code block that doesn't look like a diff should be ignored.
	response := "```\npackage main\nfunc main() {}\n```"
	got := extractDiff(response)
	if got != "" {
		t.Errorf("expected empty for non-diff plain block, got %q", got)
	}
}
