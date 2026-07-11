import { useState } from 'react'
import { Plus, Swords, Trash2 } from 'lucide-react'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader } from '@/shared/ui/card'
import {
  ConnectionChip,
  type ConnectionStatus,
} from '@/shared/ui/connection-chip'
import { HpBar } from '@/shared/ui/hp-bar'
import { Input } from '@/shared/ui/input'
import { MpBar } from '@/shared/ui/mp-bar'
import { NumberInput } from '@/shared/ui/number-input'
import { SectionHeading } from '@/shared/ui/section-heading'
import { useSessionSocket, type InitiativeEntry } from '@/shared/realtime/realtime'

// Maps realtime hook state onto ConnectionChip's tri-state. The socket
// hook only reports `isConnected` + `error`; we infer 'reconnecting' as
// "not connected AND no fatal error yet" so a flicker between attempts
// shows the spinner instead of the offline glyph.
function deriveConnectionStatus(
  isConnected: boolean,
  error: string | null,
): ConnectionStatus {
  if (isConnected) return 'connected'
  if (error) return 'offline'
  return 'reconnecting'
}

export function InitiativeCard({
  campaignId,
  sessionId,
}: {
  campaignId: number
  sessionId: number
}) {
  const rt = useSessionSocket(campaignId, sessionId)
  const status = deriveConnectionStatus(rt.isConnected, rt.error)
  const [addLabel, setAddLabel] = useState('')
  const [addInit, setAddInit] = useState(10)
  const [addType, setAddType] = useState<'character' | 'npc'>('npc')

  const submitAdd = () => {
    const label = addLabel.trim()
    if (!label) return
    rt.addEntry({ label, initiative: addInit, type: addType })
    setAddLabel('')
    setAddInit(10)
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-3">
        <div className="space-y-2">
          <SectionHeading variant="kallyadranoch" as="h2">
            Iniciativa
          </SectionHeading>
          <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
            <ConnectionChip status={status} dirty={rt.hasPersistenceWarning} />
            <span className="font-hud tabular-nums">
              Rodada {rt.state.round}
            </span>
          </div>
        </div>
        <div className="flex gap-2">
          <Button size="sm" onClick={rt.nextTurn} disabled={!rt.isConnected}>
            Próximo turno
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={rt.resetInitiative}
            disabled={!rt.isConnected}
          >
            Reset
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {rt.error && (
          <p className="text-sm text-destructive">
            Erro realtime: {rt.error}
          </p>
        )}

        {rt.state.initiative.length === 0 && (
          <p className="text-sm text-muted-foreground">
            Sem combatentes ainda. Adicione abaixo.
          </p>
        )}

        <div className="space-y-2">
          {rt.state.initiative.map((entry, idx) => (
            <InitiativeRow
              key={entry.id}
              entry={entry}
              onTurn={idx === rt.state.turnIndex}
              onDeltaHp={(delta) =>
                rt.deltaVitals(entry.id, { hpDelta: delta })
              }
              onRemove={() => rt.removeEntry(entry.id)}
            />
          ))}
        </div>

        <div className="mt-3 flex flex-wrap items-end gap-2 rounded-md border border-dashed p-3">
          <div className="min-w-[160px] flex-1">
            <label className="text-xs font-medium" htmlFor="add-label">
              Nome
            </label>
            <Input
              id="add-label"
              value={addLabel}
              onChange={(e) => setAddLabel(e.target.value)}
              placeholder="Goblin salteador…"
            />
          </div>
          <div className="w-24">
            <label className="text-xs font-medium" htmlFor="add-init">
              Iniciativa
            </label>
            <NumberInput
              id="add-init"
              min={-5}
              max={40}
              value={addInit}
              onChange={setAddInit}
            />
          </div>
          <div className="flex gap-1">
            <Button
              variant={addType === 'character' ? 'default' : 'outline'}
              size="sm"
              onClick={() => setAddType('character')}
            >
              PC
            </Button>
            <Button
              variant={addType === 'npc' ? 'default' : 'outline'}
              size="sm"
              onClick={() => setAddType('npc')}
            >
              NPC
            </Button>
          </div>
          <Button
            onClick={submitAdd}
            disabled={!rt.isConnected || !addLabel.trim()}
          >
            <Plus className="mr-1 size-4" /> Adicionar
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

function InitiativeRow({
  entry,
  onTurn,
  onDeltaHp,
  onRemove,
}: {
  entry: InitiativeEntry
  onTurn: boolean
  onDeltaHp: (delta: number) => void
  onRemove: () => void
}) {
  const hasHp = entry.hpMax !== undefined && entry.hpCurrent !== undefined
  const hasMp = entry.mpMax !== undefined
  return (
    <div
      data-on-turn={onTurn ? 'true' : 'false'}
      className={
        'flex flex-col gap-2 rounded-md border p-2.5 text-sm sm:flex-row sm:items-center sm:gap-3 ' +
        (onTurn
          ? 'border-[color:var(--primary)]/60 bg-[color-mix(in_oklch,var(--primary)_6%,transparent)]'
          : 'border-border/60')
      }
    >
      <div className="flex items-center gap-2">
        <Badge variant="outline" className="font-hud tabular-nums">
          {entry.initiative}
        </Badge>
        <div className="min-w-0 flex-1">
          <p className="flex flex-wrap items-center gap-1 truncate font-medium">
            <span className="truncate">{entry.label}</span>
            <Badge
              variant={entry.type === 'character' ? 'default' : 'secondary'}
              className="text-[10px] uppercase tracking-widest"
            >
              {entry.type === 'character' ? 'PC' : 'NPC'}
            </Badge>
            {onTurn && (
              <Badge className="gap-1 text-[10px] uppercase tracking-widest">
                <Swords className="size-3" /> Na vez
              </Badge>
            )}
          </p>
        </div>
      </div>

      {(hasHp || hasMp) && (
        <div className="flex-1 space-y-1.5 sm:min-w-[180px]">
          {hasHp && (
            <HpBar
              current={entry.hpCurrent!}
              max={entry.hpMax!}
              size="sm"
              label="PV"
            />
          )}
          {hasMp && (
            <MpBar
              current={entry.mpCurrent ?? 0}
              max={entry.mpMax!}
              size="sm"
              label="PM"
            />
          )}
        </div>
      )}

      <div className="flex items-center justify-end gap-1">
        {[-5, -1, 1, 5].map((delta) => (
          <Button
            key={delta}
            size="sm"
            variant="outline"
            onClick={() => onDeltaHp(delta)}
            className="h-9 min-w-9 font-hud tabular-nums sm:h-8 sm:min-w-8"
            aria-label={`Ajustar PV em ${delta}`}
          >
            {delta > 0 ? `+${delta}` : delta}
          </Button>
        ))}
        <Button
          size="sm"
          variant="ghost"
          onClick={onRemove}
          className="h-9 w-9 sm:h-8 sm:w-8"
          aria-label={`Remover ${entry.label}`}
        >
          <Trash2 className="size-4" />
        </Button>
      </div>
    </div>
  )
}
