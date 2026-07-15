package cedar

import (
	"strings"
	"testing"
)

func TestIsValid(t *testing.T) {
	valid := []string{
		`permit (principal, action, resource);`,
		`permit (principal == User::"alice", action == Action::"view", resource) when { resource.owner == principal };`,
		// A multi-policy document (TinyTodo subset).
		`permit (principal, action in [Action::"CreateList", Action::"GetLists"], resource == Application::"TinyTodo");
permit (principal, action, resource) when { resource has owner && resource.owner == principal };`,
	}
	for _, s := range valid {
		if !IsValid(s) {
			t.Errorf("IsValid(%q) = false, want true", s)
		}
	}
	invalid := []string{
		`permit (principal, action, resource)`,                             // missing semicolon
		`permit (principal == User::, action, resource);`,                  // malformed entity literal
		`allow (principal, action, resource);`,                             // "allow" is not a Cedar keyword
		`permit (principal, action, resource) when { resource.owner == };`, // dangling operator
	}
	for _, s := range invalid {
		if IsValid(s) {
			t.Errorf("IsValid(%q) = true, want false", s)
		}
	}
}

func TestFormat(t *testing.T) {
	got, err := Format(`permit(principal==User::"alice",action==Action::"view",resource);`)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	if !strings.Contains(got, "\n") {
		t.Errorf("expected multi-line canonical output, got %q", got)
	}
	if !IsValid(got) {
		t.Errorf("formatted output is not valid: %q", got)
	}
	// Formatting is idempotent.
	got2, err := Format(got)
	if err != nil {
		t.Fatalf("Format (second pass) error: %v", err)
	}
	if got2 != got {
		t.Errorf("Format not idempotent:\n%q\nvs\n%q", got, got2)
	}
	// Invalid input fails.
	if _, err := Format(`permit (`); err == nil {
		t.Error("Format of invalid input should error")
	}
}
