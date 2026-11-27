/// <reference types="chai" />
// The shipped @testing-library/jest-dom/vitest augmentation reopens an Assertion with
// one type parameter, while vitest 5 declares Assertion<R, T>. A mismatched reopening
// does not merge, and skipLibCheck hides that, so the matchers never reach the app.
// This reopens the current vitest Matchers instead.
import type { TestingLibraryMatchers } from '@testing-library/jest-dom/matchers';

declare module 'vitest' {
  interface Matchers<R extends void | Promise<void> = void | Promise<void>, T = unknown>
    extends TestingLibraryMatchers<any, R> {}
}
