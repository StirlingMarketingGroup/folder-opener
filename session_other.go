//go:build !windows

package main

// Only Windows has the non-interactive session 0.
func inSessionZero() bool {
	return false
}
