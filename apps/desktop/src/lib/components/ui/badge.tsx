import { cva, type VariantProps } from 'class-variance-authority';
import type { ComponentProps } from 'react';
import { cn } from '@/utils';

// Figma "Badge" (7:144): status and metadata label in JetBrains Mono.
export const badgeVariants = cva(
  'inline-flex items-center gap-2xs rounded-full px-sm py-2xs font-mono text-[11.5px] leading-[15px] font-medium whitespace-nowrap',
  {
    variants: {
      tone: {
        ok: 'bg-ok-soft text-moss-text',
        warn: 'bg-warn-soft text-warn-text',
        postit: 'bg-postit text-on-tone',
        neutral: 'bg-paper text-ink-soft',
        danger: 'bg-danger-soft text-danger-text',
      },
    },
    defaultVariants: { tone: 'neutral' },
  },
);

export type BadgeProps = ComponentProps<'span'> & VariantProps<typeof badgeVariants>;

export function Badge({ className, tone, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ tone }), className)} {...props} />;
}
