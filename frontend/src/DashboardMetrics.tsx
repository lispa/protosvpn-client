import {useEffect,useState,type ReactNode} from 'react';
export interface ConnectionMetrics {startedAt:string;sampledAt?:string;latencyMs?:number|null;downloadBps?:number|null;uploadBps?:number|null;downloadBytes?:number|null;uploadBytes?:number|null}
function valid(value:unknown):value is number {return typeof value==='number' && Number.isFinite(value) && value>=0 && value<=Number.MAX_SAFE_INTEGER;}
function bytes(value:unknown,rate=false):string {
 if(!valid(value))return '—';
 const unit=value>=1048576?2:value>=1024?1:0;
 return `${Number((value/1024**unit).toFixed(2))} ${['B','KiB','MiB'][unit]}${rate?'/s':''}`;
}
export function DashboardMetrics({connected,metrics,t,children}:{connected:boolean;metrics?:ConnectionMetrics;t:(text:string)=>string;children?:ReactNode}) {
 const [now,setNow]=useState(Date.now);
 const started=connected && metrics?.startedAt ? Date.parse(metrics.startedAt):NaN;
 useEffect(()=>{if(!connected || !Number.isFinite(started))return;setNow(Date.now());const timer=setInterval(()=>setNow(Date.now()),1000);return ()=>clearInterval(timer);},[connected,started]);
 const current=connected?metrics:undefined;
 const seconds=Math.max(0,Math.floor((now-started)/1000));
 const time=Number.isFinite(started) && started<=now ? [Math.floor(seconds/3600),Math.floor(seconds/60)%60,seconds%60].map(value=>String(value).padStart(2,'0')).join(':'):'—';
 return <>
 <div className="readout">{children}<div className="session" aria-label={t('Время сеанса')}><span>{t('Время сеанса')}</span><strong>{time}</strong></div></div>
 <div className="network-stats">
  <div className="stat"><span>{t('↓ Загрузка')}</span><strong>{bytes(current?.downloadBps,true)}</strong></div>
  <div className="stat"><span>{t('↑ Отдача')}</span><strong>{bytes(current?.uploadBps,true)}</strong></div>
  <div className="stat" title={t('Длительность начальной HTTPS-проверки через прокси; не ICMP ping и не непрерывный мониторинг.')}><span>{t('HTTPS-проверка')}</span><strong>{valid(current?.latencyMs)?`${current.latencyMs} ${t('мс')}`:'—'}</strong></div>
  <div className="traffic"><span>{t('Трафик')}</span><p>↓ {bytes(current?.downloadBytes)} · ↑ {bytes(current?.uploadBytes)}</p><small>{t('График не ведётся. Показаны только измеренные значения; — означает, что данные недоступны.')}</small></div>
 </div></>;
}
