import { createFileRoute } from '@tanstack/solid-router'
import { PendingScene } from '@/pages/pending-scene'
import { requireSession } from './-guards'

export const Route = createFileRoute('/campaigns/$id/')({
  beforeLoad: requireSession,
  component: () => <PendingScene title="Crônica" issue="ALE-72" />,
})
