import { createFileRoute } from '@tanstack/solid-router'
import { CharactersListPage } from '@/pages/characters/character-list-page'
import { requireSession } from './-guards'

export const Route = createFileRoute('/characters/')({
  beforeLoad: requireSession,
  component: CharactersListPage,
})
