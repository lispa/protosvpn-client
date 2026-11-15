import ts from 'typescript';
import {render,screen,cleanup} from '@testing-library/react';
import {afterEach,expect,it,vi} from 'vitest';
import App from './App';
import {english} from './en';

// Language names are intentionally shown in their native form.
const nativeNames=new Set(['Русский']);
const sources=import.meta.glob<string>(['./**/*.ts','./**/*.tsx','!./**/*.test.ts','!./**/*.test.tsx','!./**/*.d.ts','!./en.ts'],{eager:true,query:'?raw',import:'default'});
it('has English translations for every Russian application string',()=>{
 const missing:string[]=[];
 for(const [path,content] of Object.entries(sources)){
  const source=ts.createSourceFile(path,content,ts.ScriptTarget.Latest,true,path.endsWith('.tsx')?ts.ScriptKind.TSX:ts.ScriptKind.TS);
  function visit(node:ts.Node){
   if((ts.isStringLiteral(node)||ts.isNoSubstitutionTemplateLiteral(node)||ts.isJsxText(node))&&/[А-Яа-яЁё]/.test(node.text)){
    const text=node.text.trim();
    if(!nativeNames.has(text)&&!Object.hasOwn(english,text)&&!Object.hasOwn(english,node.text))missing.push(`${path.split('/').pop()}: ${text}`);
   }
   ts.forEachChild(node,visit);
  }
  visit(source);
 }
 expect(missing).toEqual([]);
});

afterEach(()=>{cleanup();localStorage.clear();vi.restoreAllMocks();vi.unstubAllGlobals();});
it('renders the connected English dashboard and storage notice without Russian fallback',async()=>{
 vi.spyOn(navigator,'language','get').mockReturnValue('en-US');
 vi.spyOn(navigator,'languages','get').mockReturnValue(['en-US']);
 vi.stubGlobal('fetch',vi.fn(async()=>new Response(JSON.stringify({version:'test',storageMode:'protected',storageStatus:'ready',csrfToken:'test',selectedId:'p1',profiles:[{id:'p1',name:'Test profile',host:'192.0.2.1',port:443,protocol:'vless',transport:'tcp',security:'tls',requiresAuth:false,warnings:[]}],connection:{status:'connected',profileId:'p1',engineAvailable:true,mode:'system-proxy',metrics:{startedAt:new Date().toISOString(),latencyMs:225,downloadBytes:4096,uploadBytes:2048}}}))));
 const view=render(<App/>);
 await screen.findByRole('button',{name:'Disconnect'});
 expect(screen.getByText('HTTPS check')).toBeInTheDocument();
 expect(screen.getByText('Traffic')).toBeInTheDocument();
 expect(screen.getByText(/No traffic history graph/)).toBeInTheDocument();
 expect(screen.getByText(/Profiles are saved in protected local storage/)).toBeInTheDocument();
 expect(view.container.textContent).not.toMatch(/[А-Яа-яЁё]/);
 const tooltips=[...view.container.querySelectorAll('[title]')].map(node=>node.getAttribute('title')).join(' ');
 expect(tooltips).not.toMatch(/[А-Яа-яЁё]/);
});
