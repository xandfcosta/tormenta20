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
	folha := compiledStylesheet(t)

	// Só o que está DENTRO de `class=`, e a primeira versão deste guarda errou
	// justamente aí: procurando `scene-…` no arquivo inteiro, ela reprovou
	// `scene-title`, `scene-shell` e `scene-content`, que são valores de
	// `data-slot` — ganchos que o `scene.ts` e o e2e consultam, e que não têm por
	// que existir na folha. Guarda que reprova o que está certo é guarda que
	// alguém desliga.
	atributoDeClasse := regexp.MustCompile(`class="([^"]*)"`)
	escopo := regexp.MustCompile(`^scene-[a-z0-9-]+$`)
	usadas := map[string][]string{}
	for _, caminho := range houseSources(t) {
		fonte, err := os.ReadFile(caminho)
		if err != nil {
			t.Fatalf("ler %s: %v", caminho, err)
		}
		for _, achado := range atributoDeClasse.FindAllStringSubmatch(semOsComentarios(string(fonte)), -1) {
			for _, classe := range strings.Fields(achado[1]) {
				if escopo.MatchString(classe) {
					usadas[classe] = append(usadas[classe], filepath.Base(caminho))
				}
			}
		}
	}

	// O DENOMINADOR. Hoje há UMA classe de escopo, e é justamente por ser uma só
	// que perdê-la derruba tudo: um piso de zero deixaria "o padrão parou de
	// casar" com a mesma cara de "está tudo certo".
	if len(usadas) == 0 {
		t.Fatal("nenhuma classe de escopo achada na fonte — o guarda ficou cego")
	}

	for classe, onde := range usadas {
		if aFolhaConhece(folha, classe) {
			continue
		}
		t.Errorf("a classe de escopo %q não existe na folha (usada em %s): sem ela NENHUM token da paleta resolve",
			classe, strings.Join(onde, ", "))
	}
}
