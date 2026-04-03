import prompts from 'prompts';

const TEST_FRAMEWORKS = {
  node: ['jest', 'vitest', 'mocha'],
  python: ['pytest', 'unittest'],
  go: ['testing'],
  java: ['junit5', 'testng'],
  rust: ['cargo-test'],
};

const SERVICE_CHOICES = [
  { title: 'S3', value: 's3' },
  { title: 'DynamoDB', value: 'dynamodb' },
  { title: 'SQS', value: 'sqs' },
  { title: 'SNS', value: 'sns' },
  { title: 'Lambda', value: 'lambda' },
];

export function parseArgs(argv) {
  const result = {};
  let i = 0;
  // First positional argument is project name
  if (argv[0] && !argv[0].startsWith('--')) {
    result.projectName = argv[0];
    i = 1;
  }
  while (i < argv.length) {
    const arg = argv[i];
    if (arg === '--lang' && argv[i + 1]) {
      result.lang = argv[i + 1];
      i += 2;
    } else if (arg === '--services' && argv[i + 1]) {
      result.services = argv[i + 1].split(',').map((s) => s.trim());
      i += 2;
    } else if (arg === '--test' && argv[i + 1]) {
      result.test = argv[i + 1];
      i += 2;
    } else {
      i++;
    }
  }
  return result;
}

export async function runPrompts(prefilled = {}) {
  const questions = [];

  if (!prefilled.projectName) {
    questions.push({
      type: 'text',
      name: 'projectName',
      message: 'Project name:',
      initial: 'my-cloudmock-app',
      validate: (v) => (v.trim().length > 0 ? true : 'Project name is required'),
    });
  }

  if (!prefilled.lang) {
    questions.push({
      type: 'select',
      name: 'lang',
      message: 'Language:',
      choices: [
        { title: 'Node.js', value: 'node' },
        { title: 'Python', value: 'python' },
        { title: 'Go', value: 'go' },
        { title: 'Java', value: 'java' },
        { title: 'Rust', value: 'rust' },
      ],
    });
  }

  if (!prefilled.services || prefilled.services.length === 0) {
    questions.push({
      type: 'multiselect',
      name: 'services',
      message: 'AWS services to use (space to select):',
      choices: SERVICE_CHOICES,
      min: 1,
      hint: '- Space to select. Return to submit',
    });
  }

  if (!prefilled.test) {
    questions.push({
      type: (prev, values) => {
        const lang = prefilled.lang || values.lang;
        const frameworks = TEST_FRAMEWORKS[lang] || [];
        return frameworks.length > 1 ? 'select' : null;
      },
      name: 'test',
      message: 'Test framework:',
      choices: (prev, values) => {
        const lang = prefilled.lang || values.lang;
        return (TEST_FRAMEWORKS[lang] || ['default']).map((f) => ({
          title: f,
          value: f,
        }));
      },
    });
  }

  const answers = questions.length > 0 ? await prompts(questions, {
    onCancel: () => {
      console.error('\nAborted.');
      process.exit(1);
    },
  }) : {};

  const lang = prefilled.lang || answers.lang;
  const defaultTest = TEST_FRAMEWORKS[lang]?.[0] || 'default';

  return {
    projectName: prefilled.projectName || answers.projectName,
    lang,
    services: prefilled.services || answers.services || [],
    test: prefilled.test || answers.test || defaultTest,
  };
}
