import { createFileRoute } from '@tanstack/solid-router'
import { SessionTrackerPage } from '@/pages/sessions/session-tracker-page'
import { requireSession } from './-guards'

export const Route = createFileRoute('/campaigns/$id/sessions/$sid')({
  beforeLoad: requireSession,
  component: SessionTrackerPage,
})
