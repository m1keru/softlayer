package main

import "testing"

func TestTranslateLegacyLease(t *testing.T) {
	got := translateLegacyArgs([]string{"-lease", "-ptr=host.example.com", "-note=host.example.com", "-force"})
	want := []string{"lease", "-ptr", "host.example.com", "-note", "host.example.com", "-force"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestTranslateLegacyList(t *testing.T) {
	got := translateLegacyArgs([]string{"-list", "-public", "-one"})
	want := []string{"list", "-public", "-one"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestTranslateLegacySet(t *testing.T) {
	got := translateLegacyArgs([]string{"-ip", "1.2.3.4", "-ptr=host.example.com", "-note=host.example.com"})
	if got[0] != "set" || got[1] != "-ip" || got[2] != "1.2.3.4" {
		t.Fatalf("unexpected set translation: %v", got)
	}
}

func TestTranslateLegacyStale(t *testing.T) {
	got := translateLegacyArgs([]string{"-liststale"})
	if len(got) != 1 || got[0] != "stale" {
		t.Fatalf("got %v", got)
	}
}

func TestHelpRequested(t *testing.T) {
	if !helpRequested([]string{"--help"}) {
		t.Fatal("expected --help to trigger help")
	}
	if helpRequested([]string{"-force"}) {
		t.Fatal("did not expect -force to trigger help")
	}
}
