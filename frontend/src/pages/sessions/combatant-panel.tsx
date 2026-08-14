import { useQuery } from '@tanstack/solid-query'
import { X } from 'lucide-solid'
import { Show, createSignal } from 'solid-js'
import { characterQueryOptions } from '@/entities/character/queries'
import { CharacterHud } from '@/features/character-sheet/character-hud'
import { CharacterSheet } from '@/features/character-sheet/character-sheet'
import { ConditionsSection } from '@/features/character-sheet/conditions-section'
import type { InitiativeEntry } from '@/shared/realtime/realtime'
import { settledQuery } from '@/shared/lib/settled-query'
import { Button } from '@/shared/ui/button'
import { Skeleton } from '@/shared/ui/skeleton'
import { VitalBar } from '@/shared/ui/vital-bar'

/**
 * O combatente selecionado, na superfície principal do mestre (ALE-122).
 *
 * O que pinta PRIMEIRO é o cartão de combate, não a leitura: os verbos do
 * mestre em combate são aplicar dano, aplicar condição e conferir um número —
 * uma ficha de seis abas não responde nenhum deles em um clique. O `CharacterHud`
 * já É esse cartão (PV/PM com o diálogo de ajuste, que digita o valor e mostra
 * a prévia, e que roteia queda de PV como DANO, drenando PV temporários).
 *
 * Antes disto, alcançar a ficha de um jogador no meio de um turno era: sair da
 * sessão → aba Membros → um link de 29×16 px → voltar → continuar a sessão.
 */
export function CombatantPanel(props: { entry: InitiativeEntry; onClose: () => void }) {
  return (
    <section class="flex min-h-0 flex-1 flex-col">
      <header class="flex shrink-0 items-center justify-between gap-3 border-b border-grimorio-iron p-3 sm:px-4">
        <h2 class="min-w-0 truncate font-heading text-lg uppercase tracking-wide text-grimorio-gold">
          {props.entry.label}
        </h2>
        <Button size="sm" variant="outline" aria-label="Fechar o combatente" onClick={props.onClose}>
          <X aria-hidden="true" class="size-4" />
        </Button>
      </header>

      {/* SEM `overflow-y-auto` aqui: envolvendo tudo num contêiner que rola, a
          barra de abas da ficha rolava junto — era preciso descer a tela
          inteira para trocar de bloco. Agora o cartão é fixo e só o bloco ativo
          da ficha rola, que é o que faz a cena caber numa tela (ALE-122). */}
      <Show when={props.entry.characterId} fallback={<NpcCard entry={props.entry} />} keyed>
        {(characterId) => <CharacterCard characterId={characterId} />}
      </Show>
    </section>
  )
}

/**
 * O PC: o cartão de combate em cima, a ficha inteira embaixo.
 *
 * A ordem é a regra — os verbos do mestre em combate são aplicar dano e
 * conferir um número, e seis abas não respondem nenhum deles em um clique. As
 * abas ficam para quando a pergunta for mesmo "quanto ele carrega na mochila?".
 *
 * A ficha entra no layout de UM BLOCO POR VEZ mesmo em janela larga: aqui ela
 * vive numa coluna de 616–936px, e a escolha automática dela olha a JANELA —
 * numa de 1920 ela pegaria o layout largo dentro da coluna e cortaria.
 */
function CharacterCard(props: { characterId: number }) {
  const [tab, setTab] = createSignal('expertises')
  const character = useQuery(() => characterQueryOptions(props.characterId))
  // `settledQuery` e não `.data`: a leitura pendente suspende e desanexa a cena
  // inteira, deixando a tela em branco no lugar do esqueleto (ALE-96/121).
  const sheet = () => settledQuery(character)

  return (
    <Show
      when={sheet()}
      fallback={
        <div class="space-y-3 p-3" role="status" aria-label="Carregando o combatente">
          <Skeleton class="h-24 w-full" />
          <Skeleton class="h-16 w-full" />
        </div>
      }
    >
      {(data) => (
        <div class="flex min-h-0 flex-1 flex-col">
          <div class="shrink-0">
            <CharacterHud character={data()} dense class="border-t-0" />
          {/* "Você está caído" é a coisa mais frequente que um mestre declara
              em combate, e o editor já existia — mas só dentro da aba Efeitos,
              a três cliques. Aqui ele fica no cartão, junto do resto do que se
              faz no turno (ALE-122). */}
            <div class="border-t border-grimorio-iron px-3 py-2 sm:px-4">
              <ConditionsSection character={data()} />
            </div>
          </div>
          <div class="min-h-0 flex-1">
            <CharacterSheet
              character={data()}
              tab={tab()}
              onTabChange={setTab}
              inSession
              compact
              hudless
            />
          </div>
        </div>
      )}
    </Show>
  )
}

/**
 * Um NPC não tem ficha atrás dele — o rastreador É o registro. Mostra o que a
 * entrada guarda; o bloco do monstro (DEF, ataques) vem da fatia que passa a
 * guardar de qual criatura do bestiário ele veio.
 */
function NpcCard(props: { entry: InitiativeEntry }) {
  return (
    <div class="space-y-3 p-3 sm:p-4">
      <Show
        when={props.entry.hpMax !== undefined}
        fallback={
          <p class="text-sm text-muted-foreground">
            Este NPC foi criado à mão e não tem vida registrada.
          </p>
        }
      >
        <VitalBar
          kind="hp"
          label="PV"
          current={props.entry.hpCurrent ?? 0}
          max={props.entry.hpMax ?? 0}
        />
      </Show>
    </div>
  )
}
