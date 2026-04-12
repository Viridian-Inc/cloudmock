import { useState, useEffect, useCallback } from 'preact/hooks';
import { ConnectionProvider, useConnection } from './lib/connection';
import { AuthProvider, useAuth } from './lib/auth';
import { setAdminBase, setAuthToken } from './lib/api';
import type { Environment } from './lib/environments';
import { IconRail } from './components/icon-rail/icon-rail';
import { SourceBar } from './components/source-bar/source-bar';
import { StatusBar } from './components/status-bar/status-bar';
import { ConnectionPicker } from './components/connection-picker/connection-picker';
import { CommandPalette } from './components/command-palette/command-palette';
import { ErrorBoundary } from './components/error-boundary';
import { useKeyboardShortcuts, type ShortcutAction } from './hooks/use-keyboard-shortcuts';
import { ActivityView } from './views/activity';
import { TopologyView } from './views/topology';
import { ServicesView } from './views/services';
import { TracesView } from './views/traces';
import { MetricsView } from './views/metrics';
import { SLOsView } from './views/slos';
import { IncidentsView } from './views/incidents';
import { ProfilerView } from './views/profiler';
import { ChaosView } from './views/chaos';
import { AIDebugView } from './views/ai-debug';
import { RoutingView } from './views/routing';
import { SettingsView } from './views/settings';
import { DashboardsView } from './views/dashboards';
import { MonitorsView } from './views/monitors';
import { TrafficView } from './views/traffic';
import { RUMView } from './views/rum';
import { S3BrowserView } from './views/s3-browser';
import { DynamoDBView } from './views/dynamodb';
import { SQSBrowserView } from './views/sqs-browser';
import { CognitoView } from './views/cognito';
import IaCDiffView from './views/iac-diff';
import { LambdaView } from './views/lambda';
import { IAMView } from './views/iam';
import { MailView } from './views/mail';
import { RegressionsView } from './views/regressions';
import { PlatformAppsView } from './views/platform-apps';
import { PlatformKeysView } from './views/platform-keys';
import { PlatformUsageView } from './views/platform-usage';
import { PlatformAuditView } from './views/platform-audit';
import { PlatformSettingsView } from './views/platform-settings';
export type ViewId =
  | 'activity'
  | 'topology'
  | 'services'
  | 'traces'
  | 'metrics'
  | 'dashboards'
  | 's3-browser'
  | 'dynamodb'
  | 'sqs-browser'
  | 'cognito'
  | 'lambda'
  | 'iam'
  | 'mail'
  | 'slos'
  | 'incidents'
  | 'monitors'
  | 'profiler'
  | 'chaos'
  | 'regressions'
  | 'ai-debug'
  | 'routing'
  | 'traffic'
  | 'rum'
  | 'settings'
  | 'platform-apps'
  | 'platform-keys'
  | 'platform-usage'
  | 'platform-audit'
  | 'platform-settings'
  | 'iac-diff';

const VIEW_COMPONENTS: Record<ViewId, () => preact.JSX.Element> = {
  activity: ActivityView,
  topology: TopologyView,
  services: ServicesView,
  traces: TracesView,
  metrics: MetricsView,
  dashboards: DashboardsView,
  's3-browser': S3BrowserView,
  dynamodb: DynamoDBView,
  'sqs-browser': SQSBrowserView,
  cognito: CognitoView,
  lambda: LambdaView,
  iam: IAMView,
  mail: MailView,
  slos: SLOsView,
  incidents: IncidentsView,
  monitors: MonitorsView,
  profiler: ProfilerView,
  chaos: ChaosView,
  regressions: RegressionsView,
  'ai-debug': AIDebugView,
  routing: RoutingView,
  traffic: TrafficView,
  rum: RUMView,
  'iac-diff': IaCDiffView,
  settings: SettingsView,
  'platform-apps': PlatformAppsView,
  'platform-keys': PlatformKeysView,
  'platform-usage': PlatformUsageView,
  'platform-audit': PlatformAuditView,
  'platform-settings': PlatformSettingsView,
};

/** View IDs reachable via Cmd+number shortcuts */
const SHORTCUT_VIEWS: Record<string, ViewId> = {
  activity: 'activity',
  topology: 'topology',
  services: 'services',
  traces: 'traces',
  metrics: 'metrics',
  slos: 'slos',
  incidents: 'incidents',
  profiler: 'profiler',
  chaos: 'chaos',
  settings: 'settings',
};

function Workspace() {
  const [activeView, setActiveView] = useState<ViewId>('activity');
  const { state, connect, isFirstLaunch } = useConnection();
  const { auth, fetchSaaSConfig } = useAuth();
  const [showPicker, setShowPicker] = useState(isFirstLaunch);

  // Sync admin URL and auth token to API client
  useEffect(() => {
    setAdminBase(state.adminUrl);
    // Fetch SaaS config to detect hosted mode
    if (state.adminUrl) {
      fetchSaaSConfig(state.adminUrl);
    }
  }, [state.adminUrl, fetchSaaSConfig]);

  // Sync auth token to API client whenever it changes
  useEffect(() => {
    setAuthToken(auth.token);
  }, [auth.token]);

  // Listen for auth expiry events from the API client
  useEffect(() => {
    const handler = () => setShowPicker(true);
    document.addEventListener('cloudmock:auth-expired', handler);
    return () => document.removeEventListener('cloudmock:auth-expired', handler);
  }, []);

  const handleConnect = (adminUrl: string, gatewayUrl: string) => {
    connect(adminUrl, gatewayUrl);
    setShowPicker(false);
  };

  const handleEnvironmentChange = useCallback(
    (env: Environment) => {
      if (env.endpoint) {
        // Derive gateway URL from admin endpoint (admin=4599, gateway=4566)
        const gatewayUrl = env.endpoint.replace(':4599', ':4566');
        connect(env.endpoint, gatewayUrl);
      }
    },
    [connect],
  );

  // Listen for navigate-activity event from topology node inspector
  useEffect(() => {
    const handler = (e: Event) => {
      const detail = (e as CustomEvent).detail;
      if (detail?.service) {
        // Set hash so ActivityView picks up the filter on mount
        window.location.hash = `service=${encodeURIComponent(detail.service)}`;
        setActiveView('activity');
      }
    };
    document.addEventListener('neureaux:navigate-activity', handler);
    return () => document.removeEventListener('neureaux:navigate-activity', handler);
  }, []);

  // Listen for navigate-traces event from activity event detail
  useEffect(() => {
    const handler = (e: Event) => {
      const detail = (e as CustomEvent).detail;
      if (detail?.traceId) {
        window.location.hash = `trace=${encodeURIComponent(detail.traceId)}`;
        setActiveView('traces');
      }
    };
    document.addEventListener('neureaux:navigate-traces', handler);
    return () => document.removeEventListener('neureaux:navigate-traces', handler);
  }, []);

  // Keyboard shortcut handler
  const handleShortcut = useCallback(
    (action: ShortcutAction) => {
      // View switching (Cmd+1-9/0)
      const viewId = SHORTCUT_VIEWS[action];
      if (viewId) {
        setActiveView(viewId);
        return;
      }

      // Cmd+K — handled by CommandPalette component
      if (action === 'search') {
        return;
      }

      // Cmd+L — snap to live mode (dispatches a custom event that topology listens for)
      if (action === 'live') {
        if (activeView === 'topology') {
          document.dispatchEvent(new CustomEvent('neureaux:snap-live'));
        }
        return;
      }

      // Escape — deselect current node/event
      if (action === 'deselect') {
        document.dispatchEvent(new CustomEvent('neureaux:deselect'));
        return;
      }
    },
    [activeView],
  );

  useKeyboardShortcuts(handleShortcut);

  const ViewComponent = VIEW_COMPONENTS[activeView];

  return (
    <>
      <CommandPalette onNavigate={setActiveView} />
      {showPicker && (
        <ConnectionPicker
          onConnect={handleConnect}
          onClose={() => setShowPicker(false)}
        />
      )}
      <div class="workspace">
        <IconRail activeView={activeView} onViewChange={setActiveView} />
        <div class="workspace-main">
          <SourceBar />
          <div class="workspace-content">
            <ErrorBoundary>
              <ViewComponent />
            </ErrorBoundary>
          </div>
          <StatusBar
            connected={state.connected}
            endpoint={state.gatewayUrl.replace('http://', '')}
            region={state.region || 'us-east-1'}
            profile={state.profile || 'minimal'}
            iamMode={state.iamMode || 'enforce'}
            serviceCount={state.serviceCount}
            lastUpdated={state.lastHealthCheck}
            onEnvironmentChange={handleEnvironmentChange}
          />
        </div>
      </div>
    </>
  );
}

export function App() {
  return (
    <AuthProvider>
      <ConnectionProvider>
        <Workspace />
      </ConnectionProvider>
    </AuthProvider>
  );
}
