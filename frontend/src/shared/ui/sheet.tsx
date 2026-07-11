import * as React from 'react'
import { Dialog as SheetPrimitive } from 'radix-ui'
import { X } from 'lucide-react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/shared/lib/utils'

/**
 * Sheet — radix Dialog rebranded as a directional drawer for the
 * responsive shell. On phone this powers the nav drawer + rare-action
 * bottom sheets; on desktop the same primitives back inspector
 * panels via a slightly different variant.
 *
 * `side` chooses the slide-in edge. `size` scales the inline axis.
 * Both are data-attributes so consumers can key layout off them
 * without leaking Tailwind class internals.
 */

function Sheet({ ...props }: React.ComponentProps<typeof SheetPrimitive.Root>) {
  return <SheetPrimitive.Root data-slot="sheet" {...props} />
}

function SheetTrigger({
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Trigger>) {
  return <SheetPrimitive.Trigger data-slot="sheet-trigger" {...props} />
}

function SheetClose({
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Close>) {
  return <SheetPrimitive.Close data-slot="sheet-close" {...props} />
}

function SheetOverlay({
  className,
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Overlay>) {
  return (
    <SheetPrimitive.Overlay
      data-slot="sheet-overlay"
      className={cn(
        'fixed inset-0 z-50 bg-background/70 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0',
        className,
      )}
      {...props}
    />
  )
}

const sheetContentVariants = cva(
  'fixed z-50 flex flex-col gap-4 border border-border/60 bg-card/95 p-6 shadow-[0_20px_60px_-30px_rgba(0,0,0,0.55)] backdrop-blur data-[state=open]:animate-in data-[state=closed]:animate-out',
  {
    variants: {
      side: {
        top: 'inset-x-0 top-0 border-t-0 data-[state=open]:slide-in-from-top data-[state=closed]:slide-out-to-top',
        bottom:
          'inset-x-0 bottom-0 border-b-0 data-[state=open]:slide-in-from-bottom data-[state=closed]:slide-out-to-bottom',
        left: 'inset-y-0 left-0 h-full border-l-0 data-[state=open]:slide-in-from-left data-[state=closed]:slide-out-to-left',
        right:
          'inset-y-0 right-0 h-full border-r-0 data-[state=open]:slide-in-from-right data-[state=closed]:slide-out-to-right',
      },
      size: {
        sm: '',
        md: '',
        lg: '',
      },
    },
    compoundVariants: [
      { side: 'left', size: 'sm', class: 'w-72' },
      { side: 'left', size: 'md', class: 'w-80' },
      { side: 'left', size: 'lg', class: 'w-96' },
      { side: 'right', size: 'sm', class: 'w-72' },
      { side: 'right', size: 'md', class: 'w-80' },
      { side: 'right', size: 'lg', class: 'w-96' },
      { side: 'top', size: 'sm', class: 'max-h-40' },
      { side: 'top', size: 'md', class: 'max-h-64' },
      { side: 'top', size: 'lg', class: 'max-h-96' },
      { side: 'bottom', size: 'sm', class: 'max-h-40' },
      { side: 'bottom', size: 'md', class: 'max-h-[50dvh]' },
      { side: 'bottom', size: 'lg', class: 'max-h-[80dvh]' },
    ],
    defaultVariants: {
      side: 'right',
      size: 'md',
    },
  },
)

type SheetContentProps = React.ComponentProps<typeof SheetPrimitive.Content> &
  VariantProps<typeof sheetContentVariants> & {
    hideCloseButton?: boolean
  }

function SheetContent({
  className,
  children,
  side,
  size,
  hideCloseButton,
  ...props
}: SheetContentProps) {
  return (
    <SheetPrimitive.Portal>
      <SheetOverlay />
      <SheetPrimitive.Content
        data-slot="sheet-content"
        data-side={side ?? 'right'}
        className={cn(sheetContentVariants({ side, size }), className)}
        {...props}
      >
        {children}
        {!hideCloseButton && (
          <SheetPrimitive.Close
            className="absolute right-4 top-4 rounded-sm opacity-70 transition-opacity hover:opacity-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
            aria-label="Fechar"
          >
            <X className="size-4" />
          </SheetPrimitive.Close>
        )}
      </SheetPrimitive.Content>
    </SheetPrimitive.Portal>
  )
}

function SheetHeader({ className, ...props }: React.ComponentProps<'div'>) {
  return (
    <div
      data-slot="sheet-header"
      className={cn('flex flex-col gap-1', className)}
      {...props}
    />
  )
}

function SheetTitle({
  className,
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Title>) {
  return (
    <SheetPrimitive.Title
      data-slot="sheet-title"
      className={cn('font-display text-lg text-foreground', className)}
      {...props}
    />
  )
}

function SheetDescription({
  className,
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Description>) {
  return (
    <SheetPrimitive.Description
      data-slot="sheet-description"
      className={cn('text-sm text-muted-foreground', className)}
      {...props}
    />
  )
}

export {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetOverlay,
  SheetTitle,
  SheetTrigger,
  sheetContentVariants,
}
