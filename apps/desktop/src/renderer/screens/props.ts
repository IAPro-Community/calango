import type { CalangoClient } from '@calango/sdk-ts';
import type { Translate } from '../i18n';

export interface ScreenProps {
  client: CalangoClient;
  t: Translate;
}
