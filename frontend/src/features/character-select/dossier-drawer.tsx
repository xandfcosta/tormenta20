import { ATTRIBUTE_ABBR, ATTRIBUTE_KEYS } from '@/shared/api/attribute-keys'
import { X } from 'lucide-solid'
import { For, type JSX, Show } from 'solid-js'
import type { Character, ComputedSheetV2 } from '@/shared/api/api'
import { cn } from '@/shared/lib/utils'
import { FramedPanel } from '@/shared/ui/framed-panel'
import type { AbilityBlurb } from './select-helpers'
import { FieldLabel } from '@/shared/ui/section-label'

export type DossierDrawerProps = {
  character: Character
  /** Server-computed sheet; null while in flight. */
  sheet: ComputedSheetV2 | null
  abilities: AbilityBlurb[]
  open: boolean
  onClose: () => void
}

/**
 * The "read" surface of the selector, so the stage stays a pure "choose"
 * moment. Slides over the right edge and PERSISTS across selection changes
 * (arrow through characters comparing dossiers). Sections: Identidade,
 * Combate, Atributos, Habilidades.
 *
 * Computed numbers come in as props (from GET /characters/:id/sheet) rather
 * than from a hook inside — same Go engine, no WASM in this scene.
 */
export function DossierDrawer(props: DossierDrawerProps) {
  return (
    <Show when={props.open}>
      <aside
        aria-label={`Dossiê de ${props.character.name}`}
        // Negative insets cancel the SceneShell dense content padding (p-4) so
        // the drawer sits flush against the scene edges instead of floating.
        // Keep in sync if that padding changes.
        class="absolute -inset-y-4 -right-4 z-10 w-full max-w-sm overflow-y-auto border-l border-border bg-card/95 p-4 backdrop-blur duration-200 ease-out animate-in fade-in-0 slide-in-from-right-6 motion-reduce:animate-none sm:rounded-l-md"
      >
        <div class="mb-3 flex items-center justify-between">
          <h3 class="font-display text-lg tracking-wide">{props.character.name}</h3>
          <button
            type="button"
            aria-label="Fechar dossiê"
            onClick={() => props.onClose()}
            class="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
          >
            <X class="size-4" />
          </button>
        </div>

        <DossierSection title="Identidade">
          <p class="text-sm">
            {props.character.races.map((r) => r.race).join(', ')} · {props.character.origin}
          </p>
          <p class="text-xs text-muted-foreground">
            {props.character.god
              ? `Devoto de ${props.character.god}${props.character.godPower ? ` (${props.character.godPower})` : ''}`
              : 'Sem devoção'}{' '}
            · {props.character.size} · Nível {props.character.level}
          </p>
        </DossierSection>

        <DossierSection title="Combate">
          <div class="flex gap-4 font-mono text-sm">
            <span>DEF {props.sheet ? props.sheet.defense.total : '—'}</span>
            <span class="text-[color:var(--hp-full)]">
              PV {props.character.hpCurrent}/{props.character.hpMax}
            </span>
            <span class={cn(props.character.mpMax === 0 && 'opacity-50')}>
              PM {props.character.mpCurrent}/{props.character.mpMax}
            </span>
          </div>
        </DossierSection>

        <DossierSection title="Atributos">
          <div class="grid grid-cols-6 gap-1 text-center">
            <For each={ATTRIBUTE_KEYS}>
              {(key) => (
                <div>
                  <FieldLabel as="p" class="text-4xs font-semibold">
                    {ATTRIBUTE_ABBR[key]}
                  </FieldLabel>
                  <p class="font-mono text-sm">
                    {props.sheet ? props.sheet.attributes[key].total : '—'}
                  </p>
                </div>
              )}
            </For>
          </div>
        </DossierSection>

        <DossierSection title="Habilidades">
          <FramedPanel variant="parchment" class="p-3">
            <AbilityList abilities={props.abilities} />
          </FramedPanel>
        </DossierSection>
      </aside>
    </Show>
  )
}

function DossierSection(props: { title: string; children: JSX.Element }) {
  return (
    <section class="mb-4 space-y-1.5">
      <FieldLabel as="p" tom="gold" class="font-semibold">
        {props.title}
      </FieldLabel>
      {props.children}
    </section>
  )
}

// Rendered on the parchment FramedPanel — colors inherit the dark ink; the
// description is dimmed via opacity rather than a light muted token (which
// would vanish on the light surface).
function AbilityList(props: { abilities: AbilityBlurb[] }) {
  return (
    <Show
      when={props.abilities.length > 0}
      fallback={<p class="text-xs italic opacity-70">Nenhuma.</p>}
    >
      <ul class="space-y-1.5">
        <For each={props.abilities}>
          {(ability) => (
            <li>
              <p class="text-xs font-semibold">{ability.name}</p>
              <p class="text-2xs leading-snug opacity-80">{ability.description}</p>
            </li>
          )}
        </For>
      </ul>
    </Show>
  )
}
