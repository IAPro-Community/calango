import { useCallback, useState } from 'react';
import type { Approval } from '@calango/sdk-ts';
import { usePoll } from '../usePoll';
import type { ScreenProps } from './props';

export function ApprovalsScreen({ client, t }: ScreenProps) {
  const load = useCallback(() => client.listApprovals('pending'), [client]);
  const { data, error, refresh } = usePoll(load, 3_000);
  const [actionError, setActionError] = useState<string | null>(null);

  async function decide(approval: Approval, decision: 'approve' | 'deny', remember: boolean) {
    setActionError(null);
    try {
      await client.decideApproval(approval.id, decision, remember);
      refresh();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : String(err));
    }
  }

  const approvals = data ?? [];
  return (
    <section className="screen">
      <header className="screen-header">
        <h1>{t('approvalTitle')}</h1>
      </header>
      {error && <p className="error">{error}</p>}
      {actionError && <p className="error">{actionError}</p>}
      <div className="card">
        {approvals.length === 0 ? (
          <p className="muted">{t('approvalEmpty')}</p>
        ) : (
          <ul className="list">
            {approvals.map((approval) => (
              <li key={approval.id} className="approval">
                <div>
                  <strong>{approval.action}</strong>
                  <div className="muted">
                    {t('approvalTarget')}: {approval.target}
                  </div>
                </div>
                <div className="actions">
                  <button type="button" onClick={() => void decide(approval, 'approve', false)}>
                    {t('actionApprove')}
                  </button>
                  <button type="button" onClick={() => void decide(approval, 'approve', true)}>
                    {t('actionAlwaysAllow')}
                  </button>
                  <button type="button" className="danger" onClick={() => void decide(approval, 'deny', false)}>
                    {t('actionDeny')}
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}
