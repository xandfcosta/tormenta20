package master

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTheGmTrailIconsExist cobre a chamada INDIRETA que o guarda de ícones do
// `web/ui` não vê.
//
// A trilha percorre `railStops` e passa `f.Icone` para o `@Icon`, e
// nenhum desses quatro nomes aparece literalmente num `.templ`. Sem isto, uma
// letra errada em "BookMarked" dá um botão de ferramenta sem desenho — o mesmo
// buraco silencioso, agora numa tabela.
//
// Provado VERMELHO trocando `Skull` por `Skul` na tabela.
func TestTheGmTrailIconsExist(t *testing.T) {
	gerado, err := os.ReadFile(filepath.Join("..", "ui", "icons.templ"))
	if err != nil {
		t.Fatalf("ler o gerado: %v", err)
	}
	for _, f := range railStops {
		if !strings.Contains(string(gerado), `case "`+f.Icone+`":`) {
			t.Errorf("a ferramenta %q pede o ícone %q e o gerado não o tem — sai SVG vazio, sem erro. "+
				"Acrescente em scripts/gen-icones-templ.mjs e rode o gerador.", f.Slug, f.Icone)
		}
	}
}
