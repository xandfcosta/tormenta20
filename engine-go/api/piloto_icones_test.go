package api

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TODO ícone pedido por uma cena EXISTE no gerado (ALE-255).
//
// O `switch` do `icone` não tem `default`, então um nome que ninguém gerou
// rende um `<svg>` VAZIO — sem erro de compilação, sem aviso, sem nada na tela
// além de um buraco do tamanho do ícone. Foi o que aconteceu com `Trash2` e
// `Flame` nesta fatia: eu os escrevi na cena, o Go compilou, e o botão de
// excluir ficou sem o desenho.
//
// É a mesma família do seeder que aceita id de item inexistente em silêncio
// (ALE-222): a peça nasce incompleta e nada reclama. O guarda é grep, e é
// barato — ele lê os mesmos arquivos que o gerador escreve.
//
// LIMITE DELE, e vale saber antes de confiar: ele só enxerga o nome ESCRITO no
// template, `@icone("Skull")`. Uma cena que passa o nome por variável —
// `@icone(f.Icone)`, que é o que a trilha do mestre faz para percorrer uma
// tabela — escapa inteira. Esse é o regime de ENUMERAÇÃO de que fala o
// CLAUDE.md: o guarda cobre por amostragem enquanto as chamadas forem
// literais, e no dia em que uma vira indireta ela precisa trazer o próprio
// guarda. O `TestOsIconesDaTrilhaDoMestreExistem`, logo abaixo, é esse guarda
// para a primeira indireta que apareceu.
func TestTodoIconePedidoExisteNoGerado(t *testing.T) {
	gerado, err := os.ReadFile("piloto_icones.templ")
	if err != nil {
		t.Fatalf("ler o gerado: %v", err)
	}
	arquivos, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	pedido := regexp.MustCompile(`@icone\("([A-Za-z0-9]+)"`)
	for _, f := range arquivos {
		nome := f.Name()
		if !strings.HasSuffix(nome, ".templ") || nome == "piloto_icones.templ" {
			continue
		}
		conteudo, err := os.ReadFile(nome)
		if err != nil {
			t.Fatalf("ler %s: %v", nome, err)
		}
		for _, m := range pedido.FindAllStringSubmatch(string(conteudo), -1) {
			if !strings.Contains(string(gerado), `case "`+m[1]+`":`) {
				t.Errorf("%s pede o ícone %q e o gerado não o tem — ele sai como SVG vazio, sem erro. "+
					"Acrescente em frontend/scripts/gen-icones-templ.mjs e rode o gerador.", nome, m[1])
			}
		}
	}
}

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
	gerado, err := os.ReadFile("piloto_icones.templ")
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
