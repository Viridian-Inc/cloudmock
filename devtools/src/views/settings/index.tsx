import { useRouteSelection, hrefFor, useRouter } from '../../lib/router';
import { Connections } from './connections';
import { RoutingView } from '../routing';
import { Config } from './config';
import { Appearance } from './appearance';
import { Domains } from './domains';
import { Account } from './account';
import { Webhooks } from './webhooks';
import { Audit } from './audit';
import './settings.css';

type Tab = 'connections' | 'routing' | 'domains' | 'webhooks' | 'config' | 'appearance' | 'audit' | 'account';

const TABS: { id: Tab; label: string }[] = [
  { id: 'connections', label: 'Connections' },
  { id: 'routing', label: 'Routing' },
  { id: 'domains', label: 'Domains' },
  { id: 'webhooks', label: 'Webhooks' },
  { id: 'config', label: 'Config' },
  { id: 'appearance', label: 'Appearance' },
  { id: 'audit', label: 'Audit' },
  { id: 'account', label: 'Account' },
];

const TAB_IDS = new Set<Tab>(TABS.map((t) => t.id));

export function SettingsView() {
  const router = useRouter();
  const [raw] = useRouteSelection();
  const activeTab: Tab = raw && TAB_IDS.has(raw as Tab) ? (raw as Tab) : 'connections';

  return (
    <div class="settings-view">
      <div class="settings-tabs">
        {TABS.map((t) => (
          <a
            key={t.id}
            href={hrefFor({ view: 'settings', segments: [t.id] }, router)}
            class={`tab ${activeTab === t.id ? 'active' : ''}`}
          >
            {t.label}
          </a>
        ))}
      </div>
      <div class="settings-content">
        {activeTab === 'connections' && <Connections />}
        {activeTab === 'routing' && <RoutingView />}
        {activeTab === 'domains' && <Domains />}
        {activeTab === 'webhooks' && <Webhooks />}
        {activeTab === 'config' && <Config />}
        {activeTab === 'appearance' && <Appearance />}
        {activeTab === 'audit' && <Audit />}
        {activeTab === 'account' && <Account />}
      </div>
    </div>
  );
}
