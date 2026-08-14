import { Database, HardDriveDownload } from 'lucide-solid'
import { For, Show, createSignal } from 'solid-js'
import type { Backup, ServerStatus } from '@/shared/api/api'
import { toSubmitFailure } from '@/shared/lib/form-errors'
import { Button } from '@/shared/ui/button'
import { FramedPanel } from '@/shared/ui/framed-panel'

export type ServerPanelProps = {
  status: ServerStatus
  backups: Backup[]
  onBackup: () => Promise<void>
}

/**
 * O painel de servidor da tela de administração (ALE-120): o que o dono da mesa
 * precisa saber sem abrir o terminal, e o botão de backup.
 *
 * Não há "resetar banco" nem "aplicar seed" aqui de propósito. Botão destrutivo
 * num celular, no meio da sessão, é acidente esperando — e é operação de uma
 * vez por ano, com o notebook aberto.
 */
export function ServerPanel(props: ServerPanelProps) {
  const [pending, setPending] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)

  const backup = async () => {
    setPending(true)
    setError(null)
    try {
      await props.onBackup()
    } catch (failure) {
      setError(toSubmitFailure(failure).formError ?? 'Erro ao fazer backup')
    } finally {
      setPending(false)
    }
  }

  return (
    <FramedPanel>
      <h2 class="font-heading text-sm uppercase tracking-[0.18em] text-grimorio-gold/80">
        Servidor
      </h2>

      <dl class="mt-3 space-y-1 text-sm">
        <Row label="Ambiente" value={props.status.environment} />
        <Row label="Banco" value={`${props.status.databasePath} · ${bytes(props.status.databaseSize)}`} />
        <Row
          label="Conteúdo"
          value={`${props.status.users} contas · ${props.status.campaigns} mesas · ${props.status.characters} fichas`}
        />
      </dl>

      <div class="mt-4 flex flex-wrap items-center justify-between gap-2">
        <p class="text-xs text-muted-foreground">
          <Show
            when={props.backups[0]}
            fallback="Nenhum backup ainda."
            keyed
          >
            {(latest) => `Último backup: ${dateLabel(latest.createdAt)} · ${bytes(latest.size)}`}
          </Show>
        </p>
        <Button disabled={pending()} onClick={backup}>
          <HardDriveDownload aria-hidden="true" class="mr-1 size-4" />
          {pending() ? 'Copiando…' : 'Fazer backup'}
        </Button>
      </div>

      <Show when={error()}>{(message) => <p class="mt-2 text-sm text-destructive">{message()}</p>}</Show>

      <Show when={props.backups.length > 0}>
        <ul class="mt-3 max-h-40 space-y-1 overflow-y-auto text-xs text-muted-foreground">
          <For each={props.backups}>
            {(file) => (
              <li class="flex items-center gap-2">
                <Database aria-hidden="true" class="size-3.5 shrink-0" />
                <span class="truncate font-mono">{file.name}</span>
                <span class="ml-auto shrink-0">{bytes(file.size)}</span>
              </li>
            )}
          </For>
        </ul>
      </Show>
    </FramedPanel>
  )
}

function Row(props: { label: string; value: string }) {
  return (
    <div class="flex flex-wrap gap-x-2">
      <dt class="text-muted-foreground">{props.label}</dt>
      <dd class="min-w-0 flex-1 truncate font-mono text-xs">{props.value}</dd>
    </div>
  )
}

/** Tamanho legível: o dono quer saber se cabe no pendrive, não o byte exato. */
function bytes(size: number): string {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(0)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

function dateLabel(iso: string): string {
  const at = new Date(iso)
  return Number.isNaN(at.getTime()) ? iso : at.toLocaleString('pt-BR')
}
