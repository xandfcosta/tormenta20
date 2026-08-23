import { useQuery } from '@tanstack/solid-query'
import { Users } from 'lucide-solid'
import { For } from 'solid-js'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import { connectedCharacterIds } from '@/features/session-tracker/tracker-rules'
import { settledQuery } from '@/shared/lib/settled-query'
import { cn } from '@/shared/lib/utils'
import type { PresenceUser } from '@/shared/realtime/realtime'
import { CharacterPortrait } from '@/shared/ui/character-portrait'

/**
 * O ELENCO recolhido: o segundo bloco do trilho esquerdo (ALE-211).
 *
 * Recolhido ele mostra só os JOGADORES, porque a pergunta que o trilho responde
 * o dia inteiro é "quem está na mesa e como ele está" — os NPCs do mestre são
 * preparação e vivem na gaveta (ALE-212).
 *
 * Este é o bloco que ROLA, e a assimetria com a fila é deliberada: a fila não
 * pode rolar porque a posição dela CARREGA significado (acima já agiu, abaixo
 * ainda vai), e rolar embaralharia isso. O elenco é uma lista sem ordem própria,
 * então rolar não custa leitura nenhuma.
 *
 * @example <CastRail campaignId={1} present={rt.present()} onOpenCharacter={abrir} />
 */
export function CastRail(props: {
  campaignId: number
  present: PresenceUser[]
  onOpenCharacter: (characterId: number) => void
  /** Abre a gaveta do elenco, onde moram os NPCs e o criar (ALE-212). */
  onExpand: () => void
  class?: string
}) {
  const members = useQuery(() => campaignMembersQueryOptions(props.campaignId))

  /** Mesmo filtro do "Adicionar grupo" no servidor (`listPlayerCombatants`). */
  const players = () => (settledQuery(members) ?? []).filter((member) => member.role === 'player')
  const online = () => connectedCharacterIds(settledQuery(members) ?? [], props.present)

  return (
    <nav
      aria-label="Elenco"
      class={cn('grimorio-frame flex flex-col gap-1 bg-grimorio-panel p-1', props.class)}
    >
      {/* O cabeçalho é o mesmo desenho do da fila: um botão de ícone que abre a
          gaveta. Dois blocos irmãos com dois cabeçalhos diferentes seriam duas
          gramáticas para a mesma ideia. */}
      <button
        type="button"
        aria-label="Abrir o elenco"
        onClick={props.onExpand}
        class="flex shrink-0 items-center justify-center rounded-sm border border-grimorio-iron py-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
      >
        <Users aria-hidden="true" class="size-4" />
      </button>

      <ul class="flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto">
        <For each={players()}>
          {(member) => (
            <li>
              <CastRailEntry
                name={member.character?.name ?? `Personagem ${member.characterId}`}
                online={online().has(member.characterId)}
                onOpen={() => props.onOpenCharacter(member.characterId)}
              />
            </li>
          )}
        </For>
      </ul>
    </nav>
  )
}

/**
 * Um jogador em 72px: o retrato de iniciais e o contorno que diz se ele está na
 * mesa AGORA.
 *
 * O contorno é anel e não ponto porque em 80px de trilho um ponto de 8px ao
 * lado do retrato disputaria a largura com ele; em volta, ele não custa espaço
 * nenhum. E ele não está sozinho: anel colorido não existe para quem usa leitor
 * de tela, então o nome acessível diz "na mesa" ou "fora da mesa" — a mesma
 * frase que a gaveta usa, para as duas telas não inventarem vocabulários.
 */
function CastRailEntry(props: { name: string; online: boolean; onOpen: () => void }) {
  return (
    <button
      type="button"
      onClick={props.onOpen}
      aria-label={`Abrir a ficha de ${props.name} — ${props.online ? 'na mesa' : 'fora da mesa'}`}
      title={props.name}
      class="flex w-full items-center justify-center rounded-sm border border-grimorio-iron p-1 transition-colors hover:bg-accent"
    >
      <CharacterPortrait
        name={props.name}
        size="sm"
        class={cn(
          'size-11 ring-1',
          props.online ? 'ring-[color:var(--hp-full)]' : 'ring-grimorio-iron',
        )}
      />
    </button>
  )
}
