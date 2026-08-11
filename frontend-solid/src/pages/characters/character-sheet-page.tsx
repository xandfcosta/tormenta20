import { useQuery } from '@tanstack/solid-query'
import { getRouteApi, useNavigate } from '@tanstack/solid-router'
import { Show } from 'solid-js'
import { characterQueryOptions } from '@/entities/character/queries'
import { CharacterSheet } from '@/features/character-sheet/character-sheet'
import { SHEET_PANELS, resolveSheetTab } from '@/features/character-sheet/sheet-sections'
import { SceneShell } from '@/shared/layout/scene-shell'
import { createSceneNav } from '@/shared/lib/scene-nav'
import { createSfx } from '@/shared/lib/sfx'
import { useUi } from '@/shared/stores/ui-context'
import { Skeleton } from '@/shared/ui/skeleton'

const routeApi = getRouteApi('/characters/$id')

/**
 * The character sheet as a scene. The selected block lives in `?tab=` and
 * nowhere else, so it deep-links, survives the back button and cannot drift out
 * of sync with the screen — see `CharacterSheet` for what the React version had
 * to do instead.
 */
export function CharacterSheetPage() {
  const params = routeApi.useParams()
  const search = routeApi.useSearch()
  const navigate = useNavigate()
  const ui = useUi()
  const sfx = createSfx(ui)

  const characterId = () => Number(params().id)
  const character = useQuery(() => characterQueryOptions(characterId()))

  const tab = () => resolveSheetTab(search().tab ?? '')

  const goToTab = (next: string) => {
    if (next === tab()) return
    sfx('select')
    navigate({ to: '.', search: { tab: next }, replace: true })
  }

  const back = () => {
    sfx('back')
    navigate({ to: '/characters' })
  }

  // PageUp/PageDown bump blocks; Esc leaves to the roster. The content is a
  // dense editing form, so arrows stay native there — only the rail declares a
  // nav region (see the desktop layout).
  const cycleBlock = (delta: number) => {
    const values = SHEET_PANELS.map((p) => p.value)
    const index = Math.max(0, values.indexOf(tab()))
    goToTab(values[(index + delta + values.length) % values.length])
  }

  createSceneNav({
    root: () => document.querySelector<HTMLElement>('[data-sheet-root]'),
    onEscape: back,
    bumpers: { prev: () => cycleBlock(-1), next: () => cycleBlock(1) },
    sfx,
    active: () => !character.isLoading && !!character.data,
  })

  return (
    <SceneShell dense onBack={back} onEnter={() => sfx('open')}>
      <Show
        when={!character.isLoading}
        fallback={
          <div class="w-full space-y-4">
            <Skeleton class="h-8 w-56" />
            <Skeleton class="h-96 w-full" />
          </div>
        }
      >
        <Show
          when={character.data}
          fallback={<p class="text-destructive">{(character.error as Error | null)?.message}</p>}
        >
          {(data) => (
            <div class="h-[calc(100dvh-7rem)] w-full">
              <CharacterSheet character={data()} tab={tab()} onTabChange={goToTab} />
            </div>
          )}
        </Show>
      </Show>
    </SceneShell>
  )
}
