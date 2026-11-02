package ui

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// iconCallShapes são as formas de pedir um ícone num `.templ`, com o nome no primeiro grupo.
var iconCallShapes = []*regexp.Regexp{
	regexp.MustCompile(`@(?:ui\.Icon|icone)\("([A-Za-z0-9]+)"`),
	regexp.MustCompile(`@(?:ds\.)?IconButton\([^,"]+,\s*"([A-Za-z0-9]+)"`),
	regexp.MustCompile(`@(?:ds\.)?(?:SideMenuLink|BottomNavItem)\([^,]+,\s*"([A-Za-z0-9]+)"`),
	regexp.MustCompile(`@readerButton\("[^"]*",\s*"([A-Za-z0-9]+)"`),
	regexp.MustCompile(`@menuItem\("([A-Za-z0-9]+)"`),
	regexp.MustCompile(`@tempestadeLevelStep\([^,]+,\s*[^,]+,\s*[^,]+,\s*"([A-Za-z0-9]+)"`),
	regexp.MustCompile(`@gmToolButton\("[^"]*",\s*"([A-Za-z0-9]+)"`),
	regexp.MustCompile(`@quantityStep\([^,]+,\s*[^,]+,\s*"([A-Za-z0-9]+)"`),
}

// iconTableShapes é o nome escrito numa tabela em Go, que chega à cena por variável.
var iconTableShapes = []*regexp.Regexp{regexp.MustCompile(`\bIcon:\s*"([A-Za-z0-9]+)"`)}

// TODO ícone pedido por uma cena EXISTE no gerado.
//
// O `switch` do `icone` não tem `default`, então um nome que ninguém gerou rende
// um `<svg>` VAZIO — sem erro de compilação, sem aviso, sem nada na tela além de
// um buraco do tamanho do ícone. O guarda é grep, e é barato: ele lê os mesmos
// arquivos que o gerador escreve.
//
// ELE TINHA DUAS FISSURAS, e as duas deixaram passar ícone quebrado por meses
// (ALE-420):
//
//   - lia os `.templ` do PRÓPRIO diretório, e as cenas moram em `serve/web/*`.
//     O `board.templ` nunca foi visitado;
//   - casava `@icone("X")`, e as cenas escrevem `@ui.Icon("X")` — o nome
//     exportado. Nem no próprio diretório ele teria pegado.
//
// Medido ao alargar: `ChevronLeft` e `PackagePlus` saíam como SVG vazio na
// tela, e nada acusava. Hoje ele varre `serve/` inteiro e casa as duas grafias,
// com DENOMINADOR — "nada reprovou" afirma junto quantos olhou, senão "verde" e
// "não mediu" são a mesma cor.
//
// A TERCEIRA FISSURA era a mesma da segunda: quem desenha o ícone por dentro recebe o nome por parâmetro, e
// o `@ui.Icon(icon, …)` do `ds.IconButton` não diz nome nenhum. O nome está em quem CHAMA, e a lista
// `iconCallShapes` é a de cada peça assim, e `iconTableShapes` a do nome escrito numa tabela em Go. Peça nova
// com parâmetro de ícone entra na lista.
//
// LIMITE DELE, e vale saber antes de confiar: ele só enxerga o nome ESCRITO no
// template. Uma cena que passa o nome por variável — `@icone(f.Icone)`, que é o
// que a trilha do mestre faz para percorrer uma tabela — escapa inteira. Esse é o regime de ENUMERAÇÃO de que fala o
// CLAUDE.md: o guarda cobre por amostragem enquanto as chamadas forem
// literais, e no dia em que uma vira indireta ela precisa trazer o próprio
// guarda. O `TestTheGmTrailIconsExist`, logo abaixo, é esse guarda
// para a primeira indireta que apareceu.
func TestEveryRequestedIconExistsInTheGeneratedFile(t *testing.T) {
	generated, err := os.ReadFile("icons.templ")
	if err != nil {
		t.Fatalf("ler o gerado: %v", err)
	}
	cenas, pedidos := 0, 0
	raiz := filepath.Join("..", "..")
	err = filepath.WalkDir(raiz, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		shapes := iconCallShapes
		switch {
		case strings.HasSuffix(d.Name(), ".templ") && d.Name() != "icons.templ":
			cenas++
		case strings.HasSuffix(d.Name(), ".go") && !strings.HasSuffix(d.Name(), "_test.go") &&
			!strings.HasSuffix(d.Name(), "_templ.go"):
			shapes = iconTableShapes
		default:
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, shape := range shapes {
			for _, m := range shape.FindAllStringSubmatch(string(content), -1) {
				pedidos++
				if !strings.Contains(string(generated), `case "`+m[1]+`":`) {
					t.Errorf("%s pede o ícone %q e o gerado não o tem — ele sai como SVG "+
						"VAZIO, sem erro de compilação e sem aviso: um buraco do tamanho do "+
						"ícone.\nAcrescente em scripts/gen-icons-templ.mjs e rode o gerador.",
						path, m[1])
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("varrer as cenas: %v", err)
	}
	// O DENOMINADOR: um `WalkDir` que não achasse `.templ` nenhum passaria verde.
	if cenas < 20 || pedidos < 50 {
		t.Fatalf("só %d cenas e %d ícones pedidos — o guarda ficou cego", cenas, pedidos)
	}
	t.Logf("cenas varridas: %d, ícones pedidos: %d", cenas, pedidos)
}
