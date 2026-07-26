import { createFileRoute } from '@tanstack/react-router'
import { ClasseStep } from '@/pages/characters/wizard/classe-step'

export const Route = createFileRoute('/characters/new/classe')({
  component: ClasseStep,
})
