import { createFileRoute } from '@tanstack/solid-router'
import { PendingScene } from '@/pages/pending-scene'
import { requireSession } from './-guards'

export const Route = createFileRoute('/campaigns/new')({
  beforeLoad: requireSession,
  component: () => <PendingScene title="Nova campanha" issue="ALE-72" />,
})
