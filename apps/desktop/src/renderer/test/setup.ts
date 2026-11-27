import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';
import { fakeBridge } from './fakes';

// bridge.ts reads window.calango when it is imported, so the fake must exist first.
window.calango = fakeBridge.api;

afterEach(() => {
  cleanup();
  fakeBridge.reset();
});
