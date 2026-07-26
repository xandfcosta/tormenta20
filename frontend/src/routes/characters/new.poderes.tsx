import { createFileRoute } from '@tanstack/react-router'
import { PoderesStep } from '@/pages/characters/wizard/poderes-step'

export const Route = createFileRoute('/characters/new/poderes')({
  component: PoderesStep,
})
