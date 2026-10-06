package model

import "testing"

func TestToggle(t *testing.T) {
	d := Deck{Servers: map[string]ServerConfig{"fetch": {}}, Profiles: map[string]ProfileConfig{"cursor": {}}}
	d.Toggle("cursor", "fetch")
	if !d.IsEnabled("cursor", "fetch") {
		t.Fatal("not enabled")
	}
	d.Toggle("cursor", "fetch")
	if d.IsEnabled("cursor", "fetch") {
		t.Fatal("not disabled")
	}
	d.Toggle("missing", "fetch")
	if len(d.Profiles) != 1 {
		t.Fatal("created unknown profile")
	}
}
func TestMissingEnvironment(t *testing.T) {
	t.Setenv("MCPDECK_MISSING", "")
	if _, err := Resolve(ServerConfig{Args: []string{"${MCPDECK_MISSING}"}}); err == nil {
		t.Fatal("missing value accepted")
	}
}
