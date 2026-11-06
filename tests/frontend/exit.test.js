function assert(condition,message){if(!condition)throw new Error(message||'Assertion failed')}
function exitFixture(){
  const state={confirmed:0,cancelled:0,focus:0,opens:0,errors:[]};
  const dialog={open:false,showModal(){this.open=true;state.opens++},close(){this.open=false}};
  const confirmButton={},cancelButton={focus(){state.focus++}};
  const lifecycle={async ConfirmExit(){state.confirmed++},async CancelExit(){state.cancelled++}};
  let request;
  TermiaShellExit.setup({dialog,confirmButton,cancelButton,lifecycle,subscribe:(name,fn)=>{assert(name==='app:close-requested');request=fn},notify:e=>state.errors.push(e)});
  return {state,dialog,confirmButton,cancelButton,lifecycle,request:()=>request()};
}
test('Confirmar saída chama encerramento uma única vez',async()=>{const f=exitFixture();f.request();await f.confirmButton.onclick();await f.confirmButton.onclick();assert(f.state.confirmed===1);assert(f.confirmButton.disabled===true);assert(f.confirmButton.textContent==='Encerrando…')});
test('Cancelar mantém aplicativo e permite nova solicitação',async()=>{const f=exitFixture();f.request();await f.cancelButton.onclick();assert(f.state.cancelled===1);assert(f.state.confirmed===0);assert(!f.dialog.open);f.request();assert(f.dialog.open);assert(f.state.opens===2)});
test('Escape apenas cancela; foco inicial fica em Cancelar',async()=>{const f=exitFixture();f.request();assert(f.state.focus===1);let prevented=false;await f.dialog.oncancel({preventDefault(){prevented=true}});assert(prevented);assert(f.state.cancelled===1);assert(f.state.confirmed===0)});
test('Pedidos repetidos não empilham diálogos',()=>{const f=exitFixture();f.request();f.request();assert(f.state.opens===1)});
test('Falha de comunicação reabilita confirmação',async()=>{const f=exitFixture();f.lifecycle.ConfirmExit=async()=>{throw new Error('bridge failed')};f.request();await f.confirmButton.onclick();assert(f.state.errors.length===1);assert(!f.confirmButton.disabled);assert(f.dialog.open)});
