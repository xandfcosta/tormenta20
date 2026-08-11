import { useQuery } from '@tanstack/solid-query'
import { Link } from '@tanstack/solid-router'
import { Crown, Users } from 'lucide-solid'
import { For, Show } from 'solid-js'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import { classLevelLine } from '@/entities/character/class-line'
import type { CampaignMember } from '@/shared/api/api'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { SkeletonRows } from '@/shared/ui/skeleton'
import { TomeSection } from './tome-section'

/** GM first (the crest leads the muster), then the rest in arrival order. */
export function sortRoster(members: readonly CampaignMember[]): CampaignMember[] {
  return [...members].sort((a, b) => (a.role === b.role ? 0 : a.role === 'gm' ? -1 : 1))
}

/** A member's display name — the character's, or a placeholder by id. */
export function memberName(member: CampaignMember): string {
  return member.character?.name ?? `Personagem ${member.characterId}`
}

/**
 * Membros section as a PARTY ROSTER — every hero as a portrait plate (the GM
 * crowned), laid out as a grid that grows columns with the tome page. Players
 * are never added here; the GM shares an invite link and each player joins
 * with a character of their own.
 */
export function MembersCard(props: { campaignId: number; isGm: boolean }) {
  const members = useQuery(() => campaignMembersQueryOptions(props.campaignId))
  const roster = () => sortRoster(members.data ?? [])

  return (
    <TomeSection eyebrow="A Mesa" title="Heróis">
      <Show when={props.isGm}>
        <p class="text-sm text-muted-foreground">
          Jogadores entram pelo link de convite com um personagem próprio.
        </p>
      </Show>
      <Show when={members.isLoading}>
        <SkeletonRows count={2} />
      </Show>
      <Show when={!members.isLoading && roster().length === 0}>
        <EmptyMuster />
      </Show>
      <Show when={roster().length > 0}>
        <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          <For each={roster()}>{(member) => <MemberPlate member={member} />}</For>
        </div>
      </Show>
    </TomeSection>
  )
}

/** No members yet — an empty muster awaiting the first hero. */
function EmptyMuster() {
  return (
    <div class="flex flex-col items-center gap-2 rounded-sm border border-dashed border-grimorio-iron px-4 py-10 text-center">
      <Users aria-hidden="true" class="size-6 text-muted-foreground" />
      <p class="text-sm text-muted-foreground">Nenhum personagem inscrito ainda.</p>
    </div>
  )
}

function MemberPlate(props: { member: CampaignMember }) {
  const name = () => memberName(props.member)
  const character = () => props.member.character
  const isGm = () => props.member.role === 'gm'

  return (
    <div class="flex items-center gap-3 rounded-sm border border-grimorio-iron bg-[var(--grimorio-panel)] p-3">
      <CharacterPortrait name={name()} size="sm" hue={hueFromName(name())} />
      <div class="min-w-0 flex-1">
        <p class="flex items-center gap-1.5 truncate font-medium text-foreground">
          {name()}
          <Show when={isGm()}>
            <Crown aria-hidden="true" class="size-3.5 shrink-0 text-grimorio-gold" />
          </Show>
        </p>
        <p class="truncate text-xs text-muted-foreground">
          {classLevelLine(character()?.classes ?? []) || `Nv ${character()?.level ?? 1}`}
        </p>
      </div>
      <Show when={character()}>
        {(char) => (
          <Link
            to="/characters/$id"
            params={{ id: String(char().id) }}
            class="shrink-0 text-xs text-grimorio-gold hover:underline"
          >
            Ficha
          </Link>
        )}
      </Show>
    </div>
  )
}
