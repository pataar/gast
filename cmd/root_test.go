package cmd

import (
	"bytes"
	"testing"
)

func TestBuildVersionPrefersLdflagsValue(t *testing.T) {
	version = "1.2.3"
	t.Cleanup(func() { version = "" })

	if got := buildVersion(); got != "1.2.3" {
		t.Errorf("buildVersion() = %q, want %q", got, "1.2.3")
	}
}

func TestBuildVersionFallsBackWhenUnset(t *testing.T) {
	if got := buildVersion(); got == "" {
		t.Error("buildVersion() returned an empty string without an ldflags version")
	}
}

func TestVersionFlagPrintsVersion(t *testing.T) {
	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetArgs([]string{"--version"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	want := "gast version " + rootCmd.Version + "\n"
	if got := output.String(); got != want {
		t.Errorf("--version output = %q, want %q", got, want)
	}
}
