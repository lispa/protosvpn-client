import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
beforeEach(()=>{localStorage.clear();sessionStorage.clear();vi.spyOn(navigator,'language','get').mockReturnValue('ru-RU');vi.spyOn(navigator,'languages','get').mockReturnValue(['ru-RU']);});
import App from './App';
import type { State } from './api';
import userEvent from '@testing-library/user-event';

const initial = { version:'0.1.0', profiles:[], selectedId:'', storageMode:'memory', csrfToken:'test-token', connection:{status:'disconnected',profileId:'',engineAvailable:false} };
function server() {
  const state = structuredClone(initial) as State;
  const fetcher = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = String(input);
    if (path === '/api/import') {
      const body = JSON.parse(String(init?.body));
      const profile = {id:'p1',name:body.name || 'Test profile',protocol:body.format,host:'192.0.2.1',port:443,transport:'tcp',security:'tls',requiresAuth:false,warnings:[]};
      state.profiles.push(profile); state.selectedId = profile.id;
      return new Response(JSON.stringify({profile}), {status:201});
    }
    if (path === '/api/select') {state.selectedId=JSON.parse(String(init?.body)).id;return new Response('{"ok":true}');}
    if (path.startsWith('/api/profiles/')) {
      const id=path.split('/').pop();
      if(init?.method==='PATCH') state.profiles.find(p=>p.id===id)!.name=JSON.parse(String(init.body)).name;
      if(init?.method==='DELETE') {state.profiles=state.profiles.filter(p=>p.id!==id);state.selectedId=state.profiles[0]?.id||'';}
      return new Response('{"ok":true}');
    }
    return new Response(JSON.stringify(state), {status:200});
  });
  vi.stubGlobal('fetch', fetcher);
  return {state, fetcher};
}
describe('Stage 1', () => {
  it('fetches diagnostics on demand and renders only allowlisted non-secret fields', async () => {
    const {fetcher}=server();fetcher.mockImplementation(async(input)=>new Response(JSON.stringify(String(input)==='/api/diagnostics'?{version:'0.1.0',os:'linux',arch:'amd64',storageMode:'memory',engineAvailable:false,profileCount:0,rawConfig:'TOP-SECRET',csrfToken:'PRIVATE'}:initial)));
    const user=userEvent.setup();render(<App/>);await screen.findByText('Нет профилей');
    await user.click(screen.getByRole('button',{name:'Диагностика'}));
    expect(await screen.findByText('linux / amd64')).toBeInTheDocument();
    expect(fetcher).toHaveBeenCalledWith('/api/diagnostics',expect.objectContaining({cache:'no-store'}));
    expect(screen.queryByText(/TOP-SECRET|PRIVATE/)).not.toBeInTheDocument();
    await user.click(screen.getByRole('button',{name:'Закрыть'}));expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
  it('saves only interface scale, exposes no pretend network switches and restores focus on cancel', async () => {
    server();localStorage.clear();const user=userEvent.setup();render(<App/>);await screen.findByText('Нет профилей');
    const trigger=screen.getByRole('button',{name:'Настройки'});await user.click(trigger);
    await user.selectOptions(screen.getByLabelText('Масштаб интерфейса'),'1.15');
    await user.click(screen.getByRole('button',{name:'Сохранить'}));
    expect(localStorage.getItem('protosvpn.uiScale')).toBe('1.15');
    expect(localStorage.length).toBe(2);expect(localStorage.getItem('protosvpn.language')).toBe('system');
    await user.click(trigger);await user.selectOptions(screen.getByLabelText('Масштаб интерфейса'),'1.3');
    expect(screen.getByRole('checkbox',{name:'Kill switch'})).toBeDisabled();
    expect(screen.getByRole('checkbox',{name:'DNS через туннель'})).toBeDisabled();
    await user.keyboard('{Escape}');expect(trigger).toHaveFocus();
    expect(localStorage.getItem('protosvpn.uiScale')).toBe('1.15');localStorage.clear();
  });
  it('imports a deliberately picked ovpn file without requesting credentials and resets the picker', async () => {
    const {fetcher}=server();const user=userEvent.setup();render(<App/>);await screen.findByText('Нет профилей');
    await user.click(screen.getByRole('button',{name:'+ Импорт'}));
    await user.selectOptions(screen.getByLabelText('Формат'),'openvpn');
    const config='client\ndev tun\nremote 192.0.2.2 1194\nauth-user-pass\n';
    await user.upload(screen.getByLabelText('Файл OpenVPN (.ovpn)'),new File([config],'test.ovpn',{type:'application/x-openvpn-profile'}));
    await user.click(screen.getByRole('button',{name:'Импортировать'}));
    await screen.findByRole('button',{name:'Выбрать Test profile'});
    expect(fetcher).toHaveBeenCalledWith('/api/import',expect.objectContaining({body:JSON.stringify({format:'openvpn',content:config})}));
    expect(screen.queryByLabelText(/пароль/i)).not.toBeInTheDocument();
    await user.click(screen.getByRole('button',{name:'+ Импорт'}));
    await user.selectOptions(screen.getByLabelText('Формат'),'openvpn');
    expect((screen.getByLabelText('Файл OpenVPN (.ovpn)') as HTMLInputElement).files).toHaveLength(0);
  });
  it('selects, renames as plain text and only deletes after explicit confirmation', async () => {
    const {state,fetcher}=server();
    state.profiles.push({id:'p1',name:'Первый',protocol:'vless',host:'192.0.2.1',port:443,transport:'tcp',security:'tls',requiresAuth:false,warnings:[]},{id:'p2',name:'Второй',protocol:'openvpn',host:'192.0.2.2',port:1194,transport:'udp',security:'tls',requiresAuth:true,warnings:[]});state.selectedId='p1';
    const user=userEvent.setup();render(<App/>);
    await user.click(await screen.findByRole('button',{name:'Выбрать Второй'}));
    expect(fetcher).toHaveBeenCalledWith('/api/select',expect.objectContaining({method:'POST',body:'{"id":"p2"}'}));
    await user.click(screen.getByRole('button',{name:'Переименовать'}));
    await user.clear(screen.getByLabelText('Название профиля'));
    await user.type(screen.getByLabelText('Название профиля'),'<img src=x onerror=alert(1)>');
    await user.click(screen.getByRole('button',{name:'Сохранить'}));
    expect(await screen.findByRole('button',{name:'Выбрать <img src=x onerror=alert(1)>'})).toBeInTheDocument();
    expect(document.querySelector('img')).toBeNull();
    await user.click(screen.getByRole('button',{name:'Удалить'}));
    await user.click(screen.getByRole('button',{name:'Отмена'}));
    expect(fetcher.mock.calls.filter(call=>call[1]?.method==='DELETE')).toHaveLength(0);
    await user.click(screen.getByRole('button',{name:'Удалить'}));
    await user.click(screen.getByRole('button',{name:'Удалить профиль'}));
    expect(await screen.findByRole('button',{name:'Выбрать Первый'})).toBeInTheDocument();
    expect(screen.queryByRole('button',{name:'Выбрать <img src=x onerror=alert(1)>'})).not.toBeInTheDocument();
    expect(fetcher).toHaveBeenCalledWith('/api/profiles/p2',expect.objectContaining({method:'DELETE',headers:expect.objectContaining({'X-ProtosVPN-Token':'test-token'})}));
  });
  it('imports pasted VLESS through authenticated API and clears secrets after success and cancel', async () => {
    const {fetcher} = server(); const user = userEvent.setup(); render(<App/>);
    await screen.findByText('Нет профилей');
    await user.click(screen.getByRole('button',{name:'+ Импорт'}));
    const secret = 'vless://00000000-0000-4000-8000-000000000001@192.0.2.1:443#Test';
    await user.type(screen.getByLabelText('Ссылка VLESS'),secret);
    await user.click(screen.getByRole('button',{name:'Импортировать'}));
    expect(await screen.findByRole('button',{name:/Выбрать Test profile/})).toBeInTheDocument();
    expect(fetcher).toHaveBeenCalledWith('/api/import',expect.objectContaining({method:'POST',headers:expect.objectContaining({'X-ProtosVPN-Token':'test-token'}),body:JSON.stringify({format:'vless',content:secret})}));
    await user.click(screen.getByRole('button',{name:'+ Импорт'}));
    expect(screen.getByLabelText('Ссылка VLESS')).toHaveValue('');
    await user.type(screen.getByLabelText('Ссылка VLESS'),secret);
    await user.keyboard('{Escape}');
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button',{name:'+ Импорт'}));
    expect(screen.getByLabelText('Ссылка VLESS')).toHaveValue('');
    expect(localStorage.length).toBe(0); expect(sessionStorage.length).toBe(0);
  });
  it('bootstraps real state, shows empty library and never claims networking', async () => {
    const { fetcher } = server();
    render(<App/>);
    expect(await screen.findByText('Нет профилей')).toBeInTheDocument();
    expect(fetcher).toHaveBeenCalledWith('/api/state', expect.objectContaining({cache:'no-store'}));
    expect(screen.getByRole('button',{name:'Подключить'})).toBeDisabled();
    expect(screen.getByText(/VPN-ядро и системный помощник не установлены/)).toBeInTheDocument();
    expect(screen.getByText(/только в памяти/)).toBeInTheDocument();
  });
});
