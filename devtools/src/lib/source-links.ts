/**
 * Maps service/resource names to source code paths for quick navigation.
 * Links open in VS Code via the vscode:// URI scheme.
 */

const SERVICE_PATHS: Record<string, string> = {
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
