import { JSX } from 'preact';

type IconProps = JSX.SVGAttributes<SVGSVGElement>;

const defaultProps: IconProps = {
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  'stroke-width': '2',
  width: '16',
  height: '16',
};

function Icon(props: IconProps & { children: preact.ComponentChildren }) {
  const { children, ...rest } = props;
  return <svg {...defaultProps} {...rest}>{children}</svg>;
}

/* ── Original dashboard icons ── */

export function HomeIcon(p: IconProps) {
  return <Icon {...p}>
    <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
    <polyline points="9 22 9 12 15 12 15 22" />
  </Icon>;
}

export function CloudIcon(p: IconProps) {
  return <Icon {...p}><path d="M18 10h-1.26A8 8 0 1 0 9 20h9a5 5 0 0 0 0-10z" /></Icon>;
}

export function ServicesIcon(p: IconProps) {
  return <Icon {...p}>
    <rect x="2" y="3" width="20" height="14" rx="2" />
    <line x1="8" y1="21" x2="16" y2="21" />
    <line x1="12" y1="17" x2="12" y2="21" />
  </Icon>;
}

export function RequestsIcon(p: IconProps) {
  return <Icon {...p}><polyline points="22 12 18 12 15 21 9 3 6 12 2 12" /></Icon>;
}

export function DatabaseIcon(p: IconProps) {
  return <Icon {...p}>
    <ellipse cx="12" cy="5" rx="9" ry="3" />
    <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3" />
    <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5" />
  </Icon>;
}

export function ResourcesIcon(p: IconProps) {
  return <Icon {...p}>
    <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
  </Icon>;
}

export function LambdaIcon(p: IconProps) {
  return <Icon {...p}>
    <polyline points="4 17 10 11 4 5" />
    <line x1="12" y1="19" x2="20" y2="19" />
  </Icon>;
}

export function ShieldIcon(p: IconProps) {
  return <Icon {...p}>
    <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
  </Icon>;
}

export function MailIcon(p: IconProps) {
  return <Icon {...p}>
    <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z" />
    <polyline points="22,6 12,13 2,6" />
  </Icon>;
}

export function TopologyIcon(p: IconProps) {
  return <Icon {...p}>
    <circle cx="18" cy="18" r="3" />
    <circle cx="6" cy="6" r="3" />
    <circle cx="18" cy="6" r="3" />
    <line x1="6" y1="9" x2="6" y2="21" />
    <path d="M9 6h6" />
    <path d="M6 21a3 3 0 0 0 3-3V9" />
  </Icon>;
}

export function ExpandIcon(p: IconProps) {
  return <Icon width="16" height="16" {...p}>
    <polyline points="15 3 21 3 21 9" />
    <polyline points="9 21 3 21 3 15" />
    <line x1="21" y1="3" x2="14" y2="10" />
    <line x1="3" y1="21" x2="10" y2="14" />
  </Icon>;
}

export function XIcon(p: IconProps) {
  return <Icon width="20" height="20" {...p}>
    <line x1="18" y1="6" x2="6" y2="18" />
    <line x1="6" y1="6" x2="18" y2="18" />
  </Icon>;
}

export function ChevDownIcon(p: IconProps) {
  return <Icon width="16" height="16" {...p}><polyline points="6 9 12 15 18 9" /></Icon>;
}

export function ChevRightIcon(p: IconProps) {
  return <Icon width="16" height="16" {...p}><polyline points="9 18 15 12 9 6" /></Icon>;
}

export function SearchIcon(p: IconProps) {
  return <Icon width="16" height="16" {...p}>
    <circle cx="11" cy="11" r="8" />
    <line x1="21" y1="21" x2="16.65" y2="16.65" />
  </Icon>;
}

export function PlusIcon(p: IconProps) {
  return <Icon width="16" height="16" {...p}>
    <line x1="12" y1="5" x2="12" y2="19" />
    <line x1="5" y1="12" x2="19" y2="12" />
  </Icon>;
}

export function TrashIcon(p: IconProps) {
  return <Icon width="16" height="16" {...p}>
    <polyline points="3 6 5 6 21 6" />
    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
  </Icon>;
}

export function CopyIcon(p: IconProps) {
  return <Icon width="14" height="14" {...p}>
    <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
  </Icon>;
}

export function PlayIcon(p: IconProps) {
  return <Icon width="14" height="14" {...p}><polygon points="5 3 19 12 5 21 5 3" /></Icon>;
}

export function RefreshIcon(p: IconProps) {
  return <Icon width="16" height="16" {...p}>
    <polyline points="23 4 23 10 17 10" />
    <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
  </Icon>;
}

export function SunIcon(p: IconProps) {
  return <Icon width="18" height="18" {...p}>
    <circle cx="12" cy="12" r="5" />
    <line x1="12" y1="1" x2="12" y2="3" />
    <line x1="12" y1="21" x2="12" y2="23" />
    <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
    <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
    <line x1="1" y1="12" x2="3" y2="12" />
    <line x1="21" y1="12" x2="23" y2="12" />
    <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
    <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
  </Icon>;
}

export function MoonIcon(p: IconProps) {
  return <Icon width="18" height="18" {...p}>
    <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
  </Icon>;
}

export function UploadIcon(p: IconProps) {
  return <Icon width="16" height="16" {...p}>
    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
    <polyline points="17 8 12 3 7 8" />
    <line x1="12" y1="3" x2="12" y2="15" />
  </Icon>;
}

export function BucketIcon(p: IconProps) {
  return <Icon {...p}>
    <path d="M22 5L2 5" />
    <path d="M4 5l1 14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2l1-14" />
    <path d="M9 9v8" />
    <path d="M15 9v8" />
    <path d="M2 5l2-2h16l2 2" />
  </Icon>;
}

export function QueueIcon(p: IconProps) {
  return <Icon {...p}>
    <line x1="8" y1="6" x2="21" y2="6" />
    <line x1="8" y1="12" x2="21" y2="12" />
    <line x1="8" y1="18" x2="21" y2="18" />
    <line x1="3" y1="6" x2="3.01" y2="6" />
    <line x1="3" y1="12" x2="3.01" y2="12" />
    <line x1="3" y1="18" x2="3.01" y2="18" />
  </Icon>;
}

export function TracesIcon(p: IconProps) {
  return <Icon {...p}>
    <line x1="6" y1="3" x2="6" y2="15" />
    <circle cx="18" cy="6" r="3" />
    <circle cx="18" cy="18" r="3" />
    <path d="M6 7a5 5 0 0 0 5 5h3.5a5 5 0 0 1 5 5" />
    <path d="M6 7a5 5 0 0 1 5-5h3.5a5 5 0 0 0 3.5 1.5" />
  </Icon>;
}

export function DownloadIcon(p: IconProps) {
  return <Icon width="16" height="16" {...p}>
    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
    <polyline points="7 10 12 15 17 10" />
    <line x1="12" y1="15" x2="12" y2="3" />
  </Icon>;
}

export function UsersIcon(p: IconProps) {
  return <Icon {...p}>
    <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
    <circle cx="9" cy="7" r="4" />
    <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
    <path d="M16 3.13a4 4 0 0 1 0 7.75" />
  </Icon>;
}

export function BugIcon(p: IconProps) {
  return <Icon {...p}>
    <path d="M8 2l1.88 1.88" />
    <path d="M14.12 3.88 16 2" />
    <path d="M9 7.13v-1a3.003 3.003 0 1 1 6 0v1" />
    <path d="M12 20c-3.3 0-6-2.7-6-6v-3a4 4 0 0 1 4-4h4a4 4 0 0 1 4 4v3c0 3.3-2.7 6-6 6z" />
    <path d="M12 20v-9" />
    <path d="M6.53 9C4.6 8.8 3 7.1 3 5" />
    <path d="M6 13H2" />
    <path d="M3 21c0-2.1 1.7-3.9 3.8-4" />
    <path d="M20.97 5c0 2.1-1.6 3.8-3.5 4" />
    <path d="M22 13h-4" />
    <path d="M17.2 17c2.1.1 3.8 1.9 3.8 4" />
  </Icon>;
}

export function ChartIcon(p: IconProps) {
  return <Icon {...p}>
    <line x1="18" y1="20" x2="18" y2="10" />
    <line x1="12" y1="20" x2="12" y2="4" />
    <line x1="6" y1="20" x2="6" y2="14" />
  </Icon>;
}

export function ZapIcon(p: IconProps) {
  return <Icon {...p}>
    <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
  </Icon>;
}

export function AlertIcon(p: IconProps) {
  return <Icon {...p}>
    <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
    <line x1="12" y1="9" x2="12" y2="13" />
    <line x1="12" y1="17" x2="12.01" y2="17" />
  </Icon>;
}

export function TrendDownIcon(p: IconProps) {
  return <Icon {...p}>
    <polyline points="23 18 13.5 8.5 8.5 13.5 1 6" />
    <polyline points="17 18 23 18 23 12" />
  </Icon>;
}

export function SettingsIcon(p: IconProps) {
  return <Icon {...p}>
    <circle cx="12" cy="12" r="3" />
    <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
  </Icon>;
}

/* ── New icons for devtools views ── */

/** Grid/layout icon for dashboards */
export function DashboardIcon(p: IconProps) {
  return <Icon {...p}>
    <rect x="3" y="3" width="7" height="7" rx="1" />
    <rect x="14" y="3" width="7" height="7" rx="1" />
    <rect x="3" y="14" width="7" height="7" rx="1" />
    <rect x="14" y="14" width="7" height="7" rx="1" />
  </Icon>;
}

/** Bell with checkmark for monitors */
export function MonitorIcon(p: IconProps) {
  return <Icon {...p}>
    <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
    <path d="M13.73 21a2 2 0 0 1-3.46 0" />
    <path d="M15 5l2 2 4-4" />
  </Icon>;
}

/** Arrows circling for traffic */
export function TrafficIcon(p: IconProps) {
  return <Icon {...p}>
    <polyline points="23 4 23 10 17 10" />
    <polyline points="1 20 1 14 7 14" />
    <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10" />
    <path d="M20.49 15a9 9 0 0 1-14.85 3.36L1 14" />
  </Icon>;
}

/** User with chart for RUM */
export function RUMIcon(p: IconProps) {
  return <Icon {...p}>
    <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
    <circle cx="12" cy="7" r="4" />
    <polyline points="18 8 20 6 22 8" />
  </Icon>;
}

/** Trending down chart for regressions */
export function RegressionIcon(p: IconProps) {
  return <Icon {...p}>
    <polyline points="23 18 13.5 8.5 8.5 13.5 1 6" />
    <polyline points="17 18 23 18 23 12" />
  </Icon>;
}

/** Key icon for Cognito */
export function CognitoIcon(p: IconProps) {
  return <Icon {...p}>
    <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" />
  </Icon>;
}

/** Queue/list icon for SQS (alias of QueueIcon) */
export const SQSIcon = QueueIcon;

/** Crosshair/target for SLOs */
export function TargetIcon(p: IconProps) {
  return <Icon {...p}>
    <circle cx="12" cy="12" r="10" />
    <circle cx="12" cy="12" r="6" />
    <circle cx="12" cy="12" r="2" />
  </Icon>;
}

/** Siren/alert for incidents */
export function IncidentIcon(p: IconProps) {
  return <Icon {...p}>
    <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
    <line x1="12" y1="9" x2="12" y2="13" />
    <line x1="12" y1="17" x2="12.01" y2="17" />
  </Icon>;
}

/** Flask for chaos engineering */
export function FlaskIcon(p: IconProps) {
  return <Icon {...p}>
    <path d="M9 3h6" />
    <path d="M10 3v6.2a1 1 0 0 1-.3.7L4 16a2 2 0 0 0 1.5 3.4h13A2 2 0 0 0 20 16l-5.7-6.1a1 1 0 0 1-.3-.7V3" />
  </Icon>;
}

/** CPU/chip for AI debug */
export function CpuIcon(p: IconProps) {
  return <Icon {...p}>
    <rect x="4" y="4" width="16" height="16" rx="2" />
    <rect x="9" y="9" width="6" height="6" />
    <line x1="9" y1="1" x2="9" y2="4" />
    <line x1="15" y1="1" x2="15" y2="4" />
    <line x1="9" y1="20" x2="9" y2="23" />
    <line x1="15" y1="20" x2="15" y2="23" />
    <line x1="20" y1="9" x2="23" y2="9" />
    <line x1="20" y1="14" x2="23" y2="14" />
    <line x1="1" y1="9" x2="4" y2="9" />
    <line x1="1" y1="14" x2="4" y2="14" />
  </Icon>;
}

/** Shuffle/routing icon */
export function RoutingIcon(p: IconProps) {
  return <Icon {...p}>
    <polyline points="16 3 21 3 21 8" />
    <line x1="4" y1="20" x2="21" y2="3" />
    <polyline points="21 16 21 21 16 21" />
    <line x1="15" y1="15" x2="21" y2="21" />
    <line x1="4" y1="4" x2="9" y2="9" />
  </Icon>;
}

/** Performance/gauge for profiler */
export function ProfilerIcon(p: IconProps) {
  return <Icon {...p}>
    <circle cx="12" cy="12" r="10" />
    <polyline points="12 6 12 12 16 14" />
  </Icon>;
}

/** Grid of app squares for Platform Apps */
export function AppsIcon(p: IconProps) {
  return <Icon {...p}>
    <rect x="3" y="3" width="7" height="7" rx="1" />
    <rect x="14" y="3" width="7" height="7" rx="1" />
    <rect x="3" y="14" width="7" height="7" rx="1" />
    <rect x="14" y="14" width="7" height="7" rx="1" />
    <path d="M10 6.5h4" stroke-dasharray="0" />
  </Icon>;
}

/** Key icon for API Keys */
export function KeyIcon(p: IconProps) {
  return <Icon {...p}>
    <circle cx="7.5" cy="15.5" r="5.5" />
    <path d="M21 2l-9.6 9.6" />
    <path d="M15.5 7.5L17 6" />
    <path d="M19 4l2 2" />
  </Icon>;
}

/** Bar chart up arrow for Usage */
export function UsageIcon(p: IconProps) {
  return <Icon {...p}>
    <line x1="18" y1="20" x2="18" y2="6" />
    <line x1="12" y1="20" x2="12" y2="10" />
    <line x1="6" y1="20" x2="6" y2="14" />
    <line x1="2" y1="20" x2="22" y2="20" />
  </Icon>;
}

/** Clock with list lines for Audit Log */
export function AuditIcon(p: IconProps) {
  return <Icon {...p}>
    <circle cx="12" cy="12" r="10" />
    <polyline points="12 6 12 12 16 14" />
    <line x1="2" y1="12" x2="4" y2="12" />
    <line x1="20" y1="12" x2="22" y2="12" />
  </Icon>;
}

/** Sliders icon for Platform Settings */
export function PlatformSettingsIcon(p: IconProps) {
  return <Icon {...p}>
    <line x1="4" y1="21" x2="4" y2="14" />
    <line x1="4" y1="10" x2="4" y2="3" />
    <line x1="12" y1="21" x2="12" y2="12" />
    <line x1="12" y1="8" x2="12" y2="3" />
    <line x1="20" y1="21" x2="20" y2="16" />
    <line x1="20" y1="12" x2="20" y2="3" />
    <line x1="1" y1="14" x2="7" y2="14" />
    <line x1="9" y1="8" x2="15" y2="8" />
    <line x1="17" y1="16" x2="23" y2="16" />
  </Icon>;
}

export function GridIcon(p: IconProps) {
  return <Icon {...p}>
    <rect x="3" y="3" width="7" height="7" />
    <rect x="14" y="3" width="7" height="7" />
    <rect x="14" y="14" width="7" height="7" />
    <rect x="3" y="14" width="7" height="7" />
  </Icon>;
}

export const iconMap: Record<string, (p: IconProps) => JSX.Element> = {
  home: HomeIcon,
  cloud: CloudIcon,
  services: ServicesIcon,
  requests: RequestsIcon,
  database: DatabaseIcon,
  resources: ResourcesIcon,
  lambda: LambdaIcon,
  shield: ShieldIcon,
  mail: MailIcon,
  topology: TopologyIcon,
  bucket: BucketIcon,
  queue: QueueIcon,
  users: UsersIcon,
  traces: TracesIcon,
  bug: BugIcon,
  chart: ChartIcon,
  zap: ZapIcon,
  alert: AlertIcon,
  trendDown: TrendDownIcon,
  settings: SettingsIcon,
  dashboard: DashboardIcon,
  monitor: MonitorIcon,
  traffic: TrafficIcon,
  rum: RUMIcon,
  regression: RegressionIcon,
  cognito: CognitoIcon,
  sqs: SQSIcon,
  target: TargetIcon,
  incident: IncidentIcon,
  flask: FlaskIcon,
  cpu: CpuIcon,
  routing: RoutingIcon,
  profiler: ProfilerIcon,
};
