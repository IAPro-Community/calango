import { randomBytes } from 'node:crypto';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';

const KEY_BYTES = 32;

/**
 * loadProfileKey reads the per-install secret that names agent browser partitions, or
 * creates it (owner-only file). It is not a credential: it only makes partition names
 * unguessable, so losing it means agents get fresh, empty profiles.
 */
export async function loadProfileKey(filePath: string): Promise<Buffer> {
  try {
    const existing = await readFile(filePath);
    if (existing.byteLength === KEY_BYTES) {
      return existing;
    }
  } catch (error) {
    if (!(error instanceof Error && 'code' in error && error.code === 'ENOENT')) {
      throw error;
    }
  }
  const key = randomBytes(KEY_BYTES);
  await mkdir(path.dirname(filePath), { recursive: true, mode: 0o700 });
  await writeFile(filePath, key, { mode: 0o600 });
  return key;
}
