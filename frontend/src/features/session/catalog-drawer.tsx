import { BookMarked } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/shared/ui/sheet'
import { CatalogBrowser } from '@/features/gm-tools/catalogs/catalog-browser'

/**
 * In-session catalog launcher for the GM rail. Opens a side sheet with the full
 * tabbed catalog browser (condições / magias / poderes / itens) so the GM can
 * do a quick rules check mid-combat without leaving the match for /gm/catalogs.
 */
export function CatalogDrawer() {
  return (
    <Sheet>
      <SheetTrigger asChild>
        <Button size="sm" variant="secondary" className="w-full gap-1.5">
          <BookMarked className="size-4" /> Catálogos
        </Button>
      </SheetTrigger>
      <SheetContent className="flex w-full flex-col gap-4 sm:max-w-md">
        <SheetHeader>
          <SheetTitle className="font-display tracking-wide">
            Catálogos
          </SheetTitle>
          <SheetDescription>
            Consulta rápida de condições, magias, poderes e itens.
          </SheetDescription>
        </SheetHeader>
        <div className="min-h-0 flex-1 overflow-hidden px-4 pb-4">
          <CatalogBrowser listClassName="max-h-[60vh] pr-1" />
        </div>
      </SheetContent>
    </Sheet>
  )
}
