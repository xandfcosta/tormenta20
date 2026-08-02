import { X } from 'lucide-react'
import { useMemo } from 'react'
import {
  ATTRIBUTE_ABBR,
  ATTRIBUTE_KEYS,
} from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'
import { cn } from '@/shared/lib/utils'
import {
  attributeTotal,
  characterEffects,
  defenseTotal,
} from '@/entities/character/derived'
import { type AbilityBlurb, raceAbilityBlurbs } from './select-helpers'

/**
 * Dossier drawer — the "read" surface of the selector, so the stage stays a
 * pure "choose" moment. Slides over the right edge and PERSISTS across
 * selection changes (arrow through characters comparing dossiers). Sections:
 * Identidade, Combate, Atributos, Habilidades.
 */
export function DossierDrawer({
  character,
  open,
  onClose,
}: {
  character: Character
  open: boolean
  onClose: () => void
}) {
  const effects = useMemo(() => characterEffects(character), [character])
  const defense = useMemo(
    () => defenseTotal(character, effects).total,
    [character, effects],
  )
  if (!open) return null
  return (
    <aside
      aria-label={`Dossiê de ${character.name}`}
      className="absolute inset-y-0 right-0 z-10 w-full max-w-sm overflow-y-auto border-l border-border bg-card/95 p-4 backdrop-blur sm:rounded-l-lg"
    >
      <div className="mb-3 flex items-center justify-between">
        <h3 className="font-display text-lg tracking-wide">{character.name}</h3>
        <button
          type="button"
          aria-label="Fechar dossiê"
          onClick={onClose}
          className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
        >
          <X className="size-4" />
        </button>
      </div>

      <DossierSection title="Identidade">
        <p className="text-sm">
          {character.races.map((r) => r.race).join(', ')} · {character.origin}
        </p>
        <p className="text-xs text-muted-foreground">
          {character.god
            ? `Devoto de ${character.god}${character.godPower ? ` (${character.godPower})` : ''}`
            : 'Sem devoção'}{' '}
          · {character.size} · Nível {character.level}
        </p>
      </DossierSection>

      <DossierSection title="Combate">
        <div className="flex gap-4 font-mono text-sm">
          <span>DEF {defense}</span>
          <span className="text-[color:var(--hp-full)]">
            PV {character.hpCurrent}/{character.hpMax}
          </span>
          <span className={cn(character.mpMax === 0 && 'opacity-50')}>
            PM {character.mpCurrent}/{character.mpMax}
          </span>
        </div>
      </DossierSection>

      <DossierSection title="Atributos">
        <div className="grid grid-cols-6 gap-1 text-center">
          {ATTRIBUTE_KEYS.map((k) => (
            <div key={k}>
              <p className="text-[9px] font-semibold uppercase text-muted-foreground">
                {ATTRIBUTE_ABBR[k]}
              </p>
              <p className="font-mono text-sm">
                {attributeTotal(character, k, effects)}
              </p>
            </div>
          ))}
        </div>
      </DossierSection>

      <DossierSection title="Habilidades">
        <AbilityList abilities={raceAbilityBlurbs(character, 8)} />
      </DossierSection>
    </aside>
  )
}

function DossierSection({
  title,
  children,
}: {
  title: string
  children: React.ReactNode
}) {
  return (
    <section className="mb-4 space-y-1.5">
      <p className="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">
        {title}
      </p>
      {children}
    </section>
  )
}

function AbilityList({ abilities }: { abilities: AbilityBlurb[] }) {
  if (abilities.length === 0) {
    return <p className="text-xs italic text-muted-foreground">Nenhuma.</p>
  }
  return (
    <ul className="space-y-1.5">
      {abilities.map((a) => (
        <li key={a.name}>
          <p className="text-xs font-semibold">{a.name}</p>
          <p className="text-[11px] leading-snug text-muted-foreground">
            {a.description}
          </p>
        </li>
      ))}
    </ul>
  )
}
