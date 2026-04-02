/**
 * Maps service/resource names to source code paths for quick navigation.
 * Links open in VS Code via the vscode:// URI scheme.
 */

const SERVICE_PATHS: Record<string, string> = {
  // AutoTend microservices
  'attendance': 'autotend-api-services/services/attendance',
  'bff': 'autotend-api-services/services/bff',
  'billing': 'autotend-api-services/services/billing',
  'calendar': 'autotend-api-services/services/calendar',
  'compliance': 'autotend-api-services/services/compliance',
  'identity': 'autotend-api-services/services/identity',
  'integrations': 'autotend-api-services/services/integrations',
  'notifications': 'autotend-api-services/services/notifications',
  'organizations': 'autotend-api-services/services/organizations',
  // Lambda / IaC resource name aliases
  'autotend-order-handler': 'autotend-api-services/services/billing',
  'autotend-attendance-handler': 'autotend-api-services/services/attendance',
  'autotend-notification-handler': 'autotend-api-services/services/notifications',
  'autotend-membership-handler': 'autotend-api-services/services/organizations',
  'autotend-stream-sync': 'autotend-api-services/services/calendar',
  'bff-service': 'autotend-api-services/services/bff',
  // AutoTend app
  'autotend-app': 'autotend-app/apps/app-native',
  'app-native': 'autotend-app/apps/app-native',
  // Infrastructure
  'autotend-infra': 'autotend-infra/pulumi',
  // CloudMock services
  'dynamodb': 'cloudmock/services/dynamodb',
  'lambda': 'cloudmock/services/lambda',
  's3': 'cloudmock/services/s3',
  'sqs': 'cloudmock/services/sqs',
  'sns': 'cloudmock/services/sns',
  'cognito-idp': 'cloudmock/services/cognito',
  'iam': 'cloudmock/services/iam',
  'sts': 'cloudmock/services/sts',
  'events': 'cloudmock/services/eventbridge',
  'logs': 'cloudmock/services/cloudwatchlogs',
  'monitoring': 'cloudmock/services/cloudwatch',
  'kms': 'cloudmock/services/kms',
  'secretsmanager': 'cloudmock/services/secretsmanager',
  'ssm': 'cloudmock/services/ssm',
  'apigateway': 'cloudmock/services/apigateway',
};

const WORKSPACE_ROOT = '/Users/megan/work/neureaux';

/**
 * Get the source code path for a service.
 * Returns null if no mapping exists.
 */
export function getSourcePath(serviceName: string): string | null {
  // Try exact match
  if (SERVICE_PATHS[serviceName]) {
    return `${WORKSPACE_ROOT}/${SERVICE_PATHS[serviceName]}`;
  }
  // Try lowercase exact match
  const lower = serviceName.toLowerCase();
  if (SERVICE_PATHS[lower]) {
    return `${WORKSPACE_ROOT}/${SERVICE_PATHS[lower]}`;
  }
  // Strip common prefixes/suffixes: "bff-service" → "bff", "autotend-order-handler" → "order"
  const stripped = lower
    .replace(/^autotend-/, '')
    .replace(/-handler$/, '')
    .replace(/-sync$/, '')
    .replace(/-service$/, '');
  if (SERVICE_PATHS[stripped]) {
    return `${WORKSPACE_ROOT}/${SERVICE_PATHS[stripped]}`;
  }
  // Try fuzzy substring match as last resort
  for (const [key, path] of Object.entries(SERVICE_PATHS)) {
    if (lower.includes(key.toLowerCase())) {
      return `${WORKSPACE_ROOT}/${path}`;
    }
  }
  return null;
}

/**
 * Get a VS Code URI to open a service's source directory.
 */
export function getVSCodeLink(serviceName: string): string | null {
  const path = getSourcePath(serviceName);
  if (!path) return null;
  return `vscode://file${path}`;
}

/**
 * Get a VS Code URI to open a specific file + line.
 */
export function getVSCodeFileLink(filePath: string, line?: number): string {
  const uri = `vscode://file${filePath}`;
  return line ? `${uri}:${line}` : uri;
}
