package tabuleiro

import (
	"testing"

	"t20engine/engine"
)

// Os guardas do GRUPO (ALE-203, item 10).
//
// Duas regras, e as duas só aparecem com peça GRANDE ou com lista suja — que é
// justamente o que ninguém testa clicando.

func comHorda() *BoardState {
	return &BoardState{Tokens: []BoardToken{
		{ID: "rato", Label: "Rato", X: 1, Y: 1},
		{ID: "dragao", Label: "Dragão", X: 10, Y: 10, Footprint: 6},
		{ID: "longe", Label: "Ogro", X: 40, Y: 40},
	}}
}

// TestTheLassoCatchesTheTokenByItsBody.
//
// A REGRA: uma Colossal ocupa 6×6 (p107), e o laço que toca QUALQUER casa do
// corpo dela a marca. Marcar só pela âncora faria o mestre desenhar em volta do
// dragão e não pegar o dragão — e ele concluiria que a ferramenta não funciona
// com peça grande, que é meia verdade pior que nenhuma.
func TestTheLassoCatchesTheTokenByItsBody(t *testing.T) {
	b := comHorda()
	// Um laço no MEIO do dragão, longe da âncora (10,10) que fica fora dele.
	pegos := TokensInRectangle(b, engine.Square{X: 13, Y: 13}, engine.Square{X: 14, Y: 14})
	if len(pegos) != 1 || pegos[0] != "dragao" {
		t.Errorf("o laço no meio do corpo pegou %v, esperado só o dragão", pegos)
	}
	// O CONTROLE: um laço FORA de tudo não pega nada. Sem ele, "pegou o dragão"
	// seria verdade também para uma função que devolve tudo sempre.
	if vazio := TokensInRectangle(b, engine.Square{X: 100, Y: 100}, engine.Square{X: 101, Y: 101}); len(vazio) != 0 {
		t.Errorf("um laço em lugar nenhum pegou %v", vazio)
	}
}

// TestTheGroupMovesTogetherOrDoesNotMove.
//
// A REGRA: coordenada absurda no meio da lista NÃO pode deixar metade do grupo
// movida. É o pior estado possível porque PARECE que o gesto funcionou — o
// mestre vê três zumbis andarem e três ficarem, e não tem como saber se foi
// regra ou defeito.
func TestTheGroupMovesTogetherOrDoesNotMove(t *testing.T) {
	b := comHorda()
	antes := map[string]engine.Square{}
	for _, t := range b.Tokens {
		antes[t.ID] = engine.Square{X: t.X, Y: t.Y}
	}

	// O DELTA É ESCOLHIDO PARA A PEÇA QUE ESTOURA NÃO SER A PRIMEIRA, e isso é o
	// que faz o caso medir alguma coisa. A primeira versão usava um delta
	// gigantesco e passava VERDE sobre a sabotagem: com ele o rato (a primeira da
	// lista) já falhava, o gesto voltava antes de escrever nada, e a passada
	// dupla nunca era exercitada.
	//
	// Rato em (1,1) e Ogro em (40,40), limite de 5000: com +4990 o rato cabe
	// (4991) e o Ogro não (5030). Uma passada só moveria o rato e recusaria
	// depois — metade do grupo andando, que é o pior estado possível.
	if err := MoveGroup(b, []string{"rato", "dragao", "longe"}, 4990, 0); err == nil {
		t.Error("um delta que estoura para a última peça foi aceito")
	}
	for _, peca := range b.Tokens {
		if antes[peca.ID] != (engine.Square{X: peca.X, Y: peca.Y}) {
			t.Errorf("%s andou apesar da recusa: %v → (%d,%d)", peca.ID, antes[peca.ID], peca.X, peca.Y)
		}
	}
}

// TestTheGroupMovesOnlyWhoWasMarkedAndRemembersWhereFrom.
//
// O `DeOndeVeio` é o que faz o "voltar para onde estava" do menu (ALE-206)
// funcionar depois de um movimento de grupo. Sem ele o verbo aparece e não faz
// nada, ou pior: devolve a peça a um lugar de duas cenas atrás.
func TestTheGroupMovesOnlyWhoWasMarkedAndRemembersWhereFrom(t *testing.T) {
	b := comHorda()
	if err := MoveGroup(b, []string{"rato", "dragao"}, 3, -2); err != nil {
		t.Fatalf("mover o grupo deu %v", err)
	}
	porID := map[string]BoardToken{}
	for _, peca := range b.Tokens {
		porID[peca.ID] = peca
	}
	if porID["rato"].X != 4 || porID["rato"].Y != -1 {
		t.Errorf("o rato foi para (%d,%d), esperado (4,-1)", porID["rato"].X, porID["rato"].Y)
	}
	if porID["longe"].X != 40 {
		t.Errorf("a peça NÃO marcada andou: %d", porID["longe"].X)
	}
	if de := porID["dragao"].DeOndeVeio; de == nil || de.X != 10 || de.Y != 10 {
		t.Errorf("o dragão não guardou de onde veio: %v", de)
	}
}

// TestTheGroupIgnoresATokenThatVanished: entre marcar e arrastar, o stream pode trazer a
// remoção de uma delas por outra pessoa. Recusar o movimento das outras cinco
// por causa disso é punir o mestre por uma corrida que não é dele.
func TestTheGroupIgnoresATokenThatVanished(t *testing.T) {
	b := comHorda()
	if err := MoveGroup(b, []string{"rato", "fantasma"}, 1, 1); err != nil {
		t.Fatalf("um id que não existe derrubou o gesto: %v", err)
	}
	if b.Tokens[0].X != 2 {
		t.Errorf("o rato não andou: %d", b.Tokens[0].X)
	}
}
