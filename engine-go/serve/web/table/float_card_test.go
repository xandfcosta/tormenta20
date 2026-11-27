package table

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// A FAIXA E O CARTÃO TÊM DONO (ver GLOSSARY, **cartão flutuante**): o `strikeBand` escreve a faixa e o `floatCard`
// escreve o cartão, com a alça e a seta. Escritos à mão, um cartão novo nascia sem a seta de recolher e a faixa
// nova sem o `role`, e nenhum teste via — a estrutura morava repetida em cada tela.
//
// O guarda lê os `.templ` do ÍNDICE do git, e não o disco, pela regra do `git add` antes do `go test`. O padrão é o
// da tag que só os dois componentes escrevem: a classe `ds-strike` sozinha e o atributo `data-float` sem sufixo
// (o `data-float-floor` da linha do turno é outra coisa).
func TestEveryBandAndCardIsDrawnByItsComponent(t *testing.T) {
	out, err := exec.Command("git", "ls-files", "-z", "--cached", "--", "*.templ").Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}
	handWritten := regexp.MustCompile(`class="ds-strike"|\bdata-float(\s|>|=)`)
	measured := 0
	for _, file := range strings.Split(strings.TrimRight(string(out), "\x00"), "\x00") {
		if file == "" || file == "float_card.templ" {
			continue
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("ler %s: %v", file, err)
		}
		measured++
		for i, line := range strings.Split(string(raw), "\n") {
			if handWritten.MatchString(line) {
				t.Errorf("%s:%d escreve a faixa ou o cartão à mão: use o `strikeBand` ou o `floatCard` (float_card.templ), "+
					"que são os donos da estrutura\n\t%s", file, i+1, strings.TrimSpace(line))
			}
		}
	}
	// O DENOMINADOR: uma leitura que não achasse arquivo nenhum passaria verde sem ter medido nada.
	if measured < 20 {
		t.Fatalf("li %d arquivos .templ nesta pasta; esperava dezenas — a leitura do índice falhou", measured)
	}
}
