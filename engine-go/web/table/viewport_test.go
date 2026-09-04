package table

import (
	"testing"
)

// Os guardas da JANELA sobre o plano infinito (ALE-203).
//
// O que se prende aqui é o que a moldura protegia por construção e passou a
// depender de uma CONVENÇÃO: enquanto o servidor desenhava um retângulo, "onde o
// dedo caiu" era uma conta sobre um elemento de tamanho conhecido. Agora é uma
// conta sobre a JANELA — o ponto do clique mais a vista, dividido pelo zoom — e
// ela aparece em cinco camadas de clique.

// TestNoLayerReadsThePointWithoutAddingTheViewport.
//
// Este é o guarda da FAMÍLIA e não de um defeito: a conta do clique tinha CINCO
// cópias, uma por camada, e o comentário de cada uma dizia que a repetição era
// deliberada. Sem moldura, todas as cinco ganharam a janela dentro delas — e uma
// que ficasse para trás clicaria no quadrado errado, sem erro nenhum, com o
// desvio crescendo à medida que a pessoa arrasta a vista.
//
// Ele varre o HTML SERVIDO e não o código: é a única forma de alcançar a camada
// que alguém escrever amanhã sem ler nada disto.
// TestTheShortcutDoesNotShiftWhenAGmToolLeavesTheRail.
//
// É a promessa que o cabeçalho do trilho faz: quem aprendeu `4 = gabarito`
// mestrando continua com `4 = gabarito` jogando. Ela vale porque o número sai da
// posição na lista COMPLETA, e o filtro por papel acontece DEPOIS.
//
// O caso é montado à mão de propósito. No trilho de hoje todas as ferramentas do
// mestre estão no FIM, então numerar depois do filtro daria o mesmo resultado — o
// guarda passaria sobre a sabotagem, que é o pior tipo de verde. O que quebra a
// promessa é uma ferramenta do mestre no MEIO, e é ela que este trilho tem.
func TestTheShortcutDoesNotShiftWhenAGmToolLeavesTheRail(t *testing.T) {
	trilho := numberRail([]mapTool{
		{ID: "mover", Rotulo: "Mover"},
		{ID: "pintar", Rotulo: "Pintar", SoMestre: true},
		{ID: "regua", Rotulo: "Régua"},
	})
	doMestre := forVisible(true, trilho)
	doJogador := forVisible(false, trilho)

	// O CONTROLE: os dois trilhos têm de DIFERIR, senão não há filtro medindo.
	if len(doMestre) == len(doJogador) {
		t.Fatalf("o filtro não tirou nada (%d entradas nos dois): não há o que comparar", len(doMestre))
	}
	if doJogador[len(doJogador)-1].Atalho != doMestre[len(doMestre)-1].Atalho {
		t.Errorf("a régua é a tecla %s para o mestre e %s para o jogador — a ferramenta que sumiu do meio renumerou o resto",
			doMestre[len(doMestre)-1].Atalho, doJogador[len(doJogador)-1].Atalho)
	}
}

// TestEachToolHasAKeyOfItsOwn.
//
// Duas ferramentas com o mesmo número não dão erro: a segunda simplesmente nunca
// liga, porque o `railKeyboard` monta um ternário encadeado e o primeiro
// ramo vence. Foi o defeito que os números escritos à mão convidavam, e é o
// motivo de o `numberRail` existir.
func TestEachToolHasAKeyOfItsOwn(t *testing.T) {
	vistas := map[string]string{}
	for _, f := range MapTools() {
		if f.Atalho == "" {
			t.Errorf("a ferramenta %q nasceu sem atalho", f.Rotulo)
			continue
		}
		if dono, tem := vistas[f.Atalho]; tem {
			t.Errorf("a tecla %s liga %q e %q — a segunda nunca acende", f.Atalho, dono, f.Rotulo)
		}
		vistas[f.Atalho] = f.Rotulo
	}
}

// TestThePanHandBelongsToBothRoles.
//
// Sem moldura não há rolagem nativa, então arrastar a vista deixou de ser
// conforto: é o único gesto de ponteiro que leva alguém ao outro lado do plano.
// Um trilho de jogador sem a mão seria um jogador preso no enquadramento em que
// a página abriu.
func TestThePanHandBelongsToBothRoles(t *testing.T) {
	for _, mestre := range []bool{true, false} {
		achou := false
		for _, f := range rail(mestre) {
			achou = achou || f.ID == ViewTool
		}
		if !achou {
			t.Errorf("mestre=%v não tem a mão de arrastar a vista, e sem ela não há como percorrer o plano", mestre)
		}
	}
}
