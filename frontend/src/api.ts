export interface Profile { id:string; name:string; protocol:'vless'|'openvpn'; host:string; port:number; transport:string; security:string; requiresAuth:boolean; warnings:string[] }
export interface State { systemLocale?:string; version:string; profiles:Profile[]; selectedId:string; storageMode:'memory'; csrfToken:string; connection:{status:'disconnected'|'connecting'|'connected'|'disconnecting'|'error'; profileId:string; engineAvailable:boolean; mode:'system-proxy'; errorCode?:string} }
export interface Diagnostics {version:string;os:string;arch:string;storageMode:'memory';engineAvailable:boolean;profileCount:number}
export const apiErrorMessages = {
  ENGINE_UNAVAILABLE: 'VPN-ядро недоступно. Подключение невозможно.',
  UNSUPPORTED_PROFILE: 'Подключение доступно только для VLESS. OpenVPN пока недоступен.',
  CONNECT_FAILED: 'Не удалось подключиться. Проверьте профиль и повторите попытку.',
  PROXY_FAILED: 'Не удалось включить системный прокси.',
  RESTORE_FAILED: 'Не удалось восстановить настройки прокси. Проверьте системные настройки вручную.',
  BUSY: 'Операция уже выполняется. Дождитесь её завершения.',
  REQUEST_FAILED: 'Не удалось выполнить запрос. Проверьте данные и повторите попытку.',
} as const;
export type ApiErrorCode = keyof typeof apiErrorMessages;
export function safeErrorCode(code:unknown):ApiErrorCode {
  return typeof code === 'string' && Object.hasOwn(apiErrorMessages,code) ? code as ApiErrorCode : 'REQUEST_FAILED';
}
export class ApiError extends Error {
  readonly code:ApiErrorCode;
  constructor(code:unknown) { const safe=safeErrorCode(code);super(apiErrorMessages[safe]);this.name='ApiError';this.code=safe; }
}
export async function request<T>(path:string, init:RequestInit = {}):Promise<T> {
  const response = await fetch(`/api${path}`, {cache:'no-store', ...init});
  if (!response.ok) {
    let code:unknown;
    try {const body:unknown=await response.json();if(body && typeof body==='object')code=(body as {code?:unknown;errorCode?:unknown}).code ?? (body as {errorCode?:unknown}).errorCode;}catch { /* Never expose response text. */ }
    throw new ApiError(code);
  }
  return response.json() as Promise<T>;
}
