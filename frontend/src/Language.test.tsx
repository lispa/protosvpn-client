import {render,screen,cleanup} from '@testing-library/react';
import {it,expect,vi,afterEach} from 'vitest';
import userEvent from '@testing-library/user-event';
import App from './App';
import {translate,resolveLanguage} from './i18n';
const state={version:'0.1.0',profiles:[],selectedId:'',storageMode:'memory',csrfToken:'test',connection:{status:'disconnected',profileId:'',engineAvailable:false}};
afterEach(()=>{cleanup();localStorage.clear();vi.restoreAllMocks();vi.unstubAllGlobals();});
function setup(locale:string,payload:unknown=state){vi.spyOn(navigator,'language','get').mockReturnValue(locale);vi.spyOn(navigator,'languages','get').mockReturnValue([locale]);vi.stubGlobal('fetch',vi.fn(async()=>new Response(JSON.stringify(payload))));}
it('defaults to system English and switches to Russian without restarting',async()=>{
 setup('en-US');const user=userEvent.setup();render(<App/>);
 expect(await screen.findByRole('heading',{name:'No profiles'})).toBeInTheDocument();
 await user.click(screen.getByRole('button',{name:'Settings'}));
 await user.selectOptions(screen.getByLabelText('Application language'),'ru');
 await user.click(screen.getByRole('button',{name:'Save'}));
 expect(await screen.findByRole('heading',{name:'Нет профилей'})).toBeInTheDocument();expect(localStorage.getItem('protosvpn.language')).toBe('ru');expect(document.documentElement.lang).toBe('ru');
});
it.each(['ru-RU','ru-FI','ru_RU.UTF-8'])('uses Russian system locale %s',async locale=>{setup(locale);render(<App/>);expect(await screen.findByRole('heading',{name:'Нет профилей'})).toBeInTheDocument();});
it('falls back to English for Finnish and invalid stored choice',async()=>{setup('fi-FI');localStorage.setItem('protosvpn.language','unsupported');render(<App/>);expect(await screen.findByRole('heading',{name:'No profiles'})).toBeInTheDocument();});
it('persists manual English across remount and preserves profile names',async()=>{
 setup('ru-RU',{...state,selectedId:'p1',profiles:[{id:'p1',name:'Настройки / Москва',host:'192.0.2.1',port:443,protocol:'vless',transport:'tcp',security:'tls',warnings:[],requiresAuth:false}]});
 localStorage.setItem('protosvpn.language','en');const view=render(<App/>);expect(await screen.findByRole('button',{name:'Select Настройки / Москва'})).toBeInTheDocument();view.unmount();render(<App/>);expect(await screen.findByRole('button',{name:'Select Настройки / Москва'})).toBeInTheDocument();
});
it('cancel does not apply language and system option restores auto detection',async()=>{
 setup('en-US');const user=userEvent.setup();render(<App/>);await screen.findByRole('heading',{name:'No profiles'});await user.click(screen.getByRole('button',{name:'Settings'}));await user.selectOptions(screen.getByLabelText('Application language'),'ru');await user.click(screen.getByRole('button',{name:'Cancel'}));expect(screen.getByRole('button',{name:'Settings'})).toBeInTheDocument();expect(localStorage.getItem('protosvpn.language')).toBeNull();
 await user.click(screen.getByRole('button',{name:'Settings'}));await user.selectOptions(screen.getByLabelText('Application language'),'ru');await user.click(screen.getByRole('button',{name:'Save'}));await user.click(screen.getByRole('button',{name:'Настройки'}));await user.selectOptions(screen.getByLabelText('Язык приложения'),'system');await user.click(screen.getByRole('button',{name:'Сохранить'}));expect(await screen.findByRole('heading',{name:'No profiles'})).toBeInTheDocument();
});
it('uses native OS language supplied by Go instead of WebView default',async()=>{setup('en-US',{...state,systemLocale:'ru'});vi.stubGlobal('runtime',{});render(<App/>);expect(await screen.findByRole('heading',{name:'Нет профилей'})).toBeInTheDocument();});
it('localizes known technical warnings without altering unknown text',()=>{expect(translate('ru','Syntax validated only; network authentication has not been tested.')).toBe('Проверен только синтаксис; подключение к серверу не проверялось.');expect(translate('en','unchanged')).toBe('unchanged');expect(resolveLanguage('fi-FI')).toBe('en');});
