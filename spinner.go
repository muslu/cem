package main

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type Spinner struct {
	msg      string
	stop     chan struct{}
	done     chan struct{}
	disabled atomic.Bool
	// once — Stop() birden fazla yerden çağrılıyor: stopWriter.Write
	// (subprocess'in ilk stderr byte'ında, AYRI goroutine) ve Run/ModePair
	// (captureTool döndükten sonra). Korumasız close(s.stop) ikinci çağrıda
	// "panic: close of closed channel" veriyordu (2026-09-09'da sahada görüldü).
	once sync.Once
}

// StartSpinner — verilen mesajla spinner başlatır. Stop() ile sonlandır.
// TTY değilse sessiz çalışır (stop/start no-op).
func StartSpinner(msg string) *Spinner {
	s := &Spinner{
		msg:  msg,
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}

	info, _ := os.Stderr.Stat()
	if info == nil || (info.Mode()&os.ModeCharDevice) == 0 {
		s.disabled.Store(true)
		close(s.done)
		return s
	}

	go s.run()
	return s
}

func (s *Spinner) run() {
	defer close(s.done)
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	i := 0
	// İmleci gizle
	fmt.Fprint(os.Stderr, "\033[?25l")
	defer fmt.Fprint(os.Stderr, "\033[?25h")

	for {
		select {
		case <-s.stop:
			// Satırı temizle
			fmt.Fprint(os.Stderr, "\r\033[K")
			return
		case <-ticker.C:
			frame := spinnerFrames[i%len(spinnerFrames)]
			fmt.Fprintf(os.Stderr, "\r  %s %s", colorAccent.Render(frame), s.msg)
			i++
		}
	}
}

// Stop — idempotent ve goroutine-safe. Kaç kez çağrılırsa çağrılsın spinner
// bir kez durur; her çağıran, spinner gerçekten durduktan sonra döner.
func (s *Spinner) Stop() {
	if s == nil || s.disabled.Load() {
		return
	}
	s.once.Do(func() {
		close(s.stop)
		<-s.done
	})
}
