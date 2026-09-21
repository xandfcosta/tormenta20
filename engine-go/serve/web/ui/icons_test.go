package ui

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TODO ícone pedido por uma cena EXISTE no gerado.
//
// O `switch` do `icone` não tem `default`, então um nome que ninguém gerou rende
// um `<svg>` VAZIO — sem erro de compilação, sem aviso, sem nada na tela além de
// um buraco do tamanho do ícone. O guarda é grep, e é barato: ele lê os mesmos
// arquivos que o gerador escreve.
//
// LIMITE DELE, e vale saber antes de confiar: ele só enxerga o nome ESCRITO no
// template, `@icone("Skull")`. Uma cena que passa o nome por variável —
// `@icone(f.Icone)`, que é o que a trilha do mestre faz para percorrer uma
// tabela — escapa inteira. Esse é o regime de ENUMERAÇÃO de que fala o
// CLAUDE.md: o guarda cobre por amostragem enquanto as chamadas forem
// literais, e no dia em que uma vira indireta ela precisa trazer o próprio
// guarda. O `TestTheGmTrailIconsExist`, logo abaixo, é esse guarda
// para a primeira indireta que apareceu.
func TestEveryRequestedIconExistsInTheGeneratedFile(t *testing.T) {
	generated, err := os.ReadFile("icons.templ")
	if err != nil {
		t.Fatalf("ler o gerado: %v", err)
	}
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	requested := regexp.MustCompile(`@icone\("([A-Za-z0-9]+)"`)
	for _, f := range files {
		name := f.Name()
		if !strings.HasSuffix(name, ".templ") || name == "icons.templ" {
			continue
		}
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("ler %s: %v", name, err)
		}
		for _, m := range requested.FindAllStringSubmatch(string(content), -1) {
			if !strings.Contains(string(generated), `case "`+m[1]+`":`) {
				t.Errorf("%s pede o ícone %q e o gerado não o tem — ele sai como SVG vazio, sem erro. "+
					"Acrescente em scripts/gen-icons-templ.mjs e rode o gerador.", name, m[1])
			}
		}
	}
}
