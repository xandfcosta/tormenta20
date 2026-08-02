import { createFileRoute } from '@tanstack/react-router'
import { CharacterViewPage } from '@/pages/characters/character-editor-page'

// Character editor at exactly `/characters/$id`. Data is prefetched by
// the parent layout route; this just renders the screen.
export const Route = createFileRoute('/characters/$id/')({
  // Sheet tab lives in the URL so tab-eviction/mis-nav doesn't lose context.
  validateSearch: (search: Record<string, unknown>): { tab?: string } =>
    typeof search.tab === 'string' ? { tab: search.tab } : {},
  component: CharacterViewPage,
})
