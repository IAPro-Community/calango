// The shipped @testing-library/jest-dom/vitest augmentation targets a one-parameter
// Assertion, while vitest 5 declares Assertion<R, T>. skipLibCheck hides that mismatch,
// so the matchers never reach the app types. This repeats the augmentation with the
// same arity vitest uses.
import type { TestingLibraryMatchers } from '@testing-library/jest-dom/matchers';

declare module 'vitest' {
  interface Assertion<R extends void | Promise<void> = void, T = unknown> extends TestingLibraryMatchers<any, R> {}
  interface AsymmetricMatchersContaining extends TestingLibraryMatchers<any, any> {}
}
