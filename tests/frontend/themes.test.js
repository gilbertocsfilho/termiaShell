function assert(condition, message) { if (!condition) throw new Error(message || 'Assertion failed'); }
function themeFixture(stored, dark) {
  const data = new Map(stored === undefined ? [] : [['theme', stored]]);
  const media = {
    matches: dark, listeners: new Set(),
    addEventListener(_, fn) { this.listeners.add(fn); },
    removeEventListener(_, fn) { this.listeners.delete(fn); },
    change(value) { this.matches = value; for (const fn of this.listeners) fn(); }
  };
  const root = { dataset: {}, style: {} };
  const storage = { getItem: k => data.get(k) ?? null, setItem: (k, v) => data.set(k, v) };
  return { root, data, media, storage, manager: TermiaShellThemes.create({ root, storage, media }) };
}

test('Sistema é padrão e acompanha mudanças do SO', () => {
  const f = themeFixture(undefined, false);
  assert(f.manager.getMode() === 'system');
  assert(f.root.dataset.theme === 'light');
  f.media.change(true);
  assert(f.root.dataset.theme === 'dark');
  assert(f.root.style.colorScheme === 'dark');
});
test('Claro/Escuro ignoram mudanças do SO e Sistema volta a acompanhar', () => {
  const f = themeFixture('light', true);
  f.media.change(false); f.media.change(true);
  assert(f.root.dataset.theme === 'light');
  f.manager.setMode('dark'); f.media.change(false);
  assert(f.root.dataset.theme === 'dark');
  f.manager.setMode('system');
  assert(f.root.dataset.theme === 'light');
});
test('Preferência persiste e é recuperada ao reabrir', () => {
  const f = themeFixture(undefined, false);
  f.manager.setMode('dark');
  const reopened = TermiaShellThemes.create({ root: { dataset: {}, style: {} }, storage: f.storage, media: f.media });
  assert(reopened.getMode() === 'dark');
  assert(reopened.getEffectiveTheme() === 'dark');
  reopened.destroy(); f.manager.destroy();
});
test('Preferências maliciosas não chegam ao DOM como HTML/CSS', () => {
  for (const input of ['<img src=x onerror=alert(1)>', 'light; background:url(https://evil)', '__proto__', {}, null]) {
    const f = themeFixture(input, false);
    assert(f.manager.getMode() === 'system');
    f.manager.setMode(input);
    assert(f.root.dataset.theme === 'light');
    assert(f.data.get('theme') === 'system');
    f.manager.destroy();
  }
});
test('Armazenamento bloqueado ou cheio não impede troca de tema', () => {
  const storage = { getItem() { throw new Error('denied'); }, setItem() { throw new Error('quota'); } };
  const f = themeFixture(undefined, true);
  const manager = TermiaShellThemes.create({ root: f.root, storage, media: f.media });
  manager.setMode('light');
  assert(f.root.dataset.theme === 'light');
  manager.destroy(); f.manager.destroy();
});
test('Paleta de terminal é independente e listeners são liberados', () => {
  const f = themeFixture('dark', true);
  let changes = 0;
  const unsubscribe = f.manager.subscribe(() => changes++);
  const palette = f.manager.getTerminalTheme(); palette.background = 'invalid';
  assert(f.manager.getTerminalTheme().background === '#0b101a');
  f.manager.setMode('light'); assert(changes === 2);
  unsubscribe(); f.manager.setMode('dark'); assert(changes === 2);
  f.manager.destroy(); assert(f.media.listeners.size === 0);
});
test('Preferências adulteradas não permitem fontes ilimitadas', () => {
  for (const value of ['99999999', '-1', 'NaN', 'Infinity', '<script>', null, {}]) {
    assert(TermiaShellThemes.normalizeFontSize(value) === 14);
  }
  assert(TermiaShellThemes.normalizeFontSize('18') === 18);
});
test('Contraste de texto e ANSI é legível nas duas paletas', () => {
  function luminance(hex) {
    const c = [1, 3, 5].map(i => parseInt(hex.slice(i, i + 2), 16) / 255).map(v => v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4);
    return c[0] * 0.2126 + c[1] * 0.7152 + c[2] * 0.0722;
  }
  for (const mode of ['light', 'dark']) {
    const f = themeFixture(mode, false), p = f.manager.getTerminalTheme();
    for (const color of ['foreground', 'red', 'green', 'yellow', 'blue', 'magenta', 'cyan']) {
      const a = luminance(p.background), b = luminance(p[color]);
      assert((Math.max(a,b)+0.05)/(Math.min(a,b)+0.05) >= 4.5, mode + ':' + color);
    }
    f.manager.destroy();
  }
});
