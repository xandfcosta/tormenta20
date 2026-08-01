import {
  ATTRIBUTE_KEYS,
  type AttributeKey,
  POINT_BUY_BUDGET,
  POINT_BUY_MAX,
  POINT_BUY_MIN,
  pointBuySpent,
  pointBuyWarnings,
} from '@tormenta20/t20-data'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { FieldGroup } from '@/shared/ui/field'
import { cn } from '@/shared/lib/utils'
import { NumberField } from '@/features/character-build/number-field'
import {
  anyRacePending,
  appliedRaceDeltas,
  draftTormentaCarismaExtra,
} from '@/features/character-build/grant-helpers'
import {
  type AttributeMode,
  useCharacterDraftStore,
} from '@/features/character-build/character-draft-store'
import { useCreationWizard } from '@/features/character-build/creation-wizard-context'

const ATTR_LABEL: Record<AttributeKey, string> = {
  strength: 'Força',
  dexterity: 'Destreza',
  constitution: 'Constituição',
  intelligence: 'Inteligência',
  wisdom: 'Sabedoria',
  charisma: 'Carisma',
}

type AttrValues = Record<AttributeKey, number> & {
  races: string[]
  classPowers: string[]
  powerChoices: Record<string, string[]>
  originChoices: string[]
}

export function AtributosStep() {
  const { form, raceChoices } = useCreationWizard()
  const mode = useCharacterDraftStore((s) => s.attributeMode)
  const setMode = useCharacterDraftStore((s) => s.setAttributeMode)
  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-display tracking-wide">Atributos</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <ModeToggle mode={mode} onMode={setMode} />
        <form.Subscribe selector={(s: { values: AttrValues }) => s.values}>
          {(v: AttrValues) => {
            const d = appliedRaceDeltas(v.races, raceChoices)
            // Perda de CAR de poderes da Tormenta escolhidos (pool/origem) além
            // do swap da Deformidade — mantém o total igual à ficha salva.
            const carExtra = draftTormentaCarismaExtra(
              v.races,
              raceChoices,
              v.classPowers ?? [],
              v.powerChoices ?? {},
              v.originChoices ?? [],
            )
            if (carExtra) d.charisma = (d.charisma ?? 0) + carExtra
            return (
              <>
                {anyRacePending(v.races, raceChoices) && (
                  <p className="mb-3 text-[11px] text-[color:var(--hp-hurt)]">
                    Há escolhas de atributo de raça pendentes — os +1 não
                    definidos não serão aplicados ao salvar.
                  </p>
                )}
                {mode === 'point-buy' && <PointBuyStatus values={v} />}
                <FieldGroup className="grid gap-4 sm:grid-cols-2 sm:gap-6 lg:grid-cols-3">
                  {ATTRIBUTE_KEYS.map((k) => (
                    <NumberField
                      key={k}
                      form={form}
                      name={k}
                      label={ATTR_LABEL[k]}
                      min={mode === 'point-buy' ? POINT_BUY_MIN : -5}
                      max={mode === 'point-buy' ? POINT_BUY_MAX : 10}
                      raceDelta={d[k]}
                    />
                  ))}
                </FieldGroup>
              </>
            )
          }}
        </form.Subscribe>
      </CardContent>
    </Card>
  )
}

/** Free edit (default) vs. book point-buy (p17). Mode is draft-only UI state. */
function ModeToggle({
  mode,
  onMode,
}: {
  mode: AttributeMode
  onMode: (m: AttributeMode) => void
}) {
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {(
        [
          ['free', 'Livre'],
          ['point-buy', `Compra de pontos (${POINT_BUY_BUDGET} pts, p17)`],
        ] as const
      ).map(([m, label]) => (
        <button
          key={m}
          type="button"
          aria-pressed={mode === m}
          onClick={() => onMode(m)}
          className={cn(
            'rounded-md border px-2.5 py-1 text-xs transition-colors',
            mode === m
              ? 'border-primary bg-accent font-medium'
              : 'border-border text-muted-foreground hover:bg-accent',
          )}
        >
          {label}
        </button>
      ))}
      <p className="basis-full text-xs text-muted-foreground">
        {mode === 'free'
          ? 'Edite a base livremente (preset da classe); o total já inclui os bônus de raça.'
          : 'Todos os atributos começam em 0; distribua os pontos da Tabela 1-1. Bônus de raça não gastam pontos.'}
      </p>
    </div>
  )
}

function PointBuyStatus({ values }: { values: Record<AttributeKey, number> }) {
  const attrs = Object.fromEntries(
    ATTRIBUTE_KEYS.map((k) => [k, values[k] ?? 0]),
  ) as Record<AttributeKey, number>
  let spent: number | null = null
  try {
    spent = pointBuySpent(attrs)
  } catch {
    spent = null // base fora do intervalo (valores do modo livre) — warnings cobrem
  }
  const warnings = pointBuyWarnings(attrs)
  const over = spent !== null && spent > POINT_BUY_BUDGET
  return (
    <div className="space-y-1">
      <p
        className={cn(
          'text-xs font-semibold',
          over ? 'text-[color:var(--hp-hurt)]' : 'text-muted-foreground',
        )}
      >
        Pontos: {spent ?? '—'} de {POINT_BUY_BUDGET}
      </p>
      {warnings.map((w) => (
        <p key={w} className="text-[11px] text-[color:var(--hp-hurt)]">
          {w}
        </p>
      ))}
    </div>
  )
}
