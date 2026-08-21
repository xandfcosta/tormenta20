import { useQueryClient } from '@tanstack/solid-query'
import { Show, createSignal, onCleanup } from 'solid-js'
import { campaignSessionQueryOptions } from '@/entities/session/queries'
import { ApiError, type Session, api } from '@/shared/api/api'
import { toggleTaskLine } from '@/shared/lib/markdown'
import { MarkdownView } from '@/shared/ui/markdown-view'
import { Textarea } from '@/shared/ui/textarea'

const AUTOSAVE_DELAY_MS = 1200

/**
 * As notas da sessão: o que aconteceu, XP, tesouro — em markdown, com o texto
 * de um lado e o resultado do outro (ALE-122).
 *
 * Antes eram um quadrado de dez linhas dentro de um diálogo pequeno, atrás de
 * um botão "Editar", sem marcação nenhuma. Agora ocupam a região inteira da aba
 * e NÃO têm botão de salvar: o mestre escreve no meio do combate e não vai
 * lembrar de confirmar. Salva sozinha depois da pausa, e ao sair da aba —
 * o Tabs desmonta o conteúdo inativo, então o rascunho iria embora com ele.
 *
 * Lado a lado por consulta de CONTÊINER, não de viewport: esta região é 7/12
 * da tela, e um `lg:` aqui promete duas colunas onde cabe uma só (ALE-122
 * tropeçou nisso três vezes).
 */
export function SessionNotes(props: { campaignId: number; session: Session }) {
  const queryClient = useQueryClient()
  const [draft, setDraft] = createSignal(props.session.notes ?? '')
  const [saved, setSaved] = createSignal(props.session.notes ?? '')
  const [saving, setSaving] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)
  let timer: ReturnType<typeof setTimeout> | undefined

  const save = async () => {
    const text = draft()
    if (text === saved()) return
    setSaving(true)
    setError(null)
    try {
      await api.sessions.update(props.campaignId, props.session.id, { notes: text })
      setSaved(text)
      queryClient.invalidateQueries({
        queryKey: campaignSessionQueryOptions(props.campaignId, props.session.id).queryKey,
      })
    } catch (failure) {
      setError(failure instanceof ApiError ? failure.message : 'Erro ao salvar')
    } finally {
      setSaving(false)
    }
  }

  const edit = (text: string) => {
    setDraft(text)
    clearTimeout(timer)
    timer = setTimeout(() => void save(), AUTOSAVE_DELAY_MS)
  }

  /** Marcar a tarefa reescreve a NOTA, e salva na hora: um clique é uma
   *  decisão inteira, não uma digitação no meio de uma frase. */
  const toggleTask = (line: number, checked: boolean) => {
    setDraft((current) => toggleTaskLine(current, line, checked))
    clearTimeout(timer)
    void save()
  }

  onCleanup(() => {
    clearTimeout(timer)
    void save() // trocar de aba desmonta esta região; o texto não pode ir junto
  })

  return (
    <section class="@container flex h-full min-h-0 flex-col gap-2">
      <div class="flex shrink-0 items-center justify-between gap-2">
        <h2 class="font-heading text-sm uppercase tracking-wide text-grimorio-gold">Notas</h2>
        <SaveStatus saving={saving()} dirty={draft() !== saved()} error={error()} />
      </div>

      <div class="grid min-h-0 flex-1 grid-rows-2 gap-2 @2xl:grid-cols-2 @2xl:grid-rows-1">
        <Textarea
          class="field-sizing-fixed h-full min-h-0 resize-none font-mono text-xs"
          value={draft()}
          onInput={(event) => edit(event.currentTarget.value)}
          aria-label="Notas da sessão"
          placeholder={'# Cena 1\n- O ogro **fugiu** pela ponte\n> "voltarei", ele disse'}
        />
        <div class="min-h-0 overflow-y-auto rounded-sm border border-grimorio-iron p-3">
          <Show
            when={draft().trim()}
            fallback={<p class="text-sm text-muted-foreground">Nenhuma nota ainda.</p>}
          >
            <MarkdownView source={draft()} onToggleTask={toggleTask} />
          </Show>
        </div>
      </div>
    </section>
  )
}

/** "Não salvo" enquanto a pausa não chega — dizer "Salvando…" ali seria mentira. */
function statusLabel(saving: boolean, dirty: boolean): string {
  if (saving) return 'Salvando…'
  return dirty ? 'Não salvo' : 'Salvo'
}

/** Sem botão de salvar, o estado do texto tem de estar VISÍVEL o tempo todo. */
function SaveStatus(props: { saving: boolean; dirty: boolean; error: string | null }) {
  return (
    <span
      class="text-xs"
      classList={{
        'text-destructive': props.error !== null,
        'text-muted-foreground': props.error === null,
      }}
      aria-live="polite"
    >
      <Show when={props.error} fallback={statusLabel(props.saving, props.dirty)}>
        {(message) => message()}
      </Show>
    </span>
  )
}
