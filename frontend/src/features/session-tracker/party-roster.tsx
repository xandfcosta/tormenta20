import { useQuery } from '@tanstack/react-query'
import { Badge } from '@/shared/ui/badge'
import { HpBar } from '@/shared/ui/hp-bar'
import { MpBar } from '@/shared/ui/mp-bar'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import type { CampaignMember } from '@/shared/api/api'

type PartyCharacter = NonNullable<CampaignMember['character']>

/**
 * The campaign party at a glance inside the Iniciativa section — every
 * player character with name, classe/nível and live PV/PM, shown to GM and
 * players alike (shared table state). Distinct from the initiative order
 * list below it, which is the compact "who acts when" view.
 */
export function PartyRoster({ campaignId }: { campaignId: number }) {
  const members = useQuery(campaignMembersQueryOptions(campaignId))
  const party = (members.data ?? []).flatMap((m) =>
    m.role === 'player' && m.character ? [m.character] : [],
  )
  if (party.length === 0) return null

  return (
    <div className="space-y-1.5">
      <h3 className="text-xs font-bold uppercase tracking-widest text-muted-foreground">
        Grupo
      </h3>
      <div className="grid gap-2 sm:grid-cols-2">
        {party.map((c) => (
          <PartyMember key={c.id} character={c} />
        ))}
      </div>
    </div>
  )
}

function PartyMember({ character }: { character: PartyCharacter }) {
  const cls = character.classes
    .map((c) => `${c.className} ${c.level}`)
    .join(' / ')
  return (
    <div className="space-y-1.5 rounded-lg border border-border/60 bg-card/60 p-2.5">
      <div className="flex items-baseline justify-between gap-2">
        <span className="truncate font-medium">{character.name}</span>
        <Badge variant="secondary" className="shrink-0">
          Nv {character.level}
        </Badge>
      </div>
      {cls && <p className="truncate text-xs text-muted-foreground">{cls}</p>}
      <HpBar current={character.hpCurrent} max={character.hpMax} size="sm" />
      <MpBar current={character.mpCurrent} max={character.mpMax} size="sm" />
    </div>
  )
}
