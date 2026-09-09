//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// detachProcess — arka plan güncellemesini kendi süreç grubuna alır. Aksi
// halde terminaldeki Ctrl+C (SIGINT foreground gruba gider) yarım kalmış bir
// npm/curl kurulumu bırakır.
func detachProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
