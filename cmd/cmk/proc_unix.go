//go:build !windows

package main

import "syscall"

// daemonSysProcAttr returns SysProcAttr to detach the child process on Unix.
func daemonSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}
