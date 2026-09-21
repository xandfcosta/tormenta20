package api

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A CLASSE DE ESCOPO NÃO É TINTA, E POR ISSO ELA ESCAPA.
//
// `scene-grimorio` é o escopo em que os tokens da paleta existem: o
// `@custom-variant dark` do `index.css` é `&:is(.dark *, .scene-grimorio, …)`.
// Sem essa classe no elemento de fora, NENHUM token resolve — e o app inteiro
// sai sem cor, sem contraste e com o realce de foco de outra receita.
//
// Ela escapou do `TestEveryHouseTintExistsInTheStylesheet` porque aquele mede
// TINTA (`text-grimorio-gold`, `bg-grimorio-panel`), e a classe de escopo não é
// uma tinta: é a CONDIÇÃO para as tintas valerem. Perdê-la deixa o guarda de
// tinta, o `go build` e o `templ generate` todos VERDES, e só o e2e denuncia.
//
// O gesto que a perde é banal: uma varredura de identificadores para inglês —
// o HÍFEN é fronteira de palavra, então `scene-grimorio` vira `scene-grimoire`
// junto com o resto.
func TestEveryScopeClassExistsInTheStylesheet(t *testing.T) {
	sheet := compiledStylesheet(t)

	// Só o que está DENTRO de `class=`, e a primeira versão deste guarda errou
	// justamente aí: procurando `scene-…` no arquivo inteiro, ela reprovou
	// `scene-title`, `scene-shell` e `scene-content`, que são valores de
	// `data-slot` — ganchos que o `scene.ts` e o e2e consultam, e que não têm por
	// que existir na folha. Guarda que reprova o que está certo é guarda que
	// alguém desliga.
	classAttribute := regexp.MustCompile(`class="([^"]*)"`)
	scope := regexp.MustCompile(`^scene-[a-z0-9-]+$`)
	used := map[string][]string{}
	for _, path := range houseSources(t) {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ler %s: %v", path, err)
		}
		for _, found := range classAttribute.FindAllStringSubmatch(semOsComentarios(string(source)), -1) {
			for _, class := range strings.Fields(found[1]) {
				if scope.MatchString(class) {
					used[class] = append(used[class], filepath.Base(path))
				}
			}
		}
	}

	// O DENOMINADOR. Hoje há UMA classe de escopo, e é justamente por ser uma só
	// que perdê-la derruba tudo: um piso de zero deixaria "o padrão parou de
	// casar" com a mesma cara de "está tudo certo".
	if len(used) == 0 {
		t.Fatal("nenhuma classe de escopo achada na fonte — o guarda ficou cego")
	}

	for class, where := range used {
		if aFolhaConhece(sheet, class) {
			continue
		}
		t.Errorf("a classe de escopo %q não existe na folha (usada em %s): sem ela NENHUM token da paleta resolve",
			class, strings.Join(where, ", "))
	}
}
