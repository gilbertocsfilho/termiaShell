package main

import "testing"

func TestExitWithoutWork(t *testing.T) {
	l := newLifecycle(func() bool { return false }, func() { t.Fatal("diálogo sem trabalho ativo") }, func() { t.Fatal("Quit inesperado") })
	if l.beforeClose() {
		t.Fatal("saída sem sessões bloqueada")
	}
}
func TestExitConfirmAllowsQuitReentry(t *testing.T) {
	dialogs, quits := 0, 0
	var l *Lifecycle
	l = newLifecycle(func() bool { return true }, func() { dialogs++ }, func() {
		quits++
		if l.beforeClose() {
			t.Fatal("confirmação voltou a bloquear Quit")
		}
	})
	l.ConfirmExit()
	if quits != 0 {
		t.Fatal("saída sem pedido pendente")
	}
	if !l.beforeClose() || !l.beforeClose() || dialogs != 1 {
		t.Fatal("pedido duplicado", dialogs)
	}
	l.ConfirmExit()
	l.ConfirmExit()
	if quits != 1 {
		t.Fatal("Quit deve ocorrer exatamente uma vez", quits)
	}
}
func TestExitCancelKeepsApplicationOpen(t *testing.T) {
	dialogs, quits := 0, 0
	l := newLifecycle(func() bool { return true }, func() { dialogs++ }, func() { quits++ })
	l.beforeClose()
	l.CancelExit()
	l.ConfirmExit()
	if quits != 0 {
		t.Fatal("cancelamento encerrou o aplicativo")
	}
	if !l.beforeClose() || dialogs != 2 {
		t.Fatal("novo pedido não abriu confirmação")
	}
}
