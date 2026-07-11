import { useQuery } from '@tanstack/react-query'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/shared/ui/sheet'
import { Skeleton } from '@/shared/ui/skeleton'
import { Badge } from '@/shared/ui/badge'
import { characterSheetQueryOptions } from '@/entities/character/queries'
import type { CharacterWithComputed, ComputedSheet } from '@/shared/api/api'

const ATTR_LABELS: Record<string, string> = {
  strength: 'For',
  dexterity: 'Des',
  constitution: 'Con',
  intelligence: 'Int',
  wisdom: 'Sab',
  charisma: 'Car',
}

function signed(n: number): string {
  return n >= 0 ? `+${n}` : `${n}`
}

/**
 * GM peek drawer — tap a combatant tied to a Character to see its
 * server-computed stats (attributes, vitals, defense, saves) without
 * leaving the tracker. Read-only; the full sheet lives at
 * /characters/:id. Reuses the same `characterSheet` query the sheet
 * page uses, so an open combatant hydrates from cache when warm.
 */
export function CombatantDrawer({
  characterId,
  onClose,
}: {
  characterId: number | null
  onClose: () => void
}) {
  const open = characterId !== null
  const query = useQuery({
    ...characterSheetQueryOptions(characterId ?? 0),
    enabled: open,
  })
  const data = query.data as CharacterWithComputed | undefined

  return (
    <Sheet open={open} onOpenChange={(o) => !o && onClose()}>
      <SheetContent className="w-full overflow-y-auto sm:max-w-md">
        <SheetHeader>
          <SheetTitle>{data?.name ?? 'Combatente'}</SheetTitle>
          <SheetDescription>
            {data ? `Nível ${data.computed.level}` : 'Carregando ficha…'}
          </SheetDescription>
        </SheetHeader>

        {query.isLoading && (
          <div className="space-y-3 p-4">
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-24 w-full" />
          </div>
        )}
        {data && <StatPeek computed={data.computed} />}
      </SheetContent>
    </Sheet>
  )
}

function StatPeek({ computed }: { computed: ComputedSheet }) {
  return (
    <div className="space-y-4 p-4 text-sm">
      <div className="grid grid-cols-3 gap-2 sm:grid-cols-6">
        {Object.entries(computed.attributes).map(([key, a]) => (
          <div key={key} className="rounded-md border p-2 text-center">
            <p className="text-xs text-muted-foreground">
              {ATTR_LABELS[key] ?? key}
            </p>
            <p className="font-hud text-lg font-semibold">{signed(a.total)}</p>
          </div>
        ))}
      </div>

      <div className="flex flex-wrap gap-2">
        <Badge variant="secondary">
          PV {computed.vitals.pvCurrent}/{computed.vitals.pvMax}
        </Badge>
        <Badge variant="secondary">
          PM {computed.vitals.pmCurrent}/{computed.vitals.pmMax}
        </Badge>
        <Badge variant="outline">Defesa {computed.defense.total}</Badge>
        <Badge variant="outline">Desloc. {computed.deslocamento}m</Badge>
      </div>

      <div className="grid grid-cols-3 gap-2">
        <SaveStat label="Fort" value={computed.saves.fortitude} />
        <SaveStat label="Ref" value={computed.saves.reflexos} />
        <SaveStat label="Von" value={computed.saves.vontade} />
      </div>
    </div>
  )
}

function SaveStat({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border p-2 text-center">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="font-hud font-semibold">{signed(value)}</p>
    </div>
  )
}
