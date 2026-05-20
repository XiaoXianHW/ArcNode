import { CategoryPage } from './Coding';
import { useI18n } from '../state/i18nContext';

export function Gaming() {
  const { t } = useI18n();
  return <CategoryPage category="gaming" title={t('gaming.title')} subtitle={t('gaming.subtitle')} />;
}
