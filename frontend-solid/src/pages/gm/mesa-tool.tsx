import type { Component } from 'solid-js'
import { Dynamic } from 'solid-js/web'
import { createCurrentTool } from '@/features/gm-tools/current-tool'
import type { ToolSlug } from '@/features/gm-tools/gm-tools'
import { BestiarioTool } from './bestiario-tool'
import { CatalogosTool } from './catalogos-tool'
import { EncontrosTool } from './encontros-tool'
import { ImprovisoTool } from './improviso-tool'

/**
 * Registry of the Mesa's tools. It holds the COMPONENT, never a `render(x)`
 * call: a function invoked with a value captures that value and the tool would
 * never see the next edit (gotcha #14 of the port).
 *
 * Total, not partial: the type makes a new entry in `GM_TOOLS` a compile error
 * until it has a screen, which is cheaper than an empty stage.
 */
const TOOL_COMPONENTS: Record<ToolSlug, Component> = {
  bestiario: BestiarioTool,
  encontros: EncontrosTool,
  improviso: ImprovisoTool,
  catalogos: CatalogosTool,
}

export function MesaTool() {
  const current = createCurrentTool()

  return <Dynamic component={TOOL_COMPONENTS[current()]} />
}
