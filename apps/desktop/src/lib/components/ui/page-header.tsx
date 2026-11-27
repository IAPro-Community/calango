import type { ComponentProps, ReactNode } from 'react';
import { cn } from '@/utils';

// Figma screen header (D1–D6): display title, mono context line and actions on the right.
export type PageHeaderProps = Omit<ComponentProps<'header'>, 'title'> & {
  title: ReactNode;
  /** Context line under the title, for example the provider chain or the budget. */
  meta?: ReactNode;
  actions?: ReactNode;
};

export function PageHeader({ title, meta, actions, className, ...props }: PageHeaderProps) {
  return (
    <header className={cn('flex flex-wrap items-start justify-between gap-md', className)} {...props}>
      <div className="flex min-w-0 flex-col gap-2xs">
        <h1 className="m-0 font-display text-[28px] leading-[34px] font-bold tracking-[-0.02em] text-ink">{title}</h1>
        {meta !== undefined && <p className="m-0 font-mono text-[12px] leading-4 text-ink-soft">{meta}</p>}
      </div>
      {actions !== undefined && <div className="flex flex-wrap items-center gap-xs">{actions}</div>}
    </header>
  );
}
