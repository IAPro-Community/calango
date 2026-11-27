import type { ComponentProps } from 'react';
import { cn } from '@/utils';

// Figma "NodePill" (7:169): state of a node (local PC or VPS), used in the sidebar and on mobile.
export type NodeKind = 'pc' | 'vps';
export type NodeState = 'online' | 'offline' | 'working';

export type NodePillProps = ComponentProps<'div'> & {
  kind: NodeKind;
  state: NodeState;
  /** Node name, for example "Seu computador". */
  label: string;
  /** Short state text, for example "ligado" or "offline · 14m". */
  status: string;
};

const dotClass: Record<NodeKind, string> = { pc: 'bg-leaf', vps: 'bg-tangerine' };
const statusClass: Record<NodeState, string> = {
  online: 'bg-ok-soft text-moss-text',
  offline: 'bg-paper text-ink-soft',
  working: 'bg-warn-soft text-warn-text',
};

export function NodePill({ kind, state, label, status, className, ...props }: NodePillProps) {
  return (
    <div
      className={cn(
        'flex h-10 items-center justify-between gap-xs rounded-sm border border-line bg-card px-sm py-xs',
        className,
      )}
      {...props}
    >
      <span className="flex min-w-0 items-center gap-xs">
        <span
          aria-hidden="true"
          className={cn('size-[9px] shrink-0 rounded-[3px]', state === 'offline' ? 'bg-line' : dotClass[kind])}
        />
        <span className="truncate font-body text-[12.5px] leading-[18px] text-ink">{label}</span>
      </span>
      <span
        className={cn(
          'shrink-0 rounded-xs px-xs py-2xs font-mono text-[11.5px] leading-[15px] font-medium',
          statusClass[state],
        )}
      >
        {status}
      </span>
    </div>
  );
}
