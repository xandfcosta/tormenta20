import { describe, expect, it } from 'vitest'
import { sessionStatusMeta } from './status'

describe('sessionStatusMeta', () => {
  /**
   * Um caso para os três estados, e não três casos (ALE-187).
   *
   * A poda mandava cortar o arquivo inteiro — "re-declara 1:1 o mapa de uma
   * função de três ifs" — e o mecanismo realmente não precisa de teste. Mas os
   * RÓTULOS precisam: são texto pt-BR que o usuário lê, o GLOSSARIO os governa,
   * e "Encerrada" e "Planejada" não aparecem em teste montado nenhum nem no
   * e2e. Cortar tudo tiraria o único pino deles.
   *
   * "Ao vivo" tem outro dono (o e2e o afirma em duas telas), e vem junto aqui
   * só porque separá-lo custaria mais linhas do que a asserção inteira.
   */
  it('nomeia os três estados com as palavras que a tela mostra', () => {
    expect(sessionStatusMeta('active')).toEqual({ label: 'Ao vivo', tone: 'live' })
    expect(sessionStatusMeta('ended')).toEqual({ label: 'Encerrada', tone: 'ended' })
    expect(sessionStatusMeta('planned')).toEqual({ label: 'Planejada', tone: 'planned' })
  })
})
