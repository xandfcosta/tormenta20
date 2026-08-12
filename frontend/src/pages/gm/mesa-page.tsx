import { Outlet, useNavigate } from '@tanstack/solid-router'
import { createCurrentTool } from '@/features/gm-tools/current-tool'
import type { ToolSlug } from '@/features/gm-tools/gm-tools'
import { ToolRail } from '@/features/gm-tools/tool-rail'
import { SceneShell } from '@/shared/layout/scene-shell'
import { createSfx } from '@/shared/lib/sfx'
import { useUi } from '@/shared/stores/ui-context'

/**
 * The Mesa do Mestre — one scene holding every GM tool, with the rail choosing
 * which is on stage. The shell never unmounts as the GM switches: walking the
 * rail changes the route, not the scene, so a filtered bestiary or a
 * half-composed encounter survives a look at the catalogue.
 */
export function MesaPage() {
  const navigate = useNavigate()
  const ui = useUi()
  const sfx = createSfx(ui)
  const current = createCurrentTool()

  const pick = (tool: ToolSlug) => {
    sfx('select')
    navigate({ to: '/gm/$tool', params: { tool } })
  }

  return (
    <SceneShell
      dense
      bleed
      title="Mesa do Mestre"
      backLabel="Hub"
      onBack={() => {
        sfx('back')
        navigate({ to: '/' })
      }}
      onEnter={() => sfx('transition')}
    >
      <div class="flex min-h-0 flex-1 flex-col gap-3 px-4 py-3 lg:flex-row lg:gap-4">
        <ToolRail current={current()} onPick={pick} />
        <div class="flex min-h-0 flex-1 flex-col">
          <Outlet />
        </div>
      </div>
    </SceneShell>
  )
}
