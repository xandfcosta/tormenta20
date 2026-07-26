import { createFileRoute } from '@tanstack/react-router'
import { OrigemStep } from '@/pages/characters/wizard/origem-step'

export const Route = createFileRoute('/characters/new/origem')({
  component: OrigemStep,
})
