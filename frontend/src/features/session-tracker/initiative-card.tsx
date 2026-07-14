import { useEffect, useRef, useState } from 'react'
import { useForm } from '@tanstack/react-form'
import { z } from 'zod'
import { Plus, Sparkles, Swords, Trash2 } from 'lucide-react'
import { SPELL_CATALOG } from '@tormenta20/t20-data'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader } from '@/shared/ui/card'
import {
  ConnectionChip,
  type ConnectionStatus,
} from '@/shared/ui/connection-chip'
import { Field, FieldError, FieldLabel } from '@/shared/ui/field'
import { HpBar } from '@/shared/ui/hp-bar'
import { Input } from '@/shared/ui/input'
import { MpBar } from '@/shared/ui/mp-bar'
import { NumberInput } from '@/shared/ui/number-input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/shared/ui/select'
import { SectionHeading } from '@/shared/ui/section-heading'
import type { InitiativeEntry, useSessionSocket } from '@/shared/realtime/realtime'
import { CombatantDrawer } from './combatant-drawer'
import { InitiativeRollButton } from './initiative-roll'
import { PartyRoster } from './party-roster'

/** Spell buffs a GM can push onto a combatant, resolved once from the catalog. */
const BUFF_SPELLS = Object.values(SPELL_CATALOG).filter((s) => s.buff)

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
  rt,
  isGm,
  campaignId,
  myCharacterIds,
}: {
  /** Shared session socket, owned by the page so tracker + bar + toasts
   * share one connection. */
  rt: ReturnType<typeof useSessionSocket>
  isGm: boolean
  campaignId: number
  myCharacterIds: Set<number>
}) {
  const status = deriveConnectionStatus(rt.isConnected, rt.error)
  // The viewer's own PC (players join with one) — for the self-roll button.
  const [myCharacterId] = myCharacterIds
  const [sheetCharId, setSheetCharId] = useState<number | null>(null)
  const [restCond, setRestCond] = useState<
    'ruim' | 'normal' | 'confortavel' | 'luxuosa'
  >('normal')

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
        {isGm && (
          <div className="flex flex-wrap justify-end gap-2">
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
        )}
      </CardHeader>
      <CardContent className="space-y-3">
        {rt.error && (
          <p className="text-sm text-destructive">
            Erro realtime: {rt.error}
          </p>
        )}

        <PartyRoster campaignId={campaignId} />

        {!isGm && myCharacterId !== undefined && (
          <InitiativeRollButton characterId={myCharacterId} rt={rt} />
        )}

        {isGm && (
          <div className="flex flex-wrap gap-2">
            <Button
              size="sm"
              variant="secondary"
              onClick={rt.populateParty}
              disabled={!rt.isConnected}
            >
              Adicionar grupo
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => rt.rest('scene')}
              disabled={!rt.isConnected}
            >
              Descanso de cena
            </Button>
            <div className="flex items-center gap-1">
              <Select
                value={restCond}
                onValueChange={(v) => setRestCond(v as typeof restCond)}
              >
                <SelectTrigger size="sm" className="w-[130px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="ruim">Ruim (½ nível)</SelectItem>
                  <SelectItem value="normal">Normal (nível)</SelectItem>
                  <SelectItem value="confortavel">Confortável (2×)</SelectItem>
                  <SelectItem value="luxuosa">Luxuosa (3×)</SelectItem>
                </SelectContent>
              </Select>
              <Button
                size="sm"
                variant="outline"
                onClick={() => rt.rest('day', restCond)}
                disabled={!rt.isConnected}
              >
                Descanso de dia
              </Button>
            </div>
          </div>
        )}

        {rt.state.initiative.length === 0 && (
          <p className="text-sm text-muted-foreground">
            {isGm
              ? 'Sem combatentes ainda. Adicione abaixo.'
              : 'Aguardando o mestre montar a iniciativa.'}
          </p>
        )}

        <div className="space-y-2">
          {rt.state.initiative.map((entry, idx) => {
            const isMine =
              entry.characterId !== undefined &&
              myCharacterIds.has(entry.characterId)
            const onTurn = idx === rt.state.turnIndex
            return (
            <InitiativeRow
              key={entry.id}
              entry={entry}
              onTurn={onTurn}
              // Pull the viewer's own combatant into view the moment its turn
              // starts — the "match" cue that pairs with the turn toast.
              focusOnTurn={onTurn && isMine}
              canEditVitals={isGm || isMine}
              canRemove={isGm}
              onOpenSheet={
                isGm && entry.characterId !== undefined
                  ? () => setSheetCharId(entry.characterId!)
                  : undefined
              }
              onDeltaHp={(delta) =>
                rt.deltaVitals(entry.id, { hpDelta: delta })
              }
              onApplyEffect={
                isGm && entry.characterId !== undefined
                  ? (spellId) => rt.applyEffect(entry.id, spellId)
                  : undefined
              }
              onRemove={() => rt.removeEntry(entry.id)}
            />
            )
          })}
        </div>

        {isGm && <AddCombatantForm rt={rt} />}
      </CardContent>
      <CombatantDrawer
        characterId={sheetCharId}
        onClose={() => setSheetCharId(null)}
      />
    </Card>
  )
}

function InitiativeRow({
  entry,
  onTurn,
  focusOnTurn,
  canEditVitals,
  canRemove,
  onOpenSheet,
  onDeltaHp,
  onApplyEffect,
  onRemove,
}: {
  entry: InitiativeEntry
  onTurn: boolean
  /** Scroll this row into view when its turn starts (viewer's own combatant). */
  focusOnTurn: boolean
  canEditVitals: boolean
  canRemove: boolean
  onOpenSheet?: () => void
  onDeltaHp: (delta: number) => void
  /** GM-only: push a spell buff onto this combatant's character. */
  onApplyEffect?: (spellId: string) => void
  onRemove: () => void
}) {
  const hasHp = entry.hpMax !== undefined && entry.hpCurrent !== undefined
  const hasMp = entry.mpMax !== undefined
  const rowRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (focusOnTurn)
      rowRef.current?.scrollIntoView({ block: 'center', behavior: 'smooth' })
  }, [focusOnTurn])
  return (
    <div
      ref={rowRef}
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
            {onOpenSheet ? (
              <button
                type="button"
                onClick={onOpenSheet}
                className="truncate text-left underline-offset-2 hover:underline"
                title="Ver ficha"
              >
                {entry.label}
              </button>
            ) : (
              <span className="truncate">{entry.label}</span>
            )}
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

      {(canEditVitals || canRemove || onApplyEffect) && (
        <div className="flex flex-wrap items-center justify-end gap-1">
          {onApplyEffect && <ApplyEffectSelect onApply={onApplyEffect} />}
          {canEditVitals &&
            [-5, -1, 1, 5].map((delta) => (
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
          {canRemove && (
            <Button
              size="sm"
              variant="ghost"
              onClick={onRemove}
              className="h-9 w-9 sm:h-8 sm:w-8"
              aria-label={`Remover ${entry.label}`}
            >
              <Trash2 className="size-4" />
            </Button>
          )}
        </div>
      )}
    </div>
  )
}

/**
 * GM picks a spell buff to push onto a combatant. Selecting a spell fires the
 * apply immediately; the Select value resets so the same buff can be re-applied
 * (e.g. refreshing a scene buff). Buffs are never auto-applied — this is the
 * explicit GM-targets-a-player affordance.
 */
function ApplyEffectSelect({ onApply }: { onApply: (spellId: string) => void }) {
  return (
    <Select
      value=""
      onValueChange={(spellId) => {
        if (spellId) onApply(spellId)
      }}
    >
      <SelectTrigger
        size="sm"
        className="h-9 w-9 justify-center p-0 sm:h-8 sm:w-8 [&>svg:last-child]:hidden"
        aria-label="Aplicar efeito"
        title="Aplicar efeito"
      >
        <Sparkles className="size-4" />
      </SelectTrigger>
      <SelectContent>
        {BUFF_SPELLS.map((spell) => (
          <SelectItem key={spell.id} value={spell.id}>
            {spell.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

/** A combatant entry: a required label, a whole-number initiative in the
 *  playable range, and whether it's a PC or an NPC. */
const addCombatantSchema = z.object({
  label: z
    .string()
    .trim()
    .min(1, 'Informe um nome.')
    .max(60, 'Máximo 60 caracteres.'),
  initiative: z
    .number()
    .int('Iniciativa deve ser inteiro.')
    .min(-5, 'Mínimo -5.')
    .max(40, 'Máximo 40.'),
  type: z.enum(['character', 'npc']),
})

/** GM-only "add combatant" row under the initiative list. Validated with
 *  zod + TanStack Form; resets to a fresh NPC row after each add. */
function AddCombatantForm({
  rt,
}: {
  rt: ReturnType<typeof useSessionSocket>
}) {
  const form = useForm({
    defaultValues: {
      label: '',
      initiative: 10,
      type: 'npc' as 'character' | 'npc',
    },
    validators: { onSubmit: addCombatantSchema },
    onSubmit: ({ value }) => {
      rt.addEntry({
        label: value.label.trim(),
        initiative: value.initiative,
        type: value.type,
      })
      form.reset()
    },
  })

  return (
    <form
      className="mt-3 flex flex-wrap items-end gap-2 rounded-md border border-dashed p-3"
      onSubmit={(e) => {
        e.preventDefault()
        e.stopPropagation()
        form.handleSubmit()
      }}
    >
      <form.Field
        name="label"
        validators={{ onChange: addCombatantSchema.shape.label }}
      >
        {(f) => {
          const invalid = f.state.meta.isTouched && !f.state.meta.isValid
          return (
            <Field data-invalid={invalid} className="min-w-[160px] flex-1">
              <FieldLabel htmlFor={f.name}>Nome</FieldLabel>
              <Input
                id={f.name}
                value={f.state.value}
                onChange={(e) => f.handleChange(e.target.value)}
                onBlur={f.handleBlur}
                placeholder="Goblin salteador…"
                aria-invalid={invalid}
              />
              {invalid && <FieldError errors={f.state.meta.errors} />}
            </Field>
          )
        }}
      </form.Field>
      <form.Field
        name="initiative"
        validators={{ onChange: addCombatantSchema.shape.initiative }}
      >
        {(f) => {
          const invalid = f.state.meta.isTouched && !f.state.meta.isValid
          return (
            <Field data-invalid={invalid} className="w-24">
              <FieldLabel htmlFor={f.name}>Iniciativa</FieldLabel>
              <NumberInput
                id={f.name}
                min={-5}
                max={40}
                value={f.state.value}
                onChange={(v) => f.handleChange(v)}
                onBlur={f.handleBlur}
                aria-invalid={invalid}
              />
              {invalid && <FieldError errors={f.state.meta.errors} />}
            </Field>
          )
        }}
      </form.Field>
      <form.Field name="type">
        {(f) => (
          <div className="flex gap-1">
            <Button
              type="button"
              variant={f.state.value === 'character' ? 'default' : 'outline'}
              size="sm"
              onClick={() => f.handleChange('character')}
            >
              PC
            </Button>
            <Button
              type="button"
              variant={f.state.value === 'npc' ? 'default' : 'outline'}
              size="sm"
              onClick={() => f.handleChange('npc')}
            >
              NPC
            </Button>
          </div>
        )}
      </form.Field>
      <form.Subscribe
        selector={(s) => s.canSubmit}
        children={(canSubmit) => (
          <Button type="submit" disabled={!rt.isConnected || !canSubmit}>
            <Plus className="mr-1 size-4" /> Adicionar
          </Button>
        )}
      />
    </form>
  )
}
