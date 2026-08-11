import { createFileRoute } from '@tanstack/solid-router'
import { PendingScene } from '@/pages/pending-scene'
import { requireSession } from './-guards'

export const Route = createFileRoute('/campaigns/join')({
  beforeLoad: requireSession,
  component: () => <PendingScene title="Entrar em campanha" issue="ALE-72" />,
})
