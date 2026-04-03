import { execSync } from 'child_process';
import { existsSync, readFileSync, rmSync } from 'fs';
import { join } from 'path';

const dir = join(import.meta.dirname, 'test-output');

function cleanup() {
  if (existsSync(dir)) rmSync(dir, { recursive: true });
}

function test(name, fn) {
  try {
    fn();
    console.log(`  PASS: ${name}`);
  } catch (e) {
    console.error(`  FAIL: ${name}: ${e.message}`);
    process.exitCode = 1;
  }
}

function assert(cond, msg) {
  if (!cond) throw new Error(msg);
}

console.log('create-cloudmock-app tests\n');

// Test 1: Node + DynamoDB
cleanup();
execSync(`node index.mjs test-output --lang node --services dynamodb --test jest`, { cwd: import.meta.dirname, stdio: 'pipe' });
test('node-dynamodb generates package.json', () => {
  assert(existsSync(join(dir, 'package.json')), 'missing package.json');
  const pkg = JSON.parse(readFileSync(join(dir, 'package.json'), 'utf8'));
  assert(pkg.name === 'test-output', `wrong name: ${pkg.name}`);
  assert(pkg.dependencies['@aws-sdk/client-dynamodb'], 'missing dynamodb dep');
});
test('node-dynamodb generates src/index.js', () => {
  assert(existsSync(join(dir, 'src', 'index.js')), 'missing src/index.js');
});
test('node-dynamodb generates test file', () => {
  assert(existsSync(join(dir, 'test', 'app.test.js')), 'missing test/app.test.js');
});
test('node-dynamodb generates docker-compose.yml', () => {
  assert(existsSync(join(dir, 'docker-compose.yml')), 'missing docker-compose.yml');
});
test('node-dynamodb generates README', () => {
  const readme = readFileSync(join(dir, 'README.md'), 'utf8');
  assert(readme.includes('test-output'), 'README missing project name');
});

// Test 2: Python + S3
cleanup();
execSync(`node index.mjs test-output --lang python --services s3 --test pytest`, { cwd: import.meta.dirname, stdio: 'pipe' });
test('python-s3 generates requirements.txt', () => {
  assert(existsSync(join(dir, 'requirements.txt')), 'missing requirements.txt');
  const reqs = readFileSync(join(dir, 'requirements.txt'), 'utf8');
  assert(reqs.includes('boto3'), 'missing boto3');
});
test('python-s3 generates app.py', () => {
  assert(existsSync(join(dir, 'app.py')), 'missing app.py');
});

// Test 3: Go + SQS
cleanup();
execSync(`node index.mjs test-output --lang go --services sqs`, { cwd: import.meta.dirname, stdio: 'pipe' });
test('go-sqs generates go.mod', () => {
  assert(existsSync(join(dir, 'go.mod')), 'missing go.mod');
});
test('go-sqs generates main.go', () => {
  assert(existsSync(join(dir, 'main.go')), 'missing main.go');
});

// Test 4: Java + DynamoDB
cleanup();
execSync(`node index.mjs test-output --lang java --services dynamodb --test junit5`, { cwd: import.meta.dirname, stdio: 'pipe' });
test('java-dynamodb generates pom.xml', () => {
  assert(existsSync(join(dir, 'pom.xml')), 'missing pom.xml');
});

// Test 5: Rust + DynamoDB
cleanup();
execSync(`node index.mjs test-output --lang rust --services dynamodb`, { cwd: import.meta.dirname, stdio: 'pipe' });
test('rust-dynamodb generates Cargo.toml', () => {
  assert(existsSync(join(dir, 'Cargo.toml')), 'missing Cargo.toml');
});

// Test 6: Placeholder substitution
cleanup();
execSync(`node index.mjs my-cool-project --lang node --services s3 --test jest`, { cwd: import.meta.dirname, stdio: 'pipe' });
const outDir = join(import.meta.dirname, 'my-cool-project');
test('substitution replaces project name', () => {
  const pkg = JSON.parse(readFileSync(join(outDir, 'package.json'), 'utf8'));
  assert(pkg.name === 'my-cool-project', `wrong name: ${pkg.name}`);
  assert(!JSON.stringify(pkg).includes('{{'), 'unreplaced placeholder');
});
rmSync(outDir, { recursive: true });

cleanup();
console.log('\nAll tests complete.');
