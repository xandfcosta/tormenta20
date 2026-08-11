import { createFileRoute } from '@tanstack/solid-router'
import { PendingScene } from '@/pages/pending-scene'
import { requireSession } from './-guards'

export const Route = createFileRoute('/campaigns/$id/sessions/$sid')({
  beforeLoad: requireSession,
  component: () => <PendingScene title="Sessão ao vivo" issue="ALE-74" />,
})
