import { resolve } from 'node:path'
import { defineConfig } from 'vite'

/**
 * O bundle do módulo das cenas em Datastar (ALE-231).
 *
 * Ele nasceu em `frontend/` porque o FONTE era o da SPA — o driver de teclado, o
 * som e as peças eram compartilhados, e duplicá-los seria a armadilha de
 * divergência. Com a SPA saindo (ALE-272, fatia 10c) não há mais o que
 * compartilhar: as fontes vieram para `api/assets/src` e esta config veio junto.
 *
 * O produto continua indo para DENTRO do `engine-go` — o `go:embed` o embute no
 * binário — e continua sem hash no nome: quem o referencia é um `<script>`
 * escrito à mão num template Go.
 */
export default defineConfig({
  // Sem `public/`: não há mais pasta pública neste pacote, e as fontes que a
  // folha pede em `/fonts/…` já vivem embutidas em `api/assets/static/fonts`
  // (ver `FontsHandler`).
  publicDir: false,
  resolve: { alias: { '@': resolve(import.meta.dirname, 'api/assets/src') } },
  build: {
    outDir: resolve(import.meta.dirname, 'api/assets/static'),
    // A pasta guarda o `datastar.js` e a folha compilada; limpá-la apagaria os
    // dois e o `go:embed` pararia de compilar.
    emptyOutDir: false,
    target: 'es2022',
    lib: {
      // QUATRO entradas e não quatro configs. Cada uma existe porque carrega
      // peso que as outras cenas não devem pagar — o `grimorio` põe canvas e
      // medição de contraste, o `reader` põe 448 KB de pdf.js, e a `table`
      // instala observador de mutação sobre o tabuleiro.
      entry: {
        scene: resolve(import.meta.dirname, 'api/assets/src/scene.ts'),
        grimorio: resolve(import.meta.dirname, 'api/assets/src/grimorio.ts'),
        // O leitor do livro (ALE-264) é a terceira entrada com o mesmo motivo
        // das duas anteriores: ele carrega o pdf.js, que são 448 KB, e só a
        // cena `/livro/ler` o pede. Pô-lo no `scene.js` seria mandar um
        // visualizador de PDF para quem abriu a ficha de um personagem.
        reader: resolve(import.meta.dirname, 'api/assets/src/reader.ts'),
        // A MESA (ALE-174) é a quarta entrada com o mesmo motivo das outras:
        // ela instala um observador de mutação sobre o tabuleiro, e pô-la no
        // `scene.js` seria pendurar isso na ficha, na porta e no grimório —
        // páginas que não têm peça nenhuma para animar.
        table: resolve(import.meta.dirname, 'api/assets/src/table.ts'),
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
        // estável também é o que mantém previsível o caminho de qualquer
        // pedaço que duas entradas venham a dividir.
        chunkFileNames: '[name].js',
      },
    },
  },
})
