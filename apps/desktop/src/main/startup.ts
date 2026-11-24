import { access, mkdir, unlink, writeFile } from 'node:fs/promises';
import path from 'node:path';

export function linuxAutostartFilePath(configHome: string | undefined, home: string): string {
  const base = configHome !== undefined && path.isAbsolute(configHome) ? configHome : path.join(home, '.config');
  return path.join(base, 'autostart', 'calango.desktop');
}

export function linuxAutostartEntry(executable: string, args: readonly string[]): string {
  const quote = (value: string): string => {
    if (/[\u0000-\u001F\u007F]/.test(value)) {
      throw new Error('desktop command arguments cannot contain control characters');
    }
    const escaped = value
      .replaceAll('\\', '\\\\')
      .replaceAll('"', '\\"')
      .replaceAll('$', '\\$')
      .replaceAll('`', '\\`')
      .replaceAll('%', '%%');
    return `"${escaped}"`;
  };
  const command = [executable, ...args].map(quote).join(' ');
  return [
    '[Desktop Entry]',
    'Type=Application',
    'Name=Calango',
    `Exec=${command}`,
    'Terminal=false',
    'X-GNOME-Autostart-enabled=true',
    '',
  ].join('\n');
}

export async function linuxAutostartEnabled(filePath: string): Promise<boolean> {
  try {
    await access(filePath);
    return true;
  } catch (error) {
    if (isMissingFile(error)) return false;
    throw error;
  }
}

export async function setLinuxAutostart(
  filePath: string,
  enabled: boolean,
  executable: string,
  args: readonly string[],
): Promise<void> {
  if (!enabled) {
    try {
      await unlink(filePath);
    } catch (error) {
      if (!isMissingFile(error)) throw error;
    }
    return;
  }

  await mkdir(path.dirname(filePath), { recursive: true, mode: 0o700 });
  await writeFile(filePath, linuxAutostartEntry(executable, args), { encoding: 'utf8', mode: 0o600 });
}

function isMissingFile(error: unknown): boolean {
  return error instanceof Error && 'code' in error && error.code === 'ENOENT';
}
