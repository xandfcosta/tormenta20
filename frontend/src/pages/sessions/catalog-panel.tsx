import { BookMarked } from 'lucide-solid'
import { createSignal } from 'solid-js'
import { CatalogBrowser } from '@/features/gm-tools/catalog-browser'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { Button } from '@/shared/ui/button'
import { SidePanel } from '@/shared/ui/side-panel'
import { MatchPeek } from './match-rail'

/**
 * Rules lookup without leaving the match. This is the panel the whole non-modal
 * decision was made for: on a laptop the tracker stays live behind it, so the
 * GM reads "Abalado: −2 em testes de perícia" on the right and applies it to
 * the goblin on the left without closing anything (ALE-75).
 */
export function CatalogPanel(props: { rt: SessionRealtime }) {
  const [open, setOpen] = createSignal(false)

  return (
    <>
      <Button
        type="button"
        size="sm"
        variant="secondary"
        class="w-full gap-1.5"
        onClick={() => setOpen(true)}
      >
        <BookMarked aria-hidden="true" class="size-4" /> Catálogos
      </Button>

      <SidePanel
        open={open()}
        onOpenChange={setOpen}
        title="Catálogos"
        description="Condições, magias, poderes e itens."
        header={<MatchPeek rt={props.rt} />}
      >
        {/* Bounded: inside the panel the list shares height with the header. */}
        <CatalogBrowser listClass="max-h-[60vh] min-h-0 flex-1 pr-1" />
      </SidePanel>
    </>
  )
}
