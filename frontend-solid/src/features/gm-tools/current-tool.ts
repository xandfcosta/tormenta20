import { useMatches } from '@tanstack/solid-router'
import { DEFAULT_TOOL, type ToolSlug, isToolSlug } from './gm-tools'

/**
 * The tool on stage, read from the matched route. The URL is the single source:
 * a deep link opens the tool it names, the back button walks the rail, and no
 * signal can drift from the address bar.
 *
 * Falls back to the first tool for anything unrecognised — the route guard
 * already redirects those, and a scene must never render a blank stage.
 *
 * @example const tool = createCurrentTool(); tool() // 'bestiario'
 */
export function createCurrentTool(): () => ToolSlug {
  const matches = useMatches()
  return () => {
    const params = matches().at(-1)?.params as { tool?: string } | undefined
    const slug = params?.tool
    return slug && isToolSlug(slug) ? slug : DEFAULT_TOOL
  }
}
