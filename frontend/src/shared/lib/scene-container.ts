import { createContext, useContext } from 'react'

/**
 * The DOM element of the current grimório scene (the `.scene-grimorio` section),
 * or null when not inside a scene. Overlays that portal to `document.body`
 * (Dialog/Popover/Sheet) read this and portal INTO the scene instead, so they
 * inherit the scene's token scope — without it a dialog opened over a grimório
 * scene renders in plain shadcn. Provided by SceneShell; consumed by the
 * shared/ui overlay wrappers. Null outside a scene → default (body) → shadcn.
 */
export const SceneContainerContext = createContext<HTMLElement | null>(null)

export function useSceneContainer(): HTMLElement | null {
  return useContext(SceneContainerContext)
}
