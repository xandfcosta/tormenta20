import { useQuery } from '@tanstack/solid-query'
import { Pencil, X } from 'lucide-solid'
import { type JSX, Show, createSignal } from 'solid-js'
import { characterQueryOptions } from '@/entities/character/queries'
import { CombatantBand } from '@/features/character-sheet/combatant-band'
import { ApplyEffectSelect } from '@/features/session-tracker/apply-effect-select'
import { CharacterSheet } from '@/features/character-sheet/character-sheet'
import { bestiaryCatalogQueryOptions } from '@/entities/catalog/queries'
import { MonsterStatBlock } from '@/features/gm-tools/monster-stat-block'
import { CreatureBlockDialog } from '@/features/gm-tools/creature-block-dialog'
import { CreatureStatBlock } from '@/features/gm-tools/creature-stat-block'
import { creatureFromMonster } from '@/features/gm-tools/creature-from-monster'
import { NpcConditions } from '@/features/gm-tools/npc-conditions'
import { campaignCreaturesQueryOptions } from '@/entities/creature/queries'
import { blankCreatureBlock } from '@/shared/api/creature-types'
import type { CampaignCreature } from '@/shared/api/creature-types'
import { settledQuery } from '@/shared/lib/settled-query'
import type { InitiativeEntry } from '@/shared/realtime/realtime'
import { Button } from '@/shared/ui/button'
import { Skeleton } from '@/shared/ui/skeleton'
import { VitalBar } from '@/shared/ui/vital-bar'
import { SectionTitle } from '@/shared/ui/section-label'

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
  /** A campanha dona dos blocos de criatura do mestre (ALE-137). */
  campaignId: number
  /** Aplicar um buff neste combatente. Saiu da linha da iniciativa, onde um
   *  select de 9px abria uma lista de 31 magias cobrindo a tela (ALE-122). */
  onApplyEffect?: (spellId: string) => void
  /** Liga esta linha ao bloco recém-criado. Recebe a criatura inteira porque
   *  quem liga também decide o que herdar dela (a vida, quando a linha não
   *  tem). */
  onLinkCreature?: (creature: CampaignCreature) => void
  /** Condições do NPC: elas moram na LINHA, porque NPC não tem ficha. */
  onConditions?: (conditions: string[]) => void
}) {
  const actions = (
    <>
      {/* `min-w-0 flex-1`: na faixa este campo divide a linha com as condições
          e com o fechar, e sem poder encolher ele empurrava o botão de fechar
          para fora da tela a 390px — media 231px fixos (ALE-147). */}
      <Show when={props.onApplyEffect}>
        {(apply) => <ApplyEffectSelect onApply={apply()} class="min-w-[7rem] flex-1" />}
      </Show>
      <Button
        size="sm"
        variant="outline"
        class="shrink-0"
        aria-label="Fechar o combatente"
        onClick={props.onClose}
      >
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
        fallback={
          <NpcHeader
            entry={props.entry}
            actions={actions}
            campaignId={props.campaignId}
            onLinkCreature={props.onLinkCreature}
            onConditions={props.onConditions}
          />
        }
        keyed
      >
        {(characterId) => <CharacterCard characterId={characterId} actions={actions} />}
      </Show>
    </section>
  )
}

/** O NPC não tem ficha atrás dele, então o nome dele mora num cabeçalho. */
function NpcHeader(props: {
  entry: InitiativeEntry
  actions: JSX.Element
  campaignId: number
  onLinkCreature?: (creature: CampaignCreature) => void
  onConditions?: (conditions: string[]) => void
}) {
  return (
    <>
      <header class="flex shrink-0 items-center justify-between gap-3 border-b border-grimorio-iron p-3 sm:px-4">
        <SectionTitle contexto="painel" class="min-w-0 truncate">
          {props.entry.label}
        </SectionTitle>
        <div class="flex shrink-0 items-center gap-2">{props.actions}</div>
      </header>
      <NpcCard
        entry={props.entry}
        campaignId={props.campaignId}
        onLinkCreature={props.onLinkCreature}
        onConditions={props.onConditions}
      />
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
 * Um NPC não tem ficha de personagem atrás dele: o livro modela NPC e monstro
 * na MESMA forma — "BANDIDO · ND 1/4 · Humanoide (humano) Médio" (p289) tem o
 * bloco do Ogro. Então aqui há três casos, do mais completo ao mais pobre:
 *
 * 1. **Bloco do MESTRE** (`creatureId`): editável, é a resposta da ALE-137 ao
 *    "o mestre não pode mudar nada do NPC".
 * 2. **Verbete do livro** (`monsterId`): consulta, imutável — e um botão para
 *    copiá-lo num bloco próprio quando o mestre quiser modificá-lo.
 * 3. **Digitado à mão**: só a barra de PV, com o convite a detalhar.
 *
 * `settledQuery` e não `.data`: a leitura pendente SUSPENDE e o `Suspense` do
 * route match desanexa a cena inteira — a armadilha que já apareceu quatro
 * vezes nesta issue (ALE-122).
 */
function NpcCard(props: {
  entry: InitiativeEntry
  campaignId: number
  /** Liga a linha ao bloco recém-criado, para o painel abrir nele da próxima. */
  onLinkCreature?: (creature: CampaignCreature) => void
  onConditions?: (conditions: string[]) => void
}) {
  const bestiary = useQuery(() => bestiaryCatalogQueryOptions)
  const creatures = useQuery(() => campaignCreaturesQueryOptions(props.campaignId))
  const monster = () =>
    props.entry.monsterId === undefined
      ? undefined
      : settledQuery(bestiary)?.find((creature) => creature.id === props.entry.monsterId)
  const creature = () =>
    props.entry.creatureId === undefined
      ? undefined
      : settledQuery(creatures)?.find((c) => c.id === props.entry.creatureId)

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

      {/* "Você está caído" é o que o mestre mais declara, e até aqui só o PC
          podia recebê-lo — o NPC não tem ficha onde guardar (ALE-122). */}
      <Show when={props.onConditions}>
        {(aplicar) => (
          <NpcConditions active={props.entry.conditions} onChange={aplicar()} />
        )}
      </Show>

      {/* O bloco do mestre ganha do verbete: se ele modificou o ogro, é o ogro
          dele que vale na mesa. */}
      <Show
        when={creature()}
        fallback={
          <Show when={monster()}>
            {(verbete) => (
              <div class="space-y-3">
                <MonsterStatBlock monster={verbete()} />
                <CreatureBlockDialog
                  campaignId={props.campaignId}
                  seed={{ name: verbete().name, block: creatureFromMonster(verbete()) }}
                  onSaved={(saved) => props.onLinkCreature?.(saved)}
                  trigger={(open) => (
                    <Button type="button" size="sm" variant="outline" onClick={open}>
                      <Pencil aria-hidden="true" class="size-3.5" /> Detalhar este {verbete().name}
                    </Button>
                  )}
                />
              </div>
            )}
          </Show>
        }
      >
        {(bloco) => (
          <div class="space-y-3">
            <CreatureStatBlock block={bloco().block} />
            <CreatureBlockDialog
              campaignId={props.campaignId}
              creature={bloco()}
              trigger={(open) => (
                <Button type="button" size="sm" variant="outline" onClick={open}>
                  <Pencil aria-hidden="true" class="size-3.5" /> Editar bloco
                </Button>
              )}
            />
          </div>
        )}
      </Show>

      {/* Digitado à mão e sem bloco: o convite a detalhar é o que faltava — o
          mestre guardava PM, perícias e itens na cabeça ou num papel. */}
      <Show when={!creature() && !monster()}>
        <CreatureBlockDialog
          campaignId={props.campaignId}
          seed={{ name: props.entry.label, block: blankCreatureBlock() }}
          onSaved={(saved) => props.onLinkCreature?.(saved)}
          trigger={(open) => (
            <Button type="button" size="sm" variant="outline" onClick={open}>
              <Pencil aria-hidden="true" class="size-3.5" /> Detalhar este NPC
            </Button>
          )}
        />
      </Show>
    </div>
  )
}
