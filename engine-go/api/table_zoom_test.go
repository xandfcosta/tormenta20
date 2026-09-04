package api

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"t20engine/web/table"
	"testing"
)

// O zoom é comportamento de NAVEGADOR — `--quadrado` muda e a cena inteira
// deriva —, então o que um teste de servidor pode segurar é pouco e é
// justamente o que envelhece sozinho: que o padrão da PÁGINA e o padrão do
// CÓDIGO são o mesmo número, e que os limites que a tela oferece são os que as
// expressões cobram. O resto é da fatia do e2e do tabuleiro.
func TestTheZoomIsBornAtTheDefaultAndRespectsTheLimits(t *testing.T) {
	f := novoPiloto(t)
	f.seedOpenBoard(t, "pedra")
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	// O CONTROLE: os controles estão na página. Sem isto, as buscas abaixo
	// falhariam por motivo errado e "não achei o limite" leria como "o limite
	// sumiu" quando a verdade seria "a cena não desenhou o zoom".
	// O nome do grupo virou "Enquadrar o mapa" na ALE-269, quando o centralizar
	// entrou ao lado do zoom: "Aproximar e afastar" descrevia dois dos três
	// botões. O CONTROLE continua sendo o mesmo — provar que a faixa está na
	// página antes de procurar coisa dentro dela.
	if !strings.Contains(tela, "Enquadrar o mapa") {
		t.Fatal("a cena não desenhou os controles de enquadramento")
	}

	// O PADRÃO é derivado e não digitado: escrever 44 no `data-signals` seria a
	// terceira cópia da mesma escolha, e a que fica para trás é sempre a da
	// página — a cena nasceria com um zoom e o botão contando outro.
	if !strings.Contains(tela, fmt.Sprintf("quadrado: %d", table.DefaultSquare)) {
		t.Errorf("a página não semeia o zoom padrão (%d)", table.DefaultSquare)
	}
	// E os limites que desabilitam os botões são os do código, não outros dois.
	//
	// A busca é pela EXPRESSÃO inteira e não pelo número: procurar "20" numa
	// página HTML acha vinte coisas — uma classe, um tamanho, um id — e a
	// asserção passaria verde com o limite trocado. Foi assim que ela nasceu, e
	// eu a apertei antes de confiar nela.
	//
	// E ela vai ESCAPADA: `<=` sai como `&lt;=` porque o valor do atributo é
	// DINÂMICO, e só valor constante sai literal (está no guia do pacote). No
	// navegador não muda nada — o parser desfaz o escape —, mas um teste que lê
	// HTML cru compara com a forma do fio. Medido antes de escrever.
	for _, expressao := range []string{table.ZoomAtLimit(-table.ZoomStep), table.ZoomAtLimit(table.ZoomStep)} {
		if !strings.Contains(tela, html.EscapeString(expressao)) {
			t.Errorf("a expressão de limite %q não está na cena", expressao)
		}
	}
	// O enquadramento é de TODO MUNDO: o jogador enquadra a própria janela, e
	// depender do mestre para aproximar no telefone não é enquadramento, é
	// pedido. Vale para o centralizar pela mesma razão — achar o grupo num plano
	// sem bordas é problema de quem está olhando, não de quem montou a cena.
	doJogador := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(doJogador, "Enquadrar o mapa") {
		t.Error("o jogador não recebeu os controles de enquadramento")
	}
	if !strings.Contains(doJogador, table.CenterTarget(table.BoardView{})) {
		t.Error("o jogador não recebeu o centralizar")
	}
}
