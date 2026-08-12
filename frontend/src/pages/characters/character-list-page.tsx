import { useQuery } from '@tanstack/solid-query'
import { Link, useNavigate } from '@tanstack/solid-router'
import { Search } from 'lucide-solid'
import { Show, createMemo, createSignal } from 'solid-js'
import { raceDefsCatalogQueryOptions } from '@/entities/catalog/queries'
import { charactersQueryOptions, characterSheetQueryOptions } from '@/entities/character/queries'
import { CharacterFilmstrip } from '@/features/character-select/character-filmstrip'
import { CharacterStage } from '@/features/character-select/character-stage'
import { CreateSlotStage } from '@/features/character-select/create-slot-stage'
import { DossierDrawer } from '@/features/character-select/dossier-drawer'
import { QuestionFrame } from '@/features/character-select/question-frame'
import {
  isCreateSlot,
  raceAbilityBlurbs,
  stepRosterIndex,
} from '@/features/character-select/select-helpers'
import type { Character } from '@/shared/api/api'
import { SceneShell } from '@/shared/layout/scene-shell'
import { matchesQuery } from '@/shared/lib/fuzzy-filter'
import { createSceneNav } from '@/shared/lib/scene-nav'
import { settledQuery } from '@/shared/lib/settled-query'
import { createSfx } from '@/shared/lib/sfx'
import { useUi } from '@/shared/stores/ui-context'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { SkeletonCardGrid } from '@/shared/ui/skeleton'

/** Fields the roster search looks at: name, primary class, origin and races. */
function searchFields(character: Character): string[] {
  return [
    character.name,
    character.classes[0]?.className ?? '',
    character.origin,
    character.races.map((r) => r.race).join(' '),
  ]
}

/**
 * Characters select screen — design "palco + dossiê": the selected character
 * on a spotlit center stage with prev/next peeking from the sides, a filmstrip
 * index for O(1) jumps across long rosters, and a dossier drawer for the
 * readable detail. Search filters the roster; the stage snaps to the first
 * match. `←/→` navigate, Enter opens the sheet, D toggles the dossier, `/`
 * focuses search.
 */
export function CharactersListPage() {
  const navigate = useNavigate()
  const ui = useUi()
  const sfx = createSfx(ui)
  const characters = useQuery(() => charactersQueryOptions)
  const raceDefs = useQuery(() => raceDefsCatalogQueryOptions)

  // `'novo'` is the trailing create slot, a real cursor position rather than a
  // link the arrows skip over (ALE-98).
  const [selectedId, setSelectedId] = createSignal<number | 'novo' | null>(null)
  const [dossierOpen, setDossierOpen] = createSignal(false)
  const [direction, setDirection] = createSignal<1 | -1>(1)
  const [query, setQuery] = createSignal('')

  const roster = () => characters.data ?? []
  const filtered = createMemo(() =>
    roster().filter((character) => matchesQuery(searchFields(character), query())),
  )
  // Selection is by id, but the cursor is an index into the FILTERED list — so
  // typing a query slides the stage onto the first match instead of stranding
  // it on a character that's no longer shown.
  const index = createMemo(() => {
    if (selectedId() === 'novo') return filtered().length
    const found = filtered().findIndex((c) => c.id === selectedId())
    return Math.max(0, found)
  })
  const atCreateSlot = () => isCreateSlot(index(), filtered().length)
  const selected = () => filtered()[index()] ?? null
  const prev = () => (index() > 0 ? filtered()[index() - 1] : null)
  const next = () => (index() < filtered().length - 1 ? filtered()[index() + 1] : null)

  // The computed sheet (defense, attribute totals) comes from the API — the
  // same Go engine the ficha will run as WASM (ALE-73). Keyed by character, so
  // arrowing through the roster reuses each one's cached sheet.
  const sheet = useQuery(() => ({
    ...characterSheetQueryOptions(selected()?.id ?? 0),
    enabled: selected() !== null,
  }))
  // Never touch `sheet.data` while it's pending: that suspends, and the nearest
  // boundary is the router's per-match Suspense, so the WHOLE scene gets
  // detached + re-inserted and every enter animation replays (ALE-95).
  const computed = () => settledQuery(sheet)

  // Same guard as the sheet above, and for the same reason (ALE-95): the race
  // catalog is still in flight on the first visit, and reading `.data` there
  // suspends the route match — which re-inserts the scene and replays every
  // animation. It showed up as a 1-in-4 flake in the E2E, not as a steady bug.
  const abilities = createMemo(() => {
    const character = selected()
    if (!character) return []
    return raceAbilityBlurbs(settledQuery(raceDefs) ?? [], character, 8)
  })

  const step = (delta: number) => {
    const list = filtered()
    if (list.length === 0) return
    sfx('hover')
    const nextIndex = stepRosterIndex(index(), delta, list.length)
    setDirection(delta >= 0 ? 1 : -1)
    setSelectedId(isCreateSlot(nextIndex, list.length) ? 'novo' : list[nextIndex].id)
  }

  const jumpTo = (id: number | 'novo') => {
    const target = id === 'novo' ? filtered().length : filtered().findIndex((c) => c.id === id)
    if (target === -1) return
    sfx('select')
    setDirection(target >= index() ? 1 : -1)
    setSelectedId(id)
  }

  const openForge = () => {
    sfx('select')
    navigate({ to: '/characters/new' })
  }

  // Enter means "activate what the cursor is on" — a hero opens its sheet, the
  // trailing slot opens the Forge. Same key, one grammar (ALE-98).
  const openSheet = () => {
    if (atCreateSlot()) return openForge()
    const character = selected()
    if (!character) return
    sfx('select')
    navigate({ to: '/characters/$id', params: { id: String(character.id) } })
  }

  const leaveScene = () => {
    sfx('back')
    navigate({ to: '/' })
  }

  // The roster is a selection scene (the spotlight stage): a `delegated`
  // scene-nav so it shares the grammar + gamepad seam while keeping its own
  // cursor. onCommand maps the standard grammar; onKey holds the keyboard-only
  // accelerators (D dossier, / search) and search Esc-to-clear.
  createSceneNav({
    root: () => document.querySelector<HTMLElement>('[data-slot="scene-shell"]'),
    delegated: true,
    sfx,
    onEscape: leaveScene,
    onCommand: (cmd) => {
      switch (cmd.type) {
        case 'move':
          if (cmd.dir === 'left') {
            step(-1)
            return true
          }
          if (cmd.dir === 'right') {
            step(1)
            return true
          }
          return false // ↑/↓ have no meaning on the horizontal stage
        case 'edge': {
          const list = filtered()
          if (list.length > 0) jumpTo(list[cmd.to === 'first' ? 0 : list.length - 1].id)
          return true
        }
        case 'bumper':
          step(cmd.dir === 'next' ? 5 : -5)
          return true
        case 'activate':
          openSheet()
          return true
        case 'back':
          if (dossierOpen()) {
            setDossierOpen(false)
            return true
          }
          if (query().length > 0) {
            setQuery('')
            return true
          }
          return false // nothing to close → onEscape leaves the scene
      }
    },
    onKey: (e) => {
      const el = e.target as HTMLElement
      if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA') {
        if (e.key !== 'Escape') return false // let the field handle its keys
        setQuery('')
        el.blur()
        return true
      }
      if (e.key === 'd' || e.key === 'D') {
        setDossierOpen((open) => !open)
        return true
      }
      if (e.key === '/') {
        e.preventDefault()
        document.querySelector<HTMLInputElement>('[data-roster-search]')?.focus()
        return true
      }
      return false
    },
  })

  const headerControls = () => (
    <Show when={roster().length > 0}>
      <div class="relative min-w-40 flex-1 sm:max-w-xs">
        <Search class="pointer-events-none absolute left-2 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          type="search"
          value={query()}
          onInput={(e) => setQuery(e.currentTarget.value)}
          placeholder="Buscar nome, classe, origem, raça"
          class="pl-8"
          aria-label="Buscar personagem"
          data-roster-search
        />
      </div>
      <Badge variant="secondary" class="shrink-0">
        {filtered().length} de {roster().length}
      </Badge>
      <Link to="/characters/new">
        <Button size="sm">+ Novo</Button>
      </Link>
    </Show>
  )

  return (
    <SceneShell
      dense
      title="Personagens"
      onBack={() => {
        sfx('select')
        navigate({ to: '/' })
      }}
      onEnter={() => sfx('transition')}
      headerRight={headerControls()}
      class="gap-2"
    >
      <Show when={characters.isLoading}>
        <SkeletonCardGrid count={3} />
      </Show>
      <Show when={characters.isError}>
        <p class="text-destructive">{(characters.error as Error | null)?.message}</p>
      </Show>
      <Show when={characters.isSuccess && roster().length === 0}>
        <EmptyStage />
      </Show>
      <Show when={roster().length > 0 && filtered().length === 0}>
        <NoMatches query={query()} onClear={() => setQuery('')} />
      </Show>

      <Show when={filtered().length > 0}>
        <div class="relative flex min-h-0 flex-1 flex-col">
          <Show
            when={!atCreateSlot() && selected()}
            fallback={<CreateSlotStage prev={prev()} onStep={step} onOpen={openForge} />}
          >
            {(character) => (
              <>
                <CharacterStage
                  selected={character()}
                  prev={prev()}
                  next={next()}
                  direction={direction()}
                  defense={computed()?.defense.total ?? null}
                  onStep={step}
                  onOpen={openSheet}
                  onDossier={() => setDossierOpen((open) => !open)}
                  dossierOpen={dossierOpen()}
                />
                <DossierDrawer
                  character={character()}
                  sheet={computed()}
                  abilities={abilities()}
                  open={dossierOpen()}
                  onClose={() => setDossierOpen(false)}
                />
              </>
            )}
          </Show>
          <CharacterFilmstrip
            roster={filtered()}
            selectedId={atCreateSlot() ? 'novo' : (selected()?.id ?? 0)}
            onSelect={jumpTo}
            onHover={() => sfx('hover')}
          />
          {/* Keyboard hints only where there's a keyboard: laptop+desktop
              (≥xl). Hidden on tablet/phone — the keys don't apply there.
              "abrir" and not "abrir ficha": on the create slot Enter opens the
              Forge, and the hint has to be true in both positions. */}
          <p class="hidden pt-1 text-center text-[11px] text-muted-foreground xl:block">
            ← → navegar · Enter abrir · D dossiê · / buscar
          </p>
        </div>
      </Show>
    </SceneShell>
  )
}

/** Empty roster: the stage itself invites the first character. */
function EmptyStage() {
  return (
    <div class="flex min-h-0 flex-1 flex-col items-center justify-center gap-4">
      <div class="aspect-[3/4] w-48 rounded-md border-2 border-dashed border-grimorio-iron">
        <QuestionFrame />
      </div>
      <p class="font-heading text-xl uppercase tracking-[0.12em] text-foreground">
        Seu grupo aguarda um herói
      </p>
      <Link to="/characters/new">
        <Button size="lg">Criar seu primeiro personagem</Button>
      </Link>
    </div>
  )
}

function NoMatches(props: { query: string; onClear: () => void }) {
  return (
    <div class="flex min-h-0 flex-1 flex-col items-center justify-center gap-3">
      <p class="font-heading text-sm uppercase tracking-widest text-grimorio-gold/70">
        Nada encontrado
      </p>
      <p class="text-sm text-muted-foreground">Nenhum personagem para “{props.query}”.</p>
      <Button variant="outline" onClick={() => props.onClear()}>
        Limpar busca<span class="hidden xl:inline"> (Esc)</span>
      </Button>
    </div>
  )
}
