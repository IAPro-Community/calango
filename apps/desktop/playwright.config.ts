import { defineConfig } from '@playwright/test';

// Desktop E2E (issue #119). Run `npm run desktop:build` first: the tests start
// dist/ and bin/calango. Failures keep a trace, a screenshot and the Electron log.
export default defineConfig({
  testDir: './e2e',
  outputDir: './e2e-results',
  fullyParallel: false,
  workers: 1,
  timeout: 60_000,
  expect: { timeout: 10_000 },
  reporter: process.env.CI ? [['list'], ['html', { open: 'never', outputFolder: 'e2e-report' }]] : 'list',
  forbidOnly: !!process.env.CI,
});
