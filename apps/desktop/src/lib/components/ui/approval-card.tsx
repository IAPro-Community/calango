import type { ComponentProps, ReactNode } from 'react';
import { cn } from '@/utils';

// Figma "ApprovalCard" (7:205): immutable approval (ADR 0011). It shows the exact action, the
// target and the SHA-256 digest, and keeps the receipt apart from the decision.
export type ApprovalState = 'pending' | 'approved' | 'failed';

export type ApprovalCardProps = Omit<ComponentProps<'article'>, 'title'> & {
  state: ApprovalState;
  /** Upper line: state, sequence, expiry. */
  eyebrow: string;
  title: ReactNode;
  /** Exact action, target and digest, shown in mono. */
  detail: ReactNode;
  /** Execution receipt, shown once the approval is decided. */
  receipt?: ReactNode;
  /** Decision buttons, shown while pending. */
  actions?: ReactNode;
};

const borderClass: Record<ApprovalState, string> = {
  pending: 'border-tangerine',
  approved: 'border-leaf',
  failed: 'border-danger',
};
const toneClass: Record<ApprovalState, string> = {
  pending: 'text-warn-text',
  approved: 'text-moss-text',
  failed: 'text-danger-text',
};

export function ApprovalCard({
  state,
  eyebrow,
  title,
  detail,
  receipt,
  actions,
  className,
  ...props
}: ApprovalCardProps) {
  return (
    <article
      data-state={state}
      className={cn(
        'flex flex-col gap-[10px] rounded-md border-2 bg-card p-md shadow-paper',
        borderClass[state],
        className,
      )}
      {...props}
    >
      <p
        className={cn(
          'm-0 font-mono text-[11px] leading-[14px] font-bold tracking-[0.08em] uppercase',
          toneClass[state],
        )}
      >
        {eyebrow}
      </p>
      <h3 className="m-0 font-display text-[16px] leading-[22px] font-semibold tracking-[-0.01em] text-ink">{title}</h3>
      <div className="rounded-xs bg-paper px-sm py-xs font-mono text-[12.5px] leading-[19px] break-words text-ink-soft">
        {detail}
      </div>
      {state !== 'pending' && receipt !== undefined && (
        <p className={cn('m-0 font-body text-[12.5px] leading-[18px]', toneClass[state])}>{receipt}</p>
      )}
      {state === 'pending' && actions !== undefined && <div className="flex flex-wrap gap-xs">{actions}</div>}
    </article>
  );
}
