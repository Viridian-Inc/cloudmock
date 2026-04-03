import fs from 'fs';
import path from 'path';
import { execFileSync } from 'child_process';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Pick the best template for the given config
function chooseTemplate(config) {
  const { lang, services } = config;

  // Primary service priority: dynamodb > s3 > sqs > sns > lambda
  const priority = ['dynamodb', 's3', 'sqs', 'sns', 'lambda'];
  const primaryService = priority.find((s) => services.includes(s)) || services[0] || 's3';

  const templateKey = `${lang}-${primaryService}`;
  const templatesDir = path.join(__dirname, 'templates');
  const available = fs.readdirSync(templatesDir);

  if (available.includes(templateKey)) {
    return templateKey;
  }

  // Fallback: any template matching the language
  const langMatch = available.find((t) => t.startsWith(`${lang}-`));
  if (langMatch) return langMatch;

  // Last resort: node-dynamodb
  return 'node-dynamodb';
}

// Recursively copy a directory, replacing {{PROJECT_NAME}} in file contents and filenames
function copyDir(src, dest, projectName) {
  fs.mkdirSync(dest, { recursive: true });
  const entries = fs.readdirSync(src, { withFileTypes: true });
  for (const entry of entries) {
    const srcPath = path.join(src, entry.name);
    const destName = entry.name.replace(/\.tmpl$/, '');
    const destPath = path.join(dest, destName);
    if (entry.isDirectory()) {
      copyDir(srcPath, destPath, projectName);
    } else {
      const content = fs.readFileSync(srcPath, 'utf8');
      const replaced = content.replaceAll('{{PROJECT_NAME}}', projectName);
      fs.writeFileSync(destPath, replaced, 'utf8');
    }
  }
}

function detectPackageManager() {
  try {
    execFileSync('pnpm', ['--version'], { stdio: 'ignore' });
    return 'pnpm';
  } catch {}
  try {
    execFileSync('yarn', ['--version'], { stdio: 'ignore' });
    return 'yarn';
  } catch {}
  return 'npm';
}

export async function scaffold(config) {
  const { projectName, lang, services, test } = config;

  if (!projectName) {
    console.error('Error: project name is required.');
    process.exit(1);
  }

  const targetDir = path.resolve(process.cwd(), projectName);
  if (fs.existsSync(targetDir)) {
    console.error(`Error: directory "${projectName}" already exists.`);
    process.exit(1);
  }

  const template = chooseTemplate(config);
  const templateDir = path.join(__dirname, 'templates', template);

  console.log(`\nCreating project "${projectName}" using template "${template}"...`);
  copyDir(templateDir, targetDir, projectName);

  console.log(`\nProject created at ./${projectName}`);
  console.log('\nNext steps:\n');

  if (lang === 'node') {
    const pm = detectPackageManager();
    console.log(`  cd ${projectName}`);
    console.log(`  ${pm} install`);
    console.log(`  ${pm} start`);
  } else if (lang === 'python') {
    console.log(`  cd ${projectName}`);
    console.log(`  pip install -r requirements.txt`);
    console.log(`  uvicorn app:app --reload`);
  } else if (lang === 'go') {
    console.log(`  cd ${projectName}`);
    console.log(`  go mod tidy`);
    console.log(`  go run .`);
  } else if (lang === 'java') {
    console.log(`  cd ${projectName}`);
    console.log(`  mvn spring-boot:run`);
  } else if (lang === 'rust') {
    console.log(`  cd ${projectName}`);
    console.log(`  cargo run`);
  }

  console.log(`\nRun tests:`);
  if (lang === 'node') console.log(`  npm test`);
  else if (lang === 'python') console.log(`  pytest`);
  else if (lang === 'go') console.log(`  go test ./...`);
  else if (lang === 'java') console.log(`  mvn test`);
  else if (lang === 'rust') console.log(`  cargo test`);

  console.log(`\nStart CloudMock locally:`);
  console.log(`  docker compose up   # or: npx cloudmock`);
  console.log('');
}
