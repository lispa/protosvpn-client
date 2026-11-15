import {render,screen,cleanup} from '@testing-library/react';
import {it,expect,vi,afterEach} from 'vitest';
import userEvent from '@testing-library/user-event';
import App from './App';
afterEach(()=>{cleanup();localStorage.clear();vi.unstubAllGlobals();});
it('allows retrying Disconnect when restoring system proxy failed',async()=>{
 localStorage.setItem('protosvpn.language','en');
 let restored=false;
 const calls:string[]=[];
 vi.stubGlobal('fetch',vi.fn(async(input:string,init?:RequestInit)=>{
  const path=String(input);calls.push(path);
  if(path==='/api/disconnect'){expect(init?.method).toBe('POST');restored=true;return new Response('{}');}
  return new Response(JSON.stringify({version:'0.2.0',profiles:[],selectedId:'',storageMode:'memory',csrfToken:'test',connection:{engineAvailable:true,mode:'system-proxy',profileId:'p1',status:restored?'disconnected':'error',errorCode:restored?undefined:'RESTORE_FAILED'}}));
 }));
 render(<App/>);const user=userEvent.setup();
 const button=await screen.findByRole('button',{name:'Disconnect'});expect(button).toBeEnabled();await user.click(button);
 expect(calls).toContain('/api/disconnect');expect(await screen.findByRole('button',{name:'Connect'})).toBeDisabled();
});
