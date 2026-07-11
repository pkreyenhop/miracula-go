package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEndToEndtestMiracula(t *testing.T) {
	// 1. Create a temp directory for compiling the binary
	tmpDir, err := os.MkdirTemp("", "miracula-e2e")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	binPath := filepath.Join(tmpDir, "miracula")
	buildCmd := exec.Command("go", "build", "-o", binPath, "main.go")
	buildCmd.Dir = "."
	var buildStderr bytes.Buffer
	buildCmd.Stderr = &buildStderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to compile binary: %v\nStderr: %s", err, buildStderr.String())
	}

	// 2. Locate the test_miracula.m script
	testScriptPath := filepath.Join("..", "..", "test_miracula.m")
	if _, err := os.Stat(testScriptPath); os.IsNotExist(err) {
		t.Fatalf("test_miracula.m not found at %s", testScriptPath)
	}

	// 3. Execute the binary, piping "main\n" to stdin from root directory
	runCmd := exec.Command(binPath, "test_miracula.m")
	runCmd.Dir = "../.."
	stdin, err := runCmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	runCmd.Stdout = &stdoutBuf
	runCmd.Stderr = &stderrBuf

	if err := runCmd.Start(); err != nil {
		t.Fatalf("Failed to start miracula process: %v", err)
	}

	_, err = io.WriteString(stdin, "main\n")
	if err != nil {
		t.Fatalf("Failed to write to stdin: %v", err)
	}
	stdin.Close()

	if err := runCmd.Wait(); err != nil {
		t.Fatalf("miracula process returned error: %v\nStderr: %s\nStdout: %s", err, stderrBuf.String(), stdoutBuf.String())
	}

	stdoutStr := stdoutBuf.String()
	if !strings.Contains(stdoutStr, "ALL TESTS PASSED!") {
		t.Errorf("Expected output to contain 'ALL TESTS PASSED!', got:\n%s", stdoutStr)
	}

	if strings.Contains(stdoutStr, "[FAIL]") {
		t.Errorf("Found failing tests in output:\n%s", stdoutStr)
	}
}
