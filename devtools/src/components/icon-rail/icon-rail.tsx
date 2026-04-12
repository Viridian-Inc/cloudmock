import type { JSX } from 'preact';
import type { ViewId } from '../../app';
import { t } from '../../lib/i18n';
import {
  ZapIcon,
  TopologyIcon,
  TracesIcon,
  ChartIcon,
  DashboardIcon,
  BucketIcon,
  DatabaseIcon,
  QueueIcon,
  CognitoIcon,
  LambdaIcon,
  ShieldIcon,
  MailIcon,
  TargetIcon,
  IncidentIcon,
  MonitorIcon,
  FlaskIcon,
  RegressionIcon,
  CpuIcon,
  RoutingIcon,
  TrafficIcon,
  RUMIcon,
  SettingsIcon,
  AppsIcon,
  KeyIcon,
  UsageIcon,
  AuditIcon,
  PlatformSettingsIcon,
  DiffIcon,
} from '../icons';
import './icon-rail.css';

interface IconRailProps {
  activeView: ViewId;
  onViewChange: (view: ViewId) => void;
}

type IconComponent = (p: JSX.SVGAttributes<SVGSVGElement>) => JSX.Element;

interface NavItem {
  id: ViewId;
  icon: IconComponent;
  i18nKey: string;
}

interface NavGroup {
  label: string;
  items: NavItem[];
}

const NAV_GROUPS: NavGroup[] = [
  {
    label: 'Observe',
    items: [
      { id: 'activity', icon: ZapIcon, i18nKey: 'nav.activity' },
      { id: 'topology', icon: TopologyIcon, i18nKey: 'nav.topology' },
      { id: 'traces', icon: TracesIcon, i18nKey: 'nav.traces' },
      { id: 'metrics', icon: ChartIcon, i18nKey: 'nav.metrics' },
      { id: 'dashboards', icon: DashboardIcon, i18nKey: 'nav.dashboards' },
    ],
  },
  {
    label: 'AWS',
    items: [
      { id: 's3-browser', icon: BucketIcon, i18nKey: 'nav.s3_browser' },
      { id: 'dynamodb', icon: DatabaseIcon, i18nKey: 'nav.dynamodb' },
      { id: 'sqs-browser', icon: QueueIcon, i18nKey: 'nav.sqs_browser' },
      { id: 'cognito', icon: CognitoIcon, i18nKey: 'nav.cognito' },
      { id: 'lambda', icon: LambdaIcon, i18nKey: 'nav.lambda' },
      { id: 'iam', icon: ShieldIcon, i18nKey: 'nav.iam' },
      { id: 'mail', icon: MailIcon, i18nKey: 'nav.mail' },
    ],
  },
  {
    label: 'Operations',
    items: [
      { id: 'slos', icon: TargetIcon, i18nKey: 'nav.slos' },
      { id: 'incidents', icon: IncidentIcon, i18nKey: 'nav.incidents' },
      { id: 'monitors', icon: MonitorIcon, i18nKey: 'nav.monitors' },
      { id: 'chaos', icon: FlaskIcon, i18nKey: 'nav.chaos' },
      { id: 'regressions', icon: RegressionIcon, i18nKey: 'nav.regressions' },
    ],
  },
  {
    label: 'Tools',
    items: [
      { id: 'ai-debug', icon: CpuIcon, i18nKey: 'nav.ai_debug' },
      { id: 'iac-diff', icon: DiffIcon, i18nKey: 'nav.iac_diff' },
      { id: 'routing', icon: RoutingIcon, i18nKey: 'nav.routing' },
      { id: 'traffic', icon: TrafficIcon, i18nKey: 'nav.traffic' },
      { id: 'rum', icon: RUMIcon, i18nKey: 'nav.rum' },
    ],
  },
  {
    label: 'Platform',
    items: [
      { id: 'platform-apps', icon: AppsIcon, i18nKey: 'nav.platform_apps' },
      { id: 'platform-keys', icon: KeyIcon, i18nKey: 'nav.platform_keys' },
      { id: 'platform-usage', icon: UsageIcon, i18nKey: 'nav.platform_usage' },
      { id: 'platform-audit', icon: AuditIcon, i18nKey: 'nav.platform_audit' },
      { id: 'platform-settings', icon: PlatformSettingsIcon, i18nKey: 'nav.platform_settings' },
    ],
  },
];

const BOTTOM_ITEMS: NavItem[] = [
  { id: 'settings', icon: SettingsIcon, i18nKey: 'nav.settings' },
];

export function IconRail({ activeView, onViewChange }: IconRailProps) {
  return (
    <nav class="icon-rail" role="navigation" aria-label="Main navigation">
      {NAV_GROUPS.map((group) => (
        <div key={group.label} class="rail-group">
          <div class="rail-group-label">{group.label}</div>
          {group.items.map((item) => {
            const label = t(item.i18nKey);
            const IconCmp = item.icon;
            return (
              <button
                key={item.id}
                class={`rail-item${activeView === item.id ? ' active' : ''}`}
                onClick={() => onViewChange(item.id)}
                title={label}
                role="tab"
                aria-selected={activeView === item.id}
                aria-label={label}
              >
                <IconCmp class="rail-item-icon" />
                <span class="rail-item-label">{label}</span>
              </button>
            );
          })}
        </div>
      ))}
      <div class="rail-spacer" />
      <div class="rail-group">
        {BOTTOM_ITEMS.map((item) => {
          const label = t(item.i18nKey);
          const IconCmp = item.icon;
          return (
            <button
              key={item.id}
              class={`rail-item${activeView === item.id ? ' active' : ''}`}
              onClick={() => onViewChange(item.id)}
              title={label}
              role="tab"
              aria-selected={activeView === item.id}
              aria-label={label}
            >
              <IconCmp class="rail-item-icon" />
              <span class="rail-item-label">{label}</span>
            </button>
          );
        })}
      </div>
    </nav>
  );
}
