import {useI18n} from "./i18n";
import { useRef, useState, type FormEvent } from 'react';
import { Modal } from './Modal';

type ImportData={format:'vless'|'openvpn';content:string;name?:string};
export function ImportDialog({onClose,onImport}:{onClose:()=>void;onImport:(data:ImportData)=>Promise<void>}) {
 const {t}=useI18n();

 const [format,setFormat]=useState<ImportData['format']>('vless');
 const [content,setContent]=useState('');const [file,setFile]=useState<File>();const picker=useRef<HTMLInputElement>(null);
 const [error,setError]=useState(''); const [busy,setBusy]=useState(false);
 function clear(){setContent('');setFile(undefined);if(picker.current) picker.current.value='';}
 function close(){clear();onClose();}
 async function submit(event:FormEvent){
  event.preventDefault();setError('');
  if(format==='vless' && !content.trim()){setError(t("Вставьте ссылку VLESS."));return;}
  if(format==='openvpn' && !file){setError(t("Выберите файл .ovpn."));return;}
  if(file && (file.size>1048576 || !file.name.toLowerCase().endsWith('.ovpn'))){setError(t("Выберите файл .ovpn размером до 1 МиБ."));return;}
  if(new TextEncoder().encode(content).byteLength>1048576){setError(t("Конфигурация превышает 1 МиБ."));return;}
  setBusy(true);
  try{
   const data=format==='vless'?content:await new Promise<string>((resolve,reject)=>{const reader=new FileReader();reader.onload=()=>resolve(String(reader.result));reader.onerror=()=>reject(new Error('File unavailable'));reader.readAsText(file!);});
   await onImport({format,content:data});clear();onClose();
  }catch{setError(t("Не удалось импортировать профиль. Проверьте формат конфигурации. Разрешены только безопасные самодостаточные .ovpn без внешних файлов, скриптов и плагинов."));}finally{setBusy(false);}
 }
 return <Modal title={t("Импорт профиля")} onClose={close} busy={busy}><form onSubmit={submit}>
 <p>{t("Конфигурация обрабатывается локально. Импорт не проверяет подключение к серверу.")}</p>
 <label htmlFor="import-format">{t("Формат")}</label><select id="import-format" value={format} disabled={busy} onChange={e=>{clear();setError('');setFormat(e.target.value as ImportData['format']);}}><option value="vless">{t("VLESS · ссылка")}</option><option value="openvpn">{t("OpenVPN · файл .ovpn")}</option></select>
 {format==='vless'?<><label htmlFor="import-content">{t("Ссылка VLESS")}</label><textarea id="import-content" value={content} onChange={e=>setContent(e.target.value)} rows={5} spellCheck={false} autoComplete="off" autoCapitalize="off" aria-invalid={!!error} aria-describedby={error?'import-error':'import-help'} disabled={busy}/></>:<><label htmlFor="import-file">{t("Файл OpenVPN (.ovpn)")}</label><input ref={picker} id="import-file" type="file" accept=".ovpn" disabled={busy} onChange={e=>{setFile(e.target.files?.[0]);setError('');}} aria-invalid={!!error} aria-describedby={error?'import-error':'import-help'}/></>}
 <p id="import-help" className="muted">{t("До 1 МиБ. Конфигурация содержит секретные данные. Не публикуйте её.")}{format==='openvpn' && t(" Только самодостаточные клиентские конфигурации со встроенными сертификатами; без внешних файлов, скриптов и плагинов. Пароли на этом этапе не запрашиваются. Криптография сертификатов не проверяется.")}</p>
 {error && <p id="import-error" role="alert">{error}</p>}
 <div className="dialog-actions"><button type="button" onClick={close} disabled={busy}>{t("Отмена")}</button><button className="primary" disabled={busy} type="submit">{busy?t("Импорт…"):t("Импортировать")}</button></div>
 </form></Modal>;
}
