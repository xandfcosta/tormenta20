import type { QueryClient } from '@tanstack/solid-query'
import { api } from '@/shared/api/api'

/** Liga ou desliga uma regra opcional da campanha (ALE-221). */
export type ToggleCampaignRule = (rule: string, emVigor: boolean) => Promise<void>

/**
 * Escreve o conjunto das regras desligadas e invalida QUEM DEPENDE DELE.
 *
 * As duas invalidações não são zelo: a regra vive na campanha e é aplicada na
 * FICHA. O servidor carimba `ignoredRules` em cada personagem, então uma
 * campanha que desliga a carga muda o deslocamento e três perícias de todo mundo
 * na mesa — e sem derrubar `['characters']` a ficha aberta ao lado continuaria
 * mostrando a penalidade que a mesa acabou de dispensar.
 *
 * Não é otimista, ao contrário das escritas da ficha, e a razão é o alcance: a
 * pintura otimista aqui teria de adivinhar o efeito da mudança em cada ficha
 * carregada — que é exatamente a conta que só o motor sabe fazer.
 *
 * @example const toggle = campaignRulesAction(queryClient, 3, ['carga'])
 */
export function campaignRulesAction(
  queryClient: QueryClient,
  campaignId: number,
  ignoradas: () => readonly string[],
): ToggleCampaignRule {
  return async (rule, emVigor) => {
    const proximas = emVigor
      ? ignoradas().filter((atual) => atual !== rule)
      : [...new Set([...ignoradas(), rule])]
    await api.campaigns.replaceRules(campaignId, proximas)
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['campaigns', campaignId] }),
      queryClient.invalidateQueries({ queryKey: ['characters'] }),
    ])
  }
}
