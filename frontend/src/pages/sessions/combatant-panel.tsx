import { useQuery } from '@tanstack/solid-query'
import { X } from 'lucide-solid'
import { type JSX, Show, createSignal } from 'solid-js'
import { characterQueryOptions } from '@/entities/character/queries'
import { CombatantBand } from '@/features/character-sheet/combatant-band'
import { ApplyEffectSelect } from '@/features/session-tracker/apply-effect-select'
import { CharacterSheet } from '@/features/character-sheet/character-sheet'
import { bestiaryCatalogQueryOptions } from '@/entities/catalog/queries'
import { MonsterStatBlock } from '@/features/gm-tools/monster-stat-block'
import { settledQuery } from '@/shared/lib/settled-query'
import type { InitiativeEntry } from '@/shared/realtime/realtime'
import { Button } from '@/shared/ui/button'
import { Skeleton } from '@/shared/ui/skeleton'
import { VitalBar } from '@/shared/ui/vital-bar'

/**
 * O combatente selecionado, na superfície principal do mestre (ALE-122).
 *
 * O que pinta PRIMEIRO é a faixa do combatente, não a leitura: os verbos do
 * mestre em combate são aplicar dano, aplicar condição e conferir um número —
 * uma ficha de sete abas não responde nenhum deles em um clique. A
 * `CombatantBand` é essa faixa (PV/PM com o diálogo de ajuste, que digita o
 * valor e mostra a prévia, e que roteia queda de PV como DANO, drenando PV
 * temporários; Defesa; condições). Ela usava o `CharacterHud` inteiro até a
 * ALE-145, e o cartão comia metade da região antes de a ficha começar.
 *
 * Antes disto, alcançar a ficha de um jogador no meio de um turno era: sair da
 * sessão → aba Membros → um link de 29×16 px → voltar → continuar a sessão.
 */
export function CombatantPanel(props: {
  entry: InitiativeEntry
  onClose: () => void
  /** Aplicar um buff neste combatente. Saiu da linha da iniciativa, onde um
   *  select de 9px abria uma lista de 31 magias cobrindo a tela (ALE-122). */
  onApplyEffect?: (spellId: string) => void
}) {
  const actions = (
    <>
      <Show when={props.onApplyEffect}>{(apply) => <ApplyEffectSelect onApply={apply()} />}</Show>
      <Button size="sm" variant="outline" aria-label="Fechar o combatente" onClick={props.onClose}>
        <X aria-hidden="true" class="size-4" />
      </Button>
    </>
  )

  return (
    <section class="flex min-h-0 flex-1 flex-col">
      {/* SEM `overflow-y-auto` aqui: envolvendo tudo num contêiner que rola, a
          barra de abas da ficha rolava junto — era preciso descer a tela
          inteira para trocar de bloco. Agora a faixa é fixa e só o bloco ativo
          da ficha rola, que é o que faz a cena caber numa tela (ALE-122). */}
      {/* O PC não tem cabeçalho próprio: a faixa já diz o nome, e um cabeçalho
          acima dela repetia a mesma palavra por 61px — quase 40% da região no
          celular deitado, onde ela mede 165px (ALE-145). O NPC, que não tem
          faixa, mantém o seu. */}
      <Show
        when={props.entry.characterId}
        fallback={<NpcHeader entry={props.entry} actions={actions} />}
        keyed
      >
        {(characterId) => <CharacterCard characterId={characterId} actions={actions} />}
      </Show>
    </section>
  )
}

/** O NPC não tem ficha atrás dele, então o nome dele mora num cabeçalho. */
function NpcHeader(props: { entry: InitiativeEntry; actions: JSX.Element }) {
  return (
    <>
      <header class="flex shrink-0 items-center justify-between gap-3 border-b border-grimorio-iron p-3 sm:px-4">
        <h2 class="min-w-0 truncate font-heading text-lg uppercase tracking-wide text-grimorio-gold">
          {props.entry.label}
        </h2>
        <div class="flex shrink-0 items-center gap-2">{props.actions}</div>
      </header>
      <NpcCard entry={props.entry} />
    </>
  )
}

/**
 * O PC: a faixa pequena em cima, a ficha inteira embaixo.
 *
 * A ordem é a regra — os verbos do mestre em combate são aplicar dano e
 * conferir um número, e sete abas não respondem nenhum deles em um clique. As
 * abas ficam para quando a pergunta for mesmo "quanto ele carrega na mochila?".
 *
 * A ficha entra no layout de UM BLOCO POR VEZ mesmo em janela larga: aqui ela
 * vive numa coluna de 616–936px, e a escolha automática dela olha a JANELA —
 * numa de 1920 ela pegaria o layout largo dentro da coluna e cortaria.
 */
function CharacterCard(props: { characterId: number; actions: JSX.Element }) {
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
          {/* A faixa é pequena por construção, então não precisa mais do teto de
              45% com rolagem por dentro que a ALE-125 pôs aqui: aquilo tratava o
              sintoma de um cartão mais alto que a tela inteira, e a causa era o
              cartão (ALE-145). Quem garante que a barra de abas da ficha
              continua alcançável é a asserção de alcance do `session.spec.ts`. */}
          <CombatantBand character={data()} actions={props.actions} />
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
        </div>
      )}
    </Show>
  )
}

/**
 * Um NPC não tem ficha atrás dele — o rastreador É o registro. A vida vem da
 * entrada; o resto (DEF, ataques, habilidades) vem do verbete do bestiário de
 * onde ele foi arrastado, pelo `monsterId` que a linha passou a guardar.
 *
 * `settledQuery` e não `bestiary.data`: a leitura pendente SUSPENDE e o
 * `Suspense` do route match desanexa a cena inteira — é a armadilha que já
 * apareceu quatro vezes nesta issue (ALE-122).
 */
function NpcCard(props: { entry: InitiativeEntry }) {
  const bestiary = useQuery(() => bestiaryCatalogQueryOptions)
  const monster = () =>
    props.entry.monsterId === undefined
      ? undefined
      : settledQuery(bestiary)?.find((creature) => creature.id === props.entry.monsterId)

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

      {/* Sem bloco quando a linha foi digitada à mão (não veio do bestiário) —
          e o silêncio é a resposta certa: inventar zeros diria o que não é. */}
      <Show when={monster()}>{(creature) => <MonsterStatBlock monster={creature()} />}</Show>
    </div>
  )
}
