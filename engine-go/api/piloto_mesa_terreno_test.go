package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/tabuleiro"
)

// Os guardas do PINCEL de terreno na Mesa (ALE-264, item 5).

// TestTheBrushPaintsTheKindItAskedFor, e só ela.
//
// Amostragem sobre `EspeciesDeTerreno`: a quinta espécie já nasce medida.
func TestTheBrushPaintsTheKindItAskedFor(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")

	for i, pincel := range tabuleiro.EspeciesDeTerreno {
		// O caminho é o TRAÇO desde a ALE-203, e um clique parado é um traço de
		// uma casa: a mesma casa nas duas pontas.
		casa := fmt.Sprintf("/%d/0/ate/%d/0", i, i)
		rec := f.pede(t, f.mestre, "POST",
			f.urlDaMesa()+"/tabuleiro/terreno/"+string(pincel.ID)+casa, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("pintar %s deu %d", pincel.ID, rec.Code)
		}
	}

	b := f.s.boards.Get(context.Background(), f.sessionID, aAbaPadrao)
	for i, pincel := range tabuleiro.EspeciesDeTerreno {
		casas := tabuleiro.QuadradosDe(b, pincel.ID)
		if len(casas) != 1 || casas[0].X != i {
			t.Errorf("%s ficou com %v, esperado só a casa %d", pincel.ID, casas, i)
		}
	}
}

// TestTheEraserClearsOnlyTheChosenKind.
//
// É por isso que ela é um MODO e não uma espécie: numa casa com duas, uma
// "espécie borracha" teria de decidir qual apagar, e a resposta certa — a que
// está selecionada — já é o que o modo faz. Folhagens são difícil E camuflagem
// (p267), então a casa com duas não é hipótese.
func TestTheEraserClearsOnlyTheChosenKind(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	base := f.urlDaMesa() + "/tabuleiro/terreno"

	for _, especie := range []string{"dificil", "camuflagem"} {
		if rec := f.pede(t, f.mestre, "POST", base+"/"+especie+"/3/3/ate/3/3", ""); rec.Code != http.StatusOK {
			t.Fatalf("pintar %s deu %d", especie, rec.Code)
		}
	}
	if rec := f.pede(t, f.mestre, "POST", base+"/camuflagem/3/3/ate/3/3?apagar=1", ""); rec.Code != http.StatusOK {
		t.Fatalf("apagar deu %d", rec.Code)
	}

	b := f.s.boards.Get(context.Background(), f.sessionID, aAbaPadrao)
	if n := len(tabuleiro.QuadradosDe(b, tabuleiro.TerrenoCamuflagem)); n != 0 {
		t.Errorf("a camuflagem não foi apagada (%d casas)", n)
	}
	if n := len(tabuleiro.QuadradosDe(b, tabuleiro.TerrenoDificil)); n != 1 {
		t.Errorf("a borracha levou o difícil junto (%d casas) — a casa tinha as duas", n)
	}
}

// TestTheFourKindsAreDrawnDistinctly.
//
// O guarda de leiaute que esta casa cobra: um traço pintado que não vira classe
// própria some no desenho das outras, e o mestre lê a cena errada sem nada
// estourar. Amostragem sobre a lista.
func TestTheFourKindsAreDrawnDistinctly(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	for i, pincel := range tabuleiro.EspeciesDeTerreno {
		if rec := f.pede(t, f.mestre, "POST",
			fmt.Sprintf("%s/tabuleiro/terreno/%s/%d/0/ate/%d/0", f.urlDaMesa(), pincel.ID, i, i), ""); rec.Code != http.StatusOK {
			t.Fatalf("pintar %s deu %d", pincel.ID, rec.Code)
		}
	}

	tela := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	// O CONTROLE: o tabuleiro desenhou. Sem ele, não achar as classes seria
	// verdade também sobre uma cena que não abriu.
	if !strings.Contains(tela, "tabuleiro-plano") {
		t.Fatal("o tabuleiro não desenhou — o guarda mediria a tela errada")
	}
	for _, pincel := range tabuleiro.EspeciesDeTerreno {
		if !strings.Contains(tela, "tabuleiro-"+string(pincel.ID)) {
			t.Errorf("a espécie %s foi pintada e não tem desenho próprio na cena", pincel.ID)
		}
	}
}

// TestTheRailSaysTheEffectOfEachKind.
//
// "Cobertura" sozinho não lembra ninguém de que são +5 na Defesa, e o mestre que
// precisa da regra sai da mesa para procurá-la no livro. É a mesma razão de o
// diálogo de abrir dizer que um quadrado são 1,5m.
func TestTheRailSaysTheEffectOfEachKind(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	tela := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()

	// "Ferramentas do mapa" e não mais "Pincel de terreno": o trilho deixou de
	// ser só do pincel quando MARCAR entrou nele (ALE-264, item 5), e um grupo
	// que se anuncia como pincel enquanto carrega outra ferramenta mente para
	// quem navega por leitor de tela. O guarda acusou a troca e foi ATUALIZADO.
	if !strings.Contains(tela, "Ferramentas do mapa") {
		t.Fatal("o mestre não tem trilho de ferramentas na cena aberta")
	}
	// O nome acessível da camada NÃO cita a espécie: ela sairia do sinal, que
	// guarda o ID, e o leitor de tela anunciaria "Pintar dificil" sem acento.
	// Quem diz qual é a espécie é o botão `aria-pressed` do trilho.
	//
	// O sinal chama-se `$ferramenta` desde a ALE-264 — e esta linha é um lembrete
	// caro: ela citava `$pincel`, que deixou de existir, e uma asserção de
	// AUSÊNCIA sobre um nome morto passa verde sobre nada. Nome de sinal em
	// asserção negativa envelhece em silêncio.
	if strings.Contains(tela, "'Pintar ' + $ferramenta") {
		t.Error("o nome acessível da camada monta o rótulo com o id da ferramenta")
	}
	for _, pincel := range tabuleiro.EspeciesDeTerreno {
		if !strings.Contains(tela, pincel.Efeito) {
			t.Errorf("o trilho não diz o que %s faz (%q)", pincel.ID, pincel.Efeito)
		}
	}
	// A PÁGINA, e não a regra: a citação vai junto para conferir sem reabrir o
	// livro. O número é da Tabela 5-3.
	if !strings.Contains(tela, "p238") {
		t.Error("o trilho não cita a página da regra")
	}

	// E o pincel é do MESTRE: o jogador não pinta chão.
	//
	// A asserção mudou de alvo na ALE-269 e vale dizer por quê: ela era sobre o
	// TRILHO inteiro ("Ferramentas do mapa"), que era do mestre porque só ele
	// tinha modo. A régua é de quem ataca, então o trilho passou a existir para
	// os dois papéis e o que ficou do mestre foram os PINCÉIS dentro dele. Fosse
	// mantida como estava, esta linha teria falhado dizendo a coisa errada — e
	// fosse apagada, o vazamento do pincel deixaria de ser medido.
	doJogador := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()
	for _, pincel := range tabuleiro.EspeciesDeTerreno {
		if strings.Contains(doJogador, pincel.Efeito) {
			t.Errorf("o pincel %q apareceu na cena do jogador", pincel.ID)
		}
	}
	if strings.Contains(doJogador, "Borracha") {
		t.Error("a borracha do mestre apareceu na cena do jogador")
	}
}

// TestOnlyTheGmPaints: a trava é do servidor, e não o botão escondido.
func TestOnlyTheGmPaints(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")

	rec := f.pede(t, f.jogador, "POST", f.urlDaMesa()+"/tabuleiro/terreno/dificil/1/1/ate/1/1", "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("o jogador pintou o chão: %d", rec.Code)
	}
	b := f.s.boards.Get(context.Background(), f.sessionID, aAbaPadrao)
	if n := len(tabuleiro.QuadradosDe(b, tabuleiro.TerrenoDificil)); n != 0 {
		t.Errorf("a pintura do jogador entrou mesmo assim (%d casas)", n)
	}
}

// TestPaintingWithoutABoardRefusesWithASentence.
//
// Não é 500 nem silêncio: pintar chão de uma cena que não está na mesa não tem
// onde acontecer, e a recusa fala no `erroDoComando` do rodapé do mestre.
func TestPaintingWithoutABoardRefusesWithASentence(t *testing.T) {
	f := novoPiloto(t)
	corpo := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/tabuleiro/terreno/dificil/1/1/ate/1/1", "").Body.String()
	if !strings.Contains(corpo, "não há tabuleiro aberto") {
		t.Errorf("a recusa não explica o que faltou; sinais = %s", trechoDeSinais(corpo))
	}
}
