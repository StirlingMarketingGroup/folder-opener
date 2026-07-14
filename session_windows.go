package main

import (
	"golang.org/x/sys/windows"
)

// inSessionZero reports whether the process is running in session 0 — the
// non-interactive services session. A GPO machine-context install at boot
// runs msiexec (and anything its custom actions launch) there: the server
// binds the port and answers /status, but every Explorer window it opens
// lands on session 0's invisible desktop — and the port squatting keeps the
// user's logon copy from starting.
func inSessionZero() bool {
	var sessionID uint32
	if err := windows.ProcessIdToSessionId(windows.GetCurrentProcessId(), &sessionID); err != nil {
		return false
	}
	return sessionID == 0
}
