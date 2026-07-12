import { Link } from '@tanstack/react-router'
import { Button } from '@/shared/ui/button'
import { PageChrome } from '@/shared/ui/page-chrome'
import { SectionHeading } from '@/shared/ui/section-heading'
import { CatalogBrowser } from '@/features/gm-tools/catalogs/catalog-browser'

/**
 * GM catalog browser — one tabbed page (condições / magias / poderes / itens)
 * over a shared search box for quick rules checks. The browser itself lives in
 * `features/gm-tools/catalogs/catalog-browser` so the in-session GM drawer can
 * reuse the identical surface.
 */
export function CatalogsPage() {
  return (
    <PageChrome className="space-y-4">
      <div className="flex items-center gap-3">
        <Link to="/gm">
          <Button variant="outline" size="sm">
            ←
          </Button>
        </Link>
        <SectionHeading variant="kallyadranoch" as="h1">
          Catálogos
        </SectionHeading>
      </div>

      <CatalogBrowser />
    </PageChrome>
  )
}
