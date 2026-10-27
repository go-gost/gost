package main

import (
	godebug "runtime/debug"
	"testing"
)

func buildInfo(version string, settings ...godebug.BuildSetting) *godebug.BuildInfo {
	return &godebug.BuildInfo{
		Main:     godebug.Module{Version: version},
		Settings: settings,
	}
}

func vcs(revision string, modified bool) []godebug.BuildSetting {
	modifiedValue := "false"
	if modified {
		modifiedValue = "true"
	}
	return []godebug.BuildSetting{
		{Key: "vcs.revision", Value: revision},
		{Key: "vcs.modified", Value: modifiedValue},
	}
}

func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name     string
		injected string
		bi       *godebug.BuildInfo
		fallback string
		want     string
	}{
		{"injected wins over module version", "3.3.2", buildInfo("v9.9.9"), "3.3.2", "3.3.2"},
		{"injected v prefix stripped", "v3.3.1", nil, "3.3.2", "3.3.1"},
		{"module version from go install", "", buildInfo("v3.4.0"), "3.3.2", "3.4.0"},
		{"module version v prefix stripped", "", buildInfo("v3.3.2"), "3.3.2", "3.3.2"},
		{"manual build appends short commit", "", buildInfo("(devel)", vcs("0123456789abcdef", false)...), "3.3.2", "3.3.2+0123456"},
		{"local build ignores go pseudo-version", "", buildInfo("0.0.0-20261021144633-84dd70f9f65e+dirty", vcs("0123456789abcdef", true)...), "3.3.2", "3.3.2+0123456-dirty"},
		{"local build at exact tag still appends commit", "", buildInfo("v3.4.0", vcs("0123456789abcdef", false)...), "3.3.2", "3.3.2+0123456"},
		{"manual build marks dirty tree", "", buildInfo("(devel)", vcs("0123456789abcdef", true)...), "3.3.2", "3.3.2+0123456-dirty"},
		{"short revision kept as is", "", buildInfo("(devel)", vcs("abc", false)...), "3.3.2", "3.3.2+abc"},
		{"devel without vcs falls back", "", buildInfo("(devel)"), "3.3.2", "3.3.2"},
		{"no build info falls back", "", nil, "3.3.2", "3.3.2"},
		{"empty injection falls through", "", buildInfo(""), "3.3.2", "3.3.2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.injected, tt.bi, tt.fallback); got != tt.want {
				t.Fatalf("resolveVersion(%q, %+v, %q) = %q, want %q", tt.injected, tt.bi, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestCleanVersion(t *testing.T) {
	tests := map[string]string{
		"3.3.2":                   "3.3.2",
		"v3.3.2":                  "3.3.2",
		"  v3.3.2  ":              "3.3.2",
		"(devel)":                 "",
		"":                        "",
		"v3.3.2-nightly.20261021": "3.3.2-nightly.20261021",
	}
	for in, want := range tests {
		if got := cleanVersion(in); got != want {
			t.Fatalf("cleanVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsTestBinary(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"normal run", []string{"/usr/bin/gost", "-L", "http://:8080"}, false},
		{"go test on linux", []string{"/tmp/go-build123/b001/gost.test", "-test.paniconexit0", "-test.timeout=10m0s"}, true},
		{"go test on windows", []string{`C:\Users\dev\AppData\Local\Temp\go-build123\b001\gost.test.exe`, "-test.paniconexit0"}, true},
		{"renamed test binary", []string{"/tmp/renamed-binary", "-test.run=TestResolveVersion"}, true},
		{"no args", []string{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTestBinary(tt.args); got != tt.want {
				t.Fatalf("isTestBinary(%q) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestVersionIsInjectable(t *testing.T) {
	// -ldflags "-X main.version=..." only overrides a string variable that is
	// declared without an initializer; an initializer silently disables every
	// release and docker build's version injection.
	if version != "" {
		t.Fatalf("version = %q, want the zero value so -X can inject", version)
	}
}
