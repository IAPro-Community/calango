import { useState } from 'react';
import type { AuditVerification } from '@calango/sdk-ts';
import type { ScreenProps } from './props';
import { Button, Card, PageHeader } from '@';

type ExportFormat = 'jsonl' | 'csv';

export function AuditScreen({ client, t }: ScreenProps) {
  const [result, setResult] = useState<AuditVerification | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);
  const [exporting, setExporting] = useState<ExportFormat | null>(null);
  const [exportError, setExportError] = useState(false);
  const [emptyExport, setEmptyExport] = useState(false);

  async function verify() {
    setLoading(true);
    setError(false);
    try {
      setResult(await client.verifyAudit());
    } catch {
      setError(true);
    } finally {
      setLoading(false);
    }
  }

  async function download(format: ExportFormat) {
    setExporting(format);
    setExportError(false);
    setEmptyExport(false);
    try {
      const text = await client.exportAudit(format);
      setEmptyExport(text.length === 0);
      const blob = new Blob([text], { type: format === 'jsonl' ? 'application/x-ndjson' : 'text/csv' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `calango-audit.${format}`;
      document.body.append(link);
      link.click();
      link.remove();
      window.setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch {
      setExportError(true);
    } finally {
      setExporting(null);
    }
  }

  return (
    <section className="flex flex-col gap-md">
      <PageHeader
        title={t('auditTitle')}
        actions={
          <>
            <Button variant="ghost" onClick={verify} disabled={loading}>
              {loading ? t('auditLoading') : t('auditVerify')}
            </Button>
            <Button variant="ghost" onClick={() => download('jsonl')} disabled={exporting !== null}>
              {exporting === 'jsonl' ? t('commonLoading') : t('auditExportJsonl')}
            </Button>
            <Button variant="ghost" onClick={() => download('csv')} disabled={exporting !== null}>
              {exporting === 'csv' ? t('commonLoading') : t('auditExportCsv')}
            </Button>
          </>
        }
      />
      {error && (
        <p className="error" role="alert">
          {t('auditError')}
        </p>
      )}
      {exportError && (
        <p className="error" role="alert">
          {t('auditExportError')}
        </p>
      )}
      <Card aria-live="polite">
        {result === null ? (
          <p className="muted">{loading ? t('auditLoading') : t('auditNeverChecked')}</p>
        ) : (
          <>
            <h2>{result.ok ? t('auditValid') : t('auditInvalid')}</h2>
            <p>
              {t('auditEntriesChecked')}: {result.entries_checked}
            </p>
            {result.entries_checked === 0 && <p className="muted">{t('auditEmptyChain')}</p>}
            {result.first_break !== null && (
              <div>
                <h3>{t('auditFirstBreak')}</h3>
                <p>
                  {t('auditBreakSequence')}: {result.first_break.seq}
                </p>
                <p>
                  {t('auditBreakId')}: <code>{result.first_break.id}</code>
                </p>
                <p>
                  {t('auditBreakReason')}: {result.first_break.reason}
                </p>
              </div>
            )}
          </>
        )}
      </Card>
      {emptyExport && <output className="muted">{t('auditExportEmpty')}</output>}
    </section>
  );
}
