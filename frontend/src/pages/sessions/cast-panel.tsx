import { useQuery } from '@tanstack/solid-query'
import { Plus, Trash2 } from 'lucide-solid'
import { For, Show, createMemo } from 'solid-js'
import { campaignCreaturesQueryOptions } from '@/entities/creature/queries'
import { connectedCharacterIds } from '@/features/session-tracker/tracker-rules'
import { CreatureBlockDialog } from '@/features/gm-tools/creature-block-dialog'
import { creatureActions } from '@/features/gm-tools/creature-mutations'
import { blankCreatureBlock } from '@/shared/api/creature-types'
import type { CampaignCreature } from '@/shared/api/creature-types'
import { settledQuery } from '@/shared/lib/settled-query'
import type { CampaignMember } from '@/shared/api/types'
import type { PresenceUser } from '@/shared/realtime/realtime'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { SectionTitle } from '@/shared/ui/section-label'
import { Skeleton } from '@/shared/ui/skeleton'
import { useQueryClient } from '@tanstack/solid-query'

/**
 * O ELENCO: quem existe na crônica fora da fila do combate (ALE-212).
 *
 * O conceito que faltava é que **iniciativa não é lista de combatentes**. Até
 * aqui a única forma de alguém existir na sessão era já estar na fila: não
 * havia onde guardar o taverneiro que não briga, nem como olhar a ficha de um
 * jogador fora do combate, nem como preparar o chefe da semana que vem sem
 * revelá-lo ao pôr uma linha na mesa.
 *
 * É da CAMPANHA e não da sessão, e a evidência é de um lado só: `campaign_creatures`
 * já vive por campanha (ALE-137), o roster também, e "os NPCs voltam semana que
 * vem" só é verdade assim. Por sessão exigiria armazenamento novo para PERDER a
 * propriedade que a issue pede.
 *
 * Quem põe um NPC na iniciativa é o mestre, QUANDO ele quiser (ALE-211) — criar
 * aqui não põe ninguém na fila. Essa separação é o assunto inteiro da issue.
 */
export function CastPanel(props: {
  campaignId: number
  /** Os membros vêm da PÁGINA, não de uma consulta daqui: abrir consulta dentro
   *  da cena desanexa a árvore (ALE-199, e visto de novo no trilho da ALE-211). */
  members: readonly CampaignMember[]
  /** A sala ao vivo, para dizer quem está na mesa AGORA. */
  present: PresenceUser[]
  /** Abre a ficha inteira de um PC — o diálogo mora na página, que é quem compõe. */
  onOpenCharacter: (characterId: number) => void
}) {
  const creatures = useQuery(() => campaignCreaturesQueryOptions(props.campaignId))
  const queryClient = useQueryClient()
  const actions = createMemo(() => creatureActions(queryClient, props.campaignId))

  /**
   * Só quem tem papel de JOGADOR, que é exatamente o filtro do "Adicionar
   * grupo" no servidor (`listPlayerCombatants`). O mestre costuma ter um PC
   * próprio no roster, e listá-lo aqui sob "Jogadores" criaria uma linha que o
   * "Adicionar grupo" nunca traz — duas telas discordando sobre quem é o grupo.
   */
  const players = () => props.members.filter((member) => member.role === 'player')
  const online = () => connectedCharacterIds(props.members, props.present)
  const npcs = () => settledQuery(creatures) ?? []

  return (
    <div class="space-y-5">
      <section class="space-y-2">
        <SectionTitle contexto="painel" class="text-sm">
          Jogadores · {players().length}
        </SectionTitle>
        <Show
          when={players().length > 0}
          fallback={<Vazio>Nenhum personagem no elenco desta crônica ainda.</Vazio>}
        >
          <ul class="space-y-1">
            <For each={players()}>
              {(member) => (
                <li>
                  <PlayerRow
                    name={member.character?.name ?? `Personagem ${member.characterId}`}
                    subtitle={classesOf(member.character?.classes)}
                    online={online().has(member.characterId)}
                    onOpen={() => props.onOpenCharacter(member.characterId)}
                  />
                </li>
              )}
            </For>
          </ul>
        </Show>
      </section>

      <section class="space-y-2">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <SectionTitle contexto="painel" class="text-sm">
            Meus NPCs · {npcs().length}
          </SectionTitle>
          {/* Criar SEM linha de iniciativa é a novidade: até aqui o bloco só
              nascia a partir do "+ Combatente completo", ou seja, o mestre era
              obrigado a pôr o vilão na mesa para poder escrevê-lo (ALE-137). */}
          <CreatureBlockDialog
            campaignId={props.campaignId}
            seed={{ name: '', block: blankCreatureBlock() }}
            trigger={(open) => (
              <Button size="sm" variant="outline" class="gap-1.5" onClick={open}>
                <Plus aria-hidden="true" class="size-4" />
                Criar NPC
              </Button>
            )}
          />
        </div>
        <Show when={!creatures.isPending} fallback={<Skeleton class="h-20 w-full" />}>
          <Show
            when={npcs().length > 0}
            fallback={<Vazio>Nada escrito ainda. O que você criar aqui só você vê.</Vazio>}
          >
            <ul class="space-y-1">
              <For each={npcs()}>
                {(creature) => (
                  <li>
                    <NpcRow creature={creature} campaignId={props.campaignId} onRemove={() => actions().remove(creature.id)} />
                  </li>
                )}
              </For>
            </ul>
          </Show>
        </Show>
      </section>
    </div>
  )
}

function Vazio(props: { children: string }) {
  return <p class="text-sm text-muted-foreground">{props.children}</p>
}

/** "Arcanista 9 · Guerreiro 2", ou nada quando o roster não trouxe classe. */
function classesOf(classes: { className: string; level: number }[] | undefined): string {
  if (!classes || classes.length === 0) return ''
  return classes.map((c) => `${c.className} ${c.level}`).join(' · ')
}

/**
 * Um PC do elenco. A linha inteira abre a ficha.
 *
 * O ponto verde diz quem está na mesa AGORA, e ele NÃO está sozinho: um ponto
 * colorido não existe para quem usa leitor de tela, e "conectado" é informação,
 * não enfeite. O nome acessível carrega a resposta junto.
 */
function PlayerRow(props: {
  name: string
  subtitle: string
  online: boolean
  onOpen: () => void
}) {
  return (
    <button
      type="button"
      onClick={props.onOpen}
      aria-label={`Abrir a ficha de ${props.name} — ${props.online ? 'na mesa' : 'fora da mesa'}`}
      class="flex w-full items-center gap-2 rounded-sm border border-grimorio-iron px-2.5 py-2 text-left transition-colors hover:border-grimorio-gold hover:bg-accent"
    >
      <span
        aria-hidden="true"
        class={
          props.online
            ? 'size-2 shrink-0 rounded-full bg-[color:var(--hp-full)]'
            : 'size-2 shrink-0 rounded-full border border-grimorio-iron'
        }
      />
      <span class="min-w-0 flex-1">
        <span class="block truncate text-sm">{props.name}</span>
        <Show when={props.subtitle}>
          <span class="block truncate text-xs text-muted-foreground">{props.subtitle}</span>
        </Show>
      </span>
    </button>
  )
}

/**
 * Um NPC do mestre. Clicar abre o bloco EDITÁVEL direto (decisão do dono): ele
 * é do mestre, ninguém mais o vê, e o gesto mais comum é consertar um número.
 *
 * O remover fica FORA do gatilho que abre — dois papéis no mesmo alvo é o que a
 * ALE-200 desfez noutro lugar — e pede confirmação, porque apagar o bloco é o
 * único caminho sem volta daqui.
 */
function NpcRow(props: { creature: CampaignCreature; campaignId: number; onRemove: () => void }) {
  return (
    <div class="flex items-center gap-1 rounded-sm border border-grimorio-iron pr-1 transition-colors hover:border-grimorio-gold">
      <CreatureBlockDialog
        campaignId={props.campaignId}
        creature={props.creature}
        trigger={(open) => (
          <button
            type="button"
            onClick={open}
            class="flex min-w-0 flex-1 items-baseline gap-2 px-2.5 py-2 text-left"
          >
            <span class="min-w-0 flex-1 truncate text-sm">{props.creature.name}</span>
            <span class="shrink-0 font-mono text-xs tabular-nums text-muted-foreground">
              {props.creature.block.hp} PV
            </span>
          </button>
        )}
      />
      <ConfirmDialog
        title={`Apagar "${props.creature.name}"?`}
        description="O bloco sai da crônica. Quem já está na fila do combate continua lá, mas sem bloco atrás dele."
        confirmLabel="Apagar"
        destructive
        onConfirm={props.onRemove}
        trigger={(open) => (
          <Button
            size="sm"
            variant="ghost"
            class="h-8 w-8 shrink-0"
            aria-label={`Apagar ${props.creature.name}`}
            onClick={open}
          >
            <Trash2 aria-hidden="true" class="size-4" />
          </Button>
        )}
      />
    </div>
  )
}
