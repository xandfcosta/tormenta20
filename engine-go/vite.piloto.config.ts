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
  // `tokens-solid.tsx` traz JSX, e sem ele o esbuild do Vite não o compila.
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
      // CINCO entradas e não cinco configs: o `grimorio.js` só é pedido pela
      // folha de especificação, e carregá-lo em toda cena seria pôr canvas e
      // medição de contraste no caminho de quem só quer jogar.
      entry: {
        scene: resolve(import.meta.dirname, 'api/piloto/src/scene.ts'),
        grimorio: resolve(import.meta.dirname, 'api/piloto/src/grimorio.ts'),
        'tokens-solid': resolve(import.meta.dirname, 'api/piloto/src/tokens-solid.tsx'),
        // O leitor do livro (ALE-264) é a terceira entrada com o mesmo motivo
        // das duas anteriores: ele carrega o pdf.js, que são 448 KB, e só a
        // cena `/livro/ler` o pede. Pô-lo no `scene.js` seria mandar um
        // visualizador de PDF para quem abriu a ficha de um personagem.
        reader: resolve(import.meta.dirname, 'api/piloto/src/reader.ts'),
        // A MESA (ALE-174) é a quarta entrada com o mesmo motivo das outras:
        // ela instala um observador de mutação sobre o tabuleiro, e pô-la no
        // `scene.js` seria pendurar isso na ficha, na porta e no grimório —
        // páginas que não têm peça nenhuma para animar.
        table: resolve(import.meta.dirname, 'api/piloto/src/table.ts'),
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
        // estável também é o que permite ao `tokens-solid.js` importá-lo por
        // caminho previsível.
        chunkFileNames: '[name].js',
      },
    },
  },
})
