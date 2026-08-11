import { createFileRoute } from '@tanstack/solid-router'
import { PendingScene } from '@/pages/pending-scene'
import { requireSession } from './-guards'

export const Route = createFileRoute('/gm')({
  beforeLoad: requireSession,
  component: () => <PendingScene title="Ferramentas do Mestre" issue="ALE-75" />,
})
