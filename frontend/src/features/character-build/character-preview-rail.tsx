import { Card } from '@/shared/ui/card'
import { StatCell } from '@/shared/ui/stat-cell'
import { cn } from '@/shared/lib/utils'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { type RaceChoiceState, raceAttributeDeltas } from './grant-helpers'
import { type StepSlug, stepReady, WIZARD_STEPS } from './wizard-steps'

// TanStack Form's API type is heavily generic; `any` keeps the seam usable
// without threading a dozen type parameters (mirrors ClassEntryRow/NumberField).
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type FormApi = any

/**
 * Live character preview — a sticky splash echoing the select screen
 * (hue gradient + keystone initials + DEF·PV·PM triple) plus an accreting
 * step checklist that lights up as each step completes. Morphs as the player
 * fills the wizard, closing the create→view visual loop. DEF is an armor-free
 * estimate (10 + final Destreza).
 */
export function CharacterPreviewRail({
  form,
  raceChoices,
  current,
}: {
  form: FormApi
  raceChoices: RaceChoiceState
  current?: StepSlug
}) {
  return (
    <form.Subscribe selector={(s: { values: WizardValues }) => s.values}>
      {(values: WizardValues) => (
        <PreviewCard
          values={values}
          raceChoices={raceChoices}
          current={current}
        />
      )}
    </form.Subscribe>
  )
}

type WizardValues = {
  name: string
  races: string[]
  origin: string
  size: string
  classes: { className: string; level: number }[]
  dexterity: number
  hpMax: number
  mpMax: number
  hpCurrent: number
  mpCurrent: number
  intelligence: number
  strength: number
  constitution: number
  wisdom: number
  charisma: number
  god?: string
  displacement: number
  classPowers: string[]
  originChoices: string[]
  trainedExpertises: string[]
  classChoices: Record<string, { devoto?: string; caminho?: string }>
  powerChoices: Record<string, string[]>
}

function PreviewCard({
  values,
  raceChoices,
  current,
}: {
  values: WizardValues
  raceChoices: RaceChoiceState
  current?: StepSlug
}) {
  const name = values.name.trim() || 'Novo personagem'
  const hue = hueFromName(name)
  const primary = values.classes[0]
  const role = primary?.className
    ? `${primary.className} · Nv ${primary.level}`
    : 'Sem classe'
  const dexDelta = raceAttributeDeltas(values.races, raceChoices).dexterity ?? 0
  const defense = 10 + values.dexterity + dexDelta

  return (
    <Card className="gap-0 overflow-hidden p-0 lg:self-start">
      <div
        className="relative flex aspect-[4/3] items-center justify-center"
        style={{
          background: `linear-gradient(155deg, oklch(0.55 0.15 ${hue}) 0%, oklch(0.30 0.09 ${hue}) 70%, oklch(0.22 0.06 ${hue}) 100%)`,
        }}
      >
        <span className="select-none font-display text-[7rem] leading-none text-white/15">
          {initials(name)}
        </span>
        <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/75 via-black/25 to-transparent p-3 pt-8">
          <p className="text-[10px] font-semibold uppercase tracking-widest text-white/70">
            {role}
          </p>
          <p className="truncate text-lg font-semibold text-white drop-shadow">
            {name}
          </p>
        </div>
      </div>
      <div className="grid grid-cols-3 gap-2 p-3">
        <StatCell label="DEF">{defense}</StatCell>
        <StatCell label="PV">{values.hpMax}</StatCell>
        <StatCell label="PM" dim={values.mpMax === 0}>
          {values.mpMax}
        </StatCell>
      </div>
      <ul className="space-y-1 border-t border-border p-3 text-sm">
        {WIZARD_STEPS.filter((s) => s.slug !== 'resumo').map((step) => {
          const done = stepReady(step.slug, values, raceChoices)
          const isCurrent = step.slug === current
          return (
            <li key={step.slug} className="flex items-center gap-2">
              <span
                className={cn(
                  'w-3 shrink-0 text-center',
                  isCurrent
                    ? 'text-foreground'
                    : done
                      ? 'text-primary'
                      : 'text-muted-foreground',
                )}
              >
                {isCurrent ? '▸' : done ? '✓' : '○'}
              </span>
              <span
                className={cn(
                  'w-20 shrink-0',
                  isCurrent ? 'font-medium' : 'text-muted-foreground',
                )}
              >
                {step.label}
              </span>
              <span className="min-w-0 flex-1 truncate text-right text-muted-foreground">
                {stepSummary(step.slug, values)}
              </span>
            </li>
          )
        })}
      </ul>
    </Card>
  )
}

function stepSummary(slug: StepSlug, v: WizardValues): string {
  switch (slug) {
    case 'raca':
      return v.races.join(', ')
    case 'classe':
      return v.classes[0]?.className ?? ''
    case 'origem':
      return v.origin
    case 'vitalidade':
      return v.hpMax ? `PV ${v.hpMax} · PM ${v.mpMax}` : ''
    case 'identidade':
      return v.name.trim() ? `${v.size}` : ''
    default:
      return ''
  }
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  return parts
    .slice(0, 2)
    .map((w) => w[0])
    .join('')
    .toUpperCase()
}
