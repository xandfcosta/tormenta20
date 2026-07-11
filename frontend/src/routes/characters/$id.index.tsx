import { createFileRoute } from '@tanstack/react-router'
import { CharacterViewPage } from '@/pages/characters/character-editor-page'

// Character editor at exactly `/characters/$id`. Data is prefetched by
// the parent layout route; this just renders the screen.
export const Route = createFileRoute('/characters/$id/')({
  component: CharacterViewPage,
})
