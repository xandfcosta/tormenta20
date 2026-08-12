import { type Accessor, type ParentProps, createContext, useContext } from 'solid-js'

/**
 * The DOM element of the current grimório scene (the `.scene-grimorio`
 * section), or null when not inside one. Overlays that portal to
 * `document.body` (Dialog/Popover/Tooltip/Select) read this and portal INTO the
 * scene instead, so they inherit the scene's token scope — without it a dialog
 * opened over a grimório scene renders in plain shadcn.
 *
 * It's an Accessor, not a plain value: the element only exists after the scene
 * mounts, and overlays must see it when it appears.
 *
 * Provided by SceneShell; consumed by the shared/ui overlay wrappers. Null
 * outside a scene → default (body) → shadcn.
 */
const SceneContainerContext = createContext<Accessor<HTMLElement | null>>(() => null)

export function SceneContainerProvider(props: ParentProps<{ element: Accessor<HTMLElement | null> }>) {
  return (
    <SceneContainerContext.Provider value={props.element}>
      {props.children}
    </SceneContainerContext.Provider>
  )
}

export function useSceneContainer(): Accessor<HTMLElement | null> {
  return useContext(SceneContainerContext)
}
