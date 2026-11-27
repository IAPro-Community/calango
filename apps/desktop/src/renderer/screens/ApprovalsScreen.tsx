import { useCallback, useState } from 'react';
import type { Approval } from '@calango/sdk-ts';
import { ApprovalCard, Button, Card, PageHeader } from '@';
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
    <section className="flex flex-col gap-md">
      <PageHeader title={t('approvalTitle')} />
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {actionError && (
        <p className="error" role="alert">
          {actionError}
        </p>
      )}
      {approvals.length === 0 ? (
        <Card>
          <p className="muted">{t('approvalEmpty')}</p>
        </Card>
      ) : (
        <ul className="m-0 flex max-w-[720px] list-none flex-col gap-sm p-0">
          {approvals.map((approval) => (
            <li key={approval.id}>
              <ApprovalCard
                state="pending"
                eyebrow={t('approvalPendingEyebrow')}
                title={approval.action}
                detail={
                  <>
                    <div>
                      {t('approvalTarget')}: {approval.target}
                    </div>
                    {approval.args !== undefined && Object.keys(approval.args).length > 0 && (
                      <pre className="m-0 mt-2xs whitespace-pre-wrap">{JSON.stringify(approval.args, null, 2)}</pre>
                    )}
                  </>
                }
                actions={
                  <>
                    <Button onClick={() => void decide(approval, 'approve', false)}>{t('actionApprove')}</Button>
                    <Button variant="ghost" onClick={() => void decide(approval, 'approve', true)}>
                      {t('actionAlwaysAllow')}
                    </Button>
                    <Button variant="danger" onClick={() => void decide(approval, 'deny', false)}>
                      {t('actionDeny')}
                    </Button>
                  </>
                }
              />
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
