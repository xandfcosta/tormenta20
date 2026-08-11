import { createFileRoute } from '@tanstack/solid-router'
import { PendingScene } from '@/pages/pending-scene'
import { requireSession } from './-guards'

export const Route = createFileRoute('/characters/$id')({
  beforeLoad: requireSession,
  component: () => <PendingScene title="Ficha" issue="ALE-73" />,
})
