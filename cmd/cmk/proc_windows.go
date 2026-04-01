//go:build windows

package main

import "syscall"

// daemonSysProcAttr returns SysProcAttr to detach the child process on Windows.
func daemonSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: 0x00000008, // CREATE_NO_WINDOW
	}
}
