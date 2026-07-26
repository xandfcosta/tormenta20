import { createFileRoute } from '@tanstack/react-router'
import { VitalidadeStep } from '@/pages/characters/wizard/vitalidade-step'

export const Route = createFileRoute('/characters/new/vitalidade')({
  component: VitalidadeStep,
})
