import { execFileSync } from 'node:child_process';

const unformatted = execFileSync('gofmt', ['-l', 'cmd', 'internal'], { encoding: 'utf8' }).trim();
if (unformatted) {
  console.error(`Arquivos Go sem formatação:\n${unformatted}\nExecute npm run format:go.`);
  process.exitCode = 1;
}
