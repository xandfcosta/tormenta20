import { Check, Copy } from 'lucide-solid'
import { type JSX, Show, createSignal, onCleanup } from 'solid-js'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'

export type CopyLinkRowProps = {
  url: string
  /** Accessible name of the field — there are two kinds of invite (ALE-120). */
  label: string
  /** Copy injected, so a test drives it without `navigator.clipboard`. */
  onCopy: (text: string) => Promise<void>
  /** The line under the field, explaining what this particular link does. */
  children?: JSX.Element
}

/**
 * A read-only link with a copy button that ACKNOWLEDGES the copy — the check
 * mark is the only feedback the clipboard gives, and without it the player is
 * left wondering whether the tap registered.
 *
 * Shared by the campaign invite and the account invite; they differ in copy and
 * in the URL they mint, not in this row.
 *
 * @example <CopyLinkRow url={url} label="Link de convite" onCopy={copyToClipboard} />
 */
export function CopyLinkRow(props: CopyLinkRowProps) {
  const [copied, setCopied] = createSignal(false)
  let timer: ReturnType<typeof setTimeout> | undefined
  onCleanup(() => clearTimeout(timer))

  const copy = async () => {
    await props.onCopy(props.url)
    setCopied(true)
    timer = setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div class="space-y-3">
      <div class="flex items-center gap-2">
        <Input readOnly value={props.url} aria-label={props.label} class="font-mono text-xs" />
        <Button type="button" variant="outline" size="sm" onClick={copy} aria-label="Copiar link">
          <Show when={copied()} fallback={<Copy aria-hidden="true" class="size-4" />}>
            <Check aria-hidden="true" class="size-4" />
          </Show>
        </Button>
      </div>
      <Show when={props.children}>
        <p class="text-xs text-muted-foreground">{props.children}</p>
      </Show>
    </div>
  )
}
