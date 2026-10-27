package main

import (
	godebug "runtime/debug"
	"strings"
)

// version is set at build time via -ldflags "-X main.version=...".
// It must stay uninitialized: the linker only overrides string variables
// declared without an initializer or with a constant one.
var version string

// buildVersionFallback is the baseline version shown by builds that carry no
// version information (plain `go build` outside a git checkout). The Makefile
// reads it to name archives, so keep the declaration on one line.
const buildVersionFallback = "3.3.2"

// getVersion returns the version of this build.
func getVersion() string {
	bi, _ := godebug.ReadBuildInfo()
	return resolveVersion(version, bi, buildVersionFallback)
}

// resolveVersion picks the first available source of truth: the injected
// version, then the fallback plus the VCS revision of a local checkout, then
// the module version recorded by `go install ...@vX`, then the bare fallback.
// Local checkout builds win over Main.Version because Go stamps a
// `0.0.0-<time>-<rev>` pseudo-version there instead of `(devel)`.
func resolveVersion(injected string, bi *godebug.BuildInfo, fallback string) string {
	if v := cleanVersion(injected); v != "" {
		return v
	}
	if bi != nil {
		if v := versionWithVCS(fallback, bi.Settings); v != "" {
			return v
		}
		if v := cleanVersion(bi.Main.Version); v != "" {
			return v
		}
	}
	return fallback
}

// cleanVersion normalizes a version string, dropping a leading "v" and the
// "(devel)" placeholder Go records for local builds.
func cleanVersion(v string) string {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if v == "(devel)" {
		return ""
	}
	return v
}

// versionWithVCS appends the short VCS revision (and a dirty marker) to the
// fallback, returning "" when no revision was stamped into the binary.
func versionWithVCS(fallback string, settings []godebug.BuildSetting) string {
	var rev string
	var dirty bool
	for _, s := range settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev == "" {
		return ""
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	v := fallback + "+" + rev
	if dirty {
		v += "-dirty"
	}
	return v
}
