import { Show, createSignal } from 'solid-js'
import type { CampaignCreature } from '@/shared/api/creature-types'
import { SceneContainerProvider } from '@/shared/lib/scene-container'
import type { InitiativeEntry } from '@/shared/realtime/realtime'
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/shared/ui/dialog'
import { CombatantPanel } from './combatant-panel'

/**
 * A ficha do combatente como OVERLAY (ALE-198).
 *
 * Era uma das cinco abas de uma região permanente, e essa região custava
 * metade da tela para, na maior parte do tempo, dizer "clique no nome de um
 * combatente". Ver a ficha é consulta: abre-se com uma pergunta na cabeça
 * ("quanto ele ainda aguenta?", "tem Reflexos treinado?") e se fecha em
 * seguida. O que fica na tela é o tabuleiro.
 *
 * Um caminho só, com dois gatilhos que fazem a MESMA coisa: o nome no trilho da
 * iniciativa e a peça no tabuleiro. Antes disto os dois gestos divergiam por
 * largura de janela — acima de 1536 a peça trocava a aba, abaixo não trocava —,
 * e a razão daquela divergência morre aqui: o overlay não rouba a região de
 * ninguém, então nunca há motivo para não abri-lo (ALE-161).
 *
 * O ✕ é o do PRÓPRIO painel, que já existia: o do diálogo seria um segundo
 * botão de fechar a um centímetro do primeiro.
 *
 * O diálogo se declara CONTÊINER DE CENA para o que abre dentro dele, e isso
 * não é estética: ele é modal, e um popup que nasce em `document.body` fica do
 * lado de fora — o Kobalte marca os irmãos do modal e o clique nunca chega. O
 * seletor de condição abria, a lista pintava por cima, e escolher "Abalado" não
 * fazia nada. Com o provider, condição e efeito nascem DENTRO e voltam a
 * responder (ALE-198).
 *
 * @example <CombatantDialog entry={selecionado()} onClose={() => setSelectedId(null)} … />
 */
export function CombatantDialog(props: {
  entry: InitiativeEntry | null
  campaignId: number
  onClose: () => void
  onApplyEffect: (entryId: string, spellId: string) => void
  onConditions: (entryId: string, conditions: string[]) => void
  onLinkCreature: (entry: InitiativeEntry, creature: CampaignCreature) => void
}) {
  const [caixa, setCaixa] = createSignal<HTMLElement | null>(null)

  return (
    <Dialog open={props.entry !== null} onOpenChange={(open) => !open && props.onClose()}>
      <DialogContent
        ref={setCaixa}
        showCloseButton={false}
        // Alto e largo por medida de OBJETO: dentro vai uma ficha inteira com a
        // barra de blocos, e é ela que decide o piso. A altura é fixa e não
        // `max-h`, senão o painel — que é `flex-1` por construção — não teria
        // de quem herdar altura e a ficha colapsaria em zero.
        //
        // A altura mora numa REGRA (`.dialogo-de-ficha` no index.css) e não num
        // valor arbitrário: como `h-[92vh]` ela sumia do CSS de propósito
        // nenhum, de forma intermitente, e derrubou o e2e três vezes. O porquê
        // está escrito lá.
        class="dialogo-de-ficha flex w-full max-w-4xl flex-col gap-0 overflow-hidden p-0 sm:max-w-4xl"
      >
        <SceneContainerProvider element={caixa}>
        <Show when={props.entry}>
          {(entry) => (
            <>
              {/* O Kobalte exige título e descrição para nomear o diálogo, e o
                  painel já os desenha no próprio cabeçalho — repetir na tela
                  seria o nome duas vezes em 61px, que é o que a ALE-145 tirou
                  daqui. Então eles ficam só para quem ouve.

                  "Ficha de X" e não "X": o nome puro empatava com o cabeçalho
                  da faixa, e a árvore acessível ficava com dois `h2` idênticos
                  na mesma caixa. O que se anuncia ao entrar no diálogo é o que
                  ele É, não só de quem ele fala. */}
              <DialogTitle class="sr-only">Ficha de {entry().label}</DialogTitle>
              <DialogDescription class="sr-only">
                A ficha do combatente, com vitais, condições e perícias.
              </DialogDescription>
              <CombatantPanel
                entry={entry()}
                campaignId={props.campaignId}
                onClose={props.onClose}
                onApplyEffect={(spellId) => props.onApplyEffect(entry().id, spellId)}
                onConditions={(conditions) => props.onConditions(entry().id, conditions)}
                onLinkCreature={(creature) => props.onLinkCreature(entry(), creature)}
              />
            </>
          )}
        </Show>
        </SceneContainerProvider>
      </DialogContent>
    </Dialog>
  )
}
