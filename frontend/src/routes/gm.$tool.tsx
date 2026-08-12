import { createFileRoute, redirect } from '@tanstack/solid-router'
import { DEFAULT_TOOL, isToolSlug } from '@/features/gm-tools/gm-tools'
import { MesaTool } from '@/pages/gm/mesa-tool'

export const Route = createFileRoute('/gm/$tool')({
  // A hand-typed or stale URL lands on the first tool instead of a blank stage
  // — the slug is data from outside, so it is checked before it is used.
  beforeLoad: ({ params }) => {
    if (!isToolSlug(params.tool)) {
      throw redirect({ to: '/gm/$tool', params: { tool: DEFAULT_TOOL } })
    }
  },
  component: MesaTool,
})
