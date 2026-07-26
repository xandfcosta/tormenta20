import { createFileRoute } from '@tanstack/react-router'
import { IdentidadeStep } from '@/pages/characters/wizard/identidade-step'

export const Route = createFileRoute('/characters/new/identidade')({
  component: IdentidadeStep,
})
