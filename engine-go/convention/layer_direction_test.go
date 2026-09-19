package convention

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// A SETA DAS CAMADAS SÓ APONTA PARA BAIXO.
//
// `serve/` → `app/` → `domain/` + `infra/`. Este guarda prende as duas pontas
// que, invertidas, desfazem a divisão inteira — e ele varre a ÁRVORE, não uma
// lista de pacotes: o `app/` tem um pacote hoje e o guarda vale para o que
// nascer amanhã, que é a diferença entre uma convenção varrida e um parágrafo.
//
// # O que cada metade impede
//
//   - **`app/` não conhece `serve/`.** Um caso de uso que importasse a
//     apresentação receberia `*http.Request` e devolveria 403 — e aí ele não
//     pode mais ser chamado de outro transporte, que é a única coisa que uma
//     camada de aplicação compra. É por isso que as recusas do `app/` são
//     tipadas (`ErrForbidden`) e o número mora em quem responde.
//   - **`domain/` não conhece `app/` nem `serve/`.** A regra é a única coisa que
//     nada deve puxar atrás de si.
//
// # O que ele deliberadamente NÃO impede
//
// `app/` importando `infra/` — o caso de uso É o lugar onde o banco encontra a
// regra, e proibi-lo empurraria a orquestração de volta para o `serve/`, que é
// de onde ela está saindo.
func TestNoLayerImportsUpwards(t *testing.T) {
	proibido := map[string][]string{
		"app":    {"t20engine/serve/"},
		"domain": {"t20engine/serve/", "t20engine/app/"},
	}

	raiz, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("achar a raiz: %v", err)
	}

	conjunto := token.NewFileSet()
	medidos := map[string]int{}
	for grupo, recusados := range proibido {
		dir := filepath.Join(raiz, grupo)
		err := filepath.WalkDir(dir, func(caminho string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(caminho, ".go") {
				return err
			}
			arquivo, err := parser.ParseFile(conjunto, caminho, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			medidos[grupo]++
			for _, imp := range arquivo.Imports {
				alvo := strings.Trim(imp.Path.Value, `"`)
				for _, r := range recusados {
					if strings.HasPrefix(alvo, r) {
						rel, _ := filepath.Rel(raiz, caminho)
						t.Errorf("%s importa %q — a seta das camadas só aponta para baixo.\n"+
							"Se %s/ precisa disso, o que falta é o valor atravessar para cá:\n"+
							"receba-o por parâmetro, ou declare aqui o tipo que você precisa.",
							rel, alvo, grupo)
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("caminhar %s: %v", grupo, err)
		}
	}

	// O DENOMINADOR, por grupo: uma lista de reprovados vazia e um diretório não
	// lido se parecem no terminal. O piso do `app/` é baixo de propósito — ele
	// existe para acusar a pasta que sumiu, não para contar arquivos.
	if medidos["app"] < 3 {
		t.Fatalf("o guarda leu só %d arquivos em `app/` — ele está medindo o diretório errado", medidos["app"])
	}
	if medidos["domain"] < 50 {
		t.Fatalf("o guarda leu só %d arquivos em `domain/` — ele está medindo o diretório errado", medidos["domain"])
	}
}
