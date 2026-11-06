package main

import "sync"

// A confirmação pertence à interface do aplicativo. O segundo pedido de saída
// feito por Quit deve atravessar OnBeforeClose sem abrir outro diálogo.
type Lifecycle struct {
	mu            sync.Mutex
	pending       bool
	confirmed     bool
	hasWork       func() bool
	requestDialog func()
	quit          func()
}

func newLifecycle(hasWork func() bool, requestDialog, quit func()) *Lifecycle {
	return &Lifecycle{hasWork: hasWork, requestDialog: requestDialog, quit: quit}
}
func (l *Lifecycle) beforeClose() bool {
	l.mu.Lock()
	if l.confirmed || !l.hasWork() {
		l.mu.Unlock()
		return false
	}
	if l.pending {
		l.mu.Unlock()
		return true
	}
	l.pending = true
	l.mu.Unlock()
	l.requestDialog()
	return true
}
func (l *Lifecycle) ConfirmExit() {
	l.mu.Lock()
	if !l.pending || l.confirmed {
		l.mu.Unlock()
		return
	}
	l.confirmed = true
	l.pending = false
	l.mu.Unlock()
	l.quit()
}
func (l *Lifecycle) CancelExit() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.confirmed {
		l.pending = false
	}
}
