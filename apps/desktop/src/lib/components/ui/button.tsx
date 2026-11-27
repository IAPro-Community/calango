import { Slot } from '@radix-ui/react-slot';
import { cva, type VariantProps } from 'class-variance-authority';
import type { ComponentProps } from 'react';
import { cn } from '@/utils';

// Figma "Button" (7:133): Primary (green), Postit (brand highlight), Ghost and Danger.
export const buttonVariants = cva(
  'inline-flex min-h-11 items-center justify-center gap-xs rounded-sm border px-md py-sm font-body text-[14px] leading-5 font-semibold whitespace-nowrap transition-[filter,background-color] duration-150 cursor-pointer focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-postit disabled:cursor-not-allowed disabled:opacity-55',
  {
    variants: {
      variant: {
        primary: 'border-transparent bg-btn text-on-btn hover:brightness-110',
        postit: 'border-postit-deep bg-postit text-on-tone hover:brightness-105',
        ghost: 'border-line bg-card text-ink hover:bg-paper',
        danger: 'border-danger bg-danger-soft text-danger-text hover:brightness-110',
      },
      size: {
        default: '',
        sm: 'min-h-8 px-sm py-2xs text-[12.5px]',
      },
    },
    defaultVariants: { variant: 'primary', size: 'default' },
  },
);

export type ButtonProps = ComponentProps<'button'> &
  VariantProps<typeof buttonVariants> & {
    /** Renders the styles on the child element (for example a link) instead of a <button>. */
    asChild?: boolean;
  };

export function Button({ className, variant, size, asChild = false, type, ...props }: ButtonProps) {
  if (asChild) {
    return <Slot className={cn(buttonVariants({ variant, size }), className)} {...props} />;
  }
  return <button type={type ?? 'button'} className={cn(buttonVariants({ variant, size }), className)} {...props} />;
}
