import { resolve } from 'node:path'
import { defineConfig } from 'vite'

/**
 * O bundle do módulo do piloto Datastar (ALE-231).
 *
 * Config separada e não uma entrada a mais no build da SPA: o produto vai para
 * DENTRO do `engine-go` (o `go:embed` o embute no binário) e não para o `dist/`,
 * e ele não pode ganhar hash no nome — quem o referencia é um `<script>` escrito
 * à mão num template Go.
 *
 * O fonte, porém, é o mesmo da SPA. É o que impede o driver de teclado de virar
 * duas cópias.
 */
export default defineConfig({
  // Sem cópia do `public/`: ele traz os 3,7 MB do `t20.wasm` junto, e eles
  // iriam para DENTRO do binário pelo `go:embed`. As fontes que moram lá
  // continuam sendo servidas pelo Vite em dev e pelo `STATIC_DIR` em produção
  // — ver a pendência anotada na ALE-229.
  publicDir: false,
  resolve: { alias: { '@': resolve(import.meta.dirname, 'src') } },
  build: {
    outDir: resolve(import.meta.dirname, '../engine-go/api/piloto/static'),
    // A pasta guarda o `datastar.js` e a folha compilada; limpá-la apagaria os
    // dois e o `go:embed` pararia de compilar.
    emptyOutDir: false,
    target: 'es2022',
    lib: {
      entry: resolve(import.meta.dirname, 'src/piloto/cena.ts'),
      formats: ['es'],
      fileName: () => 'cena.js',
    },
  },
})
