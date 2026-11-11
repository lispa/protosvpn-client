import {createContext,useContext,useState,useEffect,type ReactNode} from 'react';
import {english} from './en';
export type Language='ru'|'en';
export type LanguagePreference='system'|Language;
const KEY='protosvpn.language';
export function resolveLanguage(locale:string):Language{return /^ru(?:[-_.@]|$)/i.test(locale)?'ru':'en';}
export function readLanguagePreference():LanguagePreference{try{const x=localStorage.getItem(KEY);if(x==='ru'||x==='en')return x;}catch{/* Session-only fallback. */}return 'system';}
const reverse:Record<string,string>=Object.fromEntries(Object.entries(english).map(([ru,en])=>[en,ru]));
export function translate(language:Language,text:string):string{const key=reverse[text]||text;return language==='ru'?key:(english[key]||text);}
type Context={t:(text:string)=>string;language:Language;preference:LanguagePreference;setPreference:(v:LanguagePreference)=>void;setSystemLocale:(locale:string)=>void};
const I18n=createContext<Context|null>(null);
export function I18nProvider({children}:{children:ReactNode}){
 const [preference,setChoice]=useState(readLanguagePreference);
 const [systemLocale,setSystemLocale]=useState('');
 const [,setEpoch]=useState(0);
 const language=preference==='system'?resolveLanguage(systemLocale||navigator.languages?.[0]||navigator.language||'en'):preference;
 useEffect(()=>{document.documentElement.lang=language;},[language]);
 useEffect(()=>{const changed=()=>setEpoch(n=>n+1);window.addEventListener('languagechange',changed);return()=>window.removeEventListener('languagechange',changed);},[]);
 function setPreference(value:LanguagePreference){if(!['system','ru','en'].includes(value))return;setChoice(value);try{localStorage.setItem(KEY,value);}catch{/* Keep the choice for this session. */}}
 return <I18n.Provider value={{language,preference,setPreference,setSystemLocale,t:text=>translate(language,text)}}>{children}</I18n.Provider>;
}
export function useI18n(){const value=useContext(I18n);if(!value)throw new Error('I18nProvider is required');return value;}
