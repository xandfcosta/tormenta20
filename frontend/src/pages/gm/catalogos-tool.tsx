import { CatalogBrowser } from '@/features/gm-tools/catalog-browser'
import { SectionTitle } from '@/shared/ui/section-label'

/**
 * Catálogos — condições, magias, poderes e itens numa busca só. The browser
 * itself is shared with the in-session panel, so a rules check reads the same
 * either side of the table.
 */
export function CatalogosTool() {
  return (
    <section class="flex min-h-0 flex-1 flex-col gap-3" aria-labelledby="mesa-catalogos">
      <SectionTitle
        id="mesa-catalogos"
       
      >
        Catálogos
      </SectionTitle>
      <CatalogBrowser />
    </section>
  )
}
