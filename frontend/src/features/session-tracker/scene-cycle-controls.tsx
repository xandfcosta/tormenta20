import { RotateCcw, Square } from 'lucide-solid'
import { type JSX, Show } from 'solid-js'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'

/**
 * O FIM do ciclo da cena: encerrar e reiniciar (ALE-210).
 *
 * O começo não está aqui de propósito — ele mora na vaga do avanço de turno
 * (`SceneStart`), porque "iniciar" e "próximo turno" são o mesmo gesto em
 * momentos diferentes e nunca podem aparecer juntos. Estes dois são o oposto:
 * usam-se uma vez por briga, e por isso pagam o clique da confirmação.
 *
 * Os dois PARECEM a mesma coisa e não são, então cada diálogo diz o que
 * sobrevive: encerrar guarda a fila para a briga recomeçar com a mesma ordem, e
 * reiniciar é o único caminho que tira os combatentes da lista. Sem nomear isso
 * o mestre teria de descobrir a diferença apagando oito goblins por engano.
 *
 * Desde a ALE-220 encerrar também EXPIRA a duração "cena" das fichas do grupo,
 * e por isso a pergunta cresceu. É a única parte do gesto que não é reversível:
 * a fila fica guardada, mas a bênção não volta. Uma confirmação que não nomeia
 * o que apaga é pior do que nenhuma, porque ela ENSINA que o clique é barato.
 *
 * "Reiniciar" SAI do menu de configurações da sessão, onde a ALE-184 o pôs por
 * falta de lugar melhor: agora existe o lugar, que é junto do resto do ciclo.
 *
 * @example <SceneCycleControls compact connected onEnd={rt.endScene} onReset={rt.resetInitiative} />
 */
export function SceneCycleControls(props: {
  connected: boolean
  /** Só há o que encerrar dentro de uma cena. */
  sceneActive: boolean
  /**
   * Só há o que reiniciar com gente na fila — e isso NÃO depende da cena: quem
   * encerrou fica com os combatentes guardados, e esvaziá-los é o gesto
   * seguinte. Amarrar os dois à cena deixaria a lista sem caminho de saída.
   */
  hasQueue: boolean
  onEnd: () => void
  onReset: () => void
  /** No trilho de 64px só cabe o ícone; no menu da sessão cabe o rótulo. */
  compact?: boolean
}) {
  return (
    <>
      <Show when={props.sceneActive}>
        <ConfirmDialog
          title="Encerrar a cena?"
          description="A rodada e a vez voltam a zero e a mesa deixa de ver a fila. Nas fichas do grupo, os efeitos de duração cena, os usos 1/cena e as posturas ACABAM. Os combatentes CONTINUAM na lista — para esvaziá-la, use Reiniciar."
          confirmLabel="Encerrar"
          /* O `ConfirmDialog` assume destrutivo, e aqui isso estava errado: o
             botão saía CRIMSON, idêntico ao "Reiniciar" logo abaixo, bem no
             diálogo cujo texto existe para dizer que os dois são diferentes.
             Pela gramática que a ALE-200 fixou, entre clicáveis o dourado FAZ e
             o crimson DESTRÓI. Continua dourado depois da ALE-220, e a escolha
             é entre dois males: encerrar passou a apagar efeito, mas pintar os
             DOIS de crimson devolveria a confusão que este diálogo existe para
             desfazer. Quem separa os dois aqui é o texto, não a cor. */
          destructive={false}
          onConfirm={props.onEnd}
          trigger={(open) => (
            <CycleButton
              compact={props.compact}
              connected={props.connected}
              label="Encerrar cena"
              onClick={open}
              icon={<Square aria-hidden="true" class="size-4" />}
            />
          )}
        />
      </Show>
      <Show when={props.hasQueue}>
        <ConfirmDialog
          title="Reiniciar o combate?"
          description="A iniciativa, a rodada e o turno voltam a zero, e os combatentes saem da lista."
          confirmLabel="Reiniciar"
          destructive
          onConfirm={props.onReset}
          trigger={(open) => (
            <CycleButton
              compact={props.compact}
              connected={props.connected}
              label="Reiniciar o combate"
              onClick={open}
              icon={<RotateCcw aria-hidden="true" class="size-4" />}
            />
          )}
        />
      </Show>
    </>
  )
}

/**
 * Um botão do ciclo nas duas formas.
 *
 * UM componente e não dois ramos: o rótulo é o nome acessível nas duas
 * larguras — no trilho ele vai no `aria-label`, e quem usa leitor de tela ouve
 * a mesma frase que o mestre no celular lê.
 */
function CycleButton(props: {
  compact?: boolean
  connected: boolean
  label: string
  icon: JSX.Element
  onClick: () => void
}) {
  return (
    <Button
      size="sm"
      variant="outline"
      class={props.compact ? 'w-full px-0' : undefined}
      disabled={!props.connected}
      aria-label={props.compact ? props.label : undefined}
      title={props.compact ? props.label : undefined}
      onClick={props.onClick}
    >
      {props.icon}
      <Show when={!props.compact}>{props.label}</Show>
    </Button>
  )
}
