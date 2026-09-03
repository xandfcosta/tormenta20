package convention

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestNoFocusAsksTheServerWithoutAKeyboardGuard: pedido disparado por FOCO é
// afordância de TECLADO, e por isso ele pede `:focus-visible`.
//
// A razão está no `engine-go/CLAUDE.md`, em "Dois pedidos de UM gesto: quem
// CHEGA por último manda": o clique do mouse também FOCA, então um nó que pede
// ao servidor nos dois manda dois pedidos por gesto, e a ordem de CHEGADA não é
// a de saída. No bestiário isso fechava a ficha que o clique tinha acabado de
// abrir — sem erro, sem log, e só na máquina carregada, então o CI pegou duas
// vezes o que a bancada nunca reproduziu.
//
// # Por que ele mora AQUI e não no pacote da cena
//
// Ele nasceu em `api/master_dialog_test.go` e varria `*.templ` do
// PRÓPRIO diretório — o que estava certo enquanto todas as cenas eram um pacote
// só. Na ALE-278 o bestiário virou `web/master`, e mudar o guarda de casa junto
// teria deixado ele varrendo QUATRO arquivos e ignorando os outros três, com o
// terminal dizendo verde. Medido no dia da mudança: dos quatro `.templ` com
// `data-on:focus`, um é do `web/master` e três são do `api` (campanhas, o
// tabuleiro da mesa e personagens).
//
// É a forma que o CLAUDE.md da raiz descreve em "Um guarda só mede o que ele
// VISITA", e o conserto é o mesmo que o guarda de tinta levou: **caminhada, não
// lista** — nem lista de cenas, nem, como aqui, o diretório em que o arquivo por
// acaso mora. O `convention/` existe exatamente para a regra que é de todos os
// pacotes e de nenhum.
func TestNoFocusAsksTheServerWithoutAKeyboardGuard(t *testing.T) {
	// Um `data-on:focus…` inteiro, com os modificadores e o valor: o Datastar
	// escreve `data-on:focus__throttle.100ms.leading={ … }`, e é o VALOR que diz
	// se há pedido e se há guarda.
	oFoco := regexp.MustCompile(`data-on:focus[a-zA-Z0-9_.]*=(\{[^}]*\}|"[^"]*")`)

	visitados, achados := 0, 0
	err := filepath.WalkDir("..", func(caminho string, entrada fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entrada.IsDir() {
			if entrada.Name() == "node_modules" || entrada.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(caminho, ".templ") {
			return nil
		}
		visitados++
		corpo, err := os.ReadFile(caminho)
		if err != nil {
			return err
		}
		for _, gesto := range oFoco.FindAllString(string(corpo), -1) {
			if !strings.Contains(gesto, "@get(") && !strings.Contains(gesto, "@post(") {
				continue // só mexe em sinal local: não pede nada, não corre com ninguém
			}
			achados++
			if !strings.Contains(gesto, ":focus-visible") {
				t.Errorf("%s: um foco pede ao servidor sem guarda de teclado — o clique do mouse "+
					"foca também, e o pedido dele chega DEPOIS do pedido do clique numa máquina "+
					"carregada, desfazendo o que o clique fez. Embrulhe em "+
					"`el.matches(':focus-visible') && (…)`: %s", caminho, gesto)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}

	// O DENOMINADOR, em duas metades, porque "nada reprovou" e "não mediu" são a
	// mesma cor no terminal.
	//
	// A primeira metade é nova e é a lição desta issue: o guarda antigo só
	// exigia `achados > 0`, então ele teria passado verde depois da mudança de
	// pacote — havia um foco com `@get` no diretório novo, e ele bastava. O piso
	// de arquivos VISITADOS é o que denuncia a caminhada que encolheu.
	if visitados < 40 {
		t.Fatalf("a caminhada viu só %d arquivos `.templ`, e o repositório tem dezenas: "+
			"a raiz da varredura é o primeiro suspeito", visitados)
	}
	if achados == 0 {
		t.Fatal("nenhum `data-on:focus…` com `@get`/`@post` foi encontrado na fonte: " +
			"o guarda não mediu nada, e o casamento do padrão é o primeiro suspeito")
	}
}
