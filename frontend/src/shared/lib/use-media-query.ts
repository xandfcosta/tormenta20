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

/**
 * True when the user asked the OS to minimize motion (WCAG 2.3.3). Scene
 * transitions and other juice gate on this so they degrade to no-motion.
 *
 * @example
 *   const reduced = usePrefersReducedMotion()
 *   <div className={reduced ? undefined : 'scene-in'} />
 */
export function usePrefersReducedMotion(): boolean {
  return useMediaQuery('(prefers-reduced-motion: reduce)')
}
