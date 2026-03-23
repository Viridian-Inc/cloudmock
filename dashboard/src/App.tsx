import { useState, useEffect, useCallback } from 'preact/hooks';
import { api } from './api';
import { useSSE } from './hooks/useSSE';
import { useRoute } from './hooks/useRoute';
import { Header } from './components/Header';
import { Sidebar, NavItem } from './components/Sidebar';
import { CommandPalette } from './components/CommandPalette';
import { Toast } from './components/Toast';
import { HomePage } from './pages/Home';
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
import { DebugAssistantPage, useDebugErrorCount } from './pages/DebugAssistant';
import { TracesPage } from './pages/Traces';
import { MetricsPage } from './pages/Metrics';
import { ChaosPage, useChaosActive } from './pages/Chaos';
import { ConsolePage } from './pages/Console';

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
  const debugErrorCount = useDebugErrorCount();
  const chaosActive = useChaosActive();

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
    { id: '/', label: 'Home', icon: 'home' },
    { id: '/services', label: 'Services', icon: 'services' },
    { id: '/requests', label: 'Requests', icon: 'requests' },
    { id: '/traces', label: 'Traces', icon: 'traces' },
    { id: '/dynamodb', label: 'DynamoDB', icon: 'database' },
    { id: '/s3', label: 'S3', icon: 'bucket' },
    { id: '/sqs', label: 'SQS', icon: 'queue' },
    { id: '/cognito', label: 'Cognito', icon: 'users' },
    { id: '/resources', label: 'Resources', icon: 'resources' },
    { id: '/lambda', label: 'Lambda', icon: 'lambda' },
    { id: '/iam', label: 'IAM', icon: 'shield' },
    { id: '/mail', label: 'Mail', icon: 'mail', badge: mailCount || null },
    { id: '/console', label: 'Console', icon: 'topology' },
    { id: '/topology', label: 'Topology', icon: 'topology' },
    { id: '/debug', label: 'Debug', icon: 'bug', badge: debugErrorCount || null },
    { id: '/metrics', label: 'Metrics', icon: 'chart' },
    { id: '/chaos', label: 'Chaos', icon: 'zap', badge: chaosActive ? 1 : null },
  ];

  function renderPage() {
    if (segments[0] === 'requests' && segments[1]) {
      return <RequestDetailPage id={segments[1]} showToast={showToast} />;
    }
    switch (activePath) {
      case '/services': return <ServicesPage services={services} stats={stats} health={health} />;
      case '/requests': return <RequestsPage sse={sse} showToast={showToast} />;
      case '/traces': return <TracesPage showToast={showToast} />;
      case '/dynamodb': return <DynamoDBPage showToast={showToast} />;
      case '/s3': return <S3BrowserPage showToast={showToast} />;
      case '/sqs': return <SQSBrowserPage showToast={showToast} />;
      case '/cognito': return <CognitoBrowserPage showToast={showToast} />;
      case '/resources': return <ResourcesPage services={services} />;
      case '/lambda': return <LambdaPage sse={sse} />;
      case '/iam': return <IAMPage showToast={showToast} />;
      case '/mail': return <MailPage />;
      case '/console': return <ConsolePage sse={sse} showToast={showToast} />;
      case '/topology': return <TopologyPage sse={sse} />;
      case '/debug': return <DebugAssistantPage showToast={showToast} />;
      case '/metrics': return <MetricsPage />;
      case '/chaos': return <ChaosPage showToast={showToast} />;
      default: return <HomePage sse={sse} showToast={showToast} />;
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
