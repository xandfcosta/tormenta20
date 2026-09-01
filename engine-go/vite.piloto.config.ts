import { resolve } from 'node:path'
import { defineConfig } from 'vite'
import solid from 'vite-plugin-solid'

/**
 * O bundle do módulo do piloto Datastar (ALE-231).
 *
 * Ele nasceu em `frontend/` porque o FONTE era o da SPA — o driver de teclado, o
 * som e as peças eram compartilhados, e duplicá-los seria a armadilha de
 * divergência. Com a SPA saindo (ALE-272, fatia 10c) não há mais o que
 * compartilhar: as fontes vieram para `api/piloto/src` e esta config veio junto.
 *
 * O produto continua indo para DENTRO do `engine-go` — o `go:embed` o embute no
 * binário — e continua sem hash no nome: quem o referencia é um `<script>`
 * escrito à mão num template Go.
 */
export default defineConfig({
  // O plugin do Solid é necessário desde a ilha das peças (ALE-251): o
  // `pecas-solid.tsx` traz JSX, e sem ele o esbuild do Vite não o compila.
  plugins: [solid()],
  // Sem `public/`: não há mais pasta pública neste pacote, e as fontes que a
  // folha pede em `/fonts/…` já vivem embutidas em `api/piloto/static/fonts`
  // (ver `FontesDoPiloto`).
  publicDir: false,
  resolve: { alias: { '@': resolve(import.meta.dirname, 'api/piloto/src') } },
  build: {
    outDir: resolve(import.meta.dirname, 'api/piloto/static'),
    // A pasta guarda o `datastar.js` e a folha compilada; limpá-la apagaria os
    // dois e o `go:embed` pararia de compilar.
    emptyOutDir: false,
    target: 'es2022',
    lib: {
      // DUAS entradas e não duas configs: o `grimorio.js` só é pedido pela
      // folha de especificação, e carregá-lo em toda cena seria pôr canvas e
      // medição de contraste no caminho de quem só quer jogar.
      entry: {
        cena: resolve(import.meta.dirname, 'api/piloto/src/cena.ts'),
        grimorio: resolve(import.meta.dirname, 'api/piloto/src/grimorio.ts'),
        'pecas-solid': resolve(import.meta.dirname, 'api/piloto/src/pecas-solid.tsx'),
        // O leitor do livro (ALE-264) é a terceira entrada com o mesmo motivo
        // das duas anteriores: ele carrega o pdf.js, que são 448 KB, e só a
        // cena `/piloto/livro/ler` o pede. Pô-lo no `cena.js` seria mandar um
        // visualizador de PDF para quem abriu a ficha de um personagem.
        leitor: resolve(import.meta.dirname, 'api/piloto/src/leitor.ts'),
      },
      formats: ['es'],
      fileName: (_formato, nome) => `${nome}.js`,
    },
    rollupOptions: {
      output: {
        // Sem HASH no nome do pedaço compartilhado, e isto não é preferência:
        // o produto é embutido por `go:embed` e versionado, então um hash novo
        // a cada build encheria o repositório de sobras e faria o guarda de
        // "regenerar e comparar" do CI acusar mudança em toda corrida. O nome
        // estável também é o que permite ao `pecas-solid.js` importá-lo por
        // caminho previsível.
        chunkFileNames: '[name].js',
      },
    },
  },
})
