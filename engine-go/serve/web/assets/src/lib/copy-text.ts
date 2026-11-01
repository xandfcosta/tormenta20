/**
 * Copia um texto para a área de transferência, e diz se copiou.
 *
 * `navigator.clipboard` só existe em contexto seguro, HTTPS ou `localhost`, e o app é
 * servido em HTTP na rede da mesa. Em `http://192.168.x.x` ele é `undefined`, e chamá-lo
 * direto estoura no clique sem nada na tela. Por isso o caminho antigo, `execCommand`,
 * fica de reserva, e não é resto de código: é o que funciona no endereço de uso real.
 *
 * @example await copyText(location.href) // true se copiou
 */
export async function copyText(text: string): Promise<boolean> {
  if (navigator.clipboard) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // A permissão pode ser recusada mesmo em contexto seguro: tenta o outro caminho.
    }
  }
  return copyBySelecting(text)
}

/**
 * Seleciona o texto num campo temporário e pede a cópia ao navegador.
 *
 * O campo entra no diálogo aberto, se houver: fora dele a página está inerte, e um campo
 * ali não recebe a seleção. `readOnly` é o que impede o teclado do celular de subir, e o
 * foco volta para onde estava porque selecionar o rouba.
 */
function copyBySelecting(text: string): boolean {
  const previous = document.activeElement as HTMLElement | null
  const holder = document.createElement('textarea')
  holder.value = text
  holder.readOnly = true
  holder.style.cssText = 'position:fixed;top:0;left:0;opacity:0'
  ;(document.querySelector('dialog[open]') ?? document.body).append(holder)
  holder.select()
  holder.setSelectionRange(0, text.length)
  let copied = false
  try {
    copied = document.execCommand('copy')
  } catch {
    copied = false
  }
  holder.remove()
  previous?.focus()
  return copied
}
