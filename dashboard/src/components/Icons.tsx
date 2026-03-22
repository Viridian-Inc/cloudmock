import { JSX } from 'preact';

type IconProps = JSX.SVGAttributes<SVGSVGElement>;

const defaultProps: IconProps = {
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  'stroke-width': '2',
  width: '18',
  height: '18',
};

function Icon(props: IconProps & { children: preact.ComponentChildren }) {
  const { children, ...rest } = props;
  return <svg {...defaultProps} {...rest}>{children}</svg>;
}

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
};
