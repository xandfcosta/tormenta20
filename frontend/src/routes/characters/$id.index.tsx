import { createFileRoute } from '@tanstack/react-router'
import { CharacterViewPage } from '@/pages/characters/character-editor-page'
import { SceneFallback } from '@/shared/layout/scene-fallback'

// Character editor at exactly `/characters/$id`. Data is prefetched by
// the parent layout route; this just renders the screen.
export const Route = createFileRoute('/characters/$id/')({
  // Sheet tab lives in the URL so tab-eviction/mis-nav doesn't lose context.
  validateSearch: (search: Record<string, unknown>): { tab?: string } =>
    typeof search.tab === 'string' ? { tab: search.tab } : {},
  // Match the layout: dark grimório ground while the code-split page chunk
  // loads, so the scene transition never flashes the white app background.
  pendingComponent: SceneFallback,
  component: CharacterViewPage,
})
