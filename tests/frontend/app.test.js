function assert(condition, message) { if (!condition) throw new Error(message || 'Assertion failed'); }

test('Configurações alteram tema de terminais já abertos e janela nativa', () => {
  const existing={term:{options:{}}};sessions.set('existing',existing);
  document.querySelector('#theme-mode').onchange({target:{value:'light'}});
  assert(existing.term.options.theme.background==='#f7f9fd');
  assert(fixtureNativeCalls.at(-1)==='light');
  document.querySelector('#theme-mode').onchange({target:{value:'dark'}});
  assert(existing.term.options.theme.background==='#0b101a');
  assert(fixtureNativeCalls.at(-1)==='dark');
  assert(fixtureStorage.get('theme')==='dark');sessions.clear();
});
test('Sistema atualiza terminais abertos ao mudar preferência do SO', () => {
  const existing={term:{options:{}}};sessions.set('existing',existing);
  document.querySelector('#theme-mode').onchange({target:{value:'system'}});
  fixtureMedia.change(false);assert(existing.term.options.theme.background==='#f7f9fd');
  fixtureMedia.change(true);assert(existing.term.options.theme.background==='#0b101a');
  assert(fixtureNativeCalls.at(-1)==='system');sessions.clear();
});
test('Nome malicioso de perfil é apresentado como texto', () => {
  profiles=[{id:'unsafe',name:'<img src=x onerror=alert(1)>',kind:'ssh',host:'example.test'}];caps.localKind='wsl';renderProfiles();
  function containsText(node,text){return node.textContent===text||node.children.some(child=>containsText(child,text))}
  assert(containsText(document.querySelector('#profiles'),profiles[0].name));profiles=[];
});
test('Saída de terminal é encaminhada como bytes, sem interpretar HTML', async () => {
  await init();let received;
  sessions.set('unsafe',{term:{write(bytes,done){received=String.fromCharCode(...bytes);done()}}});
  fixtureEvents['terminal:data']({id:'unsafe',data:'PGltZyBzcmM9eCBvbmVycm9yPWFsZXJ0KDEpPg=='});
  assert(received==='<img src=x onerror=alert(1)>');sessions.clear();
});
test('Linux apresenta terminal local; Windows apresenta WSL', async () => {
  fixtureOS='linux';await init();
  assert(fixtureLocalCard.dataset.kind==='local');
  assert(document.querySelector('#quick-local').textContent==='›_ Abrir terminal');
  fixtureOS='windows';await init();assert(fixtureLocalCard.dataset.kind==='wsl');
  assert(document.querySelector('#quick-local').textContent==='›_ Abrir WSL');
});
test('Novo terminal herda tema e bloqueia sequências OSC 52', async () => {
  window.appTheme.setMode('light');
  await connect({name:'Nova sessão',kind:'ssh',host:'example.test',port:22,user:'tester'},'','');
  const term=fixtureTerminals.at(-1);
  assert(term.options.theme.background==='#f7f9fd');
  assert(term.osc.get(52)('malicious-clipboard-payload')===true);
  window.appTheme.setMode('dark');assert(term.options.theme.background==='#0b101a');
  sessions.clear();active=null;
});
