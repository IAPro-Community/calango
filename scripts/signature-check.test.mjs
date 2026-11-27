// Verificação de assinatura de artefato nativo (issue #120). Isolada atrás de variável de
// ambiente: a cadeia de assinatura e notarização ainda não existe, então na CI de PR o teste
// é pulado com motivo. Na homologação, aponte CALANGO_SIGNED_ARTIFACT para o binário ou app
// assinado. Ver DEV/CI_NATIVA.md.
import assert from 'node:assert/strict';
import { execFile } from 'node:child_process';
import { test } from 'node:test';
import { promisify } from 'node:util';

const execFileAsync = promisify(execFile);
const artifact = process.env.CALANGO_SIGNED_ARTIFACT ?? '';

const skipReason =
  artifact === ''
    ? 'assinatura não exercitada: defina CALANGO_SIGNED_ARTIFACT com um artefato assinado (gate de homologação, DEV/CI_NATIVA.md)'
    : process.platform === 'linux'
      ? 'Linux não tem assinatura de código nativa; a homologação usa checksum e assinatura destacada (DEV/CI_NATIVA.md)'
      : false;

test('artefato nativo tem assinatura válida', { skip: skipReason }, async () => {
  if (process.platform === 'darwin') {
    // codesign sai com código diferente de zero se a assinatura for inválida ou ausente.
    await execFileAsync('codesign', ['--verify', '--strict', '--verbose=2', artifact]);
    return;
  }
  if (process.platform === 'win32') {
    const { stdout } = await execFileAsync(
      'powershell.exe',
      [
        '-NoProfile',
        '-NonInteractive',
        '-Command',
        '(Get-AuthenticodeSignature -LiteralPath $env:CALANGO_SIGNED_ARTIFACT).Status',
      ],
      { env: process.env },
    );
    assert.equal(stdout.trim(), 'Valid');
    return;
  }
  assert.fail(`plataforma sem verificação de assinatura: ${process.platform}`);
});
