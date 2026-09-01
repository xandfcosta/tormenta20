package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOsIconesDaTrilhaDoMestreExistem cobre a chamada INDIRETA que o guarda
// acima não vê.
//
// A trilha percorre `ferramentasDoMestre` e passa `f.Icone` para o `@icone`, e
// nenhum desses quatro nomes aparece literalmente num `.templ`. Sem isto, uma
// letra errada em "BookMarked" dá um botão de ferramenta sem desenho — o mesmo
// buraco silencioso, agora numa tabela.
//
// Provado VERMELHO trocando `Skull` por `Skul` na tabela.
func TestOsIconesDaTrilhaDoMestreExistem(t *testing.T) {
	gerado, err := os.ReadFile(filepath.Join("..", "web", "ui", "icons.templ"))
	if err != nil {
		t.Fatalf("ler o gerado: %v", err)
	}
	for _, f := range ferramentasDoMestre {
		if !strings.Contains(string(gerado), `case "`+f.Icone+`":`) {
			t.Errorf("a ferramenta %q pede o ícone %q e o gerado não o tem — sai SVG vazio, sem erro. "+
				"Acrescente em frontend/scripts/gen-icones-templ.mjs e rode o gerador.", f.Slug, f.Icone)
		}
	}
}
