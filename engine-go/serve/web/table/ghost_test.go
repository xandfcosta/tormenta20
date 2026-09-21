package table

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"t20engine/domain/board"
	"t20engine/domain/engine"
)

// Os guardas do FANTASMA e da SETA.
//
// O pedido do dono: ao soltar a peça ela é desenhada onde foi solta, o início do
// movimento fica marcado por uma peça transparente, e a seta liga os dois pontos.
//
// O que se prende aqui é a DIVISA: a peça é DESENHADA no fim
// do caminho e continua GRAVADA na origem. As duas metades precisam de guarda,
// porque cada uma sozinha passa verde sobre o defeito da outra — desenhar sem
// gravar seria a peça andando sem confirmação, e gravar sem desenhar é o defeito
// que o dono relatou.

// onBoardAt põe a peça do jogador numa casa escolhida, e devolve o id dela.
//
// Irmã da `onBoard`, que sempre põe em 0,0: aqui a origem precisa ser um
// número que não se confunda com "não preenchido" — com a peça em 0,0 um
// fantasma desenhado na quina do plano por engano passaria despercebido.
// element acha o primeiro elemento cujo atributo `attribute` contém `excerpt`,
// e devolve os atributos dele.
//
// Um parser de HTML de verdade e não uma expressão regular, e a razão é o guarda
// do fim deste arquivo: as expressões do Datastar carregam `<`, `>` e aspas
// dentro dos valores, e um `<[^>]*>` corta um elemento no meio de um `data-on:`
// sem avisar — a busca devolveria menos e a ausência viraria conclusão.
func element(t *testing.T, screen, attribute, excerpt string) map[string]string {
	t.Helper()
	z := html.NewTokenizer(strings.NewReader(screen))
	for {
		switch z.Next() {
		case html.ErrorToken:
			return nil
		case html.StartTagToken, html.SelfClosingTagToken:
			attrs := map[string]string{}
			for {
				key, value, more := z.TagAttr()
				attrs[string(key)] = string(value)
				if !more {
					break
				}
			}
			if strings.Contains(attrs[attribute], excerpt) {
				return attrs
			}
		}
	}
}

// `Stops` NULO é valor legítimo: o `ProposeMove` deixa o caminho pronto sem
// passar por paradas. Deduzir as dobras do `Path` não é possível — um trecho
// legítimo já dobra sozinho, porque a diagonal vem primeiro —, então a seta vira
// a reta entre o começo e o fim.
func TestTheArrowWithoutStopsJoinsBothEndsOfThePath(t *testing.T) {
	noStops := &board.PendingMove{
		Path: []engine.Square{{}, {X: 1}, {X: 2}, {X: 3}},
	}
	folds := moveFolds(noStops)
	if len(folds) != 2 || folds[0] != (engine.Square{}) || folds[1] != (engine.Square{X: 3}) {
		t.Fatalf("as dobras de um caminho sem paradas saíram %+v", folds)
	}
	// A perna anda 3 e a ponta recua meio quadrado: de 0,5 até 3,0. Sem orçamento
	// (-1) ela sai inteira de dourado — o vermelho tem guarda próprio em
	// `move_drawing_test.go`.
	if wire, _, _ := moveWires(folds, []int{3}, -1); wire != "M 0.5 0.5 L 3 0.5" {
		t.Errorf("a seta reta saiu %q", wire)
	}
	// E com uma dobra só não há o que ligar: `d` vazio é o jeito de o `<path>`
	// não desenhar sem um `data-show` a mais, que é a combinação que congela a aba.
	if wire, blue, beyond := moveWires([]engine.Square{{}}, nil, -1); wire != "" || blue != "" || beyond != "" {
		t.Errorf("uma dobra só desenhou %q, %q e %q", wire, blue, beyond)
	}
}

// classesThatReceiveBox lê a folha COMPILADA e devolve as classes de toda
// regra que resolve o `--col` em pixels.
//
// A folha compilada e não a fonte, porque é ela que o navegador recebe: uma
// classe que o scanner do Tailwind não viu não existe na folha, e é justamente
// esse o modo de falhar que não dá erro (ver o `engine-go/CLAUDE.md`).
func classesThatReceiveBox(t *testing.T) map[string]bool {
	t.Helper()
	sheet, err := os.ReadFile("../assets/static/app.css")
	if err != nil {
		t.Fatalf("ler a folha compilada: %v", err)
	}
	classes := map[string]bool{}
	for _, rule := range strings.Split(string(sheet), "}") {
		opens := strings.Index(rule, "{")
		if opens < 0 || !strings.Contains(rule[opens:], "left:calc(var(--col)") {
			continue
		}
		for _, selector := range strings.Split(rule[:opens], ",") {
			if name := strings.TrimPrefix(strings.TrimSpace(selector), "."); name != "" {
				classes[name] = true
			}
		}
	}
	return classes
}

// temAlgumaClasse: basta UMA classe posicionada, porque o elemento veste várias —
// o fantasma é `board-token board-token-ghost`, e quem lhe dá caixa é a
// primeira.
func temAlgumaClasse(list string, sought map[string]bool) bool {
	for _, c := range strings.Fields(list) {
		if sought[c] {
			return true
		}
	}
	return false
}
