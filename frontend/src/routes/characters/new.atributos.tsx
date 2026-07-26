import { createFileRoute } from '@tanstack/react-router'
import { AtributosStep } from '@/pages/characters/wizard/atributos-step'

export const Route = createFileRoute('/characters/new/atributos')({
  component: AtributosStep,
})
