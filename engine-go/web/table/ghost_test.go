package table

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"t20engine/engine"
	"t20engine/tabuleiro"
)

// Os guardas do FANTASMA e da SETA (ALE-203, item 4 da lista do dono).
//
// As palavras dele: *"Movimentar a peça arrastando somente cria um ponto para o
// primeiro movimento. Logo ao soltar a peça, ela voltar para o início do
// movimento. A ideia é, ao soltar a peça, ela vai ser renderizada no lugar que
// foi solta e o início mostra a peça transparente para marcar o início do
// movimento. A seta da régua conecta os dois pontos."*
//
// O que se prende aqui é a DIVISA que a fatia abriu: a peça é DESENHADA no fim
// do caminho e continua GRAVADA na origem. As duas metades precisam de guarda,
// porque cada uma sozinha passa verde sobre o defeito da outra — desenhar sem
// gravar seria a peça andando sem confirmação, e gravar sem desenhar é o defeito
// que o dono relatou.

// onBoardAt põe a peça do jogador numa casa escolhida, e devolve o id dela.
//
// Irmã da `onBoard`, que sempre põe em 0,0: aqui a origem precisa ser um
// número que não se confunda com "não preenchido" — com a peça em 0,0 um
// fantasma desenhado na quina do plano por engano passaria despercebido.
// element acha o primeiro elemento cujo atributo `atributo` contém `trecho`,
// e devolve os atributos dele.
//
// Um parser de HTML de verdade e não uma expressão regular, e a razão é o guarda
// do fim deste arquivo: as expressões do Datastar carregam `<`, `>` e aspas
// dentro dos valores, e um `<[^>]*>` corta um elemento no meio de um `data-on:`
// sem avisar — a busca devolveria menos e a ausência viraria conclusão.
func element(t *testing.T, tela, atributo, trecho string) map[string]string {
	t.Helper()
	z := html.NewTokenizer(strings.NewReader(tela))
	for {
		switch z.Next() {
		case html.ErrorToken:
			return nil
		case html.StartTagToken, html.SelfClosingTagToken:
			attrs := map[string]string{}
			for {
				chave, valor, mais := z.TagAttr()
				attrs[string(chave)] = string(valor)
				if !mais {
					break
				}
			}
			if strings.Contains(attrs[atributo], trecho) {
				return attrs
			}
		}
	}
}

// TestTheArrowWithoutStopsJoinsBothEndsOfThePath.
//
// `Stops` NULO é valor legítimo: o `ProposeMove` deixa o caminho pronto sem
// passar por paradas. Deduzir as dobras do `Path` não é possível — um trecho
// legítimo já dobra sozinho, porque a diagonal vem primeiro —, então a seta vira
// a reta entre o começo e o fim.
func TestTheArrowWithoutStopsJoinsBothEndsOfThePath(t *testing.T) {
	semParadas := &tabuleiro.PendingMove{
		Path: []engine.Square{{}, {X: 1}, {X: 2}, {X: 3}},
	}
	dobras := moveFolds(semParadas)
	if len(dobras) != 2 || dobras[0] != (engine.Square{}) || dobras[1] != (engine.Square{X: 3}) {
		t.Fatalf("as dobras de um caminho sem paradas saíram %+v", dobras)
	}
	// A perna anda 3 e a ponta recua meio quadrado: de 0,5 até 3,0. Sem orçamento
	// (-1) ela sai inteira de dourado — o vermelho do item 13 tem guarda próprio
	// em `move_drawing_test.go`.
	if fio, _, _ := moveWires(dobras, []int{3}, -1); fio != "M 0.5 0.5 L 3 0.5" {
		t.Errorf("a seta reta saiu %q", fio)
	}
	// E com uma dobra só não há o que ligar: `d` vazio é o jeito de o `<path>`
	// não desenhar sem um `data-show` a mais, que é a combinação que congela a aba.
	if fio, azul, alem := moveWires([]engine.Square{{}}, nil, -1); fio != "" || azul != "" || alem != "" {
		t.Errorf("uma dobra só desenhou %q, %q e %q", fio, azul, alem)
	}
}

// boxReceiveClasses lê a folha COMPILADA e devolve as classes de toda
// regra que resolve o `--col` em pixels.
//
// A folha compilada e não a fonte, porque é ela que o navegador recebe: uma
// classe que o scanner do Tailwind não viu não existe na folha, e é justamente
// esse o modo de falhar que não dá erro (ver o `engine-go/CLAUDE.md`).
func classesThatReceiveBox(t *testing.T) map[string]bool {
	t.Helper()
	folha, err := os.ReadFile("piloto/static/piloto.css")
	if err != nil {
		t.Fatalf("ler a folha compilada: %v", err)
	}
	classes := map[string]bool{}
	for _, regra := range strings.Split(string(folha), "}") {
		abre := strings.Index(regra, "{")
		if abre < 0 || !strings.Contains(regra[abre:], "left:calc(var(--col)") {
			continue
		}
		for _, seletor := range strings.Split(regra[:abre], ",") {
			if nome := strings.TrimPrefix(strings.TrimSpace(seletor), "."); nome != "" {
				classes[nome] = true
			}
		}
	}
	return classes
}

// temAlgumaClasse: basta UMA classe posicionada, porque o elemento veste várias —
// o fantasma é `tabuleiro-peca tabuleiro-peca-fantasma`, e quem lhe dá caixa é a
// primeira.
func temAlgumaClasse(lista string, procuradas map[string]bool) bool {
	for _, c := range strings.Fields(lista) {
		if procuradas[c] {
			return true
		}
	}
	return false
}
