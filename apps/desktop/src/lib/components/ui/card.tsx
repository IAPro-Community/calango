import { type ComponentProps, type ReactNode, useId } from 'react';
import { cn } from '@/utils';

// Figma panels (D1 "Conversas & rotinas", D5 cards): card surface with an optional mono eyebrow.
// With a title the section is named by it, so screen readers list it as a region.
export type CardProps = Omit<ComponentProps<'section'>, 'title'> & {
  /** Short uppercase heading in JetBrains Mono, rendered as an h2. */
  title?: ReactNode;
  /** Content aligned to the right of the title, for example a badge or a button. */
  aside?: ReactNode;
};

export function Card({ title, aside, className, children, ...props }: CardProps) {
  const titleId = useId();
  return (
    <section
      aria-labelledby={title !== undefined ? titleId : undefined}
      className={cn('flex min-w-0 flex-col gap-sm rounded-lg border border-line bg-card p-md', className)}
      {...props}
    >
      {(title !== undefined || aside !== undefined) && (
        <header className="flex flex-wrap items-center justify-between gap-xs">
          {title !== undefined && (
            <h2
              id={titleId}
              className="m-0 font-mono text-[11px] leading-[14px] font-bold tracking-[0.08em] text-ink-soft uppercase"
            >
              {title}
            </h2>
          )}
          {aside}
        </header>
      )}
      {children}
    </section>
  );
}
