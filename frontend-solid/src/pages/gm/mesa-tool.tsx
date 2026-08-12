import type { Component } from 'solid-js'
import { Dynamic } from 'solid-js/web'
import { createCurrentTool } from '@/features/gm-tools/current-tool'
import type { ToolSlug } from '@/features/gm-tools/gm-tools'
import { BestiarioTool } from './bestiario-tool'
import { ToolPending } from './tool-pending'

/**
 * Registry of the Mesa's tools. It holds the COMPONENT, never a `render(x)`
 * call: a function invoked with a value captures that value and the tool would
 * never see the next edit (gotcha #14 of the port).
 *
 * Partial while the scene lands slice by slice (ALE-75) — an unregistered tool
 * says so on stage instead of rendering nothing.
 */
const TOOL_COMPONENTS: Partial<Record<ToolSlug, Component>> = {
  bestiario: BestiarioTool,
}

export function MesaTool() {
  const current = createCurrentTool()
  const component = () => TOOL_COMPONENTS[current()]

  return <Dynamic component={component() ?? (() => <ToolPending slug={current()} />)} />
}
