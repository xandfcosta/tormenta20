import { createFileRoute, redirect } from '@tanstack/solid-router'
import { DEFAULT_TOOL } from '@/features/gm-tools/gm-tools'

// /gm names no tool, so it opens the first one. The Mesa always lives at an
// addressable tool — there is no toolless Mesa to look at.
export const Route = createFileRoute('/gm/')({
  beforeLoad: () => {
    throw redirect({ to: '/gm/$tool', params: { tool: DEFAULT_TOOL } })
  },
})
