import { useState, useEffect, useCallback } from 'preact/hooks';
import { api } from './api';
import { useSSE } from './hooks/useSSE';
import { useRoute } from './hooks/useRoute';
import { Header } from './components/Header';
import { Sidebar, NavItem } from './components/Sidebar';
import { CommandPalette } from './components/CommandPalette';
import { Toast } from './components/Toast';
import { ServicesPage } from './pages/Services';
import { RequestsPage } from './pages/Requests';
import { RequestDetailPage } from './pages/RequestDetail';
import { DynamoDBPage } from './pages/DynamoDB';
import { ResourcesPage } from './pages/Resources';
import { LambdaPage } from './pages/Lambda';
import { IAMPage } from './pages/IAM';
import { MailPage } from './pages/Mail';
import { TopologyPage } from './pages/Topology';
import { S3BrowserPage } from './pages/S3Browser';
import { SQSBrowserPage } from './pages/SQSBrowser';
import { CognitoBrowserPage } from './pages/CognitoBrowser';

let toastTimer: ReturnType<typeof setTimeout> | null = null;

export function App() {
  const { segments, activePath } = useRoute();
  const sse = useSSE();
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [services, setServices] = useState<any[]>([]);
  const [stats, setStats] = useState<any>({});
  const [health, setHealth] = useState<any>(null);
  const [toast, setToast] = useState('');
  const [mailCount, setMailCount] = useState(0);

  const showToast = useCallback((msg: string) => {
    setToast(msg);
    if (toastTimer) clearTimeout(toastTimer);
    toastTimer = setTimeout(() => setToast(''), 3000);
  }, []);

  useEffect(() => {
    api('/api/services').then(setServices).catch(() => {});
    api('/api/stats').then(setStats).catch(() => {});
    api('/api/health').then(setHealth).catch(() => {});
    api('/api/ses/emails').then((e: any) => setMailCount(Array.isArray(e) ? e.length : 0)).catch(() => {});
  }, []);

  useEffect(() => {
    const iv = setInterval(() => {
      api('/api/stats').then(setStats).catch(() => {});
      api('/api/ses/emails').then((e: any) => setMailCount(Array.isArray(e) ? e.length : 0)).catch(() => {});
    }, 5000);
    return () => clearInterval(iv);
  }, []);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setPaletteOpen(p => !p);
      }
      if (e.key === 'Escape') setPaletteOpen(false);
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, []);

  const navItems: NavItem[] = [
    { id: '/', label: 'Services', icon: 'services' },
    { id: '/requests', label: 'Requests', icon: 'requests' },
    { id: '/dynamodb', label: 'DynamoDB', icon: 'database' },
    { id: '/s3', label: 'S3', icon: 'bucket' },
    { id: '/sqs', label: 'SQS', icon: 'queue' },
    { id: '/cognito', label: 'Cognito', icon: 'users' },
    { id: '/resources', label: 'Resources', icon: 'resources' },
    { id: '/lambda', label: 'Lambda', icon: 'lambda' },
    { id: '/iam', label: 'IAM', icon: 'shield' },
    { id: '/mail', label: 'Mail', icon: 'mail', badge: mailCount || null },
    { id: '/topology', label: 'Topology', icon: 'topology' },
  ];

  function renderPage() {
    if (segments[0] === 'requests' && segments[1]) {
      return <RequestDetailPage id={segments[1]} showToast={showToast} />;
    }
    switch (activePath) {
      case '/requests': return <RequestsPage sse={sse} showToast={showToast} />;
      case '/dynamodb': return <DynamoDBPage showToast={showToast} />;
      case '/s3': return <S3BrowserPage showToast={showToast} />;
      case '/sqs': return <SQSBrowserPage showToast={showToast} />;
      case '/cognito': return <CognitoBrowserPage showToast={showToast} />;
      case '/resources': return <ResourcesPage services={services} />;
      case '/lambda': return <LambdaPage sse={sse} />;
      case '/iam': return <IAMPage showToast={showToast} />;
      case '/mail': return <MailPage />;
      case '/topology': return <TopologyPage sse={sse} />;
      default: return <ServicesPage services={services} stats={stats} health={health} />;
    }
  }

  return (
    <div class="layout">
      <Header connected={sse.connected} health={health} onOpenPalette={() => setPaletteOpen(true)} />
      <div class="body-wrap">
        <Sidebar items={navItems} activePath={activePath} serviceCount={services.length} />
        <main class="main">
          {renderPage()}
        </main>
      </div>
      {paletteOpen && <CommandPalette services={services} onClose={() => setPaletteOpen(false)} />}
      <Toast message={toast} />
    </div>
  );
}
