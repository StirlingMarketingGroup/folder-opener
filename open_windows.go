package main

import (
	"fmt"
	"os/exec"
	"syscall"
)

func openDir(path string) error {
	return runExplorer(fmt.Sprintf(`explorer.exe "%s"`, path))
}

func revealFile(path string) error {
	// /select must be passed as one comma-joined token with the path quoted;
	// Go's default argv quoting would wrap the whole thing in quotes and
	// break Explorer's parsing, so we hand it the raw command line.
	return runExplorer(fmt.Sprintf(`explorer.exe /select,"%s"`, path))
}

// runExplorer fires Explorer and deliberately ignores its exit code:
// explorer.exe famously exits 1 even on success. openPath has already
// verified the target exists, which is the error case we care about.
//
// Do NOT set HideWindow here: Explorer honors the STARTF_USESHOWWINDOW /
// SW_HIDE hint and opens the folder window invisible.
func runExplorer(cmdLine string) error {
	cmd := exec.Command("explorer.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: cmdLine}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start explorer: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
