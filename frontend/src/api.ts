export interface Profile { id:string; name:string; protocol:'vless'|'openvpn'; host:string; port:number; transport:string; security:string; requiresAuth:boolean; warnings:string[] }
import type {ConnectionMetrics} from './DashboardMetrics';
export interface State { systemLocale?:string; version:string; profiles:Profile[]; selectedId:string; storageMode:'memory'|'protected'; storageStatus?:'ready'|'empty'|'error'; storageErrorCode?:'STORE_LOAD_FAILED'|'STORE_SAVE_FAILED'; csrfToken:string; connection:{status:'disconnected'|'connecting'|'connected'|'disconnecting'|'error'; profileId:string; engineAvailable:boolean; mode:'system-proxy'; errorCode?:string; compatibility?:boolean; metrics?:ConnectionMetrics} }
export interface Diagnostics {version:string;os:string;arch:string;storageMode:'memory'|'protected';engineAvailable:boolean;profileCount:number}
export const apiErrorMessages = {
  STORE_LOAD_FAILED: 'Не удалось загрузить сохранённые профили. Существующий файл не перезаписан. Перезапустите приложение или проверьте доступ к хранилищу.',
  STORE_SAVE_FAILED: 'Не удалось сохранить профили. Изменение не сохранено. Проверьте доступ к хранилищу и повторите попытку.',
  COMPATIBILITY_UNAVAILABLE: 'Старое ядро совместимости недоступно. Проверьте установку приложения или используйте современный профиль с TLS/Reality.',
  UNSUPPORTED_COMPATIBILITY: 'Совместимость доступна только для VLESS без TLS/Reality (security=none). Для этого профиля используйте обычное подключение.',
  ENGINE_START_FAILED: '[ENGINE_START_FAILED] Xray не запустился или локальный прокси не готов. Сбой до проверки сервера.',
  EGRESS_CHECK_FAILED: '[EGRESS_CHECK_FAILED] Xray запущен, но HTTPS-проверка через сервер не прошла. Прокси Windows ещё не включён.',
  HELPER_START_FAILED: '[HELPER_START_FAILED] Windows не удалось запустить процесс-помощник.',
  HELPER_EXITED: '[HELPER_EXITED] Процесс-помощник завершился без корректного ответа.',
  HELPER_TIMEOUT: '[HELPER_TIMEOUT] Процесс-помощник не ответил за 30 секунд.',
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
    try {
      const body:unknown=await response.json();
      if(body && typeof body==='object') {
        const payload=body as {error?:unknown;code?:unknown;errorCode?:unknown};
        const nested=payload.error;
        code=nested && typeof nested==='object'
          ? (nested as {code?:unknown}).code
          : payload.code ?? payload.errorCode;
      }
    } catch { /* Never expose response text. */ }
    throw new ApiError(code);
  }
  return response.json() as Promise<T>;
}
