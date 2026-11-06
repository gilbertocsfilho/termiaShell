// Fixture de unidade. Não abre navegador nem substitui revisão visual de layout.
class FixtureElement {
  constructor() { this.children=[]; this.dataset={}; this.style={}; this.value=''; this.textContent=''; this.classList={toggle(){},add(){},remove(){}}; }
  append(...items) { this.children.push(...items); }
  setAttribute(name,value) { this[name]=value; }
  replaceChildren(...items) { this.children = items; }
  addEventListener() {}
  querySelector() { return new FixtureElement(); }
  set innerHTML(_) { throw new Error('HTML não confiável foi inserido'); }
}
const fixtureElements = new Map();
const fixtureSSHButton = new FixtureElement(); fixtureSSHButton.dataset.type='ssh';
const fixtureLocalCard = new FixtureElement(); fixtureLocalCard.dataset.kind='wsl';
const fixtureSSHCard = new FixtureElement(); fixtureSSHCard.dataset.kind='ssh';
globalThis.window=globalThis;
globalThis.document={
  createElement:()=>new FixtureElement(),
  createElementNS:()=>new FixtureElement(),
  querySelector(selector){
    if(selector==='.connect-card[data-kind="wsl"]')return fixtureLocalCard;
    if(!fixtureElements.has(selector))fixtureElements.set(selector,new FixtureElement());
    return fixtureElements.get(selector);
  },
  querySelectorAll(selector){if(selector==='[data-type]')return [fixtureSSHButton];if(selector==='.connect-card')return [fixtureLocalCard,fixtureSSHCard];return []},
  addEventListener(){}
};
const fixtureForm=document.querySelector('#connection-form');
fixtureForm.elements={};for(const key of ['auth','name','host','user','port','keyPath','password','passphrase','target','directory'])fixtureForm.elements[key]=new FixtureElement();
const fixtureStorage=new Map();
globalThis.localStorage={getItem:k=>fixtureStorage.get(k)??null,setItem:(k,v)=>fixtureStorage.set(k,String(v))};
const fixtureMedia={matches:false,listeners:new Set(),addEventListener(_,f){this.listeners.add(f)},removeEventListener(_,f){this.listeners.delete(f)},change(value){this.matches=value;for(const f of this.listeners)f()}};
globalThis.appTheme=TermiaShellThemes.create({root:{dataset:{},style:{}},storage:localStorage,media:fixtureMedia});
const fixtureNativeCalls=[],fixtureEvents={};
globalThis.runtime={WindowSetLightTheme(){fixtureNativeCalls.push('light')},WindowSetDarkTheme(){fixtureNativeCalls.push('dark')},WindowSetSystemDefaultTheme(){fixtureNativeCalls.push('system')},EventsOn:(event,fn)=>fixtureEvents[event]=fn};
let fixtureOS='windows';
globalThis.go={app:{App:{GetCapabilities:async()=>({os:fixtureOS,localKind:fixtureOS==='windows'?'wsl':'local'}),ListProfiles:async()=>[],Ack(){},OpenSession:async()=>({id:'new-terminal',directory:'/',hasFiles:false}),StartReading:async()=>{}}}};
globalThis.addEventListener=()=>{};
globalThis.go.main={Lifecycle:{ConfirmExit:async()=>{},CancelExit:async()=>{}}};
globalThis.ResizeObserver=class{observe(){}};
globalThis.setTimeout=()=>1;globalThis.clearTimeout=()=>{};
globalThis.requestAnimationFrame=fn=>fn();
const fixtureTerminals=[];
globalThis.Terminal=class{
  constructor(options){this.options=options;this.osc=new Map();this.parser={registerOscHandler:(code,handler)=>this.osc.set(code,handler)};fixtureTerminals.push(this)}
  loadAddon(){} open(){} onData(){} onResize(){} attachCustomKeyEventHandler(){} focus(){}
};
globalThis.FitAddon={FitAddon:class{fit(){}}};
