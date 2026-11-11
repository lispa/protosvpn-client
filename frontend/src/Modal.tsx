import {useI18n} from "./i18n";
import { useEffect, useRef, type ReactNode } from 'react';

export function Modal({title, children, onClose, busy=false}:{title:string;children:ReactNode;onClose:()=>void;busy?:boolean}) {
 const {t}=useI18n();

 const ref = useRef<HTMLDivElement>(null);
 useEffect(()=>{
  const previous = document.activeElement as HTMLElement | null;
  const first = ref.current?.querySelector<HTMLElement>('input,textarea,select,button'); first?.focus();
  return ()=>previous?.focus();
 },[]);
 return <div className="modal-backdrop" onMouseDown={event=>{if(event.target===event.currentTarget && !busy) onClose();}}>
  <div className="modal" role="dialog" aria-modal="true" aria-labelledby="modal-title" ref={ref} onKeyDown={event=>{
   if(event.key==='Escape'){event.preventDefault(); if(!busy) onClose();}
   if(event.key==='Tab'){
    const targets = ref.current?.querySelectorAll<HTMLElement>('input:not(:disabled),textarea:not(:disabled),select:not(:disabled),button:not(:disabled),[tabindex="0"]');
    if(!targets?.length) return;
    const first=targets[0], last=targets[targets.length-1];
    if(event.shiftKey && document.activeElement===first){event.preventDefault();last.focus();}
    else if(!event.shiftKey && document.activeElement===last){event.preventDefault();first.focus();}
   }
  }}><header className="modal-heading"><span className="brandmark" aria-hidden="true">P</span><h2 id="modal-title">{title}</h2></header>{children}</div>
 </div>;
}
