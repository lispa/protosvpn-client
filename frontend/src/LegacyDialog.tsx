import {Modal} from './Modal';
import {useI18n} from './i18n';

export const legacyRisks = 'Используется старый официальный Xray v25.12.8. Старая версия может содержать уязвимости и не иметь новых исправлений безопасности. VLESS сам по себе не шифрует трафик; у этого профиля нет TLS/Reality между клиентом и сервером. HTTPS-сайты сохраняют собственное TLS-шифрование, но это не защищает весь трафик прокси. Рекомендуем современный профиль с TLS/Reality.';

export function LegacyDialog({onClose,onContinue}:{onClose:()=>void;onContinue:()=>void}) {
 const {t}=useI18n();
 return <Modal title={t('Небезопасная совместимость')} onClose={onClose}>
  <p className="error-banner" role="alert">{t(legacyRisks)}</p>
  <div className="dialog-actions"><button onClick={onClose}>{t('Отмена')}</button><button className="primary" onClick={onContinue}>{t('Продолжить')}</button></div>
 </Modal>;
}
