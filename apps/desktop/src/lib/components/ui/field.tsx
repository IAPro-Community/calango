import type { ComponentProps, ReactNode } from 'react';
import { cn } from '@/utils';

// Form controls of the Figma composer (D1) and settings panels. The clay border keeps the
// 3:1 contrast that WCAG 1.4.11 asks of input boundaries.
const control =
  'w-full min-h-11 rounded-sm border border-clay bg-paper px-sm py-xs font-body text-[14px] leading-5 text-ink placeholder:text-ink-soft focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-postit disabled:cursor-not-allowed disabled:opacity-55';

export function Input({ className, ...props }: ComponentProps<'input'>) {
  return <input className={cn(control, className)} {...props} />;
}

export function Textarea({ className, ...props }: ComponentProps<'textarea'>) {
  return <textarea className={cn(control, 'min-h-20 resize-y', className)} {...props} />;
}

export function Select({ className, ...props }: ComponentProps<'select'>) {
  return <select className={cn(control, 'cursor-pointer', className)} {...props} />;
}

export type FieldProps = Omit<ComponentProps<'label'>, 'children'> & {
  label: ReactNode;
  /** Help text under the control. */
  hint?: ReactNode;
  children: ReactNode;
};

// Field wraps the control in its label, so the label names it without ids.
export function Field({ label, hint, className, children, ...props }: FieldProps) {
  return (
    <label className={cn('flex min-w-0 flex-col gap-2xs font-body text-[12.5px] leading-[18px] text-ink-soft', className)} {...props}>
      <span>{label}</span>
      {children}
      {hint !== undefined && <span className="text-[11.5px] text-ink-soft">{hint}</span>}
    </label>
  );
}
