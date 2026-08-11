import { useQuery, useQueryClient } from '@tanstack/solid-query'
import { Link } from '@tanstack/solid-router'
import { Crown, Trash2, Users } from 'lucide-solid'
import { For, Show, createSignal } from 'solid-js'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import { classLevelLine } from '@/entities/character/class-line'
import { type CampaignMember, api } from '@/shared/api/api'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { Button } from '@/shared/ui/button'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { SkeletonRows } from '@/shared/ui/skeleton'
import { InviteButton } from './invite-button'
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
  const queryClient = useQueryClient()

  /** Drops a member and clears the character's now-stale "Campanhas" list. */
  const remove = async (member: CampaignMember) => {
    await api.members.remove(props.campaignId, member.id)
    await queryClient.invalidateQueries({
      queryKey: campaignMembersQueryOptions(props.campaignId).queryKey,
    })
    await queryClient.invalidateQueries({
      queryKey: ['characters', member.characterId, 'campaigns'],
    })
  }

  return (
    <TomeSection
      eyebrow="A Mesa"
      title="Heróis"
      action={<Show when={props.isGm}>{<InviteButton campaignId={props.campaignId} />}</Show>}
    >
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
          <For each={roster()}>
            {(member) => (
              <MemberPlate
                member={member}
                canRemove={props.isGm}
                onRemove={() => remove(member)}
              />
            )}
          </For>
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

export type MemberPlateProps = {
  member: CampaignMember
  /** Only the GM may drop someone from the muster. */
  canRemove: boolean
  onRemove: () => Promise<void>
}

export function MemberPlate(props: MemberPlateProps) {
  const name = () => memberName(props.member)
  const character = () => props.member.character
  const isGm = () => props.member.role === 'gm'

  return (
    <div class="group relative flex items-center gap-3 rounded-sm border border-grimorio-iron bg-[var(--grimorio-panel)] p-3">
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
      <Show when={props.canRemove}>
        <RemoveMemberButton name={name()} onRemove={props.onRemove} />
      </Show>
    </div>
  )
}

/**
 * Drops a hero from the muster. Hidden until hover/focus so the roster reads as
 * a roster, and `data-nav-skip` keeps it out of the arrow-key cursor — nobody
 * should land on "remove" by walking the grid. On a touch screen there IS no
 * hover, so it stays visible there or the GM could never remove anyone.
 */
function RemoveMemberButton(props: { name: string; onRemove: () => Promise<void> }) {
  const [pending, setPending] = createSignal(false)

  const remove = async () => {
    setPending(true)
    try {
      await props.onRemove()
    } finally {
      setPending(false)
    }
  }

  return (
    <Button
      variant="ghost"
      size="icon-sm"
      data-nav-skip
      class="absolute top-1.5 right-1.5 opacity-0 transition-opacity pointer-coarse:opacity-100 focus-visible:opacity-100 group-hover:opacity-100"
      onClick={remove}
      disabled={pending()}
      aria-label={`Remover ${props.name}`}
    >
      <Trash2 aria-hidden="true" class="size-4" />
    </Button>
  )
}
