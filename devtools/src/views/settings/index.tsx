import { useState } from 'preact/hooks';
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

export function SettingsView() {
  const [activeTab, setActiveTab] = useState<Tab>('connections');

  return (
    <div class="settings-view">
      <div class="settings-tabs">
        {TABS.map((t) => (
          <button
            key={t.id}
            class={`tab ${activeTab === t.id ? 'active' : ''}`}
            onClick={() => setActiveTab(t.id)}
          >
            {t.label}
          </button>
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
