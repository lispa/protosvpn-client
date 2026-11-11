const fs=require('node:fs'),path=require('node:path'),ts=require('../frontend/node_modules/typescript');
const names=process.argv[2]?[process.argv[2]]:['App','ImportDialog','ProfileDialog','SettingsDialog','DiagnosticsDialog','Modal'];const patches=[],keys=new Set();
for(const name of names){const file=path.resolve(__dirname,'../frontend/src/'+name+'.tsx'),old=fs.readFileSync(file,'utf8'),sf=ts.createSourceFile(file,old,ts.ScriptTarget.Latest,true,ts.ScriptKind.TSX),edits=[];
 function walk(n){
  if(ts.isJsxText(n)&&/[А-Яа-яЁё]/.test(n.text)){const key=n.text.trim().replace(/\s+/g,' ');keys.add(key);edits.push([n.getStart(sf),n.end,'{t('+JSON.stringify(key)+')'+(n.text.endsWith(' ')?'+" "':'')+'}']);return;}
  if(ts.isStringLiteral(n)&&/[А-Яа-яЁё]/.test(n.text)){keys.add(n.text);const str='t('+JSON.stringify(n.text)+')';edits.push([n.getStart(sf),n.end,ts.isJsxAttribute(n.parent)?'{'+str+'}':str]);return;}
  if(ts.isTemplateExpression(n)&&n.head.text==='Выбрать '){keys.add('Выбрать');edits.push([n.getStart(sf),n.end,'t("Выбрать")+" "+profile.name']);return;}
  ts.forEachChild(n,walk);
 }walk(sf);
 let text=old;for(const [start,end,s]of edits.sort((a,b)=>b[0]-a[0]))text=text.slice(0,start)+s+text.slice(end);
 text='import {useI18n'+(name==='App'?', I18nProvider':'')+'} from "./i18n";\n'+text;
 const pattern=name==='App'?'export default function App() {':'export function '+name+'(';
 if(name==='App')text=text.replace(pattern,'export default function App(){return <I18nProvider><ClientApp/></I18nProvider>;}\nfunction ClientApp() {\n const {t,setSystemLocale}=useI18n();');
 else{const sf2=ts.createSourceFile(file,text,ts.ScriptTarget.Latest,true,ts.ScriptKind.TSX);let pos;ts.forEachChild(sf2,n=>{if(ts.isFunctionDeclaration(n)&&n.name?.text===name)pos=n.body.getStart(sf2)+1;});if(!pos)throw Error('Missing component '+name);text=text.slice(0,pos)+'\n const {t}=useI18n();\n'+text.slice(pos);}
 if(name==='App'){text=text.replace('setState(await request<State>(\'/state\'));','const next=await request<State>(\'/state\'); if("runtime" in window && next.systemLocale) setSystemLocale(next.systemLocale); setState(next);');text=text.replace('{warning}</p>','{t(warning)}</p>').replace('<span>{error}</span>','<span>{t(error)}</span>');}
 patches.push({file,old,text});
}
if(process.env.APPLY==='1'){for(const p of patches){if(p.old.includes('from "./i18n"'))throw Error('Already localized: '+p.file);fs.writeFileSync(p.file,p.text);}console.log('Localized '+patches.length+' components');}else console.log(JSON.stringify({patches,keys:[...keys].sort()}));
