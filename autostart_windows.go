package main

import (
	"golang.org/x/sys/windows/registry"
)

const (
	runKeyPath  = `Software\Microsoft\Windows\CurrentVersion\Run`
	runKeyValue = "Folder Opener"
)

func enableAutostart() error {
	exe, err := executablePath()
	if err != nil {
		return err
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue(runKeyValue, `"`+exe+`"`)
}

func disableAutostart() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if err := key.DeleteValue(runKeyValue); err != nil && err != registry.ErrNotExist {
		return err
	}
	return nil
}

// autostartEnabled also checks HKLM: the MSI installs a machine-wide Run
// value there. enable/disable only manage the per-user HKCU value.
func autostartEnabled() bool {
	for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		key, err := registry.OpenKey(root, runKeyPath, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		_, _, err = key.GetStringValue(runKeyValue)
		key.Close()
		if err == nil {
			return true
		}
	}
	return false
}
