import { useQueryClient } from '@tanstack/solid-query'
import { For, Show, createSignal } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import { DialogInlineError } from '@/shared/ui/dialog-inline-error'
import { Panel } from '@/shared/ui/panel'
import { SectionLabel } from '@/shared/ui/section-label'
import { campaignRulesAction } from './campaign-rules-write'

/**
 * As regras opcionais da campanha — o catálogo que a TELA mostra (ALE-221).
 *
 * O identificador é o mesmo do motor (`engine.RuleCarga`) e do banco; o texto é
 * daqui, porque é pt-BR de tela e o Go não escreve tela. A página do livro entra
 * no rótulo de propósito: quem desliga a regra quer conferir o que está
 * dispensando.
 */
const REGRAS_OPCIONAIS = [
  {
    id: 'carga',
    titulo: 'Limites de carga',
    descricao:
      'Passar do limite sobrecarrega: −5 de penalidade de armadura e −3m de deslocamento (p141). Os espaços continuam somados na mochila mesmo com a regra desligada.',
  },
] as const

/**
 * O painel dos interruptores. Ele é UM e mora em dois lugares — a aba de
 * configuração da campanha e a gaveta de dentro da sessão —, porque o valor é
 * um só: duas portas para a mesma chave, e não um override de sessão.
 *
 * A falha aparece INLINE e não por toast: este painel também é montado dentro de
 * um overlay na cena da sessão, e um toast disparado de dentro de modal não é
 * anunciado (o Kobalte marca os irmãos com `aria-hidden`).
 *
 * `heading={false}` quando o host já nomeia o painel — é o que a gaveta da
 * sessão faz, e sem isso o título aparece duas vezes a vinte pixels de
 * distância, que é o defeito que a ALE-145 consertou noutro lugar.
 *
 * @example <CampaignRulesCard campaignId={3} ignoredRules={campaign().ignoredRules ?? []} />
 */
export function CampaignRulesCard(props: {
  campaignId: number
  /** Os identificadores DESLIGADOS hoje. Vazio = tudo em vigor. */
  ignoredRules: readonly string[]
  /** O host já escreveu o nome do painel? Padrão: não, e o cartão o escreve. */
  heading?: boolean
}) {
  const queryClient = useQueryClient()
  const [pendente, setPendente] = createSignal<string | null>(null)
  const [erro, setErro] = createSignal<string | null>(null)
  const alternar = campaignRulesAction(queryClient, props.campaignId, () => props.ignoredRules)

  const emVigor = (id: string) => !props.ignoredRules.includes(id)

  const clicar = async (id: string) => {
    setErro(null)
    setPendente(id)
    try {
      await alternar(id, !emVigor(id))
    } catch {
      setErro('Não foi possível salvar a regra. Tente de novo.')
    } finally {
      setPendente(null)
    }
  }

  return (
    <section class="space-y-3" aria-label="Regras da campanha">
      <Show when={props.heading !== false}>
        <SectionLabel class="font-semibold">Regras da campanha</SectionLabel>
      </Show>
      <p class="text-xs text-muted-foreground">
        O livro deixa estas na sua mão. Desligar vale para todos os personagens da
        campanha, na hora.
      </p>
      <For each={REGRAS_OPCIONAIS}>
        {(regra) => (
          <Panel class="flex flex-wrap items-start justify-between gap-3 p-3">
            <div class="min-w-[12rem] flex-1 space-y-1">
              <p class="text-sm text-foreground">{regra.titulo}</p>
              <p class="text-xs text-muted-foreground">{regra.descricao}</p>
            </div>
            <button
              type="button"
              aria-label={regra.titulo}
              aria-pressed={emVigor(regra.id)}
              disabled={pendente() === regra.id}
              onClick={() => clicar(regra.id)}
              class={cn(
                'shrink-0 rounded-sm border px-3 py-1 text-xs transition-colors disabled:opacity-60',
                emVigor(regra.id)
                  ? 'border-grimorio-gold bg-accent text-grimorio-gold'
                  : 'border-grimorio-iron text-muted-foreground hover:bg-accent',
              )}
            >
              {emVigor(regra.id) ? 'Em vigor' : 'Desligada'}
            </button>
          </Panel>
        )}
      </For>
      <DialogInlineError message={erro()} />
    </section>
  )
}
