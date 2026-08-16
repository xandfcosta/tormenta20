/**
 * Os lugares que o mestre pode escolher, desenhados em CSS (ALE-124).
 *
 * Sintético e não imagem: o app não tem upload nem armazenamento de arquivo, e
 * a primeira fatia do tabuleiro não pode ficar esperando essa esteira nascer. A
 * grade por cima é a mesma em todos — o que muda é o chão.
 *
 * O `repeating-linear-gradient` da grade vem por último na pilha e usa
 * `background-size` calculado pelo número de quadrados, então a linha cai
 * exatamente na borda da célula em qualquer tamanho de tela.
 */
export const TERRAIN_STYLE: Record<string, string> = {
  pedra: 'bg-[repeating-linear-gradient(0deg,rgba(255,255,255,.07)_0_1px,transparent_1px_100%),repeating-linear-gradient(90deg,rgba(255,255,255,.07)_0_1px,transparent_1px_100%),linear-gradient(180deg,#2a2723,#1d1b18)]',
  taverna:
    'bg-[repeating-linear-gradient(0deg,rgba(255,255,255,.06)_0_1px,transparent_1px_100%),repeating-linear-gradient(90deg,rgba(255,255,255,.06)_0_1px,transparent_1px_100%),linear-gradient(180deg,#3a2a1c,#241a11)]',
  floresta:
    'bg-[repeating-linear-gradient(0deg,rgba(255,255,255,.06)_0_1px,transparent_1px_100%),repeating-linear-gradient(90deg,rgba(255,255,255,.06)_0_1px,transparent_1px_100%),linear-gradient(180deg,#1e2c1e,#121a12)]',
  ermo: 'bg-[repeating-linear-gradient(0deg,rgba(255,255,255,.06)_0_1px,transparent_1px_100%),repeating-linear-gradient(90deg,rgba(255,255,255,.06)_0_1px,transparent_1px_100%),linear-gradient(180deg,#332c20,#1f1a13)]',
  cripta:
    'bg-[repeating-linear-gradient(0deg,rgba(255,255,255,.08)_0_1px,transparent_1px_100%),repeating-linear-gradient(90deg,rgba(255,255,255,.08)_0_1px,transparent_1px_100%),linear-gradient(180deg,#232228,#141317)]',
  papel:
    'bg-[repeating-linear-gradient(0deg,rgba(0,0,0,.10)_0_1px,transparent_1px_100%),repeating-linear-gradient(90deg,rgba(0,0,0,.10)_0_1px,transparent_1px_100%),linear-gradient(180deg,#e8ddc4,#d6c9ab)]',
}

/** Rótulo de cada chão, para o seletor do mestre. */
export const TERRAIN_LABEL: Record<keyof typeof TERRAIN_STYLE, string> = {
  pedra: 'Pedra',
  taverna: 'Taverna',
  floresta: 'Floresta',
  ermo: 'Ermo',
  cripta: 'Cripta',
  papel: 'Papel',
}

export const TERRAIN_IDS = Object.keys(TERRAIN_STYLE)
