import { createFileRoute } from '@tanstack/solid-router'
import { GrimorioPage } from '@/pages/grimorio/grimorio-page'
import { requireSession } from './-guards'

/**
 * A folha de especificação do sistema de desenho (ALE-173). Atrás do mesmo
 * guarda das outras cenas: ela não expõe nada do jogo, mas é ferramenta de
 * quem constrói, não de quem joga.
 */
export const Route = createFileRoute('/grimorio')({
  beforeLoad: requireSession,
  component: GrimorioPage,
})
