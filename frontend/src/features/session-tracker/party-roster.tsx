import { useQuery } from '@tanstack/solid-query'
import { For, Show, createMemo } from 'solid-js'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import type { CampaignMember } from '@/shared/api/api'
import { Skeleton } from '@/shared/ui/skeleton'
import { VitalBar } from '@/shared/ui/vital-bar'
import { settledQuery } from '@/shared/lib/settled-query'
import { FieldLabel } from '@/shared/ui/section-label'

type PartyCharacter = NonNullable<CampaignMember['character']>

/**
 * The party at a glance: every player character with class/level and live
 * PV/PM, shown to GM and players alike. Distinct from the initiative order
 * below it — this answers "how is everyone doing", not "who acts when".
 */
export function PartyRoster(props: { campaignId: number }) {
  const members = useQuery(() => campaignMembersQueryOptions(props.campaignId))
  const party = createMemo(() =>
    (settledQuery(members) ?? []).flatMap((member) =>
      member.role === 'player' && member.character ? [member.character] : [],
    ),
  )

  return (
    <Show when={!members.isLoading} fallback={<PartyRosterSkeleton />}>
      <Show when={party().length > 0}>
        <div class="space-y-1.5">
          <FieldLabel as="h3" class="text-xs font-bold">
            Grupo
          </FieldLabel>
          <div class="grid gap-2 sm:grid-cols-2">
            <For each={party()}>{(character) => <PartyMember character={character} />}</For>
          </div>
        </div>
      </Show>
    </Show>
  )
}

/** Keeps the panel's height while the roster loads. */
function PartyRosterSkeleton() {
  return (
    <div class="space-y-1.5">
      <Skeleton class="h-3 w-16" />
      <div class="grid gap-2 sm:grid-cols-2">
        <Skeleton class="h-24 w-full rounded-none" />
        <Skeleton class="h-24 w-full rounded-none" />
      </div>
    </div>
  )
}

function PartyMember(props: { character: PartyCharacter }) {
  const classes = () =>
    props.character.classes.map((c) => `${c.className} ${c.level}`).join(' / ')

  return (
    <div class="space-y-1.5 rounded-none border border-border/60 bg-card/60 p-2.5">
      <div class="flex items-baseline justify-between gap-2">
        <span class="truncate font-medium">{props.character.name}</span>
        <FieldLabel class="shrink-0 rounded-none bg-muted px-1.5">
          Nv {props.character.level}
        </FieldLabel>
      </div>
      <Show when={classes()}>
        {(line) => <p class="truncate text-xs text-muted-foreground">{line()}</p>}
      </Show>
      <VitalBar
        kind="hp"
        label="PV"
        current={props.character.hpCurrent}
        max={props.character.hpMax}
      />
      <VitalBar
        kind="mp"
        label="PM"
        current={props.character.mpCurrent}
        max={props.character.mpMax}
      />
    </div>
  )
}
