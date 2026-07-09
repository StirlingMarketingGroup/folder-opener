package main

import (
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func openDir(path string) error {
	return runExplorer(fmt.Sprintf(`explorer.exe "%s"`, path))
}

// openDenied handles a path os.Stat can't access (ERROR_ACCESS_DENIED).
// Explorer hands new windows to the session shell, which runs as the desktop
// user — so even when THIS process is denied (it may hold a different token
// than the desktop user; see the MSI LaunchApp comment), the folder still
// opens with the user's own permissions, and if even they lack access
// Explorer shows its native permission error.
//
// FindFirstFile needs only list permission on the PARENT directory, so it
// usually still reveals whether the target is a folder even when opening the
// target itself is denied. When it too is denied the target is revealed
// (/select) rather than opened — reveal is safe for files and folders alike,
// while `explorer.exe "<file>"` would EXECUTE a file, which a localhost
// caller must never be able to trigger on a path this process can't even
// stat.
func openDenied(path string) (action string, ok bool) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", false
	}
	var data windows.Win32finddata
	handle, err := windows.FindFirstFile(pathPtr, &data)
	if err != nil {
		if err := revealFile(path); err != nil {
			return "", false
		}
		return "revealed", true
	}
	_ = windows.FindClose(handle)

	if data.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 {
		if err := revealFile(path); err != nil {
			return "", false
		}
		return "revealed", true
	}
	if err := openDir(path); err != nil {
		return "", false
	}
	return "opened", true
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
//
// Even shown, the window opens BEHIND everything with only a taskbar
// flash: Windows' foreground lock denies SetForegroundWindow to a
// background process like this server. So after launch we find the new
// Explorer window and deliberately pull it to the foreground.
func runExplorer(cmdLine string) error {
	before := explorerWindowSet()

	cmd := exec.Command("explorer.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: cmdLine}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start explorer: %w", err)
	}
	go func() { _ = cmd.Wait() }()

	go focusNewExplorerWindow(before)
	return nil
}

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	procEnumWindows         = user32.NewProc("EnumWindows")
	procGetClassNameW       = user32.NewProc("GetClassNameW")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procShowWindow          = user32.NewProc("ShowWindow")
	procIsIconic            = user32.NewProc("IsIconic")
	procKeybdEvent          = user32.NewProc("keybd_event")
	procSwitchToThisWindow  = user32.NewProc("SwitchToThisWindow")
	procBringWindowToTop    = user32.NewProc("BringWindowToTop")
	enumExplorerWindowsProc = windows.NewCallback(appendExplorerWindow)
)

const (
	swRestore          = 9
	vkMenu             = 0x12
	keyeventfKeyup     = 0x0002
	maxClassNameLength = 256
)

// enumeratedWindows collects results from the EnumWindows callback, which
// can't carry a Go pointer through LPARAM without tripping vet's
// unsafe.Pointer rules. EnumWindows is synchronous, so a mutex around each
// enumeration keeps concurrent /open requests from interleaving.
var (
	enumeratedWindowsMu sync.Mutex
	enumeratedWindows   []windows.HWND
)

// appendExplorerWindow is the EnumWindows callback: collects top-level
// windows of Explorer's folder-window classes. Visibility isn't required —
// a just-created window can enumerate before it's shown, and that's exactly
// the one we want.
func appendExplorerWindow(hwnd windows.HWND, _ uintptr) uintptr {
	var class [maxClassNameLength]uint16
	n, _, _ := procGetClassNameW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&class[0])), maxClassNameLength)
	if n == 0 {
		return 1 // keep enumerating
	}
	switch windows.UTF16ToString(class[:n]) {
	case "CabinetWClass", "ExploreWClass":
		enumeratedWindows = append(enumeratedWindows, hwnd)
	}
	return 1
}

func explorerWindows() []windows.HWND {
	enumeratedWindowsMu.Lock()
	defer enumeratedWindowsMu.Unlock()
	enumeratedWindows = nil
	_, _, _ = procEnumWindows.Call(enumExplorerWindowsProc, 0)
	return enumeratedWindows
}

func explorerWindowSet() map[windows.HWND]bool {
	seen := make(map[windows.HWND]bool)
	for _, hwnd := range explorerWindows() {
		seen[hwnd] = true
	}
	return seen
}

// focusNewExplorerWindow polls for an Explorer folder window that didn't
// exist before the launch and forces it to the foreground. Explorer opens a
// new window per invocation, so the diff finds ours; if none shows up (e.g.
// the shell recycled an existing window) we leave focus alone — the folder
// is still open, just not pulled to the front.
func focusNewExplorerWindow(before map[windows.HWND]bool) {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for _, hwnd := range explorerWindows() {
			if before[hwnd] {
				continue
			}
			forceForeground(hwnd)
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// forceForeground brings hwnd to the front from a process that has no
// foreground rights. Plain SetForegroundWindow is denied for background
// processes (the window just flashes on the taskbar), so on failure it uses
// the classic unlock: a synthetic ALT press counts as recent user input,
// which lifts the foreground lock for the next call. SwitchToThisWindow is
// the last resort — same effect as the Alt-Tab switcher.
func forceForeground(hwnd windows.HWND) {
	if isIconic, _, _ := procIsIconic.Call(uintptr(hwnd)); isIconic != 0 {
		_, _, _ = procShowWindow.Call(uintptr(hwnd), swRestore)
	}

	_, _, _ = procBringWindowToTop.Call(uintptr(hwnd))
	_, _, _ = procSetForegroundWindow.Call(uintptr(hwnd))
	if foreground, _, _ := procGetForegroundWindow.Call(); windows.HWND(foreground) == hwnd {
		return
	}

	_, _, _ = procKeybdEvent.Call(vkMenu, 0, 0, 0)
	_, _, _ = procSetForegroundWindow.Call(uintptr(hwnd))
	_, _, _ = procKeybdEvent.Call(vkMenu, 0, keyeventfKeyup, 0)
	if foreground, _, _ := procGetForegroundWindow.Call(); windows.HWND(foreground) == hwnd {
		return
	}

	_, _, _ = procSwitchToThisWindow.Call(uintptr(hwnd), 1)
}
