import { useSyncExternalStore } from 'react'

/**
 * Subscribes to a CSS media query and returns whether it currently matches.
 * Wraps `window.matchMedia` behind a hook so components never touch the
 * browser API directly. Uses `useSyncExternalStore` so it's tear-free under
 * concurrent rendering and SSR-safe (server snapshot is always `false`).
 *
 * @example
 *   const isDesktop = useMediaQuery('(min-width: 1024px)')
 */
export function useMediaQuery(query: string): boolean {
  return useSyncExternalStore(
    (onChange) => {
      const list = window.matchMedia(query)
      list.addEventListener('change', onChange)
      return () => list.removeEventListener('change', onChange)
    },
    () => window.matchMedia(query).matches,
    () => false,
  )
}
