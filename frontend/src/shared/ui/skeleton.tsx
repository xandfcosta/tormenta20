import { For, type ComponentProps, splitProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'

/** A pulsing placeholder block. Size it with utility classes. */
export function Skeleton(props: ComponentProps<'div'>) {
  const [local, rest] = splitProps(props, ['class'])
  return (
    <div data-slot="skeleton" class={cn('animate-pulse rounded-md bg-accent', local.class)} {...rest} />
  )
}

/** Card-shaped skeletons for grid pages (roster, chronicles). */
export function SkeletonCardGrid(props: { count?: number; class?: string }) {
  const slots = () => Array.from({ length: props.count ?? 3 })
  return (
    <div class={cn('grid gap-4 sm:grid-cols-2 lg:grid-cols-3', props.class)}>
      <For each={slots()}>
        {() => (
          <div data-slot="skeleton-card" class="rounded-md border p-4">
            <Skeleton class="h-5 w-3/4" />
            <Skeleton class="mt-3 h-3 w-full" />
            <Skeleton class="mt-2 h-3 w-2/3" />
            <div class="mt-3 flex gap-2">
              <Skeleton class="h-4 w-12" />
              <Skeleton class="h-4 w-16" />
            </div>
          </div>
        )}
      </For>
    </div>
  )
}

/** Row-shaped skeletons for table-ish lists. */
export function SkeletonRows(props: { count?: number; class?: string }) {
  const slots = () => Array.from({ length: props.count ?? 4 })
  return (
    <div class={cn('space-y-2', props.class)}>
      <For each={slots()}>
        {() => (
          <div data-slot="skeleton-row" class="flex items-center justify-between rounded-md border p-3">
            <div class="flex-1 space-y-1.5">
              <Skeleton class="h-4 w-1/3" />
              <Skeleton class="h-3 w-2/3" />
            </div>
            <Skeleton class="h-5 w-16" />
          </div>
        )}
      </For>
    </div>
  )
}
