import { Mail } from 'lucide-solid'
import { Match, Switch } from 'solid-js'
import { Skeleton } from '@/shared/ui/skeleton'

export type InviteLetterProps = {
  /** Still resolving the token against the backend. */
  loading: boolean
  /** The token resolved to nothing — expired or rotated away. */
  invalid: boolean
  campaignName: string | undefined
}

/**
 * The invite as a letter tucked into the tome: it tells the player WHICH table
 * they were called to before they commit a character to it. A dead token says
 * so plainly here, so the player asks the GM for a new link instead of
 * wondering why the button won't submit.
 *
 * @example <InviteLetter loading={q.isLoading} invalid={q.isError} campaignName={q.data?.campaignName} />
 */
export function InviteLetter(props: InviteLetterProps) {
  return (
    <div class="flex items-start gap-3 rounded-sm border border-grimorio-gold/40 bg-[var(--grimorio-panel)] p-4">
      <Mail aria-hidden="true" class="mt-0.5 size-4 shrink-0 text-grimorio-gold" />
      <div class="min-w-0 flex-1 space-y-1">
        <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">
          Convite
        </p>
        <Switch>
          <Match when={props.loading}>
            <Skeleton class="h-5 w-64" />
          </Match>
          <Match when={props.invalid}>
            <p class="text-sm text-destructive">
              Convite inválido ou expirado. Peça um novo link ao mestre.
            </p>
          </Match>
          <Match when={props.campaignName}>
            {(name) => (
              <p class="text-sm">
                Você foi convidado para{' '}
                <span class="font-semibold text-grimorio-gold">{name()}</span>.
              </p>
            )}
          </Match>
        </Switch>
      </div>
    </div>
  )
}
