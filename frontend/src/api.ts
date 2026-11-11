export interface Profile { id:string; name:string; protocol:'vless'|'openvpn'; host:string; port:number; transport:string; security:string; requiresAuth:boolean; warnings:string[] }
export interface State { systemLocale?:string; version:string; profiles:Profile[]; selectedId:string; storageMode:'memory'; csrfToken:string; connection:{status:'disconnected'; profileId:string; engineAvailable:false} }
export interface Diagnostics {version:string;os:string;arch:string;storageMode:'memory';engineAvailable:boolean;profileCount:number}
export async function request<T>(path:string, init:RequestInit = {}):Promise<T> {
  const response = await fetch(`/api${path}`, {cache:'no-store', ...init});
  if (!response.ok) throw new Error('Не удалось выполнить запрос. Проверьте данные и повторите попытку.');
  return response.json() as Promise<T>;
}
