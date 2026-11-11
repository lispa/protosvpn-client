import {useI18n} from "./i18n";
import { useState, type FormEvent } from 'react';
import { Modal } from './Modal';
import type { Profile } from './api';
export function ProfileDialog({kind,profile,onClose,onSave}:{kind:'rename'|'delete';profile:Profile;onClose:()=>void;onSave:(name?:string)=>Promise<void>}) {
 const {t}=useI18n();

 const [name,setName]=useState(profile.name);const [error,setError]=useState('');const [busy,setBusy]=useState(false);
 async function submit(event:FormEvent){event.preventDefault();setError('');if(kind==='rename' && (!name.trim() || [...name.trim()].length>80)){setError(t("Введите название от 1 до 80 символов."));return;}setBusy(true);try{await onSave(name.trim());onClose();}catch{setError(t("Не удалось сохранить изменение. Повторите попытку."));}finally{setBusy(false);}}
 return <Modal title={kind==='rename'?t("Переименовать профиль"):t("Удалить профиль?")} onClose={onClose} busy={busy}><form onSubmit={submit}>{kind==='rename'?<><label htmlFor="profile-name">{t("Название профиля")}</label><input id="profile-name" value={name} onChange={e=>setName(e.target.value)} disabled={busy} aria-describedby={error?'profile-error':undefined}/></>:<><p className="delete-name">{profile.name}</p><p>{t("Профиль и его конфигурация будут удалены из памяти приложения. Отменить это действие нельзя.")}</p></>}{error && <p role="alert" id="profile-error">{error}</p>}<div className="dialog-actions"><button type="button" onClick={onClose} disabled={busy}>{t("Отмена")}</button><button className={kind==='delete'?'danger':'primary'} disabled={busy} type="submit">{busy?t("Сохранение…"):kind==='rename'?t("Сохранить"):t("Удалить профиль")}</button></div></form></Modal>;
}
