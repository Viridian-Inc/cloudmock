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

interface NavGroup {
  label: string;
  ids: string[];
}

const navGroups: NavGroup[] = [
  { label: 'Observe', ids: ['/console', '/services', '/requests', '/traces', '/metrics'] },
  { label: 'Respond', ids: ['/incidents', '/regressions', '/debug'] },
  { label: 'Resources', ids: ['/dynamodb', '/s3', '/sqs', '/lambda', '/cognito', '/iam', '/mail'] },
  { label: 'Cloud-Native', ids: ['/kubernetes', '/argocd'] },
];

const bottomIds = new Set(['/settings', '/chaos']);

export function Sidebar({ items, activePath, serviceCount }: SidebarProps) {
  const itemMap = new Map(items.map(i => [i.id, i]));

  // Collect IDs that appear in groups or bottom
  const groupedIds = new Set<string>();
  navGroups.forEach(g => g.ids.forEach(id => groupedIds.add(id)));
  bottomIds.forEach(id => groupedIds.add(id));

  // Ungrouped items (like Home, Topology, Resources) go after groups
  const ungrouped = items.filter(i => !groupedIds.has(i.id) && !bottomIds.has(i.id));
  const bottomItems = items.filter(i => bottomIds.has(i.id));

  function renderItem(item: NavItem) {
    const IconComp = iconMap[item.icon];
    const isActive = activePath === item.id;
    return (
      <a
        key={item.id}
        class={`sidebar-item ${isActive ? 'active' : ''}`}
        href={`#${item.id}`}
        onClick={(e) => {
          e.preventDefault();
          location.hash = item.id;
        }}
      >
        {IconComp && <IconComp />}
        <span>{item.label}</span>
        {item.badge ? <span class="sidebar-badge">{item.badge}</span> : null}
      </a>
    );
  }

  return (
    <nav class="sidebar">
      <div class="sidebar-header">
        <div class="sidebar-logo">C</div>
        <div>
          <div class="sidebar-title">CloudMock</div>
          <div class="sidebar-subtitle">Console</div>
        </div>
      </div>

      {navGroups.map(group => {
        const groupItems = group.ids
          .map(id => itemMap.get(id))
          .filter(Boolean) as NavItem[];
        if (groupItems.length === 0) return null;
        return (
          <div class="sidebar-section" key={group.label}>
            <div class="sidebar-section-label">{group.label}</div>
            {groupItems.map(renderItem)}
          </div>
        );
      })}

      {ungrouped.length > 0 && (
        <div class="sidebar-section">
          <div class="sidebar-section-label">Other</div>
          {ungrouped.map(renderItem)}
        </div>
      )}

      <div class="sidebar-footer">
        {bottomItems.map(renderItem)}
        <div style={{ padding: '8px 12px', fontSize: '11px', color: 'var(--text-tertiary)' }}>
          {serviceCount} services
        </div>
      </div>
    </nav>
  );
}
