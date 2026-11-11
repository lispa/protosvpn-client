import {useI18n} from "./i18n";
import {useEffect,useState} from 'react';
import {Modal} from './Modal';
import {request,type Diagnostics} from './api';
export function DiagnosticsDialog({onClose}:{onClose:()=>void}){
 const {t}=useI18n();

 const [data,setData]=useState<Diagnostics>();const [error,setError]=useState('');const [loading,setLoading]=useState(false);
 async function load(){setLoading(true);setError('');try{setData(await request<Diagnostics>('/diagnostics'));}catch{setError(t("Не удалось получить диагностику. Локальная служба недоступна."));}finally{setLoading(false);}}
 useEffect(()=>{void load();},[]);
 return <Modal title={t("Диагностика")} onClose={onClose}><p>{t("Только техническое состояние приложения. Без конфигураций, адресов серверов и секретов.")}</p>{loading && <p role="status">{t("Загрузка…")}</p>}{error && <p role="alert">{error}</p>}{data && <dl className="diagnostics"><dt>{t("Версия")}</dt><dd>{data.version}</dd><dt>{t("Платформа")}</dt><dd>{data.os} / {data.arch}</dd><dt>{t("Хранилище")}</dt><dd>{data.storageMode==='memory'?t("Оперативная память"):t("Неизвестно")}</dd><dt>{t("VPN-ядро")}</dt><dd>{data.engineAvailable?t("Обнаружено"):t("Недоступно")}</dd><dt>{t("Профилей")}</dt><dd>{data.profileCount}</dd></dl>}<p className="muted">{t("Соединение, DNS и маршруты не изменяются. Данные диагностики не отправляются во внешние сервисы.")}</p><div className="dialog-actions"><button onClick={()=>void load()} disabled={loading}>{t("Обновить")}</button><button className="primary" onClick={onClose}>{t("Закрыть")}</button></div></Modal>;
}
