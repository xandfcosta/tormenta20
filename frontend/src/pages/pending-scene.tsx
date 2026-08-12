import { useNavigate } from '@tanstack/solid-router'
import { SceneShell } from '@/shared/layout/scene-shell'

/**
 * Scaffolding for a scene that hasn't been ported yet (ALE-63). It exists so
 * the Hub's menu points at real routes — a menu entry that can't navigate is
 * worse than one that says "not here yet".
 *
 * Deliberately says so on screen: nobody should mistake this for a finished
 * scene, and each of these is deleted by the issue that ports its scene.
 *
 * @example <PendingScene title="Personagens" issue="ALE-70" />
 */
export function PendingScene(props: { title: string; issue: string }) {
  const navigate = useNavigate()
  return (
    <SceneShell dense title={props.title} onBack={() => navigate({ to: '/' })} backLabel="Hub">
      <div class="flex flex-1 flex-col items-center justify-center gap-2 text-center">
        <p class="font-heading text-lg tracking-wide text-muted-foreground">
          Cena ainda não portada para SolidJS.
        </p>
        <p class="text-sm text-muted-foreground">Chega em {props.issue}.</p>
      </div>
    </SceneShell>
  )
}
