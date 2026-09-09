//go:build windows

package main

import "os/exec"

// detachProcess — Windows'ta ayrı süreç grubu gerekmez; başlatılan süreç
// zaten ebeveyn çıkınca sonlanmaz.
func detachProcess(cmd *exec.Cmd) {}
