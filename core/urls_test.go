package core

import (
	"fmt"
	"testing"
)

func TestTargetKeyNormalizesDeduplicationInput(t *testing.T) {
	pairs := map[string]string{
		"HTTP://Example.COM":             "http://example.com/",
		"http://example.com:80/":         "http://example.com/",
		"https://example.com:443/path#x": "https://example.com/path",
		"EXAMPLE.COM":                    "example.com",
	}

	for input, expected := range pairs {
		if actual := TargetKey(input); actual != expected {
			t.Fatalf("TargetKey(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestDeduplicateTargetsWith500UniqueTargets(t *testing.T) {
	targets := make([]string, 0, 600)
	for i := 0; i < 500; i++ {
		targets = append(targets, fmt.Sprintf("https://target-%03d.example.com/%d", i, i))
	}
	for i := 0; i < 100; i++ {
		targets = append(targets, fmt.Sprintf("HTTPS://TARGET-%03d.EXAMPLE.COM:443/%d#duplicate", i, i))
	}

	unique := DeduplicateTargets(targets)
	if len(unique) != 500 {
		t.Fatalf("DeduplicateTargets returned %d targets, want 500", len(unique))
	}
	if unique[0] != "https://target-000.example.com/0" {
		t.Fatalf("first target changed: %q", unique[0])
	}
}
