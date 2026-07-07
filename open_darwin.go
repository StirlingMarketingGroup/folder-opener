package main

import "os/exec"

func openDir(path string) error {
	return runQuiet(exec.Command("open", path))
}

func revealFile(path string) error {
	return runQuiet(exec.Command("open", "-R", path))
}
