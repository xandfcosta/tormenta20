import { useQueryClient } from '@tanstack/solid-query'
import { For, Show, createSignal, onCleanup } from 'solid-js'
import { campaignSessionQueryOptions } from '@/entities/session/queries'
import { ApiError, type Session, api } from '@/shared/api/api'
import { toggleTaskLine } from '@/shared/lib/markdown'
import { MarkdownView } from '@/shared/ui/markdown-view'
import { Textarea } from '@/shared/ui/textarea'
import { SectionTitle } from '@/shared/ui/section-label'
import { Button } from '@/shared/ui/button'
import { createElementSize } from '@/shared/lib/element-size'
import {
  NOTES_VIEW_KEY,
  type NotesView,
  cabeLadoALado,
  efetivo,
  readNotesView,
} from './notes-view'

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
 * O ARRANJO é escolha do mestre desde a ALE-139 — escrever, ler ou lado a lado
 * —, porque lado a lado fixo desperdiça a tela nos dois extremos: com uma nota
 * de duas linhas são duas colunas quase vazias, e com uma nota longa se escreve
 * numa coluna estreita. A escolha GRUDA, porque é preferência de trabalho e não
 * estado da sessão.
 *
 * A largura que decide se o lado a lado cabe é a da REGIÃO, não a da janela:
 * esta região é 7/12 da tela na cena do mestre e a tela inteira na gaveta, e um
 * `lg:` aqui prometeria duas colunas onde cabe uma só (a ALE-122 tropeçou nisso
 * três vezes). Medida por observador em vez de consulta de contêiner porque a
 * FAIXA precisa da resposta: com o lado a lado fora de alcance, ela tem de
 * mostrar "Escrever" como o modo ativo — CSS esconde o botão, mas não conserta
 * o que os outros dois anunciam.
 */
export function SessionNotes(props: {
  campaignId: number
  session: Session
  /** Injetável para teste; em produção é o `localStorage`. */
  storage?: Storage
}) {
  const queryClient = useQueryClient()
  const armazem = () => props.storage ?? globalThis.localStorage
  const [regiao, setRegiao] = createSignal<HTMLElement>()
  const tamanho = createElementSize(regiao)
  const [escolhido, setEscolhido] = createSignal<NotesView>(
    readNotesView(armazem()?.getItem(NOTES_VIEW_KEY) ?? null),
  )
  const modo = () => efetivo(escolhido(), cabeLadoALado(tamanho().width))

  const escolher = (proximo: NotesView) => {
    setEscolhido(proximo)
    armazem()?.setItem(NOTES_VIEW_KEY, proximo)
  }

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
    <section ref={setRegiao} class="flex h-full min-h-0 flex-col gap-2">
      <div class="flex shrink-0 flex-wrap items-center justify-between gap-2">
        <SectionTitle contexto="painel" class="text-sm">Notas</SectionTitle>
        <div class="flex items-center gap-2">
          <ViewSwitch
            modo={modo()}
            cabeDuplo={cabeLadoALado(tamanho().width)}
            onModo={escolher}
          />
          <SaveStatus saving={saving()} dirty={draft() !== saved()} error={error()} />
        </div>
      </div>

      <div
        class="grid min-h-0 flex-1 gap-2"
        classList={{ 'grid-cols-2': modo() === 'duplo' }}
      >
        <Show when={modo() !== 'ler'}>
          <Textarea
            class="field-sizing-fixed h-full min-h-0 resize-none font-mono text-xs"
            value={draft()}
            onInput={(event) => edit(event.currentTarget.value)}
            aria-label="Notas da sessão"
            placeholder={'# Cena 1\n- O ogro **fugiu** pela ponte\n> "voltarei", ele disse'}
          />
        </Show>
        <Show when={modo() !== 'escrever'}>
          <div class="min-h-0 overflow-y-auto rounded-sm border border-grimorio-iron p-3">
            <Show
              when={draft().trim()}
              fallback={<p class="text-sm text-muted-foreground">Nenhuma nota ainda.</p>}
            >
              <MarkdownView source={draft()} onToggleTask={toggleTask} />
            </Show>
          </div>
        </Show>
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

const ROTULO: Record<NotesView, string> = {
  escrever: 'Escrever',
  ler: 'Ler',
  duplo: 'Lado a lado',
}

/**
 * A faixa de modos, no canto do cabeçalho (ALE-139).
 *
 * "Lado a lado" só aparece onde cabe. Escondê-lo é metade do trabalho: sem a
 * outra metade — o modo EFETIVO — a faixa mostraria os dois botões restantes
 * sem nenhum marcado, e o mestre não saberia o que está vendo.
 */
function ViewSwitch(props: {
  modo: NotesView
  cabeDuplo: boolean
  onModo: (modo: NotesView) => void
}) {
  const modos = () =>
    (props.cabeDuplo ? ['escrever', 'ler', 'duplo'] : ['escrever', 'ler']) as NotesView[]

  return (
    // `fieldset` e não `div role="group"`: o elemento nativo já É o grupo, e a
    // regra de semântica da casa pede o elemento quando ele existe. Precisa de
    // `min-w-0`, porque o padrão do navegador para fieldset é `min-content` e
    // ele se recusa a encolher dentro de um flex.
    <fieldset
      class="flex min-w-0 shrink-0 gap-1"
      aria-label="Modo de visualização das notas"
    >
      <For each={modos()}>
        {(valor) => (
          <Button
            size="sm"
            variant={props.modo === valor ? 'default' : 'outline'}
            aria-pressed={props.modo === valor}
            class="h-7 px-2 text-2xs"
            onClick={() => props.onModo(valor)}
          >
            {ROTULO[valor]}
          </Button>
        )}
      </For>
    </fieldset>
  )
}
