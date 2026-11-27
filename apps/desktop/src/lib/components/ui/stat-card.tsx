import type { ComponentProps, ReactNode } from 'react';
import { cn } from '@/utils';

// Figma "StatCard" (7:206): metric or budget card (monthly cost, tokens, active routines).
export type StatCardProps = ComponentProps<'section'> & {
  label: string;
  value: ReactNode;
  hint?: ReactNode;
  /** Colour of the hint line; "warn" when the metric needs attention. */
  hintTone?: 'ok' | 'warn' | 'danger';
};

const hintClass = { ok: 'text-moss-text', warn: 'text-warn-text', danger: 'text-danger-text' };

export function StatCard({ label, value, hint, hintTone = 'ok', className, ...props }: StatCardProps) {
  return (
    <section className={cn('flex flex-col gap-[6px] rounded-md border border-line bg-card p-md', className)} {...props}>
      <h3 className="m-0 font-mono text-[11px] leading-[14px] font-bold tracking-[0.08em] text-ink-soft uppercase">
        {label}
      </h3>
      <p className="m-0 font-display text-[28px] leading-[34px] font-bold tracking-[-0.02em] text-ink">{value}</p>
      {hint !== undefined && (
        <p className={cn('m-0 font-mono text-[11.5px] leading-[15px] font-medium', hintClass[hintTone])}>{hint}</p>
      )}
    </section>
  );
}
