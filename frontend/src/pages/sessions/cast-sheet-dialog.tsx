import { useQuery } from '@tanstack/solid-query'
import { X } from 'lucide-solid'
import { Show, createSignal } from 'solid-js'
import { characterQueryOptions } from '@/entities/character/queries'
import { CharacterSheet } from '@/features/character-sheet/character-sheet'
import { CombatantBand } from '@/features/character-sheet/combatant-band'
import { SceneContainerProvider } from '@/shared/lib/scene-container'
import { settledQuery } from '@/shared/lib/settled-query'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/shared/ui/dialog'
import { Skeleton } from '@/shared/ui/skeleton'

/**
 * A ficha de um PC aberta pelo ELENCO, inteira e editável (ALE-212).
 *
 * A decisão do dono é que o mestre edita tudo, não só os vitais: se um número
 * está errado no meio da noite, ele conserta ali. O servidor já concordava —
 * `authorizedCharacter` deixa passar o dono, o admin **e o mestre da campanha**
 * —, então isto não abriu porta nenhuma; abriu o caminho até uma porta que já
 * estava destrancada.
 *
 * **A linha da crônica não é enfeite.** O que o mestre edita é o SNAPSHOT desta
 * campanha e não o personagem do jogador (ALE-33): o mesmo herói em duas
 * crônicas tem duas fichas. O risco que a issue nomeia não é ele estragar o
 * personagem — é ele ACHAR que estragou, ou achar que consertou o de outra
 * mesa. A frase fica no cabeçalho, junto do que ele está prestes a mexer, e não
 * na gaveta que o diálogo cobre.
 *
 * Difere do `CombatantDialog` no que ele NÃO tem: aqui não há linha de
 * iniciativa atrás, então não há aplicar efeito nem condição de NPC. É ficha,
 * não combatente — e é essa a distinção que a issue inteira persegue.
 *
 * @example <CastSheetDialog characterId={aberto()} chronicle="Snapshot Test" onClose={…} />
 */
export function CastSheetDialog(props: {
  characterId: number | null
  /** O nome da crônica, que é o que a linha de aviso precisa dizer. */
  chronicle: string
  onClose: () => void
}) {
  const [caixa, setCaixa] = createSignal<HTMLElement | null>(null)

  return (
    <Dialog open={props.characterId !== null} onOpenChange={(open) => !open && props.onClose()}>
      <DialogContent
        ref={setCaixa}
        showCloseButton={false}
        // A mesma medida de OBJETO do diálogo do combatente: dentro vai uma
        // ficha inteira com a barra de blocos, e é ela que decide o piso. A
        // altura mora na regra `.dialogo-de-ficha` pelo motivo escrito lá
        // (ALE-200) — como valor arbitrário ela sumia do CSS de forma
        // intermitente e derrubou o e2e três vezes.
        class="dialogo-de-ficha flex w-full max-w-4xl flex-col gap-0 overflow-hidden p-0 sm:max-w-4xl"
      >
        {/* O provider é o mesmo remédio da ALE-198: o diálogo é modal, e um
            popup nascido em `document.body` fica do lado de fora — o Kobalte
            marca os irmãos e o clique nunca chega. A ficha tem seletor de
            condição e combobox dentro. */}
        <SceneContainerProvider element={caixa}>
          <Show when={props.characterId} keyed>
            {(characterId) => (
              <CastSheet characterId={characterId} chronicle={props.chronicle} onClose={props.onClose} />
            )}
          </Show>
        </SceneContainerProvider>
      </DialogContent>
    </Dialog>
  )
}

function CastSheet(props: { characterId: number; chronicle: string; onClose: () => void }) {
  const [tab, setTab] = createSignal('expertises')
  const character = useQuery(() => characterQueryOptions(props.characterId))
  // `settledQuery` e não `.data`: a leitura pendente SUSPENDE e desanexa a cena
  // inteira, deixando a tela em branco no lugar do esqueleto (ALE-96/121).
  const sheet = () => settledQuery(character)

  return (
    <Show
      when={sheet()}
      fallback={
        <div class="space-y-3 p-4" role="status" aria-label="Carregando a ficha">
          <Skeleton class="h-24 w-full" />
          <Skeleton class="h-16 w-full" />
        </div>
      }
    >
      {(data) => (
        <section class="flex min-h-0 flex-1 flex-col">
          {/* O nome fica SÓ na faixa, que já o desenha: um cabeçalho com o nome
              acima dela era a mesma palavra duas vezes em 40px — exatamente o
              que a ALE-145 tirou do painel do combatente, e que eu reintroduzi
              aqui até ver a tela. Para quem OUVE, o nome continua: ele está no
              título do diálogo, que é o que se anuncia ao abrir. */}
          <DialogTitle class="sr-only">
            Ficha de {data().name} nesta crônica
          </DialogTitle>
          <header class="flex shrink-0 items-center justify-between gap-3 border-b border-grimorio-iron px-3 py-2 sm:px-4">
            {/* A frase que responde "qual ficha é esta?" antes de a dúvida
                existir. Ela é a `DialogDescription`, então quem ouve o diálogo
                abrir ouve o aviso junto do nome. */}
            <DialogDescription class="min-w-0 truncate text-xs text-muted-foreground">
              Ficha desta crônica · {props.chronicle}
            </DialogDescription>
            <Button
              size="sm"
              variant="outline"
              class="shrink-0"
              aria-label="Fechar a ficha"
              onClick={props.onClose}
            >
              <X aria-hidden="true" class="size-4" />
            </Button>
          </header>

          {/* A MESMA composição do combatente: faixa pequena em cima, ficha
              inteira embaixo. A ordem é a regra — os verbos do mestre são
              conferir um número e ajustar vitais, e sete abas não respondem
              nenhum deles em um clique (ALE-145). */}
          <CombatantBand character={data()} />
          <div class="min-h-0 flex-1">
            <CharacterSheet
              character={data()}
              tab={tab()}
              onTabChange={setTab}
              inSession
              compact
              hudless
              glance
            />
          </div>
        </section>
      )}
    </Show>
  )
}
