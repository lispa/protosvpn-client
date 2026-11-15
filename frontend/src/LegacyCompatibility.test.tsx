import {act, render, screen, within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest';
import App from './App';
import type {State} from './api';

beforeEach(()=>{localStorage.clear();sessionStorage.clear();vi.spyOn(navigator,'language','get').mockReturnValue('ru-RU');vi.spyOn(navigator,'languages','get').mockReturnValue(['ru-RU']);});
afterEach(()=>{vi.useRealTimers();vi.restoreAllMocks();vi.unstubAllGlobals();});
function server(security='none') {
 const state:State={version:'test',storageMode:'memory',csrfToken:'synthetic-token',selectedId:'p1',profiles:[
  {id:'p1',name:'Legacy',protocol:'vless',host:'192.0.2.1',port:80,transport:'tcp',security,requiresAuth:false,warnings:[]},
  {id:'p2',name:'Other',protocol:'vless',host:'192.0.2.2',port:443,transport:'tcp',security:'none',requiresAuth:false,warnings:[]},
 ],connection:{status:'disconnected',profileId:'',engineAvailable:true,mode:'system-proxy'}};
 const fetcher=vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{
  if(String(input)==='/api/select')state.selectedId=JSON.parse(String(init?.body)).id;
  if(String(input)==='/api/connect')return new Response('{}');
  return new Response(JSON.stringify(state));
 });
 vi.stubGlobal('fetch',fetcher);
 return {state,fetcher,connects:()=>fetcher.mock.calls.filter(([path])=>String(path)==='/api/connect')};
}
const label='Совместимость со старым Xray (небезопасно)';
describe('Explicit legacy compatibility',()=>{
 it('shows accurate English risks before confirmation',async()=>{
  server();localStorage.setItem('protosvpn.language','en');const user=userEvent.setup();render(<App/>);
  await user.click(await screen.findByRole('checkbox',{name:'Legacy Xray compatibility (unsafe)'}));
  await user.click(screen.getByRole('button',{name:'Connect'}));
  const dialog=screen.getByRole('dialog',{name:'Unsafe compatibility'});
  expect(dialog).toHaveTextContent('official Xray v25.12.8');
  expect(dialog).toHaveTextContent('vulnerabilities');expect(dialog).toHaveTextContent('VLESS itself does not encrypt');
  expect(dialog).toHaveTextContent('HTTPS sites retain their own TLS');expect(dialog).toHaveTextContent('modern profile with TLS/Reality');
  await user.click(within(dialog).getByRole('button',{name:'Continue'}));
 });
 it.each([
  ['COMPATIBILITY_UNAVAILABLE','ru','Старое ядро совместимости недоступно'],
  ['UNSUPPORTED_COMPATIBILITY','ru','Совместимость доступна только для VLESS без TLS/Reality'],
  ['COMPATIBILITY_UNAVAILABLE','en','The legacy compatibility engine is unavailable'],
  ['UNSUPPORTED_COMPATIBILITY','en','Compatibility is available only for VLESS without TLS/Reality'],
 ])('uses safe static text for %s in %s and never falls back',async(code,language,message)=>{
  localStorage.setItem('protosvpn.language',language);
  const {fetcher,connects}=server();const normal=fetcher.getMockImplementation()!;
  fetcher.mockImplementation(async(input,init)=>String(input)==='/api/connect'?new Response(JSON.stringify({code,message:'SECRET server details'}),{status:400}):normal(input,init));
  const user=userEvent.setup();render(<App/>);await screen.findByRole('checkbox',{name:language==='ru'?label:'Legacy Xray compatibility (unsafe)'});
  await user.click(screen.getByRole('button',{name:language==='ru'?'Подключить':'Connect'}));
  expect(await screen.findByRole('alert')).toHaveTextContent(message);
  expect(connects()).toHaveLength(1);expect(connects()[0][1]?.body).toBe('{"id":"p1"}');
  expect(screen.queryByText(/SECRET/)).not.toBeInTheDocument();
 });
 it.each(['none','','tls','reality'])('defaults OFF and keeps normal request unchanged for security=%s',async security=>{
  const {connects}=server(security);const user=userEvent.setup();render(<App/>);await screen.findByRole('button',{name:'Выбрать Legacy'});
  if(security==='tls'||security==='reality')expect(screen.queryByRole('checkbox',{name:label})).not.toBeInTheDocument();
  else expect(screen.getByRole('checkbox',{name:label})).not.toBeChecked();
  await user.click(screen.getByRole('button',{name:'Подключить'}));
  expect(screen.queryByRole('dialog')).not.toBeInTheDocument();expect(connects()).toHaveLength(1);
  expect(connects()[0][1]?.body).toBe('{"id":"p1"}');
 });
 it('disables compatibility while a connect request is busy',async()=>{
  const {fetcher}=server();const normal=fetcher.getMockImplementation()!;
  let release!:(response:Response)=>void;
  fetcher.mockImplementation(async(input,init)=>String(input)==='/api/connect'?new Promise<Response>(resolve=>{release=resolve;}):normal(input,init));
  const user=userEvent.setup();render(<App/>);await screen.findByRole('checkbox',{name:label});
  await user.click(screen.getByRole('button',{name:'Подключить'}));
  expect(screen.getByRole('checkbox',{name:label})).toBeDisabled();
  await act(async()=>release(new Response('{}')));
 });
 it('keeps the active legacy warning from backend state despite unchecked controls and selection changes',async()=>{
  const {state}=server();Object.assign(state.connection,{status:'connected',profileId:'p1',compatibility:true});
  const user=userEvent.setup();render(<App/>);
  const checkbox=await screen.findByRole('checkbox',{name:label});expect(checkbox).not.toBeChecked();expect(checkbox).toBeDisabled();
  expect(screen.getByRole('alert')).toHaveTextContent('Активно небезопасное подключение');
  expect(screen.getByRole('alert')).toHaveTextContent('v25.12.8');
  await user.click(screen.getByRole('button',{name:'Выбрать Other'}));
  expect(screen.getByRole('alert')).toHaveTextContent('Активно небезопасное подключение');
  expect(screen.getByRole('checkbox',{name:label})).not.toBeChecked();
  expect(screen.getByRole('checkbox',{name:label})).toBeDisabled();
 });
 it('resets opt-in on selecting another profile and returning, including blank security',async()=>{
  server('');const user=userEvent.setup();render(<App/>);
  await user.click(await screen.findByRole('checkbox',{name:label}));
  await user.click(screen.getByRole('button',{name:'Выбрать Other'}));
  expect(screen.getByRole('checkbox',{name:label})).not.toBeChecked();
  await user.click(screen.getByRole('button',{name:'Выбрать Legacy'}));
  expect(screen.getByRole('checkbox',{name:label})).not.toBeChecked();
 });
 it('requires a fresh explicit confirmation, allows cancel without any request, then sends true only on Continue',async()=>{
  const {connects}=server();const user=userEvent.setup();render(<App/>);
  const checkbox=await screen.findByRole('checkbox',{name:label});expect(checkbox).not.toBeChecked();
  await user.click(checkbox);await user.click(screen.getByRole('button',{name:'Подключить'}));
  const dialog=screen.getByRole('dialog');expect(connects()).toHaveLength(0);
  expect(dialog).toHaveTextContent('v25.12.8');expect(dialog).toHaveTextContent('HTTPS');
  expect(within(dialog).getByRole('button',{name:'Отмена'})).toHaveFocus();
  await user.click(within(dialog).getByRole('button',{name:'Отмена'}));expect(connects()).toHaveLength(0);
  await user.click(screen.getByRole('button',{name:'Подключить'}));expect(connects()).toHaveLength(0);
  await user.click(within(screen.getByRole('dialog')).getByRole('button',{name:'Продолжить'}));
  expect(connects()).toHaveLength(1);expect(JSON.parse(String(connects()[0][1]?.body))).toEqual({id:'p1',compatibility:true});
  expect(localStorage.length).toBe(0);expect(sessionStorage.length).toBe(0);
 });
});
