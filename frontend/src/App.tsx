import {useI18n, I18nProvider} from "./i18n";
import {WindowControls, isDesktop} from './WindowControls';
import { useEffect, useState, type CSSProperties } from 'react';
import { request, type State } from './api';
import { ImportDialog } from './ImportDialog';
import { ProfileDialog } from './ProfileDialog';
import { readScale, SettingsDialog } from './SettingsDialog';
import { DiagnosticsDialog } from './DiagnosticsDialog';
import styles from './App.module.css';
import './desktop.css';

type Dialog = 'import' | 'rename' | 'delete' | 'settings' | 'diagnostics' | null;
export default function App(){return <I18nProvider><ClientApp/></I18nProvider>;}
function ClientApp() {
 const {t,setSystemLocale}=useI18n();
  const [state, setState] = useState<State>();
  const [error, setError] = useState('');
  const [modal, setModal] = useState<Dialog>(null);
  const [busy, setBusy] = useState(false);
  const [scale, setScale] = useState(readScale);
  const selected = state?.profiles.find(profile => profile.id === state.selectedId);
  async function refresh() { const next=await request<State>('/state'); if("runtime" in window && next.systemLocale) setSystemLocale(next.systemLocale); setState(next); }
  useEffect(() => { void refresh().catch(() => setError(t("Локальная служба недоступна."))); }, []);
  async function mutate(path: string, method: string, body?: unknown) {
    await request(path, {
      method,
      headers: { 'Content-Type': 'application/json', 'X-ProtosVPN-Token': state?.csrfToken || '' },
      ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    });
    await refresh();
  }
  async function select(id: string) {
    setBusy(true); setError('');
    try { await mutate('/select', 'POST', { id }); }
    catch { setError(t("Не удалось выбрать профиль. Повторите попытку.")); }
    finally { setBusy(false); }
  }
  const close = () => setModal(null);
  return <div className={`${styles.workspace}${isDesktop() ? ' desktop-workspace' : ''}`} style={{ '--ui-scale': scale } as CSSProperties}>
    <main className="app-shell" inert={modal !== null} aria-hidden={modal !== null}>
      <header className="titlebar">
        <span className="brandmark" aria-hidden="true">P</span><h1>ProtosVPN</h1>
        <span className="gold-rule" aria-hidden="true"/><span className="edition">RETRO NETWORK <span>/ 01</span></span>
        <WindowControls/>
      </header>
      <div className="app-inner">
        {error && <div className="error-banner" role="alert"><span>{t(error)}</span><button onClick={() => { setError(''); void refresh().catch(() => setError(t("Локальная служба недоступна."))); }}>{t("Повторить")}</button></div>}
        <section className="display" aria-labelledby="connection-heading">
          <div className="display-top"><h2 id="connection-heading">{t("Выбранный профиль")}</h2><span className="connection-badge"><i aria-hidden="true"/>{t("Не подключено")}</span></div>
          <div className="readout">
            <div className="profile-readout"><div className="location" title={selected?.name}>{selected?.name || t("Профиль не выбран")}</div><div className="protocol">{selected ? `${selected.protocol.toUpperCase()} · ${selected.transport.toUpperCase()}${selected.security ? ` / ${selected.security.toUpperCase()}` : ''}` : 'VLESS / OPENVPN'}<span>{selected ? `${selected.host.includes(':') ? `[${selected.host}]` : selected.host}:${selected.port}` : t("Импортируйте конфигурацию для начала работы")}</span></div></div>
            <div className="session"><span>{t("Время сеанса")}</span><strong>—</strong></div>
          </div>
          <div className="network-stats">
            <div className="stat"><span>{t("↓ Загрузка")}</span><strong>— <small>{t("КБ/с")}</small></strong></div>
            <div className="stat"><span>{t("↑ Отдача")}</span><strong>— <small>{t("КБ/с")}</small></strong></div>
            <div className="stat"><span>{t("Задержка")}</span><strong>— <small>{t("мс")}</small></strong></div>
            <div className="traffic"><span>{t("Трафик / нет данных")}</span><div className="empty-graph" aria-label={t("Данные трафика недоступны")}><span>—</span></div></div>
          </div>
        </section>
        <div className="connection-actions"><button className="connect" disabled aria-describedby="engine"><span aria-hidden="true">⏻</span> {t("Подключить")}</button><span className="no-tunnel">{t("Туннель не создан")}</span><button onClick={() => setModal('settings')}>{t("Настройки")}</button></div>
        <p id="engine" className="engine-note">{t("VPN-ядро и системный помощник не установлены. Подключение пока недоступно.")}</p>
        <section className="profile-section" aria-labelledby="profiles-heading">
          <div className="section-title"><h2 id="profiles-heading">{t("Подключения")}</h2><span className="section-rule"/><span className="profile-count">{t("Профилей:")+" "}{state ? state.profiles.length : '—'}</span></div>
          <div className="profiles" aria-busy={busy || !state}>
            <div className="profile-columns" aria-hidden="true"><span>№</span><span>{t("Профиль / сервер")}</span><span>{t("Протокол")}</span><span className="ping">Ping</span><span className="row-status">{t("Состояние")}</span></div>
            {!state ? <div className="empty-state"><span className="empty-symbol" aria-hidden="true">{error ? '!' : '·'}</span><h3>{error ? t("Нет связи с приложением") : t("Загрузка профилей…")}</h3><p>{error ? t("Проверьте, запущена ли локальная служба.") : t("Читаем локальное состояние.")}</p></div> : state.profiles.length === 0 ? <div className="empty-state"><span className="empty-symbol" aria-hidden="true">+</span><h3>{t("Нет профилей")}</h3><p>{t("Добавьте ссылку VLESS или файл OpenVPN.")}<br/>{t("Конфигурация останется на этом устройстве.")}</p><button className="text-button" onClick={() => setModal('import')}>{t("Импортировать первый профиль")+" "}<span aria-hidden="true">→</span></button></div> : <ul className="profile-list" aria-label={t("Профили")}>{state.profiles.map((profile, index) => <li key={profile.id}><button className={`profile-row ${profile.id === state.selectedId ? 'selected' : ''}`} aria-label={t("Выбрать")+" "+profile.name} aria-pressed={profile.id === state.selectedId} disabled={busy} onClick={() => void select(profile.id)}><span className="row-index">{String(index + 1).padStart(2, '0')}</span><span className="row-name"><b>{profile.name}</b><small>{profile.host}{profile.requiresAuth ? t(" · Нужны учётные данные") : ''}</small></span><span className="row-protocol">{profile.protocol.toUpperCase()}<small>{profile.security || profile.transport}</small></span><span className="ping">—</span><span className="row-status">{profile.id === state.selectedId ? t("○ Выбран") : t("Не подключён")}</span></button></li>)}</ul>}
          </div>
          {selected?.requiresAuth && <p className="profile-note">{t("Профилю нужны учётные данные. Ввод пароля станет доступен после интеграции VPN-ядра.")}</p>}
          {selected?.warnings?.map((warning, index) => <p className="profile-note" key={index}>{t(warning)}</p>)}
          <div className="library-actions"><button disabled={!state || busy} onClick={() => setModal('import')}>{t("+ Импорт")}</button><button disabled={!selected || busy} onClick={() => setModal('rename')}>{t("Переименовать")}</button><button className="subtle-button" disabled={!selected || busy} onClick={() => setModal('delete')}>{t("Удалить")}</button><span className="formats">.ovpn <span>/</span> vless://</span></div>
        </section>
        <aside className="memory-notice"><span aria-hidden="true">i</span><p><strong>{t("Первый этап · без VPN-подключения")}</strong>{t("Профили хранятся только в памяти и исчезнут при закрытии приложения. Трафик, DNS и системные маршруты не изменяются.")}</p></aside>
      </div>
      <footer className="statusbar"><span>PROTOSVPN <span className="footer-extra">· CLASSIC INSPIRED</span></span><button className="text-button" onClick={() => setModal('diagnostics')}>{t("Диагностика")}</button><span className="footer-extra">{state ? `v${state.version}` : 'LOCAL CLIENT'}</span></footer>
    </main>
    {modal === 'import' && <ImportDialog onClose={close} onImport={data => mutate('/import', 'POST', data)}/>}
    {modal === 'settings' && <SettingsDialog scale={scale} onClose={close} onSave={value => { setScale(value); try { localStorage.setItem('protosvpn.uiScale', value); } catch { /* Preference remains session-only when storage is unavailable. */ } }}/>}
    {modal === 'diagnostics' && <DiagnosticsDialog onClose={close}/>}
    {(modal === 'rename' || modal === 'delete') && selected && <ProfileDialog kind={modal} profile={selected} onClose={close} onSave={name => mutate(`/profiles/${encodeURIComponent(selected.id)}`, modal === 'rename' ? 'PATCH' : 'DELETE', modal === 'rename' ? { name } : undefined)}/>}
  </div>;
}
