import { createFileRoute } from '@tanstack/react-router'
import { RacaStep } from '@/pages/characters/wizard/raca-step'

export const Route = createFileRoute('/characters/new/raca')({
  component: RacaStep,
})
