import { createFileRoute } from '@tanstack/solid-router'
import { PendingScene } from '@/pages/pending-scene'
import { requireSession } from './-guards'

export const Route = createFileRoute('/campaigns/')({
  beforeLoad: requireSession,
  component: () => <PendingScene title="Crônicas" issue="ALE-71" />,
})
