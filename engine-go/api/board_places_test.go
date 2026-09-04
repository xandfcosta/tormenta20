package api

import "t20engine/tabuleiro"

import (
	"context"
	"strings"
	"testing"
)

/*
Lugares da crônica (ALE-124, fatia 5).

Até esta fatia, encerrar o tabuleiro DESTRUÍA a scene: o `Close` apagava a linha
e a taverna que o mestre montou peça por peça morria junto. A épica prometia o
contrário — "encerrar ARQUIVA, e devolve o tabuleiro à lista de Lugares da
crônica" —, e era a única promessa que o código contradizia.

O que se prova aqui é o ciclo que a table vive: montar, encerrar, voltar semana
que vem e achar tudo onde estava.
*/

func mesaComTaverna(t *testing.T) (*Server, int64, int64) {
	t.Helper()
	s := newTestServer(t)
	campanha := seedCampaign(t, s, seedUser(t, s, "gm@t.com"))
	sessao := seedSession(t, s, campanha)
	ctx := context.Background()

	s.boards.Open(ctx, sessao, "Taverna do Javali", "taverna")
	if _, err := s.boards.AddToken(ctx, sessao, defaultTab, tabuleiro.BoardToken{Label: "Ogro", X: 3, Y: 4, Footprint: 2}, true); err != nil {
		t.Fatalf("adicionar peça: %v", err)
	}
	return s, campanha, sessao
}

func TestEndingArchivesTheSceneInsteadOfDestroyingIt(t *testing.T) {
	s, campanha, sessao := mesaComTaverna(t)
	ctx := context.Background()

	if err := s.boards.Archive(ctx, campanha, s.boards.Get(ctx, sessao, defaultTab)); err != nil {
		t.Fatalf("arquivar: %v", err)
	}
	s.boards.Close(ctx, sessao, defaultTab)

	lugares := s.boards.Places(ctx, campanha)
	if len(lugares) != 1 {
		t.Fatalf("depois de encerrar, a crônica tem %d lugares: %+v", len(lugares), lugares)
	}
	if lugares[0].Name != "Taverna do Javali" {
		t.Errorf("o lugar guardado se chama %q", lugares[0].Name)
	}
	// A contagem existe para o mestre escolher onde jogar sem baixar o acervo.
	if lugares[0].Tokens != 1 {
		t.Errorf("a taverna guardada tem %d peças, esperado 1", lugares[0].Tokens)
	}
	// E a mesa fica MESMO sem tabuleiro: arquivar não é deixar a cena aberta.
	if b := s.boards.Get(ctx, sessao, defaultTab); b != nil {
		t.Errorf("a sessão continuou com tabuleiro depois de encerrar: %+v", b)
	}
}

func TestReopeningBringsTheTokensBackWhereTheyWere(t *testing.T) {
	s, campanha, sessao := mesaComTaverna(t)
	ctx := context.Background()
	if err := s.boards.Archive(ctx, campanha, s.boards.Get(ctx, sessao, defaultTab)); err != nil {
		t.Fatalf("arquivar: %v", err)
	}
	s.boards.Close(ctx, sessao, defaultTab)
	guardado := s.boards.Places(ctx, campanha)[0]

	volta, err := s.boards.OpenPlace(ctx, campanha, sessao, guardado.ID)
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}

	if volta.Place != "Taverna do Javali" {
		t.Errorf("reabriu como %q", volta.Place)
	}
	if len(volta.Tokens) != 1 || volta.Tokens[0].X != 3 || volta.Tokens[0].Y != 4 {
		t.Fatalf("as peças não voltaram onde estavam: %+v", volta.Tokens)
	}
	if volta.Tokens[0].Footprint != 2 {
		t.Errorf("o tamanho da peça se perdeu: %d", volta.Tokens[0].Footprint)
	}
}

// Encerrar a MESMA taverna de novo sobrescreve: quem reabre, move duas peças e
// encerra espera UMA taverna, não uma pilha de tavernas quase iguais.
func TestArchivingTwiceDoesNotStackTheSamePlace(t *testing.T) {
	s, campanha, sessao := mesaComTaverna(t)
	ctx := context.Background()

	if err := s.boards.Archive(ctx, campanha, s.boards.Get(ctx, sessao, defaultTab)); err != nil {
		t.Fatalf("arquivar: %v", err)
	}
	if _, err := s.boards.AddToken(ctx, sessao, defaultTab, tabuleiro.BoardToken{Label: "Bandido", X: 9, Y: 9}, true); err != nil {
		t.Fatalf("segunda peça: %v", err)
	}
	if err := s.boards.Archive(ctx, campanha, s.boards.Get(ctx, sessao, defaultTab)); err != nil {
		t.Fatalf("arquivar de novo: %v", err)
	}

	lugares := s.boards.Places(ctx, campanha)
	if len(lugares) != 1 {
		t.Fatalf("a crônica ficou com %d tavernas: %+v", len(lugares), lugares)
	}
	if lugares[0].Tokens != 2 {
		t.Errorf("o lugar guardou %d peças, esperado 2 (a versão mais recente)", lugares[0].Tokens)
	}
}

// O provisório é de uma cena que já acabou: a mesa que reabre a taverna não
// deve nada a um movimento proposto na semana passada.
func TestThePendingMoveDoesNotComeBackWithThePlace(t *testing.T) {
	s, campanha, sessao := mesaComTaverna(t)
	ctx := context.Background()
	board := s.boards.Get(ctx, sessao, defaultTab)
	board.Pending = &tabuleiro.PendingMove{TokenID: "t1", Cost: 3, Budget: 6}

	if err := s.boards.Archive(ctx, campanha, board); err != nil {
		t.Fatalf("arquivar: %v", err)
	}
	s.boards.Close(ctx, sessao, defaultTab)
	volta, err := s.boards.OpenPlace(ctx, campanha, sessao, s.boards.Places(ctx, campanha)[0].ID)
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}

	if volta.Pending != nil {
		t.Errorf("o provisório voltou junto com o lugar: %+v", volta.Pending)
	}
}

// O id do lugar vem do cliente: sem conferir a crônica, um mestre apagaria a
// cena de OUTRA mesa mandando um id que não é dele.
func TestAPlaceFromAnotherCampaignCannotBeDeleted(t *testing.T) {
	s, campanha, sessao := mesaComTaverna(t)
	ctx := context.Background()
	if err := s.boards.Archive(ctx, campanha, s.boards.Get(ctx, sessao, defaultTab)); err != nil {
		t.Fatalf("arquivar: %v", err)
	}
	guardado := s.boards.Places(ctx, campanha)[0]
	outra := seedCampaign(t, s, seedUser(t, s, "outro@t.com"))

	if err := s.boards.RemovePlace(ctx, outra, guardado.ID); err == nil {
		t.Fatal("apagou o lugar de outra crônica")
	}
	if len(s.boards.Places(ctx, campanha)) != 1 {
		t.Error("o lugar sumiu mesmo com a recusa")
	}
}

/*
Aqui moravam o TestSwitchingScenesArchivesTheOneOnTheTable e o
TestASceneFromAnotherCampaignCannotReachTheTable, os dois dirigindo o `ShowPlace`
(ALE-191) — a porta que a ALE-205 aposentou e que nenhuma rota chamava havia três
fatias.

O PRIMEIRO é a razão de esta lápide existir. Ele afirmava, EM VERDE, que trocar
de cena arquiva a que estava na mesa; o GLOSSARIO diz o contrário desde a
ALE-205 — "Reabrir não troca mais nada de lugar: ele acrescenta uma aba, e a
cena que estava na mesa continua onde estava". Ele não ficou obsoleto junto com
a porta que dirigia: passou a afirmar o OPOSTO do produto, e continuou verde
porque a porta morta ainda respondia. É a forma mais cara desta família — o
teste não avisa que envelheceu, ele mente com cara de cobertura (ALE-289).

O SEGUNDO prendia uma regra VIVA na porta errada, e por isso mudou de casa em
vez de morrer: virou o TestASceneFromAnotherCampaignCannotReachTheTableThroughOpenPlace,
logo abaixo, com o controle que ele não tinha.
*/
// O id do lugar vem do cliente, e o `OpenPlace` é a porta VIVA: sem conferir a
// crônica, um mestre puxaria para a própria mesa a cena de OUTRA campanha.
//
// Ele existe porque a regra estava presa só no `ShowPlace`, que nenhuma rota
// chama desde a ALE-205 — o verde era sobre a porta MORTA, e apagá-la como
// código morto levaria junto a única prova da regra (ALE-289).
func TestASceneFromAnotherCampaignCannotReachTheTableThroughOpenPlace(t *testing.T) {
	s, campanha, sessao := mesaComTaverna(t)
	ctx := context.Background()
	neighbourCampaign := seedCampaign(t, s, seedUser(t, s, "vizinho-openplace@t.com"))
	if err := s.boards.Archive(ctx, neighbourCampaign, &tabuleiro.BoardState{Version: 1, Place: "Cripta alheia"}); err != nil {
		t.Fatalf("guardar a cena da outra mesa: %v", err)
	}
	theirPlace := s.boards.Places(ctx, neighbourCampaign)[0]

	if _, err := s.boards.OpenPlace(ctx, campanha, sessao, theirPlace.ID); err == nil {
		t.Fatal("abriu na mesa a cena de outra crônica")
	}
	if naMesa := s.boards.Get(ctx, sessao, defaultTab); naMesa == nil || naMesa.Place != "Taverna do Javali" {
		t.Errorf("a recusa mexeu na cena que estava na mesa: %+v", naMesa)
	}

	// O CONTROLE, e sem ele o guarda não vale nada: um `OpenPlace` que recusasse
	// TODO lugar passaria nas asserções acima. O da própria crônica tem de abrir.
	if err := s.boards.Archive(ctx, campanha, s.boards.Get(ctx, sessao, defaultTab)); err != nil {
		t.Fatalf("arquivar a taverna: %v", err)
	}
	myPlace := placeNamed(t, s.boards.Places(ctx, campanha), "Taverna do Javali")
	if _, err := s.boards.OpenPlace(ctx, campanha, sessao, myPlace.ID); err != nil {
		t.Fatalf("o lugar da PRÓPRIA crônica foi recusado: %v", err)
	}
}

func placeNamed(t *testing.T, lugares []tabuleiro.Place, nome string) tabuleiro.Place {
	t.Helper()
	for _, lugar := range lugares {
		if lugar.Name == nome {
			return lugar
		}
	}
	t.Fatalf("%q não está no acervo: %+v", nome, lugares)
	return tabuleiro.Place{}
}

/*
Montar o lugar sem pôr nada na table (ALE-191, fatia 2).

É o único ponto do tabuleiro onde o estado inteiro chega pelo CLIENTE — nos
outros ele manda a intenção e o servidor produz o estado. O rascunho não tem
concorrência, broadcast nem vez, então um handler por gesto seria protocolo para
nada; o preço é conferir o que chega antes de virar acervo.
*/

// A cena montada volta inteira na próxima vez que o mestre a abrir — e a peça
// nova, que nasceu sem id no cliente, ganha um.
func TestBuildingThePlaceStoresTheSceneWithAnIdForTheNewToken(t *testing.T) {
	s, campanha, sessao := mesaComTaverna(t)
	ctx := context.Background()
	if err := s.boards.Archive(ctx, campanha, s.boards.Get(ctx, sessao, defaultTab)); err != nil {
		t.Fatalf("guardar a taverna: %v", err)
	}
	lugar := s.boards.Places(ctx, campanha)[0]

	montada := &tabuleiro.BoardState{Place: "nome que o cliente inventou", Tokens: []tabuleiro.BoardToken{
		{Label: "Necromante", X: 4, Y: 4, Footprint: 2},
	}}
	if err := s.boards.SavePlaceScene(ctx, campanha, lugar.ID, montada); err != nil {
		t.Fatalf("guardar a cena montada: %v", err)
	}

	volta, err := s.boards.PlaceScene(ctx, campanha, lugar.ID)
	if err != nil {
		t.Fatalf("reabrir para montar: %v", err)
	}
	if len(volta.Tokens) != 1 || volta.Tokens[0].Label != "Necromante" {
		t.Fatalf("a cena montada não voltou: %+v", volta.Tokens)
	}
	if volta.Tokens[0].ID == "" {
		t.Error("a peça nova voltou sem id: nada consegue selecioná-la depois")
	}
	// O NOME é da coluna: o rascunho não renomeia o lugar por baixo do pano.
	if volta.Place != "Taverna do Javali" {
		t.Errorf("o lugar passou a se chamar %q", volta.Place)
	}
	// E a MESA não foi tocada: montar é preparação.
	if naMesa := s.boards.Get(ctx, sessao, defaultTab); naMesa == nil || len(naMesa.Tokens) != 1 {
		t.Errorf("montar o lugar mexeu na cena que está na mesa: %+v", naMesa)
	}
}

// O estado chega do cliente, então o que ele afirma é conferido: uma peça em
// table.Coordinate absurda estouraria a serialização e a tela de todo mundo quando a
// cena chegasse à mesa — e o erro tem de dizer o valor ofensor.
func TestASceneBuiltWithAnAbsurdCoordinateIsRefused(t *testing.T) {
	s, campanha, sessao := mesaComTaverna(t)
	ctx := context.Background()
	if err := s.boards.Archive(ctx, campanha, s.boards.Get(ctx, sessao, defaultTab)); err != nil {
		t.Fatalf("guardar a taverna: %v", err)
	}
	lugar := s.boards.Places(ctx, campanha)[0]

	absurda := &tabuleiro.BoardState{Tokens: []tabuleiro.BoardToken{{Label: "Fantasma", X: 9_000_000, Y: 0}}}
	err := s.boards.SavePlaceScene(ctx, campanha, lugar.ID, absurda)

	if err == nil {
		t.Fatal("guardou uma peça fora do limite de sanidade")
	}
	if !strings.Contains(err.Error(), "9000000") {
		t.Errorf("o erro não diz o valor ofensor: %v", err)
	}
	if depois := s.boards.Places(ctx, campanha)[0]; depois.Tokens != 1 {
		t.Errorf("a recusa mexeu no acervo: o lugar ficou com %d peças", depois.Tokens)
	}
}

// A mesma posse do apagar e do mostrar à mesa: o id vem do cliente.
func TestAPlaceFromAnotherCampaignCannotBeBuilt(t *testing.T) {
	s, campanha, _ := mesaComTaverna(t)
	ctx := context.Background()
	outra := seedCampaign(t, s, seedUser(t, s, "vizinha@t.com"))
	if err := s.boards.Archive(ctx, outra, &tabuleiro.BoardState{Version: 1, Place: "Cripta alheia"}); err != nil {
		t.Fatalf("guardar a cena da outra mesa: %v", err)
	}
	alheia := s.boards.Places(ctx, outra)[0]

	if _, err := s.boards.PlaceScene(ctx, campanha, alheia.ID); err == nil {
		t.Error("leu a cena de outra crônica")
	}
	if err := s.boards.SavePlaceScene(ctx, campanha, alheia.ID, &tabuleiro.BoardState{}); err == nil {
		t.Error("escreveu na cena de outra crônica")
	}
}
