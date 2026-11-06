'use strict';

(function (scope) {
  function setup({ dialog, confirmButton, cancelButton, lifecycle, subscribe, notify }) {
    let busy = false;
    subscribe('app:close-requested', () => {
      if (dialog.open) return;
      busy = false;
      confirmButton.disabled = false;
      confirmButton.textContent = 'Encerrar';
      cancelButton.disabled = false;
      dialog.showModal();
      cancelButton.focus();
    });
    async function cancel() {
      if (busy) return;
      busy = true;
      try { await lifecycle.CancelExit(); dialog.close(); }
      catch (e) { notify(String(e)); }
      finally { busy = false; }
    }
    confirmButton.onclick = async () => {
      if (busy) return;
      busy = true;
      confirmButton.disabled = true;
      cancelButton.disabled = true;
      confirmButton.textContent = 'Encerrando…';
      try { await lifecycle.ConfirmExit(); }
      catch (e) {
        busy = false;
        confirmButton.disabled = false;
        cancelButton.disabled = false;
        confirmButton.textContent = 'Encerrar';
        notify(String(e));
      }
    };
    cancelButton.onclick = cancel;
    dialog.oncancel = e => { e.preventDefault(); return cancel(); };
  }
  scope.TermiaShellExit = Object.freeze({ setup });
})(globalThis);
