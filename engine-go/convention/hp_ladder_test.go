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

	allowed := map[string]bool{}
	raw, err := os.ReadFile(filepath.Join("testdata", "hp_tints_outside_the_ladder.txt"))
	if err != nil {
		t.Fatalf("ler a lista de permitidos: %v", err)
	}
	for _, row := range strings.Split(string(raw), "\n") {
		row = strings.TrimSpace(row)
		if row == "" || strings.HasPrefix(row, "#") {
			continue
		}
		allowed[row] = true
	}
	// O DENOMINADOR DA LISTA: um arquivo vazio — ou um caminho que o `ReadFile`
	// achou pela metade — faria todo sítio da árvore reprovar de uma vez, com
	// cara de descoberta.
	if len(allowed) < 5 {
		t.Fatalf("a lista de permitidos tem %d entradas, e ela é o denominador", len(allowed))
	}

	output, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached", "*.templ", "*.go").Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}

	// As DUAS grafias da mesma tinta: a classe da paleta (`bg-hp-full`) e o valor
	// arbitrário (`bg-[color:var(--hp-full)]`). Casar só uma faria o guarda
	// passar verde sobre um renome de grafia — e as duas conviviam na árvore em
	// que ele nasceu.
	vitalInk := regexp.MustCompile(`(?:--|-)hp-(?:full|hurt|critical)\b`)

	var failed []string
	read, withPaint := 0, 0
	for _, rel := range strings.Split(strings.TrimRight(string(output), "\x00"), "\x00") {
		if rel == "" || strings.HasSuffix(rel, "_templ.go") || strings.HasSuffix(rel, "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("ler %s: %v", rel, err)
		}
		read++

		// COMENTÁRIO não pinta nada, e uma prosa que EXPLICA a escada citando os
		// nomes dela viraria violação. É a contaminação que a ALE-313 mediu no
		// guarda de payload, evitada de saída.
		var code []string
		for _, row := range strings.Split(string(body), "\n") {
			if s := strings.TrimSpace(row); strings.HasPrefix(s, "//") {
				continue
			}
			code = append(code, row)
		}
		if !vitalInk.MatchString(strings.Join(code, "\n")) {
			continue
		}
		withPaint++
		if !allowed[rel] {
			failed = append(failed, rel)
		}
	}

	// O PISO dos ARQUIVOS LIDOS, e o dos que de fato escrevem tinta: a caminhada
	// pode encolher sem zerar — foi como o guarda do foco quase passou verde
	// medindo metade (ALE-278).
	if read < 200 {
		t.Fatalf("o guarda leu só %d arquivos, e a árvore tem centenas", read)
	}
	if withPaint < len(allowed) {
		t.Fatalf("só %d arquivos escrevem tinta vital, e a lista declara %d: "+
			"a lista envelheceu ou a varredura encolheu", withPaint, len(allowed))
	}

	for _, rel := range failed {
		t.Errorf("%s escreve uma tinta vital à mão.\n"+
			"    A COR de um vital é da escada: `ui.HpFillTone` pinta a faixa e\n"+
			"    `ui.HpInkTone` escreve o número. Se esta tinta NÃO está dizendo\n"+
			"    quanto de vida sobrou, declare o arquivo em\n"+
			"    `convention/testdata/hp_tints_outside_the_ladder.txt`, com a razão.", rel)
	}
}
