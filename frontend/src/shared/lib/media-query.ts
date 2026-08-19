import { type Accessor, createSignal, onCleanup } from 'solid-js'

/**
 * Subscribes to a CSS media query as a signal. Wraps `window.matchMedia` so
 * components never touch the browser API directly.
 *
 * The React version needed `useSyncExternalStore` to stay tear-free under
 * concurrent rendering; Solid has no re-render to tear, so a signal plus a
 * listener is the whole thing.
 *
 * @example const isDesktop = createMediaQuery('(min-width: 1024px)')
 */
export function createMediaQuery(query: string): Accessor<boolean> {
  // Ambiente sem `matchMedia` responde FALSO em vez de explodir: o jsdom da
  // suíte não o implementa, e um componente que só usa a media query para
  // decidir juice não pode derrubar a tela inteira por causa disso. É o mesmo
  // guarda que o `sfx.ts` já fazia com `(pointer: coarse)`.
  if (typeof window.matchMedia !== 'function') return () => false
  const list = window.matchMedia(query)
  const [matches, setMatches] = createSignal(list.matches)
  const onChange = () => setMatches(list.matches)
  list.addEventListener('change', onChange)
  onCleanup(() => list.removeEventListener('change', onChange))
  return matches
}

/**
 * True when the user asked the OS to minimize motion (WCAG 2.3.3). Scene
 * transitions and other juice gate on this so they degrade to no-motion.
 */
export function createPrefersReducedMotion(): Accessor<boolean> {
  return createMediaQuery('(prefers-reduced-motion: reduce)')
}
