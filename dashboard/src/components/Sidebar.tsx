import { iconMap } from './Icons';

export interface NavItem {
  id: string;
  label: string;
  icon: string;
  badge?: number | null;
}

interface SidebarProps {
  items: NavItem[];
  activePath: string;
  serviceCount: number;
}

export function Sidebar({ items, activePath, serviceCount }: SidebarProps) {
  return (
    <nav class="sidebar">
      {items.map((item) => {
        const IconComp = iconMap[item.icon];
        return (
          <a
            class={`nav-item ${activePath === item.id ? 'active' : ''}`}
            href={`#${item.id}`}
            onClick={(e) => {
              e.preventDefault();
              location.hash = item.id;
            }}
          >
            {IconComp && <IconComp />}
            <span>{item.label}</span>
            {item.badge ? <span class="nav-badge">{item.badge}</span> : null}
          </a>
        );
      })}
      <div class="nav-divider" />
      <div class="nav-footer">
        <div>v0.1.0</div>
        <div>{serviceCount} services</div>
      </div>
    </nav>
  );
}
