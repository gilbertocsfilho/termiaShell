'use strict';

// Apenas valores enumerados chegam ao DOM. Nenhuma preferência é CSS ou HTML.
(function (scope) {
  const modes = Object.freeze(['system', 'light', 'dark']);
  const palettes = Object.freeze({
    dark: Object.freeze({
      background: '#0b101a', foreground: '#dce5f5', cursor: '#a0c4ff',
      selectionBackground: '#35517d', black: '#192331', red: '#f08080',
      green: '#82cca5', yellow: '#e5cb8b', blue: '#8db5ed',
      magenta: '#ba9fe5', cyan: '#80caca', white: '#dce5f5'
    }),
    light: Object.freeze({
      background: '#f7f9fd', foreground: '#243247', cursor: '#245bb0',
      selectionBackground: '#c5dafa', black: '#243247', red: '#ad2935',
      green: '#24683e', yellow: '#7b5900', blue: '#245bb0',
      magenta: '#74439a', cyan: '#146b74', white: '#5a677b',
      brightBlack: '#5a677b', brightRed: '#b72b38', brightGreen: '#23693d',
      brightYellow: '#795900', brightBlue: '#205db5', brightMagenta: '#7c429d',
      brightCyan: '#176b75', brightWhite: '#53627a'
    })
  });

  function normalizeMode(value) {
    return modes.includes(value) ? value : 'system';
  }

  function normalizeFontSize(value) {
    const n = Number(value);
    return Number.isFinite(n) && n >= 11 && n <= 24 ? Math.round(n) : 14;
  }

  function create({ root, storage, media }) {
    let mode;
    try { mode = normalizeMode(storage?.getItem('theme')); }
    catch { mode = 'system'; }
    let effective;
    const listeners = new Set();

    function apply() {
      effective = mode === 'system' ? (media.matches ? 'dark' : 'light') : mode;
      root.dataset.theme = effective;
      root.style.colorScheme = effective;
      for (const listener of listeners) listener(effective, mode);
    }
    function systemChanged() { if (mode === 'system') apply(); }
    media.addEventListener('change', systemChanged);
    apply();

    return Object.freeze({
      getMode: () => mode,
      getEffectiveTheme: () => effective,
      getTerminalTheme: () => ({ ...palettes[effective] }),
      setMode(value) {
        mode = normalizeMode(value);
        try { storage?.setItem('theme', mode); } catch { /* Quota indisponível: mantém a sessão atual. */ }
        apply();
      },
      subscribe(listener) {
        listeners.add(listener);
        listener(effective, mode);
        return () => listeners.delete(listener);
      },
      destroy() {
        media.removeEventListener('change', systemChanged);
        listeners.clear();
      }
    });
  }
  scope.TermiaShellThemes = Object.freeze({ normalizeMode, normalizeFontSize, create });
})(globalThis);
