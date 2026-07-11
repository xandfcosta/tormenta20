import { createFileRoute, redirect } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { BookOpen, Package, Shield, Shirt, Star, ToggleRight } from 'lucide-react'
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/shared/ui/tabs'
import { cn } from '@/shared/lib/utils'
import { sheetBg } from '@/shared/lib/sheet-theme'
import {
  characterQueryOptions,
  meQueryOptions,
} from '@/shared/lib/queries'
import { AbilitiesPanel } from './character-sheet/abilities-panel'
import { CampaignsPanel } from './character-sheet/campaigns-panel'
import { EffectsCountBadge } from './character-sheet/effects-count-badge'
import { EffectsPanel } from './character-sheet/effects-panel'
import { EquipmentPanel } from './character-sheet/equipment-panel'
import { ExpertisesPanel } from './character-sheet/expertises-panel'
import { InventoryPanel } from './character-sheet/inventory-panel'
import { ProficienciesPanel } from './character-sheet/proficiencies-panel'
import { SheetHeader } from './character-sheet/sheet-header'
import { SpellbookPanel } from './character-sheet/spellbook-panel'
import { VitalsAside } from './character-sheet/vitals-aside'
import type { Character } from '@/shared/api/api'

export const Route = createFileRoute('/characters/$id')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user) throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context, params }) =>
    context.queryClient.ensureQueryData(characterQueryOptions(Number(params.id))),
  component: CharacterViewPage,
})

function CharacterViewPage() {
  const { id } = Route.useParams()
  const character = useQuery(characterQueryOptions(Number(id)))

  if (character.isLoading) return <p className="p-6">Loading…</p>
  if (character.isError) {
    return (
      <p className="p-6 text-destructive">{(character.error as Error).message}</p>
    )
  }
  if (!character.data) return null

  return <CharacterSheet character={character.data} />
}

function CharacterSheet({ character }: { character: Character }) {
  return (
    <div
      className={cn(
        'grid h-full grid-rows-[auto_auto_1fr] gap-2 overflow-hidden p-2 sm:gap-3 sm:p-3 lg:grid-cols-[minmax(300px,26rem)_1fr] lg:grid-rows-[auto_1fr]',
        sheetBg,
      )}
    >
      <SheetHeader character={character} className="lg:col-span-2" />
      <VitalsAside character={character} />
      <RightPanel character={character} />
    </div>
  )
}

function RightPanel({ character }: { character: Character }) {
  return (
    <Tabs
      defaultValue="expertises"
      className="flex min-h-0 flex-col gap-2 overflow-hidden"
    >
      <TabsList className="self-start">
        <TabsTrigger value="expertises">Perícias</TabsTrigger>
        <TabsTrigger value="equipment" className="gap-1.5">
          <Shirt className="size-3.5" /> Equipado
        </TabsTrigger>
        <TabsTrigger value="inventory" className="gap-1.5">
          <Package className="size-3.5" /> Inventário
        </TabsTrigger>
        <TabsTrigger value="conditionals" className="gap-1.5">
          <ToggleRight className="size-3.5" /> Efeitos
          <EffectsCountBadge character={character} />
        </TabsTrigger>
        <TabsTrigger value="proficiencies" className="gap-1.5">
          <Shield className="size-3.5" /> Proficiências
        </TabsTrigger>
        <TabsTrigger value="abilities" className="gap-1.5">
          <Star className="size-3.5" /> Habilidades
        </TabsTrigger>
        <TabsTrigger value="spells" className="gap-1.5">
          <BookOpen className="size-3.5" /> Magias
        </TabsTrigger>
        <TabsTrigger value="campaigns" className="gap-1.5">
          Campanhas
        </TabsTrigger>
      </TabsList>
      <TabsContent
        value="expertises"
        className="flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <ExpertisesPanel character={character} />
      </TabsContent>
      <TabsContent
        value="equipment"
        className="flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <EquipmentPanel character={character} />
      </TabsContent>
      <TabsContent
        value="inventory"
        className="flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <InventoryPanel character={character} />
      </TabsContent>
      <TabsContent
        value="conditionals"
        className="flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <EffectsPanel character={character} />
      </TabsContent>
      <TabsContent
        value="proficiencies"
        className="flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <ProficienciesPanel character={character} />
      </TabsContent>
      <TabsContent
        value="abilities"
        className="flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <AbilitiesPanel character={character} />
      </TabsContent>
      <TabsContent
        value="spells"
        className="flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <SpellbookPanel character={character} />
      </TabsContent>
      <TabsContent
        value="campaigns"
        className="flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <CampaignsPanel characterId={character.id} />
      </TabsContent>
    </Tabs>
  )
}
