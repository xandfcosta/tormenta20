import { Link } from '@tanstack/react-router'
import { ATTRIBUTE_ABBR, ATTRIBUTE_KEYS } from '@tormenta20/t20-data'
import type { ReactNode } from 'react'
import { Card } from '@/shared/ui/card'
import { StatCell } from '@/shared/ui/stat-cell'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { useCreationWizard } from './creation-wizard-context'
import {
  anyRacePending,
  type RaceChoiceState,
  appliedRaceDeltas,
  raceGrant,
  resolveRaceDeltas,
} from './grant-helpers'
import {
  AbilityDisclosure,
  ClassGrantPanel,
  DeltaBadges,
  OriginGrantPanel,
} from './grant-panels'
import { deriveDraftVitals } from './draft-vitals'
import type { CharacterFormValues, StepSlug } from './wizard-steps'

/**
 * Final Resumo — a select-screen-style hero (hue splash + DEF·PV·PM triple)
 * over grouped, expandable sections that reuse the mid-flow grant renderers,
 * each with an "Editar" jump back to its step. Closes the create→view loop.
 */
export function CreationSummary() {
  const { form, raceChoices } = useCreationWizard()
  return (
    <form.Subscribe selector={(s: { values: CharacterFormValues }) => s.values}>
      {(values: CharacterFormValues) => (
        <SummaryBody values={values} raceChoices={raceChoices} />
      )}
    </form.Subscribe>
  )
}

function SummaryBody({
  values,
  raceChoices,
}: {
  values: CharacterFormValues
  raceChoices: RaceChoiceState
}) {
  const name = values.name.trim() || 'Novo personagem'
  const hue = hueFromName(name)
  const primary = values.classes[0]
  const deltas = appliedRaceDeltas(values.races, raceChoices)
  const defense = 10 + values.dexterity + (deltas.dexterity ?? 0)
  const { pvMax, pmMax } = deriveDraftVitals(values, raceChoices)
  const flavor = [
    values.races.join(', '),
    values.origin,
    values.god ? `devoto de ${values.god}` : null,
    values.size,
  ]
    .filter(Boolean)
    .join(' · ')

  return (
    <div className="lg:grid lg:h-full lg:grid-cols-[20rem_1fr] lg:gap-4">
      <div className="space-y-3 lg:self-start">
      <Card className="gap-0 overflow-hidden p-0">
        <div
          className="relative flex aspect-[5/3] items-center justify-center"
          style={{
            background: `linear-gradient(155deg, oklch(0.55 0.15 ${hue}) 0%, oklch(0.30 0.09 ${hue}) 70%, oklch(0.22 0.06 ${hue}) 100%)`,
          }}
        >
          <span className="select-none font-display text-[8rem] leading-none text-white/15">
            {initials(name)}
          </span>
          <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/75 via-black/25 to-transparent p-4 pt-10">
            <p className="text-[11px] font-semibold uppercase tracking-widest text-white/70">
              {primary
                ? `${primary.className} · Nv ${primary.level}`
                : 'Sem classe'}
            </p>
            <p className="truncate text-2xl font-semibold text-white drop-shadow">
              {name}
            </p>
            {flavor && (
              <p className="truncate text-sm text-white/80">{flavor}</p>
            )}
          </div>
        </div>
        <div className="grid grid-cols-3 gap-2 p-4">
          <StatCell label="DEF">{defense}</StatCell>
          <StatCell label="PV">
            {Math.min(values.hpCurrent, pvMax)}/{pvMax}
          </StatCell>
          <StatCell label="PM" dim={pmMax === 0}>
            {Math.min(values.mpCurrent, pmMax)}/{pmMax}
          </StatCell>
        </div>
      </Card>

      {anyRacePending(values.races, raceChoices) && (
        <p className="text-[11px] text-[color:var(--hp-hurt)]">
          Há escolhas de atributo de raça pendentes — volte à etapa Raça para
          distribuí-las, ou crie assim mesmo (os +1 faltantes não se aplicam).
        </p>
      )}
      </div>

      <div className="mt-4 space-y-3 lg:mt-0 lg:max-h-full lg:overflow-y-auto lg:pr-1">
      <SummaryRow slug="raca" label="Raça">
        {values.races.length === 0 ? (
          <Empty />
        ) : (
          values.races.map((r) => (
            <div key={r} className="space-y-1">
              <p className="text-sm font-medium">{r}</p>
              <DeltaBadges deltas={resolveRaceDeltas(r, raceChoices[r])} />
              <RaceAbilities name={r} />
            </div>
          ))
        )}
      </SummaryRow>

      <SummaryRow slug="classe" label="Classe">
        {values.classes.length === 0 ? (
          <Empty />
        ) : (
          values.classes.map((c) => (
            <div key={c.className} className="space-y-1">
              <p className="text-sm font-medium">
                {c.className} · Nv {c.level}
              </p>
              <ClassGrantPanel className={c.className} level={c.level} />
            </div>
          ))
        )}
      </SummaryRow>

      <SummaryRow slug="poderes" label="Poderes & Caminho">
        {values.classes.map((c) => {
          const blob = values.classChoices[c.className]
          if (!blob?.caminho && !blob?.devoto) return null
          return (
            <p key={c.className} className="text-sm">
              {c.className}:{blob.caminho ? ` caminho ${blob.caminho}` : ''}
              {blob.devoto ? ` · devoto de ${blob.devoto}` : ''}
            </p>
          )
        })}
        <p className="text-sm text-muted-foreground">
          {values.classPowers.length} poder(es) escolhido(s)
        </p>
      </SummaryRow>

      <SummaryRow slug="origem" label="Origem">
        {values.origin ? (
          <>
            <p className="mb-1 text-xs text-muted-foreground">
              {values.originChoices.length}/2 benefícios escolhidos
            </p>
            <OriginGrantPanel originId={values.origin} collapsible />
          </>
        ) : (
          <Empty />
        )}
      </SummaryRow>

      <SummaryRow slug="atributos" label="Atributos">
        <div className="grid grid-cols-3 gap-2 sm:grid-cols-6">
          {ATTRIBUTE_KEYS.map((k) => {
            const d = deltas[k] ?? 0
            return (
              <StatCell key={k} label={ATTRIBUTE_ABBR[k]}>
                {values[k] + d}
              </StatCell>
            )
          })}
        </div>
        <p className="mt-1 text-[11px] text-muted-foreground">
          base (preset) + bônus de raça = final.
        </p>
      </SummaryRow>

      <SummaryRow slug="pericias" label="Perícias">
        {values.trainedExpertises.length === 0 ? (
          <Empty />
        ) : (
          <div className="flex flex-wrap gap-1.5">
            {values.trainedExpertises.map((name) => (
              <span
                key={name}
                className="rounded-md border border-border px-2 py-0.5 text-xs"
              >
                {name}
              </span>
            ))}
          </div>
        )}
      </SummaryRow>

      <SummaryRow slug="vitalidade" label="Vitalidade">
        <p className="text-sm">
          PV {values.hpCurrent}/{values.hpMax} · PM {values.mpCurrent}/
          {values.mpMax}
        </p>
      </SummaryRow>

      <SummaryRow slug="identidade" label="Identidade">
        <p className="text-sm">
          Tamanho {values.size} · Deslocamento {values.displacement}m
          {values.god ? ` · Deus: ${values.god}` : ''}
        </p>
      </SummaryRow>
      </div>
    </div>
  )
}

function RaceAbilities({ name }: { name: string }) {
  const grant = raceGrant(name)
  if (!grant) return null
  return (
    <AbilityDisclosure
      label="habilidades"
      singular="habilidade"
      lines={grant.abilities}
    />
  )
}

function SummaryRow({
  slug,
  label,
  children,
}: {
  slug: StepSlug
  label: string
  children: ReactNode
}) {
  return (
    <div className="space-y-2 rounded-lg border border-border p-3">
      <div className="flex items-center justify-between">
        <p className="text-xs font-semibold uppercase tracking-widest text-muted-foreground">
          {label}
        </p>
        <Link
          to={`/characters/new/${slug}` as string}
          className="text-xs text-muted-foreground underline-offset-2 hover:text-foreground hover:underline"
        >
          Editar ›
        </Link>
      </div>
      {children}
    </div>
  )
}

function Empty() {
  return <p className="text-xs italic text-muted-foreground">Não definido.</p>
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
