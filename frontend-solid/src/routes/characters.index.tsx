import { createFileRoute } from '@tanstack/solid-router'
import { PendingScene } from '@/pages/pending-scene'
import { requireSession } from './-guards'

export const Route = createFileRoute('/characters/')({
  beforeLoad: requireSession,
  component: () => <PendingScene title="Personagens" issue="ALE-70" />,
})
