import {useI18n} from "./i18n";
import {useState} from 'react';
import {Modal} from './Modal';
export const scaleOptions=['1','1.15','1.3'] as const;
export function readScale(){try{const value=localStorage.getItem('protosvpn.uiScale');return scaleOptions.includes(value as typeof scaleOptions[number])?value!:'1';}catch{return '1';}}
export function SettingsDialog({scale,onSave,onClose}:{scale:string;onSave:(value:string)=>void;onClose:()=>void}){
 const {t,preference,setPreference}=useI18n();
 const [languageChoice,setLanguageChoice]=useState(preference);

 const [value,setValue]=useState(scale);
 return <Modal title={t("Настройки")} onClose={onClose}><form onSubmit={e=>{e.preventDefault();onSave(value);setPreference(languageChoice);onClose();}}><label htmlFor="language-choice">{t('Язык приложения')}</label><select id="language-choice" value={languageChoice} onChange={e=>setLanguageChoice(e.target.value as typeof languageChoice)}><option value="system">{t('Системный')}</option><option value="ru">Русский</option><option value="en">English</option></select><p className="muted">{t('При выборе системного языка используются русский или английский. Для других языков — английский.')}</p><label htmlFor="ui-scale">{t("Масштаб интерфейса")}</label><select id="ui-scale" value={value} onChange={e=>setValue(e.target.value)}><option value="1">{t("100% · обычный")}</option><option value="1.15">{t("115% · крупный")}</option><option value="1.3">{t("130% · очень крупный")}</option></select><p className="muted">{t("Сохраняются только масштаб интерфейса и выбор языка. Конфигурации и секреты в настройки не записываются.")}</p><fieldset disabled><legend>{t("Сетевые функции · недоступны")}</legend><label className="check"><input type="checkbox"/>Kill switch</label><label className="check"><input type="checkbox"/>{t("DNS через туннель")}</label></fieldset><p className="muted">{t("Для этих функций нужны VPN-ядро и системный помощник. Защита трафика сейчас не выполняется.")}</p><div className="dialog-actions"><button type="button" onClick={onClose}>{t("Отмена")}</button><button type="submit" className="primary">{t("Сохранить")}</button></div></form></Modal>;
}
