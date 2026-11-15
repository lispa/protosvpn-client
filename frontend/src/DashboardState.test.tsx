import {render,screen,fireEvent} from '@testing-library/react';
import {afterEach,beforeEach,it,expect,vi} from 'vitest';
import App from './App';
const syntax='Проверен только синтаксис; подключение к серверу не проверялось.';
function state(){return {version:'test',storageMode:'protected',storageStatus:'ready',csrfToken:'token',selectedId:'stable-id',profiles:[{id:'stable-id',name:'Saved',host:'192.0.2.1',port:443,protocol:'vless',transport:'tcp',security:'tls',requiresAuth:false,warnings:[syntax]}],connection:{status:'connected',profileId:'stable-id',engineAvailable:true,mode:'system-proxy',metrics:{startedAt:new Date().toISOString(),downloadBps:2048}}};}
beforeEach(()=>{localStorage.clear();vi.spyOn(navigator,'language','get').mockReturnValue('ru-RU');vi.spyOn(navigator,'languages','get').mockReturnValue(['ru-RU']);});
afterEach(()=>{vi.restoreAllMocks();vi.unstubAllGlobals();});
it('renders actual metrics and replaces syntax-only warning only for the connected selected profile',async()=>{
 const data=state();vi.stubGlobal('fetch',vi.fn(async()=>new Response(JSON.stringify(data))));render(<App/>);
 await screen.findByRole('button',{name:'Выбрать Saved'});
 expect(screen.getByText('2 KiB/s')).toBeInTheDocument();
 expect(screen.queryByText(syntax)).not.toBeInTheDocument();
 expect(screen.getByText(/Начальная HTTPS-проверка через прокси пройдена/)).toBeInTheDocument();
});
it('reports protected persistence and reloads stable server IDs without browser profile storage',async()=>{
 const data=state();const fetcher=vi.fn(async()=>new Response(JSON.stringify(data)));vi.stubGlobal('fetch',fetcher);
 const view=render(<App/>);await screen.findByRole('button',{name:'Выбрать Saved'});
 expect(screen.getByText('Профили сохраняются в защищённом локальном хранилище и доступны после перезапуска.')).toBeInTheDocument();
 expect(screen.queryByText(/только в памяти/)).not.toBeInTheDocument();
 view.unmount();render(<App/>);
 expect(await screen.findByRole('button',{name:'Выбрать Saved'})).toHaveAttribute('aria-pressed','true');
 expect(localStorage.length).toBe(0);expect(sessionStorage.length).toBe(0);
});
it('explicitly explains an empty first-launch protected store',async()=>{
 const data=state();data.profiles=[];data.selectedId='';data.storageStatus='empty';
 vi.stubGlobal('fetch',vi.fn(async()=>new Response(JSON.stringify(data))));render(<App/>);
 expect(await screen.findByText('Сохранённых профилей нет. Импортируйте ссылку VLESS или файл OpenVPN.')).toBeInTheDocument();
});
it('retains unsafe legacy warning and disconnect after proxy restoration failure',async()=>{
 const data={...state(),connection:{...state().connection,status:'error',errorCode:'RESTORE_FAILED',compatibility:true}};
 vi.stubGlobal('fetch',vi.fn(async()=>new Response(JSON.stringify(data))));render(<App/>);
 expect(await screen.findByText('Активно небезопасное подключение')).toBeInTheDocument();
 expect(screen.getByRole('button',{name:'Отключить'})).toBeEnabled();
});
it('diagnostics reports protected storage rather than memory or unknown',async()=>{
 const data=state();vi.stubGlobal('fetch',vi.fn(async(input)=>new Response(JSON.stringify(String(input)==='/api/diagnostics'?{version:'test',os:'windows',arch:'amd64',storageMode:'protected',engineAvailable:true,profileCount:1}:data))));
 render(<App/>);await screen.findByRole('button',{name:'Выбрать Saved'});fireEvent.click(screen.getByRole('button',{name:'Диагностика'}));
 expect(await screen.findByText('Защищённое локальное хранилище')).toBeInTheDocument();
});
it('shows safe prominent storage-load failure without claiming a fresh empty store',async()=>{
 const data={...state(),profiles:[],selectedId:'',storageStatus:'error',storageErrorCode:'STORE_LOAD_FAILED',rawError:'SECRET path'};
 vi.stubGlobal('fetch',vi.fn(async()=>new Response(JSON.stringify(data))));render(<App/>);
 expect(await screen.findByRole('alert')).toHaveTextContent('Существующий файл не перезаписан.');
 expect(screen.queryByText(/SECRET/)).not.toBeInTheDocument();
 expect(screen.queryByText('Сохранённых профилей нет. Импортируйте ссылку VLESS или файл OpenVPN.')).not.toBeInTheDocument();
 expect(screen.getByRole('button',{name:'+ Импорт'})).toBeDisabled();
});
