#!/usr/bin/env node
import { parseArgs, runPrompts } from './prompts.mjs';
import { scaffold } from './scaffold.mjs';

const args = parseArgs(process.argv.slice(2));
const config = args.projectName ? await runPrompts(args) : await runPrompts({});
await scaffold(config);
