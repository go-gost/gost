package main

import (
	"bufio"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// sshHostEntry holds the resolved settings for a single SSH host alias.
type sshHostEntry struct {
	hostname     string // HostName directive
	port         string // Port directive
	user         string // User directive
	identityFile string // first IdentityFile directive
}

// parseSSHConfigLine splits an SSH config line into (key, value).
// Handles both "Key Value" and "Key=Value" (with optional spaces around =).
func parseSSHConfigLine(line string) (key, value string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	// Find first whitespace or '='
	i := 0
	for i < len(line) && line[i] != ' ' && line[i] != '\t' && line[i] != '=' {
		i++
	}
	if i == len(line) {
		return "", "", false
	}
	key = line[:i]
	rest := strings.TrimLeft(line[i:], " \t=")
	// Strip trailing inline comment (must be preceded by whitespace)
	if idx := strings.Index(rest, " #"); idx >= 0 {
		rest = strings.TrimSpace(rest[:idx])
	}
	value = strings.Trim(rest, `"'`)
	return key, value, value != ""
}

// readSSHConfig parses ~/.ssh/config and returns a map of host alias/pattern
// to resolved entry. Only exact-match patterns (no wildcards) are indexed so
// they can be looked up directly; wildcard patterns are skipped for now.
func readSSHConfig() map[string]*sshHostEntry {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	path := filepath.Join(home, ".ssh", "config")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	entries := make(map[string]*sshHostEntry)
	var current *sshHostEntry
	var currentPatterns []string

	commit := func() {
		if current == nil {
			return
		}
		for _, pat := range currentPatterns {
			// Skip wildcard patterns — they can't be used for exact lookup.
			if strings.ContainsAny(pat, "*?") {
				continue
			}
			if _, exists := entries[pat]; !exists {
				entries[pat] = current
			}
		}
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		key, value, ok := parseSSHConfigLine(scanner.Text())
		if !ok {
			continue
		}
		switch strings.ToLower(key) {
		case "host":
			commit()
			currentPatterns = strings.Fields(value)
			current = &sshHostEntry{}
		case "hostname":
			if current != nil && current.hostname == "" {
				current.hostname = value
			}
		case "port":
			if current != nil && current.port == "" {
				current.port = value
			}
		case "user":
			if current != nil && current.user == "" {
				current.user = value
			}
		case "identityfile":
			if current != nil && current.identityFile == "" {
				if strings.HasPrefix(value, "~/") {
					home, _ := os.UserHomeDir()
					value = filepath.Join(home, value[2:])
				}
				current.identityFile = value
			}
		}
	}
	commit()

	return entries
}

// expandSSHNode rewrites a single node URL string using ~/.ssh/config when the
// scheme is "ssh" and the host matches a Host entry. Fields already present in
// the URL are never overridden.
func expandSSHNode(raw string, cfg map[string]*sshHostEntry) string {
	if cfg == nil || !strings.HasPrefix(raw, "ssh://") {
		return raw
	}

	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "ssh" {
		return raw
	}

	host := u.Hostname()
	port := u.Port()

	entry, ok := cfg[host]
	if !ok {
		return raw
	}

	// Resolve hostname.
	newHost := host
	if entry.hostname != "" {
		newHost = entry.hostname
	}

	// Resolve port: only substitute when the URL carries no explicit port or
	// carries the SSH default (22) and the config specifies a different one.
	newPort := port
	if newPort == "" || newPort == "22" {
		if entry.port != "" && entry.port != "22" {
			newPort = entry.port
		}
	}

	if newPort != "" && newPort != "22" {
		u.Host = net.JoinHostPort(newHost, newPort)
	} else {
		u.Host = newHost
	}

	// Apply user if not already present in the URL.
	if entry.user != "" && (u.User == nil || u.User.Username() == "") {
		var pw string
		if u.User != nil {
			pw, _ = u.User.Password()
		}
		if pw != "" {
			u.User = url.UserPassword(entry.user, pw)
		} else {
			u.User = url.User(entry.user)
		}
	}

	// Apply identity file as the "key" query parameter when not already set.
	if entry.identityFile != "" {
		q := u.Query()
		if q.Get("privateKeyFile") == "" {
			q.Set("privateKeyFile", entry.identityFile)
			u.RawQuery = q.Encode()
		}
	}

	return u.String()
}

// expandSSHNodes applies expandSSHNode to every element of the node list and
// returns the result as a new slice. The original slice is not modified.
func expandSSHNodes(nodes []string) []string {
	cfg := readSSHConfig()
	result := make([]string, len(nodes))
	for i, n := range nodes {
		result[i] = expandSSHNode(n, cfg)
	}
	return result
}
