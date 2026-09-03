package characters

import (
	"strings"
	"t20engine/book"
	"testing"
)

// Os guardas da cena de PERSONAGENS (ALE-239).
//
// O que se protege é o que o SERVIDOR passou a fazer e a SPA pedia por
// requisição: a Defesa saindo da mesma `ComputeSheetV2` da ficha, e os textos
// de raça saindo do catálogo embutido. Mais a gramática do cursor, que é o que
// a ALE-98 estabeleceu e que um porte distraído quebra sem perceber.

// Este é o dividendo da ALE-107 aparecendo: os textos vêm do catálogo EMBUTIDO.
// O guarda é sobre o texto existir, não sobre qual é — o conteúdo é dado
// transcrito e quem o valida é o schema, não um `expect` por verbete.
func TestTheDossierCarriesTheRaceTextsFromTheEmbeddedCatalog(t *testing.T) {
	if len(book.RaceAbilities("Humano", 8)) == 0 {
		t.Fatal("o catálogo embutido não devolveu habilidade nenhuma para Humano")
	}
	for _, h := range book.RaceAbilities("Humano", 8) {
		if h.Name == "" || h.Description == "" {
			t.Errorf("habilidade %q sem nome ou sem descrição — o dossiê ficaria com linha vazia", h.ID)
		}
	}
}

// Raça que não está no catálogo não derruba nada: o herói abre sem as linhas de
// sabor. Um personagem antigo com raça renomeada é caso normal, não erro.
func TestAnUnknownRaceDoesNotBringTheDossierDown(t *testing.T) {
	if got := book.RaceAbilities("Não Existe", 8); got != nil {
		t.Errorf("raça desconhecida devolveu %v", got)
	}
}

func TestTheDossierRespectsTheLimit(t *testing.T) {
	if got := len(book.RaceAbilities("Humano", 1)); got != 1 {
		t.Errorf("limite 1 devolveu %d", got)
	}
}

// ── a gramática do cursor ────────────────────────────────────────────────────

// corpoDoBotao devolve o conteúdo visível de um botão achado pelo `aria-label`.
//
// O `>` que fecha a tag de abertura é o primeiro do trecho porque o templ
// escapa `>` dentro de valor de atributo — então o corte é seguro.
func corpoDoBotao(t *testing.T, html, rotulo string) string {
	t.Helper()
	i := strings.Index(html, `aria-label="`+rotulo+`"`)
	if i < 0 {
		t.Fatalf("nenhum botão com rótulo %q no HTML", rotulo)
	}
	resto := html[i:]
	abre := strings.Index(resto, ">")
	fecha := strings.Index(resto, "</button>")
	if abre < 0 || fecha < 0 || abre > fecha {
		t.Fatalf("botão %q malformado no HTML", rotulo)
	}
	return resto[abre+1 : fecha]
}
