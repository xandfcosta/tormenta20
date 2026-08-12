import { createFileRoute } from '@tanstack/solid-router'
import { ForgePage } from '@/pages/characters/forge/forge-page'
import { requireSession } from './-guards'

// LAYOUT route: it has children (`/characters/new/$step`), so it must render an
// Outlet — the Forja shell does. A detail component here would swallow the
// outlet and no step would ever mount (reference_tanstack_nested_routes).
export const Route = createFileRoute('/characters/new')({
  beforeLoad: requireSession,
  component: ForgePage,
})
