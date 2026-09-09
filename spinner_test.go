package main

import (
	"sync"
	"testing"
)

// newTestSpinner — run() goroutine'ini başlatmadan, Stop()'un kanal mantığını
// test edilebilir hâle getirir (done kapalı → Stop hemen döner).
func newTestSpinner() *Spinner {
	s := &Spinner{stop: make(chan struct{}), done: make(chan struct{})}
	close(s.done)
	return s
}

// TestSpinnerStopIdempotent — sahada görülen panic: pair modunda stopWriter
// (subprocess ilk stderr byte'ı) ve Run ardışık Stop() çağırıyordu →
// "panic: close of closed channel", spinner.go:67.
func TestSpinnerStopIdempotent(t *testing.T) {
	s := newTestSpinner()
	s.Stop()
	s.Stop() // panic etmemeli
	s.Stop()
}

func TestSpinnerStopEsZamanli(t *testing.T) {
	s := newTestSpinner()
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); s.Stop() }()
	}
	wg.Wait()
}

func TestSpinnerStopNilGuvenli(t *testing.T) {
	var s *Spinner
	s.Stop() // nil receiver panic etmemeli
}

func TestSpinnerDisabledStopNoOp(t *testing.T) {
	s := &Spinner{stop: make(chan struct{}), done: make(chan struct{})}
	s.disabled.Store(true)
	s.Stop()
	s.Stop()
	select {
	case <-s.stop:
		t.Error("disabled spinner'da stop kanalı kapatıldı")
	default:
	}
}
