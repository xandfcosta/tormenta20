import { X } from 'lucide-react'
import {
  ATTRIBUTE_ABBR,
  ATTRIBUTE_KEYS,
} from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'
import { cn } from '@/shared/lib/utils'
import { useComputedSheet } from '@/entities/character/computed-sheet'
import { FramedPanel } from '@/shared/ui/framed-panel'
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
  const sheet = useComputedSheet(character)
  const defense = sheet.defense.total
  if (!open) return null
  return (
    <aside
      aria-label={`Dossiê de ${character.name}`}
      // Negative insets cancel the SceneShell dense content padding (p-4) so the
      // drawer sits flush against the scene edges instead of floating with a
      // gap. Keep in sync if that padding changes.
      className="absolute -inset-y-4 -right-4 z-10 w-full max-w-sm overflow-y-auto border-l border-border bg-card/95 p-4 backdrop-blur duration-200 ease-out animate-in fade-in-0 slide-in-from-right-6 motion-reduce:animate-none sm:rounded-l-lg"
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
                {sheet.attributes[k].total}
              </p>
            </div>
          ))}
        </div>
      </DossierSection>

      <DossierSection title="Habilidades">
        <FramedPanel variant="parchment" className="p-3">
          <AbilityList abilities={raceAbilityBlurbs(character, 8)} />
        </FramedPanel>
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
      <p className="text-[10px] font-semibold uppercase tracking-widest text-grimorio-gold">
        {title}
      </p>
      {children}
    </section>
  )
}

// Rendered on the parchment FramedPanel — colors inherit the dark ink; the
// description is dimmed via opacity rather than a light muted token (which
// would vanish on the light surface).
function AbilityList({ abilities }: { abilities: AbilityBlurb[] }) {
  if (abilities.length === 0) {
    return <p className="text-xs italic opacity-70">Nenhuma.</p>
  }
  return (
    <ul className="space-y-1.5">
      {abilities.map((a) => (
        <li key={a.name}>
          <p className="text-xs font-semibold">{a.name}</p>
          <p className="text-[11px] leading-snug opacity-80">{a.description}</p>
        </li>
      ))}
    </ul>
  )
}
