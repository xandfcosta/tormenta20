/**
 * O app INSTALADO: o manifest, os ícones e as metas que o `index.html` aponta
 * (ALE-118). Ver a linha "app instalado" do GLOSSARIO.md — não é a Tela cheia.
 *
 * Por que isto é teste e não confiança no arquivo estático: a instalação falha
 * em SILÊNCIO. Um `href` de ícone que não existe mais, um `rel="manifest"` que
 * sumiu num passe pelo `index.html`, um ícone abaixo de 512 — em nenhum desses
 * casos há erro na tela, no console do app ou no build. O que acontece é que o
 * Chrome deixa de oferecer instalar, e ninguém descobre até a próxima mesa.
 *
 * Lê os arquivos do disco de propósito: são eles que o Vite copia crus para o
 * `dist`, então é neles que a asserção tem valor.
 */
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

// A raiz do pacote. `import.meta.url` NÃO serve aqui: sob o transform do vitest
// ele não é uma URL `file:`, e o `fileURLToPath` recusa. O `cwd` do processo é
// a raiz da config do vitest, que é este pacote.
const noPacote = (caminho: string): string => resolve(process.cwd(), caminho)

function ler(caminho: string): string {
  return readFileSync(noPacote(caminho), 'utf8')
}

/** O manifest, já parseado — só os campos que esta suíte afirma. */
type Manifest = {
  display: string
  theme_color: string
  icons: { src: string; sizes: string; purpose?: string }[]
}

function manifest(): Manifest {
  return JSON.parse(ler('public/manifest.webmanifest')) as Manifest
}

function cabeca(): Document {
  return new DOMParser().parseFromString(ler('index.html'), 'text/html')
}

/** O arquivo que um `href` absoluto da página aponta dentro do `public/`. */
function existeEmPublic(href: string): boolean {
  return existsSync(noPacote(`public${href}`))
}

/** O maior lado declarado em `sizes` ("512x512" → 512). */
function lado(icone: { sizes: string }): number {
  return Number.parseInt(icone.sizes.split('x')[0], 10)
}

describe('o app instalado', () => {
  it('o index aponta para o manifest e para o ícone do iPhone', () => {
    const doc = cabeca()

    // Sem esta linha o "Adicionar à Tela de Início" do Android devolve um
    // ATALHO — uma aba comum com a barra de endereço de volta.
    expect(doc.querySelector('link[rel="manifest"]')?.getAttribute('href')).toBe('/manifest.webmanifest')
    // O iOS ignora os ícones do manifest; sem este, a Tela de Início ganha um
    // print da página.
    const apple = doc.querySelector('link[rel="apple-touch-icon"]')?.getAttribute('href')
    expect(apple, 'o iPhone precisa do apple-touch-icon').toBeTruthy()
    expect(existeEmPublic(apple as string), `${apple} não existe em public/`).toBe(true)
  })

  it('a cor da barra é a MESMA nos dois lugares que a declaram', () => {
    // A meta vale na aba do navegador e o manifest na janela instalada. São dois
    // arquivos dizendo a mesma decisão, e é assim que elas divergem.
    const meta = cabeca().querySelector('meta[name="theme-color"]')?.getAttribute('content')

    expect(meta).toBe(manifest().theme_color)
  })

  it('abre sem barra de endereço', () => {
    // NÃO fixa "standalone" contra "fullscreen": qual dos dois é decisão do dono
    // e muda numa palavra. O que este teste protege é que o valor continua sendo
    // um que abre em janela própria — `browser` devolveria a aba comum.
    expect(['standalone', 'fullscreen', 'minimal-ui']).toContain(manifest().display)
  })

  it('os ícones que o manifest promete existem, e cobrem 192 e 512', () => {
    const icones = manifest().icons

    for (const icone of icones) {
      expect(existeEmPublic(icone.src), `${icone.src} não existe em public/`).toBe(true)
    }
    // O mínimo que o Chrome exige para OFERECER instalar.
    expect(icones.some((i) => lado(i) >= 192)).toBe(true)
    expect(icones.some((i) => lado(i) >= 512)).toBe(true)
    // Sem um `maskable`, o lançador do Android recorta o desenho dentro de um
    // círculo branco em vez de compor sobre o fundo do grimório.
    expect(icones.some((i) => i.purpose === 'maskable')).toBe(true)
  })
})
