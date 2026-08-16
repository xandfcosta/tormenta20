/**
 * O link que o admin entrega ao jogador.
 *
 * Vive fora dos dois lugares que o mostram — o diálogo do Hub e o painel de
 * convites em aberto — porque a chave `?convite=` é um CONTRATO com a rota
 * `/register`, que a lê. Com a montagem repetida em cada tela, renomear a chave
 * num lugar só deixava o admin copiando um link que o registro ignora em
 * silêncio (ALE-123).
 *
 * @example inviteRegisterUrl(window.location.origin, token)
 */
export function inviteRegisterUrl(origin: string, token: string): string {
  return `${origin}/register?convite=${encodeURIComponent(token)}`
}
