package convention

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A COR de um vital é da ESCADA, e de mais ninguém (ALE-316).
//
// # O defeito que o originou
//
// A escada de PV — crítico até 25%, ferido até 50%, cheio acima — nasceu na SPA
// e foi portada para a Mesa, com guarda de limiares desde a ALE-214. A FICHA e a
// LISTA DE HERÓIS não a portaram: as duas escreviam `--hp-full` fixo, sem ramo
// nenhum. Medido no navegador, um herói a 10/57 desenhava o verde de vida cheia
// com a barra a um quinto da largura.
//
// Nenhum guarda podia pegar. Os casos de vital afirmavam o NÚMERO, que sempre
// esteve certo, e a largura também — o que estava errado era a única coisa que
// ninguém media. O conserto (a escada em `ui.HpFillTone`/`ui.HpInkTone`) não
// impede a próxima tela de escrever a tinta à mão outra vez; quem impede é este.
//
// # Por que uma lista de PERMITIDOS
//
// As três tintas têm um segundo emprego que nada tem com vida: o verde é
// "presente/copiado", e o âmbar e o vermelho são atenção e perigo em contextos
// próprios. Uma lista de PROIBIDOS teria de adivinhar quais sítios são de PV, e
// ela subconta em silêncio — é o achado do `route_params.txt` (ALE-310).
//
// Aqui a pergunta se inverte: todo arquivo que escreve uma tinta vital reprova
// até alguém declará-lo, e declarar é o ato de dizer "esta tinta não responde
// quanto de vida sobrou".
func TestNoSurfacePaintsAVitalTintOutsideTheLadder(t *testing.T) {
	root := filepath.Join("..", "..")

	permitidos := map[string]bool{}
	bruto, err := os.ReadFile(filepath.Join("testdata", "hp_tints_outside_the_ladder.txt"))
	if err != nil {
		t.Fatalf("ler a lista de permitidos: %v", err)
	}
	for _, linha := range strings.Split(string(bruto), "\n") {
		linha = strings.TrimSpace(linha)
		if linha == "" || strings.HasPrefix(linha, "#") {
			continue
		}
		permitidos[linha] = true
	}
	// O DENOMINADOR DA LISTA: um arquivo vazio — ou um caminho que o `ReadFile`
	// achou pela metade — faria todo sítio da árvore reprovar de uma vez, com
	// cara de descoberta.
	if len(permitidos) < 5 {
		t.Fatalf("a lista de permitidos tem %d entradas, e ela é o denominador", len(permitidos))
	}

	saida, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached", "*.templ", "*.go").Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}

	// As DUAS grafias da mesma tinta: a classe da paleta (`bg-hp-full`) e o valor
	// arbitrário (`bg-[color:var(--hp-full)]`). Casar só uma faria o guarda
	// passar verde sobre um renome de grafia — e as duas conviviam na árvore em
	// que ele nasceu.
	tintaVital := regexp.MustCompile(`(?:--|-)hp-(?:full|hurt|critical)\b`)

	var reprovados []string
	lidos, comTinta := 0, 0
	for _, rel := range strings.Split(strings.TrimRight(string(saida), "\x00"), "\x00") {
		if rel == "" || strings.HasSuffix(rel, "_templ.go") || strings.HasSuffix(rel, "_test.go") {
			continue
		}
		corpo, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("ler %s: %v", rel, err)
		}
		lidos++

		// COMENTÁRIO não pinta nada, e uma prosa que EXPLICA a escada citando os
		// nomes dela viraria violação. É a contaminação que a ALE-313 mediu no
		// guarda de payload, evitada de saída.
		var codigo []string
		for _, linha := range strings.Split(string(corpo), "\n") {
			if s := strings.TrimSpace(linha); strings.HasPrefix(s, "//") {
				continue
			}
			codigo = append(codigo, linha)
		}
		if !tintaVital.MatchString(strings.Join(codigo, "\n")) {
			continue
		}
		comTinta++
		if !permitidos[rel] {
			reprovados = append(reprovados, rel)
		}
	}

	// O PISO dos ARQUIVOS LIDOS, e o dos que de fato escrevem tinta: a caminhada
	// pode encolher sem zerar — foi como o guarda do foco quase passou verde
	// medindo metade (ALE-278).
	if lidos < 200 {
		t.Fatalf("o guarda leu só %d arquivos, e a árvore tem centenas", lidos)
	}
	if comTinta < len(permitidos) {
		t.Fatalf("só %d arquivos escrevem tinta vital, e a lista declara %d: "+
			"a lista envelheceu ou a varredura encolheu", comTinta, len(permitidos))
	}

	for _, rel := range reprovados {
		t.Errorf("%s escreve uma tinta vital à mão.\n"+
			"    A COR de um vital é da escada: `ui.HpFillTone` pinta a faixa e\n"+
			"    `ui.HpInkTone` escreve o número. Se esta tinta NÃO está dizendo\n"+
			"    quanto de vida sobrou, declare o arquivo em\n"+
			"    `convention/testdata/hp_tints_outside_the_ladder.txt`, com a razão.", rel)
	}
}
