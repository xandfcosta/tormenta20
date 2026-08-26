/**
 * Gera `engine-go/api/testdata/markdown-do-js.json` a partir do markdown da SPA
 * (ALE-269).
 *
 * POR QUE UM ORÁCULO E NÃO UMA LISTA DE `expect` EM GO: as notas da sessão são
 * desenhadas pelas DUAS telas enquanto a migração durar, a partir do MESMO texto
 * no banco. Um port "parecido" faz a mesma nota do mestre sair diferente em cada
 * uma — e as diferenças desta gramática são justamente as decisões que custaram
 * caro, não detalhes:
 *
 *   - o parágrafo guarda LINHA A LINHA, porque numa nota de mesa a quebra é
 *     intencional (ALE-122: trinta linhas viravam um parágrafo só);
 *   - `####` não é título, é texto — o que ele não conhece nunca é comido;
 *   - link só `http(s)`; um `javascript:` volta a ser TEXTO, não link morto.
 *
 * Escrever os esperados à mão em Go seria uma SEGUNDA TRANSCRIÇÃO da gramática,
 * e a segunda transcrição é onde as duas telas passam a discordar em silêncio. O
 * oráculo é medido rodando o JS de verdade, como o `matizDoNome` já fez.
 *
 * O Node apaga os tipos sozinho (>= 23.6), então o `.ts` da SPA roda direto.
 *
 * Uso: node scripts/dump-markdown-oracle.ts
 *
 * Regenerar é ATO DELIBERADO, como o `genoracle`: se um caso mudar, o diff diz
 * que a gramática da SPA mudou — e aí o Go acompanha, nunca o contrário.
 */
import { writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { parseMarkdown, toggleTaskLine } from '../frontend/src/shared/lib/markdown.ts'

const AQUI = dirname(fileURLToPath(import.meta.url))
const SAIDA = resolve(AQUI, '../engine-go/api/testdata/markdown-do-js.json')

/**
 * Os casos, e cada um está aqui por uma razão nomeada — um oráculo de casos
 * bonitos passaria verde sobre a armadilha que ele deveria prender.
 */
const CASOS: { nota: string; fonte: string }[] = [
  { nota: 'o cabeçalho e o negrito, o caso comum', fonte: '# Cena 1\n- O ogro **fugiu** pela ponte' },
  { nota: 'ALE-122: a quebra de linha é INTENCIONAL, não junta', fonte: 'linha um\nlinha dois\nlinha tres' },
  { nota: 'tarefa carrega a LINHA de origem, senão o clique não reescreve a nota', fonte: '- [ ] pagar o taverneiro\n- [x] dar o XP' },
  { nota: 'citação', fonte: '> "voltarei", ele disse' },
  { nota: 'lista ordenada', fonte: '1. primeiro\n2. segundo' },
  { nota: 'régua', fonte: '---' },
  { nota: 'link http passa, javascript: vira TEXTO', fonte: 'um `codigo` e um [elo](https://t20.com) e um [mau](javascript:alert(1))' },
  { nota: 'título fundo demais NÃO é título — o que ele não conhece vira texto', fonte: '#### fundo demais' },
  { nota: 'itálico com asterisco e com sublinhado', fonte: '*so italico* e **negrito** e _sublinhado_' },
  { nota: 'nota vazia é caminho normal', fonte: '' },
  { nota: 'só espaços', fonte: '   \n  ' },
  { nota: 'marcação aberta e não fechada volta inteira como texto', fonte: 'um **negrito que nao fecha e um [elo(sem parenteses' },
  { nota: 'lista colada num parágrafo', fonte: 'antes\n- item\ndepois' },
  { nota: 'tarefa maiúscula [X] conta como marcada', fonte: '- [X] gritou' },
  { nota: 'asterisco também abre item de lista', fonte: '* com asterisco' },
]

const ALTERNA: { nota: string; fonte: string; linha: number; marcada: boolean }[] = [
  { nota: 'marca', fonte: '- [ ] dar XP', linha: 0, marcada: true },
  { nota: 'desmarca', fonte: '- [x] dar XP', linha: 0, marcada: false },
  { nota: 'linha que não é tarefa volta INTACTA', fonte: 'sem tarefa', linha: 0, marcada: true },
  { nota: 'mexe só na linha pedida', fonte: '- [ ] a\n- [ ] b', linha: 1, marcada: true },
  { nota: 'linha fora da faixa volta intacta', fonte: '- [ ] a', linha: 9, marcada: true },
  // DOIS quadrinhos na mesma linha: o `String.replace` sem `g` troca só o
  // PRIMEIRO, e um `ReplaceAll` no Go reescreveria também o que o mestre
  // digitou no meio do texto. Sem este caso a divergência passa despercebida —
  // conferido sabotando o port, que ficou VERDE até esta linha existir.
  { nota: 'só o PRIMEIRO quadrinho da linha muda', fonte: '- [ ] anotar [ ] no mapa', linha: 0, marcada: true },
]

writeFileSync(
  SAIDA,
  `${JSON.stringify(
    {
      _: 'GERADO por scripts/dump-markdown-oracle.ts a partir de frontend/src/shared/lib/markdown.ts. Não edite à mão.',
      arvores: CASOS.map((c) => ({ ...c, blocos: parseMarkdown(c.fonte) })),
      alterna: ALTERNA.map((a) => ({ ...a, saida: toggleTaskLine(a.fonte, a.linha, a.marcada) })),
    },
    null,
    2,
  )}\n`,
)
console.log(`escrito: ${SAIDA}`)
