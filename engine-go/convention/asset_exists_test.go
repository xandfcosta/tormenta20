package convention

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TODO ESTÁTICO PEDIDO EXISTE NA PASTA (ALE-301).
//
// O `EstaticoDoPiloto` monta o endereço por CONCATENAÇÃO — `"/static/" +
// arquivo + "?v=" + digito` —, então um nome que não existe produz uma URL de
// aparência perfeita e um 404 só na hora do pedido. O modo de falhar é o pior
// desta casa: a ilha de JS não instala, a cena funciona QUASE toda (sem a
// animação, sem o leitor, sem o deslize da peça) e nada no Go acusa. O
// `go:embed piloto/static/*` é curinga e compila do mesmo jeito; o
// `TestEveryStaticAddressOnThePageIsVersioned` mede a VERSÃO no endereço e
// passaria verde sobre um 404.
//
// Ele nasceu ao renomear as quatro entradas do bundle (`cena`→`scene`,
// `leitor`→`reader`, `mesa`→`table`, `pecas-solid`→`tokens-solid`): a corrente
// tem cinco elos — fonte, chave do `vite.piloto.config.ts`, artefato commitado,
// chamada de `Asset` e o `<script src>` da cena — e nada liga um ao outro. A
// suíte de e2e inteira passa verde com um elo faltando, porque só o caso que
// exercita AQUELA ilha percebe.
var assetCall = regexp.MustCompile(`Asset\("([^"]+)"\)`)

func TestEveryAssetAskedForExists(t *testing.T) {
	staticDir := filepath.Join("..", "api", "piloto", "static")
	if _, err := os.Stat(staticDir); err != nil {
		t.Fatalf("a pasta dos estáticos não está em %s: %v", staticDir, err)
	}

	asked := map[string][]string{}
	filesRead := 0
	root := filepath.Join("..", "..")
	err := filepath.WalkDir(root, func(path string, item fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if item.IsDir() {
			switch item.Name() {
			case ".git", "node_modules", "test-results", "playwright-report", "dist", "backups", "data":
				return fs.SkipDir
			}
			return nil
		}
		// O `_templ.go` fica de FORA: ele repete o que o `.templ` já disse, e
		// contá-lo duplicaria cada pedido sem cobrir nada a mais.
		ext := filepath.Ext(path)
		if (ext != ".go" && ext != ".templ") || strings.HasSuffix(path, "_templ.go") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		filesRead++
		relative := strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(path, root)), "/")
		for _, m := range assetCall.FindAllStringSubmatch(string(body), -1) {
			asked[m[1]] = append(asked[m[1]], relative)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("varrer: %v", err)
	}

	// O DENOMINADOR, porque "nenhum pedido quebrado" e "nenhum pedido achado"
	// são a mesma cor no terminal. São seis hoje — a folha, o Datastar, o
	// `scene.js`, o `reader.js`, o `table.js`, o `tokens-solid.js`, o
	// `grimorio.js` e o `pdf.worker.js` —, e o piso é folgado de propósito.
	if filesRead < 300 || len(asked) < 5 {
		t.Fatalf("a varredura leu %d arquivos e achou %d estáticos pedidos — a raiz é o primeiro suspeito",
			filesRead, len(asked))
	}

	var missing []string
	for name, onde := range asked {
		if _, err := os.Stat(filepath.Join(staticDir, name)); err != nil {
			sort.Strings(onde)
			missing = append(missing, name+" — pedido em "+strings.Join(onde, ", "))
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("estático pedido que NÃO existe em api/piloto/static — %d de %d:\n  %s\n"+
			"O endereço sai montado do mesmo jeito e o navegador leva 404: a ilha de JS não "+
			"instala e a cena funciona quase toda, em silêncio. Se o nome mudou, mude os CINCO "+
			"elos — fonte, `vite.piloto.config.ts`, artefato, `Asset(…)` e o `<script src>` — e "+
			"rode `scripts/build-piloto-js.sh` ANTES de tirar o artefato velho.",
			len(missing), len(asked), strings.Join(missing, "\n  "))
	}
	t.Logf("estáticos pedidos: %d, %d sem arquivo, de %d arquivos varridos", len(asked), len(missing), filesRead)
}
