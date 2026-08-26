/**
 * Gera `engine-go/api/piloto_icones.templ` a partir do lucide INSTALADO (ALE-231).
 *
 * Por que gerar em vez de transcrever: as páginas do Datastar precisam do SVG
 * embutido, e copiar `d="M19 17V5a2 2..."` à mão doze vezes é transcrição — a
 * mesma classe de erro que a auditoria do bestiário encontrou 44 vezes, e que
 * nenhum teste pega, porque um path errado é um path válido.
 *
 * Ele resolve o APELIDO: a SPA escreve `Users2`, e o lucide 1.x guarda esse
 * ícone em `users-round.mjs`. Listar o nome que a SPA usa e deixar o gerador
 * achar o arquivo é o que impede as duas portas de desenharem ícones
 * diferentes quando o pacote renomeia alguma coisa.
 *
 *   node scripts/gen-icones-templ.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const AQUI = dirname(fileURLToPath(import.meta.url))
const LUCIDE = resolve(AQUI, '../node_modules/lucide-solid/dist/esm')
const SAIDA = resolve(AQUI, '../../engine-go/api/piloto_icones.templ')

/** Os nomes como a SPA os escreve. Acrescentar aqui e rodar o gerador. */
const QUERIDOS = [
  'Users2',
  'Scroll',
  'Wand2',
  'PlayCircle',
  'Settings',
  'Volume2',
  'VolumeX',
  'Maximize',
  'Minimize',
  'UserPlus',
  'ShieldCheck',
  'LogOut',
  'ChevronRight',
  'Search',
  'Plus',
  'KeyRound',
  'Mail',
  'Crown',
  'Trash2',
  'Flame',
  // A trilha de ferramentas do mestre (ALE-257): a mesma escolha de ícones da
  // SPA, para o mestre que aprendeu a trilha num lado a reconhecer no outro.
  'Skull',
  'Swords',
  'Dices',
  'BookMarked',
  // Os oito catálogos viraram paradas do trilho (ALE-264): cada um precisa do
  // ícone dele. `Skull` (bestiário) e `Wand2` (magias) já estavam na lista.
  'Shield',
  'HeartPulse',
  'Church',
  'Zap',
  'Backpack',
  'Star',
  // A MESA em forma de shell (ALE-269). Ela era a única tela do piloto ainda
  // desenhada com EMOJI, e emoji não obedece `currentColor`: o 🐉 e o 📝 saíam
  // coloridos sobre a paleta de ouro e ferro, não escureciam quando o botão
  // ficava desabilitado e não acendiam em dourado na vez. Os nomes abaixo são
  // os MESMOS que a `session-gm-view` importa do lucide, para o mestre que
  // aprendeu um lado reconhecer o outro.
  'NotebookPen',
  'PanelLeftOpen',
  'X',
  'Settings2',
  'Hourglass',
  'Moon',
  'Theater',
  'Library',
  'Minus',
  'Pencil',
  'Eye',
  'EyeOff',
  'Eraser',
  'MapPin',
  'Square',
  'SquareCheck',
  // A RÉGUA e o GABARITO (ALE-269, superfície 8), e o `MousePointer2` que passou
  // a nomear o modo de MOVER: com ferramentas na mão do jogador, "nenhuma
  // ferramenta" deixou de ser um estado que se adivinha e virou um botão.
  'Ruler',
  'Radar',
  'MousePointer2',
  // O CENTRALIZADOR (ALE-269, item 9), o mesmo ícone que a `ViewControls` da SPA.
  'Crosshair',
  // DESFAZER A PARADA (ALE-269, item 10), o mesmo da `MoveBar` da SPA.
  'CornerUpLeft',
]

const indice = readFileSync(resolve(LUCIDE, 'lucide-solid.mjs'), 'utf8')

/** Acha o arquivo do ícone seguindo o apelido exportado pelo pacote. */
function arquivoDe(nome) {
  const linha = indice
    .split('\n')
    .find((l) => l.includes(`as ${nome},`) && l.includes("from './icons/"))
  if (!linha) throw new Error(`lucide não exporta ${nome}`)
  return linha.match(/from '\.\/(icons\/[^']+)'/)[1]
}

/** Extrai o `iconNode` do módulo. É um literal JS válido, avaliado como tal. */
function nosDe(arquivo) {
  const fonte = readFileSync(resolve(LUCIDE, arquivo), 'utf8')
  const bruto = fonte.match(/const iconNode = (\[[\s\S]*?\]);\n/)
  if (!bruto) throw new Error(`sem iconNode em ${arquivo}`)
  // biome-ignore lint/security/noGlobalEval: dado do pacote que nós instalamos, lido do disco, nunca de rede.
  return eval(bruto[1])
}

function atributos(props) {
  return Object.entries(props)
    .filter(([k]) => k !== 'key')
    .map(([k, v]) => ` ${k}="${String(v).replaceAll('"', '&quot;')}"`)
    .join('')
}

const casos = QUERIDOS.map((nome) => {
  const arquivo = arquivoDe(nome)
  const filhos = nosDe(arquivo)
    .map(([tag, props]) => `\t\t\t\t<${tag}${atributos(props)}></${tag}>`)
    .join('\n')
  return `\t\t\tcase "${nome}":\n${filhos}\n\t\t\t\t// lucide: ${arquivo}`
}).join('\n')

const conteudo = `package api

// GERADO por \`node scripts/gen-icones-templ.mjs\` — não edite à mão.
//
// Os SVGs vêm do lucide INSTALADO, não transcritos: copiar path à mão é a classe
// de erro que nenhum teste pega, porque um path errado é um path válido. O
// gerador segue o APELIDO do pacote (a SPA escreve \`Users2\`, o lucide 1.x
// guarda em \`users-round.mjs\`), então as duas portas do app desenham o mesmo
// desenho mesmo quando o pacote renomeia alguma coisa.
//
// Ícone novo: acrescente o nome em \`QUERIDOS\` e rode o gerador.

// icone desenha um ícone do lucide. \`nome\` é o nome que a SPA usa.
//
// Sempre \`aria-hidden\`: nos doze usos deste app o ícone acompanha um rótulo em
// texto, e um nome acessível a mais faria o leitor de tela dizer a mesma coisa
// duas vezes. Ícone que precisar de nome próprio pede um componente próprio.
templ icone(nome, classe string) {
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width="24"
		height="24"
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		aria-hidden="true"
		class={ classe }
	>
		switch nome {
${casos}
		}
	</svg>
}
`

writeFileSync(SAIDA, conteudo)
console.log(`${QUERIDOS.length} ícones → ${SAIDA}`)
