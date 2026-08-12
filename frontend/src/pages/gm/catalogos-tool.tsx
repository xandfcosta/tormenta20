import { CatalogBrowser } from '@/features/gm-tools/catalog-browser'

/**
 * Catálogos — condições, magias, poderes e itens numa busca só. The browser
 * itself is shared with the in-session panel, so a rules check reads the same
 * either side of the table.
 */
export function CatalogosTool() {
  return (
    <section class="flex min-h-0 flex-1 flex-col gap-3" aria-labelledby="mesa-catalogos">
      <h2
        id="mesa-catalogos"
        class="font-heading text-lg uppercase tracking-[0.16em] text-grimorio-gold"
      >
        Catálogos
      </h2>
      <CatalogBrowser />
    </section>
  )
}
