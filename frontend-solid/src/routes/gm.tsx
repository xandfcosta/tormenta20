import { createFileRoute } from '@tanstack/solid-router'
import { MesaPage } from '@/pages/gm/mesa-page'
import { requireSession } from './-guards'

/**
 * `/gm` has children, so it MUST be a layout that renders an Outlet — a detail
 * screen here would swallow the outlet and `/gm/$tool` would never mount
 * ([[reference_tanstack_nested_routes]]). `MesaPage` is the shell: rail plus
 * Outlet, and it never unmounts as the GM walks between tools.
 */
export const Route = createFileRoute('/gm')({
  beforeLoad: requireSession,
  component: MesaPage,
})
