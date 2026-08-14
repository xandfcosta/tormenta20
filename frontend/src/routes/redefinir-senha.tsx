import { createFileRoute } from '@tanstack/solid-router'
import { ResetPasswordPage } from '@/pages/auth/reset-password-page'

type ResetSearch = { token?: string }

/**
 * A ponta do jogador no link que o admin gera (ALE-120). Anônima de propósito:
 * quem esqueceu a senha não consegue autenticar para trocá-la — o que guarda a
 * rota é o token de uso único, verificado no servidor.
 */
export const Route = createFileRoute('/redefinir-senha')({
  validateSearch: (search: Record<string, unknown>): ResetSearch =>
    typeof search.token === 'string' ? { token: search.token } : {},
  component: ResetPasswordPage,
})
