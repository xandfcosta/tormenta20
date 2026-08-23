import { useQuery } from '@tanstack/solid-query'
import { Link, useNavigate } from '@tanstack/solid-router'
import { Plus, Search, UserPlus } from 'lucide-solid'
import { For, Show, createMemo, createSignal } from 'solid-js'
import { campaignsQueryOptions } from '@/entities/campaign/queries'
import { createActiveSessionByCampaign } from '@/features/campaign-select/active-sessions'
import { CampaignBook } from '@/features/campaign-select/campaign-book'
import { CampaignRail } from '@/features/campaign-select/campaign-rail'
import type { Campaign } from '@/shared/api/api'
import { SceneShell } from '@/shared/layout/scene-shell'
import { matchesQuery } from '@/shared/lib/fuzzy-filter'
import { createSceneNav } from '@/shared/lib/scene-nav'
import { createSfx } from '@/shared/lib/sfx'
import { cn } from '@/shared/lib/utils'
import { useUi } from '@/shared/stores/ui-context'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { SkeletonCardGrid } from '@/shared/ui/skeleton'

type RoleFilter = 'all' | 'gm' | 'player'

const ROLE_FILTERS: { value: RoleFilter; label: string }[] = [
  { value: 'all', label: 'Todas' },
  { value: 'gm', label: 'Mestrando' },
  { value: 'player', label: 'Jogando' },
]

/**
 * Campanhas — the campaigns roster as a cinematic "chapter select" scene: the
 * focused chronicle on a stage (emblem, synopsis, your PC, live status, primary
 * action) with a rail to switch focus, mirroring the character selector.
 * `←/→` (or `↑/↓`) move focus, Enter opens (resumes a live session or opens the
 * chronicle), O opens the chronicle, `/` focuses search, Esc clears.
 */
export function CampaignsListPage() {
  const navigate = useNavigate()
  const ui = useUi()
  const sfx = createSfx(ui)
  const campaigns = useQuery(() => campaignsQueryOptions)

  const [selectedId, setSelectedId] = createSignal<number | null>(null)
  const [query, setQuery] = createSignal('')
  const [role, setRole] = createSignal<RoleFilter>('all')

  const roster = () => campaigns.data ?? []
  const hasData = () => roster().length > 0

  const filtered = createMemo(() =>
    roster().filter((campaign) => {
      const matchesRole = role() === 'all' || (campaign.role ?? 'player') === role()
      return matchesRole && matchesQuery([campaign.name, campaign.description ?? ''], query())
    }),
  )
  const index = createMemo(() => Math.max(0, filtered().findIndex((c) => c.id === selectedId())))
  const selected = () => filtered()[index()] ?? null
  // Gives each page turn its direction: a later marker turns forward.
  const orderIds = createMemo(() => filtered().map((c) => c.id))

  const activeByCampaign = createActiveSessionByCampaign(() => roster().map((c) => c.id))

  const step = (delta: number) => {
    const list = filtered()
    if (list.length === 0) return
    sfx('hover')
    const nextIndex = Math.min(list.length - 1, Math.max(0, index() + delta))
    setSelectedId(list[nextIndex].id)
  }
  const jumpTo = (id: number) => {
    sfx('select')
    setSelectedId(id)
  }
  const openDetail = (campaign: Campaign) => {
    sfx('select')
    navigate({ to: '/campaigns/$id', params: { id: String(campaign.id) } })
  }
  const resume = (campaign: Campaign) => {
    const sid = activeByCampaign()[campaign.id]
    if (sid == null) return openDetail(campaign)
    sfx('select')
    navigate({
      to: '/campaigns/$id/sessions/$sid',
      params: { id: String(campaign.id), sid: String(sid) },
    })
  }
  const openFocused = () => {
    const campaign = selected()
    if (!campaign) return
    if (activeByCampaign()[campaign.id] != null) resume(campaign)
    else openDetail(campaign)
  }

  // A selection scene (the 1-D book rail): a `delegated` scene-nav so it shares
  // the grammar + gamepad seam while keeping its own cursor. onKey holds the
  // keyboard-only accelerators (/ search, O open).
  createSceneNav({
    root: () => document.querySelector<HTMLElement>('[data-slot="scene-shell"]'),
    delegated: true,
    sfx,
    onEscape: () => {
      sfx('back')
      navigate({ to: '/' })
    },
    onCommand: (cmd) => {
      switch (cmd.type) {
        case 'move':
          // A 1-D rail: both axes step (←/↑ back, →/↓ forward).
          step(cmd.dir === 'right' || cmd.dir === 'down' ? 1 : -1)
          return true
        case 'edge': {
          const list = filtered()
          if (list.length > 0) jumpTo(list[cmd.to === 'first' ? 0 : list.length - 1].id)
          return true
        }
        case 'bumper':
          step(cmd.dir === 'next' ? 5 : -5)
          return true
        case 'activate':
          openFocused()
          return true
        case 'back':
          if (query().length > 0) {
            setQuery('')
            return true
          }
          return false // nothing to clear → onEscape leaves the scene
      }
    },
    onKey: (e) => {
      const el = e.target as HTMLElement
      if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA') {
        if (e.key !== 'Escape') return false
        setQuery('')
        el.blur()
        return true
      }
      if (e.key === 'o' || e.key === 'O') {
        const campaign = selected()
        if (campaign) openDetail(campaign)
        return true
      }
      if (e.key === '/') {
        e.preventDefault()
        document.querySelector<HTMLInputElement>('[data-campaign-search]')?.focus()
        return true
      }
      return false
    },
  })

  const headerControls = () => (
    <div class="flex w-full flex-wrap items-center gap-2 sm:w-auto sm:flex-1 sm:justify-end">
      <Show when={hasData()}>
        <div class="relative w-full sm:w-56 md:w-64">
          <Search class="pointer-events-none absolute left-2 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="search"
            value={query()}
            onInput={(e) => setQuery(e.currentTarget.value)}
            placeholder="Buscar campanha"
            class="pl-8"
            aria-label="Buscar campanha"
            data-campaign-search
          />
        </div>
      </Show>
      <div class="flex flex-1 flex-wrap items-center gap-2 sm:flex-none">
        <Show when={hasData()}>
          <RoleFilterButtons role={role()} onRole={setRole} />
        </Show>
        <div class="ml-auto flex gap-2 sm:ml-0">
          <Link to="/campaigns/join">
            <Button variant="outline" size="sm" aria-label="Entrar em campanha">
              <UserPlus class="size-4 sm:mr-1" />
              <span class="hidden sm:inline">Entrar</span>
            </Button>
          </Link>
          <Link to="/campaigns/new">
            <Button size="sm" aria-label="Nova campanha">
              <Plus class="size-4 sm:mr-1" />
              <span class="hidden sm:inline">Nova</span>
            </Button>
          </Link>
        </div>
      </div>
    </div>
  )

  return (
    <SceneShell
      dense
      title="Campanhas"
      onBack={() => {
        sfx('select')
        navigate({ to: '/' })
      }}
      onEnter={() => sfx('transition')}
      headerRight={headerControls()}
    >
      <Show when={campaigns.isLoading}>
        <SkeletonCardGrid count={3} />
      </Show>
      <Show when={campaigns.isError}>
        <p class="text-destructive">{(campaigns.error as Error | null)?.message}</p>
      </Show>
      <Show when={campaigns.isSuccess && !hasData()}>
        <NoCampaigns />
      </Show>
      <Show when={hasData() && filtered().length === 0}>
        <NoMatches onClear={() => setQuery('')} />
      </Show>

      <Show when={selected()}>
        {(campaign) => (
          <div class="flex min-h-0 flex-1 items-center justify-center">
            <div class="flex min-h-0 w-full max-w-7xl flex-col gap-2 lg:flex-row lg:items-stretch lg:gap-0">
              {/* Mirrors the bookmarks' width so the book itself centers on the
                  viewport (not the book+tabs group). Hidden on phones. */}
              <div aria-hidden="true" class="hidden lg:block lg:w-56 lg:shrink-0" />
              <CampaignBook
                campaign={campaign()}
                isLive={activeByCampaign()[campaign().id] != null}
                orderIds={orderIds()}
                onOpen={() => openDetail(campaign())}
                onResume={() => resume(campaign())}
              />
              <CampaignRail
                campaigns={filtered()}
                selectedId={campaign().id}
                activeByCampaign={activeByCampaign()}
                onSelect={jumpTo}
                onHover={() => sfx('hover')}
                class="lg:-ml-px lg:w-56 lg:shrink-0 lg:self-stretch"
              />
            </div>
          </div>
        )}
      </Show>
    </SceneShell>
  )
}

/** Role segmented filter (Todas / Mestrando / Jogando). */
function RoleFilterButtons(props: { role: RoleFilter; onRole: (role: RoleFilter) => void }) {
  return (
    <div class="flex gap-1">
      <For each={ROLE_FILTERS}>
        {(filter) => (
          <Button
            type="button"
            size="sm"
            variant={props.role === filter.value ? 'default' : 'outline'}
            aria-pressed={props.role === filter.value}
            onClick={() => props.onRole(filter.value)}
            class={cn(props.role === filter.value && 'pointer-events-none')}
          >
            {filter.label}
          </Button>
        )}
      </For>
    </div>
  )
}

/** Empty roster: theatrical CTA matching the grimório scene. */
function NoCampaigns() {
  return (
    <div class="flex min-h-0 flex-1 flex-col items-center justify-center gap-4 py-10">
      <div class="flex size-24 items-center justify-center rounded-sm border-2 border-dashed border-grimorio-iron">
        <span class="select-none font-heading text-5xl text-grimorio-gold/30">✦</span>
      </div>
      <p class="font-heading text-xl uppercase tracking-[0.12em] text-foreground">
        Nenhuma campanha ainda
      </p>
      <Link to="/campaigns/new">
        <Button size="lg">Criar sua primeira campanha</Button>
      </Link>
    </div>
  )
}

function NoMatches(props: { onClear: () => void }) {
  return (
    <div class="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 py-10">
      <p class="font-heading text-sm uppercase tracking-widest text-grimorio-gold/70">
        Nada encontrado
      </p>
      <p class="text-sm text-muted-foreground">Nenhuma campanha corresponde ao filtro.</p>
      <Button variant="outline" onClick={() => props.onClear()}>
        Limpar busca<span class="hidden xl:inline"> (Esc)</span>
      </Button>
    </div>
  )
}
