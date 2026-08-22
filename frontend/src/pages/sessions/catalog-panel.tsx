import { CatalogBrowser } from '@/features/gm-tools/catalog-browser'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { SidePanel } from '@/shared/ui/side-panel'
import { MatchPeek } from './match-rail'

/**
 * Rules lookup without leaving the match. This is the panel the whole non-modal
 * decision was made for: on a laptop the tracker stays live behind it, so the
 * GM reads "Abalado: −2 em testes de perícia" on the right and applies it to
 * the goblin on the left without closing anything (ALE-75).
 *
 * Quem abre é o TRILHO das consultas, não um gatilho interno: o trilho garante
 * um overlay por vez, e um botão morando dentro do painel não teria como saber
 * que outro está aberto (ALE-198).
 */
export function CatalogPanel(props: {
  rt: SessionRealtime
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  return (
      <SidePanel
        open={props.open}
        onOpenChange={props.onOpenChange}
        title="Catálogos"
        description="Condições, magias, poderes e itens."
        header={<MatchPeek rt={props.rt} />}
      >
        {/* Sem teto de altura: dentro do painel a lista É o filho que rola, e
            o `max-h-[60vh]` que estava aqui deixava um vão morto embaixo dela —
            o scroll parava antes do fim do side sheet (ALE-122). */}
        <CatalogBrowser listClass="min-h-0 flex-1 pr-1" />
      </SidePanel>
  )
}
