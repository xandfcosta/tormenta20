import { createFileRoute } from '@tanstack/solid-router'
import { z } from 'zod'
import { CharacterSheetPage } from '@/pages/characters/character-sheet-page'
import { requireSession } from './-guards'

// The selected block lives in the URL so it deep-links and survives the back
// button — and, unlike the React version, it is the ONLY place it lives.
const searchSchema = z.object({ tab: z.string().optional() })

export const Route = createFileRoute('/characters/$id')({
  validateSearch: searchSchema,
  beforeLoad: requireSession,
  component: CharacterSheetPage,
})
