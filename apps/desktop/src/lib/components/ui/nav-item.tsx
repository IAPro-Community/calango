import type { ComponentProps } from 'react';
import { cn } from '@/utils';

// Figma "NavItem" (7:180): side navigation item of the desktop shell.
export type NavItemProps = ComponentProps<'button'> & {
  active?: boolean;
  /** Pending items shown in a post-it counter; hidden when zero or absent. */
  count?: number;
};

export function NavItem({ active = false, count, className, children, type, ...props }: NavItemProps) {
  return (
    <button
      type={type ?? 'button'}
      aria-current={active ? 'page' : undefined}
      className={cn(
        'flex h-[38px] w-full cursor-pointer items-center justify-between gap-xs rounded-sm border px-sm py-xs text-left font-body text-[14px] text-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-postit',
        active
          ? 'border-leaf bg-ok-soft leading-5 font-semibold'
          : 'border-transparent bg-transparent leading-[21px] hover:bg-card',
        className,
      )}
      {...props}
    >
      <span className="flex min-w-0 items-center gap-xs">
        <span aria-hidden="true" className={cn('size-2 shrink-0 rounded-[2px]', active ? 'bg-leaf' : 'bg-clay')} />
        <span className="truncate">{children}</span>
      </span>
      {count !== undefined && count > 0 && (
        <span className="shrink-0 rounded-full bg-postit px-xs py-2xs font-mono text-[11.5px] leading-[15px] font-medium text-on-tone">
          {count}
        </span>
      )}
    </button>
  );
}
