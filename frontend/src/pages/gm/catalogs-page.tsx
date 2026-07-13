import { PageChrome } from '@/shared/ui/page-chrome'
import { CatalogBrowser } from '@/features/gm-tools/catalogs/catalog-browser'
import { GmPageHeader } from '@/features/gm-tools/gm-page-header'

/**
 * GM catalog browser — one tabbed page (condições / magias / poderes / itens)
 * over a shared search box for quick rules checks. The browser itself lives in
 * `features/gm-tools/catalogs/catalog-browser` so the in-session GM drawer can
 * reuse the identical surface.
 */
export function CatalogsPage() {
  return (
    <PageChrome className="space-y-4">
      <GmPageHeader title="Catálogos" />

      <CatalogBrowser />
    </PageChrome>
  )
}
