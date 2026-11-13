package convention

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// O FLUXO LONGO ABRE PELO `ui.LiveStream`, OU DESISTE NO MEIO DA SESSÃO (ALE-636).
//
// O Datastar 1.0.5 conta as novas tentativas de um `@get` pela vida inteira da página: a espera dobra a
// cada queda até 30 s, e depois da décima o fluxo não volta mais. Medido com treze reinícios do
// servidor, como num deploy: 0,6 s, 1,6 s, 3,6 s… 29,6 s, e depois nada. Na 1.0.4, todo retorno levava
// ~0,6 s. O `ui.LiveStream` passa ao `@get` as opções que tiram o teto e o recuo.
//
// O guarda lê o `Init:` da página, que é por onde os fluxos longos abrem (a Mesa, a ficha ao vivo e a
// tela de exibição), e não o handler: o defeito mora no TEXTO do `@get`, e nenhum teste de Go vê o
// runtime reconectar. Quem prova o comportamento é o `live-stream-reconnects.spec.ts`, no e2e.
func TestEveryLiveStreamOpensThroughLiveStream(t *testing.T) {
	root := filepath.Join("..", "..")
	output, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached", "engine-go/serve/*.go").Output()
	if err != nil {
		t.Fatalf("git ls-files em %s: %v", root, err)
	}

	initsRead := 0
	var offenders []string
	for _, relative := range strings.Split(strings.TrimRight(string(output), "\x00"), "\x00") {
		if relative == "" || strings.HasSuffix(relative, "_templ.go") || strings.HasSuffix(relative, "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("ler %s: %v", relative, err)
		}
		for n, line := range strings.Split(string(body), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || !strings.HasPrefix(trimmed, "Init:") {
				continue
			}
			initsRead++
			if strings.Contains(trimmed, "@get(") {
				offenders = append(offenders, relative+":"+strconv.Itoa(n+1)+" — "+trimmed)
			}
		}
	}

	// O DENOMINADOR: são três fluxos longos hoje, e um guarda que não leu nenhum `Init:` não provou nada.
	if initsRead < 3 {
		t.Fatalf("o guarda leu %d `Init:` de página e esperava ao menos os três fluxos longos — a varredura "+
			"encolheu (arquivo movido, campo renomeado?)", initsRead)
	}
	for _, offender := range offenders {
		t.Errorf("%s\n    abre o fluxo com `@get` cru: use `ui.LiveStream(endereço)`, senão o Datastar 1.0.5 "+
			"desiste dele depois de dez quedas", offender)
	}
}
