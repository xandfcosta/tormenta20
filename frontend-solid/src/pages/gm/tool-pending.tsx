import { type ToolSlug, toolLabel } from '@/features/gm-tools/gm-tools'

/** Scaffolding: deleted as each remaining tool of ALE-75 lands. */
export function ToolPending(props: { slug: ToolSlug }) {
  return (
    <div class="flex flex-1 flex-col items-center justify-center gap-2 text-center">
      <p class="font-heading text-lg uppercase tracking-[0.16em] text-muted-foreground">
        {toolLabel(props.slug)}
      </p>
      <p class="text-sm text-muted-foreground">
        Esta ferramenta ainda está sendo montada (ALE-75).
      </p>
    </div>
  )
}
