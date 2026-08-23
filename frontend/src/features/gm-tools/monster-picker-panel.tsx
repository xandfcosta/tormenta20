import type { Monster } from '@/shared/api/catalog-types'
import type { JSX } from 'solid-js'
import { SidePanel } from '@/shared/ui/side-panel'
import { MonsterPickerList } from './monster-picker-list'

export type MonsterPickerPanelProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: string
  /** Live context pinned above the list (the tracker's peek, in a session). */
  header?: JSX.Element
  onPick: (monster: Monster) => void
  /** Ver `MonsterPickerListProps.itemVerbo`. */
  itemVerbo?: string
  /** Whether picking closes the panel. Adding several creatures in a row is the
   *  normal case, so the default keeps it open. */
  closeOnPick?: boolean
}

/**
 * `MonsterPickerList` in a side panel — for surfaces whose main content is
 * something else (the Mesa's encounter stage, the session's tracker) and that
 * want the bestiary beside it rather than in place of it.
 */
export function MonsterPickerPanel(props: MonsterPickerPanelProps) {
  const pick = (monster: Monster) => {
    props.onPick(monster)
    if (props.closeOnPick) props.onOpenChange(false)
  }

  return (
    <SidePanel
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={props.title}
      description={props.description}
      header={props.header}
    >
      <MonsterPickerList onPick={pick} idPrefix="picker" itemVerbo={props.itemVerbo} />
    </SidePanel>
  )
}
