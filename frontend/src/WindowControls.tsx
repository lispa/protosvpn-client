import {WindowMinimise, WindowToggleMaximise, Quit} from '../wailsjs/runtime/runtime';
import {useI18n} from './i18n';

export const isDesktop = () => 'runtime' in window;

export function WindowControls() {
 const {t} = useI18n();
 if (!isDesktop()) return null;
 return <div className="window-controls">
  <button type="button" aria-label={t('Свернуть окно')} title={t('Свернуть окно')} onClick={WindowMinimise}><span aria-hidden="true">─</span></button>
  <button type="button" aria-label={t('Развернуть / восстановить окно')} title={t('Развернуть / восстановить окно')} onClick={WindowToggleMaximise}><span aria-hidden="true">□</span></button>
  <button type="button" className="window-close" aria-label={t('Закрыть приложение')} title={t('Закрыть приложение')} onClick={Quit}><span aria-hidden="true">×</span></button>
 </div>;
}
