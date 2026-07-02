package player

import "testing"

func TestFirstAvailable(t *testing.T) {
	lookPath := func(name string) (string, error) {
		if name == "paplay" {
			return "/usr/bin/paplay", nil
		}
		return "", errNotFound
	}

	cmd, ok := FirstAvailable(lookPath)
	if !ok {
		t.Fatal("FirstAvailable() ok = false")
	}
	if cmd.Name != "paplay" {
		t.Fatalf("command name = %q, want paplay", cmd.Name)
	}
}
