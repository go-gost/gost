package main

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func writeSSHConfig(t *testing.T, content string) (cleanup func()) {
	t.Helper()
	dir := t.TempDir()
	sshDir := filepath.Join(dir, ".ssh")
	if err := os.Mkdir(sshDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	old := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	return func() { os.Setenv("HOME", old) }
}

func TestParseSSHConfigLine(t *testing.T) {
	cases := []struct {
		line, wantKey, wantVal string
		wantOk                 bool
	}{
		{"  HostName example.com", "HostName", "example.com", true},
		{"HostName=example.com", "HostName", "example.com", true},
		{"HostName = example.com", "HostName", "example.com", true},
		{"# comment", "", "", false},
		{"", "", "", false},
		{`IdentityFile "~/.ssh/id_rsa"`, "IdentityFile", "~/.ssh/id_rsa", true},
		{"Host myalias", "Host", "myalias", true},
		{"Host alias1 alias2", "Host", "alias1 alias2", true},
	}
	for _, c := range cases {
		k, v, ok := parseSSHConfigLine(c.line)
		if ok != c.wantOk || k != c.wantKey || v != c.wantVal {
			t.Errorf("parseSSHConfigLine(%q) = (%q, %q, %v), want (%q, %q, %v)",
				c.line, k, v, ok, c.wantKey, c.wantVal, c.wantOk)
		}
	}
}

func TestExpandSSHNode_NoMatch(t *testing.T) {
	cfg := map[string]*sshHostEntry{
		"myalias": {hostname: "real.host.com", port: "2222", user: "alice"},
	}
	if got := expandSSHNode("http://myalias:8080", cfg); got != "http://myalias:8080" {
		t.Errorf("non-SSH URL was modified: %q", got)
	}
	if got := expandSSHNode("ssh://other:22", cfg); got != "ssh://other:22" {
		t.Errorf("unknown SSH host was modified: %q", got)
	}
}

func TestExpandSSHNode_FullExpansion(t *testing.T) {
	keyPath := "/home/alice/.ssh/id_ed25519"
	cfg := map[string]*sshHostEntry{
		"myalias": {
			hostname:     "real.host.com",
			port:         "2222",
			user:         "alice",
			identityFile: keyPath,
		},
	}

	got := expandSSHNode("ssh://myalias", cfg)
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse result URL: %v", err)
	}
	if u.Hostname() != "real.host.com" {
		t.Errorf("hostname: got %q, want %q", u.Hostname(), "real.host.com")
	}
	if u.Port() != "2222" {
		t.Errorf("port: got %q, want %q", u.Port(), "2222")
	}
	if u.User.Username() != "alice" {
		t.Errorf("user: got %q, want %q", u.User.Username(), "alice")
	}
	if q := u.Query().Get("privateKeyFile"); q != keyPath {
		t.Errorf("privateKeyFile param: got %q, want %q", q, keyPath)
	}
}

func TestExpandSSHNode_PreserveExisting(t *testing.T) {
	cfg := map[string]*sshHostEntry{
		"myalias": {
			hostname:     "real.host.com",
			port:         "2222",
			user:         "alice",
			identityFile: "/home/alice/.ssh/id_ed25519",
		},
	}

	got := expandSSHNode("ssh://bob@myalias:22?key=/tmp/other_key", cfg)
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse result URL: %v", err)
	}
	if u.Hostname() != "real.host.com" {
		t.Errorf("hostname: got %q, want %q", u.Hostname(), "real.host.com")
	}
	// Port 22 is default; config has 2222 → should be substituted.
	if u.Port() != "2222" {
		t.Errorf("port: got %q, want %q", u.Port(), "2222")
	}
	// Existing user in URL must be preserved.
	if u.User.Username() != "bob" {
		t.Errorf("user: got %q, want %q", u.User.Username(), "bob")
	}
	// Existing TLS key param must be preserved unchanged.
	if q := u.Query().Get("key"); q != "/tmp/other_key" {
		t.Errorf("key param: got %q, want %q", q, "/tmp/other_key")
	}
	// privateKeyFile from config must be added (key is TLS, not SSH).
	if q := u.Query().Get("privateKeyFile"); q != "/home/alice/.ssh/id_ed25519" {
		t.Errorf("privateKeyFile param: got %q, want %q", q, "/home/alice/.ssh/id_ed25519")
	}
}

func TestExpandSSHNode_PrivateKeyFileInURL_Preserved(t *testing.T) {
	cfg := map[string]*sshHostEntry{
		"srv": {hostname: "1.2.3.4", identityFile: "/home/user/.ssh/id_rsa"},
	}
	got := expandSSHNode("ssh://srv?privateKeyFile=/custom/key", cfg)
	u, _ := url.Parse(got)
	// privateKeyFile already set → identityFile from config should NOT override.
	if q := u.Query().Get("privateKeyFile"); q != "/custom/key" {
		t.Errorf("privateKeyFile param: got %q, want %q", q, "/custom/key")
	}
}

func TestReadSSHConfig(t *testing.T) {
	cleanup := writeSSHConfig(t, `
# global defaults
Host *
    ServerAliveInterval 60

Host bastion
    HostName bastion.example.com
    Port 2222
    User deploy
    IdentityFile ~/.ssh/bastion_key

Host dev
    HostName 10.0.0.5
    User dev
`)
	defer cleanup()

	cfg := readSSHConfig()
	if cfg == nil {
		t.Fatal("readSSHConfig returned nil")
	}

	e, ok := cfg["bastion"]
	if !ok {
		t.Fatal("expected 'bastion' entry")
	}
	if e.hostname != "bastion.example.com" {
		t.Errorf("hostname: got %q", e.hostname)
	}
	if e.port != "2222" {
		t.Errorf("port: got %q", e.port)
	}
	if e.user != "deploy" {
		t.Errorf("user: got %q", e.user)
	}
	if filepath.Base(e.identityFile) != "bastion_key" {
		t.Errorf("identityFile: got %q", e.identityFile)
	}

	dev, ok := cfg["dev"]
	if !ok {
		t.Fatal("expected 'dev' entry")
	}
	if dev.hostname != "10.0.0.5" {
		t.Errorf("dev hostname: got %q", dev.hostname)
	}

	// Wildcard Host * must not be indexed by exact key.
	if _, ok := cfg["*"]; ok {
		t.Error("wildcard Host * should not appear in entries map")
	}
}

func TestExpandSSHNodes_Integration(t *testing.T) {
	cleanup := writeSSHConfig(t, `
Host jump
    HostName jump.corp.example.com
    Port 2200
    User ops
    IdentityFile ~/.ssh/jump_key
`)
	defer cleanup()

	nodes := []string{
		"ssh://jump",
		"http://proxy:8080",
		"ssh://realhost:22",
	}
	got := expandSSHNodes(nodes)

	u0, _ := url.Parse(got[0])
	if u0.Hostname() != "jump.corp.example.com" {
		t.Errorf("[0] hostname: got %q", u0.Hostname())
	}
	if u0.Port() != "2200" {
		t.Errorf("[0] port: got %q", u0.Port())
	}
	if u0.User.Username() != "ops" {
		t.Errorf("[0] user: got %q", u0.User.Username())
	}
	if got[1] != "http://proxy:8080" {
		t.Errorf("[1] non-SSH URL changed: %q", got[1])
	}
	// ssh://realhost:22 — not in config, unchanged.
	if got[2] != "ssh://realhost:22" {
		t.Errorf("[2] unknown host changed: %q", got[2])
	}
}
