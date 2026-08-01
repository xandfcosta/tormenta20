import { createFileRoute } from '@tanstack/react-router'
import { EquipamentoStep } from '@/pages/characters/wizard/equipamento-step'

export const Route = createFileRoute('/characters/new/equipamento')({
  component: EquipamentoStep,
})
