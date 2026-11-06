'use strict';

// Executado antes do CSS para aplicar a preferência sem piscar o tema oposto.
let themeStorage;
try { themeStorage = window.localStorage; } catch { /* WebView sem armazenamento. */ }
window.appTheme = TermiaShellThemes.create({
  root: document.documentElement,
  storage: themeStorage,
  media: window.matchMedia('(prefers-color-scheme: dark)')
});
