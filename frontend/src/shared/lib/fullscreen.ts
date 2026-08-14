import { type Accessor, createSignal, onCleanup } from 'solid-js'

/**
 * Safari shipped the unprefixed API only in 16.4; older iPads and macOS builds
 * still expose the `webkit` names, so the wrapper reads both. iPhone Safari has
 * NEITHER for elements (only `<video>`), which is exactly what `supported`
 * reports — there the path to a chrome-less app is Add to Home Screen.
 */
type FullscreenDocument = Document & {
  webkitFullscreenElement?: Element | null
  webkitFullscreenEnabled?: boolean
  webkitExitFullscreen?: () => Promise<void> | void
}

type FullscreenElement = HTMLElement & {
  webkitRequestFullscreen?: () => Promise<void> | void
}

/** Both spellings, because a browser fires only the one it implements. */
const CHANGE_EVENTS = ['fullscreenchange', 'webkitfullscreenchange'] as const

export type FullscreenController = {
  /** False on iPhone Safari — the UI hides the control instead of lying. */
  supported: boolean
  active: Accessor<boolean>
  toggle: () => void
}

function currentElement(doc: FullscreenDocument): Element | null {
  return doc.fullscreenElement ?? doc.webkitFullscreenElement ?? null
}

function enter(el: FullscreenElement): Promise<void> | void {
  return el.requestFullscreen ? el.requestFullscreen() : el.webkitRequestFullscreen?.()
}

function leave(doc: FullscreenDocument): Promise<void> | void {
  return doc.exitFullscreen ? doc.exitFullscreen() : doc.webkitExitFullscreen?.()
}

/**
 * The app's only door to the Fullscreen API (root CLAUDE.md: third-party — and
 * browser — surfaces get a thin owned interface). Takes the document as a
 * parameter so a test can hand it a fake instead of monkey-patching the real one.
 *
 * HOLDS state (a signal + a listener), so per the frontend guide it must be born
 * ONCE in a component body, never per event.
 *
 * The browser only grants `requestFullscreen` under a user gesture, so there is
 * deliberately no "restore on load": a persisted preference could not re-enter
 * without a tap, and a toggle that silently does nothing on startup is worse
 * than no toggle. Fullscreen survives client-side navigation (same document),
 * which is why the Hub can own the control for the whole app.
 *
 * @example const fs = createFullscreen(); <button onClick={fs.toggle} disabled={!fs.supported} />
 */
export function createFullscreen(doc: Document = document): FullscreenController {
  const target = doc as FullscreenDocument
  const [active, setActive] = createSignal(currentElement(target) !== null)

  // Esc and the system gesture also leave fullscreen, so the label follows the
  // browser's truth instead of what the last click intended.
  const sync = () => setActive(currentElement(target) !== null)
  for (const event of CHANGE_EVENTS) target.addEventListener(event, sync)
  onCleanup(() => {
    for (const event of CHANGE_EVENTS) target.removeEventListener(event, sync)
  })

  const toggle = () => {
    const root = target.documentElement as FullscreenElement
    // A refused request (gesture expired, embedded without `allow`) rejects;
    // there is nothing to recover — the screen simply stays as it is.
    void Promise.resolve(currentElement(target) ? leave(target) : enter(root)).catch(
      () => undefined,
    )
  }

  return {
    supported: target.fullscreenEnabled ?? target.webkitFullscreenEnabled ?? false,
    active,
    toggle,
  }
}
