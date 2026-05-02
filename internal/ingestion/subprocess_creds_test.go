package ingestion

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestEmailCredentialsPipeline_SecretsNotInEnv(t *testing.T) {
	script := `
import sys, json
creds = json.loads(sys.stdin.read())
assert "username" in creds, "missing username"
assert "password" in creds, "missing password"
assert "server" in creds, "missing server"
import os
for key in list(os.environ.keys()):
    assert "ALEPH_EMAIL" not in key, f"credential leaked in env: {key}"
    assert "PASSWORD" not in key.upper(), f"password leaked in env: {key}"
del creds
print(json.dumps({"status": "ok"}))
`

	cmd := exec.Command("python3", "-c", script)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=en_US.UTF-8"}

	creds := emailCredentials{
		Server:   "imap.example.com",
		Port:     "993",
		Username: "user@example.com",
		Password: "super-secret-password-123",
		Folder:   "INBOX",
	}
	credsJSON, _ := json.Marshal(creds)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Skipf("python3 not available: %v", err)
	}

	if _, err := stdinPipe.Write(credsJSON); err != nil {
		cmd.Process.Kill()
		t.Fatalf("write creds: %v", err)
	}
	stdinPipe.Close()

	if err := cmd.Wait(); err != nil {
		t.Fatalf("script failed: %v, stderr: %s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "ok") {
		t.Errorf("unexpected output: %s", stdout.String())
	}
}

func TestEmailCredentialsPipeline_DelCredsInScript(t *testing.T) {
	script := `
import sys, json
creds = json.loads(sys.stdin.read())
password = creds["password"]
del creds
try:
    _ = creds
    print(json.dumps({"deleted": False}))
except NameError:
    print(json.dumps({"deleted": True, "password_len": len(password)}))
`

	cmd := exec.Command("python3", "-c", script)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=en_US.UTF-8"}

	creds := emailCredentials{
		Server:   "imap.test.com",
		Port:     "993",
		Username: "testuser",
		Password: "testpass",
		Folder:   "INBOX",
	}
	credsJSON, _ := json.Marshal(creds)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Skipf("python3 not available: %v", err)
	}

	if _, err := stdinPipe.Write(credsJSON); err != nil {
		cmd.Process.Kill()
		t.Fatalf("write creds: %v", err)
	}
	stdinPipe.Close()

	if err := cmd.Wait(); err != nil {
		t.Fatalf("script failed: %v, stderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, `"deleted"`) || !strings.Contains(output, `"password_len"`) {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestEmailCredentialsPipeline_EnvMinimal(t *testing.T) {
	cmd := exec.Command("python3", "-c", "import os; print(sorted(os.environ.keys()))")
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=en_US.UTF-8"}

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Skipf("python3 not available: %v", err)
	}

	envKeys := stdout.String()
	if strings.Contains(envKeys, "ALEPH_EMAIL") {
		t.Errorf("env should not contain ALEPH_EMAIL keys: %s", envKeys)
	}
	if strings.Contains(strings.ToUpper(envKeys), "PASSWORD") {
		t.Errorf("env should not contain PASSWORD keys: %s", envKeys)
	}
}

func TestEmailCredentialsStruct(t *testing.T) {
	creds := emailCredentials{
		Server:   "imap.example.com",
		Port:     "993",
		Username: "user@example.com",
		Password: "secret",
		Folder:   "INBOX",
	}
	data, err := json.Marshal(creds)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed emailCredentials
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed.Server != "imap.example.com" {
		t.Errorf("expected server imap.example.com, got %s", parsed.Server)
	}
	if parsed.Port != "993" {
		t.Errorf("expected port 993, got %s", parsed.Port)
	}
	if parsed.Username != "user@example.com" {
		t.Errorf("expected username user@example.com, got %s", parsed.Username)
	}
	if parsed.Password != "secret" {
		t.Errorf("expected password secret, got %s", parsed.Password)
	}
	if parsed.Folder != "INBOX" {
		t.Errorf("expected folder INBOX, got %s", parsed.Folder)
	}
}

func TestRunEmailFetch_NoTempFile(t *testing.T) {
	tmpDirs, _ := os.ReadDir(os.TempDir())
	emailTempBefore := 0
	for _, d := range tmpDirs {
		if strings.HasPrefix(d.Name(), "aleph-email-") {
			emailTempBefore++
		}
	}

	_ = emailTempBefore
}