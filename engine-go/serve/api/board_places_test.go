package api

import "t20engine/domain/board"

import (
	"context"
	"t20engine/app/boards"

	"strings"
	"t20engine/infra/events"
	"testing"
)

/*
Lugares da crônica: o ciclo que a mesa vive — montar, encerrar, voltar semana que
vem e achar tudo onde estava.
*/

func mesaComTaverna(t *testing.T) (*Server, int64, int64) {
	t.Helper()
	s := newTestServer(t)
	campaign := seedCampaign(t, s, seedUser(t, s, "gm@t.com"))
	session := seedSession(t, s, campaign)
	ctx := context.Background()

	s.boards.Open(ctx, session, "Taverna do Javali", "tavern")
	if _, err := s.boards.AddToken(ctx, session, defaultTab, board.BoardToken{Label: "Ogro", X: 3, Y: 4, Footprint: 2}); err != nil {
		t.Fatalf("adicionar peça: %v", err)
	}
	return s, campaign, session
}

func TestEndingArchivesTheSceneInsteadOfDestroyingIt(t *testing.T) {
	s, campaign, session := mesaComTaverna(t)
	ctx := context.Background()

	if err := s.boards.Archive(ctx, campaign, s.boards.Get(ctx, session, defaultTab)); err != nil {
		t.Fatalf("arquivar: %v", err)
	}
	s.boards.Close(ctx, session, defaultTab)

	places := s.boards.Places(ctx, campaign)
	if len(places) != 1 {
		t.Fatalf("depois de encerrar, a crônica tem %d lugares: %+v", len(places), places)
	}
	if places[0].Name != "Taverna do Javali" {
		t.Errorf("o lugar guardado se chama %q", places[0].Name)
	}
	// A contagem existe para o mestre escolher onde jogar sem baixar o acervo.
	if places[0].Tokens != 1 {
		t.Errorf("a taverna guardada tem %d peças, esperado 1", places[0].Tokens)
	}
	// E a mesa fica MESMO sem tabuleiro: arquivar não é deixar a cena aberta.
	if b := s.boards.Get(ctx, session, defaultTab); b != nil {
		t.Errorf("a sessão continuou com tabuleiro depois de encerrar: %+v", b)
	}
}

func TestReopeningBringsTheTokensBackWhereTheyWere(t *testing.T) {
	s, campaign, session := mesaComTaverna(t)
	ctx := context.Background()
	if err := s.boards.Archive(ctx, campaign, s.boards.Get(ctx, session, defaultTab)); err != nil {
		t.Fatalf("arquivar: %v", err)
	}
	s.boards.Close(ctx, session, defaultTab)
	saved := s.boards.Places(ctx, campaign)[0]

	back, err := s.boards.OpenPlace(ctx, campaign, session, saved.ID)
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}

	if back.Place != "Taverna do Javali" {
		t.Errorf("reabriu como %q", back.Place)
	}
	if len(back.Tokens) != 1 || back.Tokens[0].X != 3 || back.Tokens[0].Y != 4 {
		t.Fatalf("as peças não voltaram onde estavam: %+v", back.Tokens)
	}
	if back.Tokens[0].Footprint != 2 {
		t.Errorf("o tamanho da peça se perdeu: %d", back.Tokens[0].Footprint)
	}
}

// Encerrar a MESMA taverna de novo sobrescreve: quem reabre, move duas peças e
// encerra espera UMA taverna, não uma pilha de tavernas quase iguais.
func TestArchivingTwiceDoesNotStackTheSamePlace(t *testing.T) {
	s, campaign, session := mesaComTaverna(t)
	ctx := context.Background()

	if err := s.boards.Archive(ctx, campaign, s.boards.Get(ctx, session, defaultTab)); err != nil {
		t.Fatalf("arquivar: %v", err)
	}
	if _, err := s.boards.AddToken(ctx, session, defaultTab, board.BoardToken{Label: "Bandido", X: 9, Y: 9}); err != nil {
		t.Fatalf("segunda peça: %v", err)
	}
	if err := s.boards.Archive(ctx, campaign, s.boards.Get(ctx, session, defaultTab)); err != nil {
		t.Fatalf("arquivar de novo: %v", err)
	}

	places := s.boards.Places(ctx, campaign)
	if len(places) != 1 {
		t.Fatalf("a crônica ficou com %d tavernas: %+v", len(places), places)
	}
	if places[0].Tokens != 2 {
		t.Errorf("o lugar guardou %d peças, esperado 2 (a versão mais recente)", places[0].Tokens)
	}
}

// O provisório é de uma cena que já acabou: a mesa que reabre a taverna não
// deve nada a um movimento proposto na semana passada.
func TestThePendingMoveDoesNotComeBackWithThePlace(t *testing.T) {
	s, campaign, session := mesaComTaverna(t)
	ctx := context.Background()
	opened := s.boards.Get(ctx, session, defaultTab)
	opened.Pending = &board.PendingMove{TokenID: "t1", Cost: 3, Budget: 6}

	if err := s.boards.Archive(ctx, campaign, opened); err != nil {
		t.Fatalf("arquivar: %v", err)
	}
	s.boards.Close(ctx, session, defaultTab)
	back, err := s.boards.OpenPlace(ctx, campaign, session, s.boards.Places(ctx, campaign)[0].ID)
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}

	if back.Pending != nil {
		t.Errorf("o provisório voltou junto com o lugar: %+v", back.Pending)
	}
}

// O id do lugar vem do cliente: sem conferir a crônica, um mestre apagaria a
// cena de OUTRA mesa mandando um id que não é dele.
func TestAPlaceFromAnotherCampaignCannotBeDeleted(t *testing.T) {
	s, campaign, session := mesaComTaverna(t)
	ctx := context.Background()
	if err := s.boards.Archive(ctx, campaign, s.boards.Get(ctx, session, defaultTab)); err != nil {
		t.Fatalf("arquivar: %v", err)
	}
	saved := s.boards.Places(ctx, campaign)[0]
	other := seedCampaign(t, s, seedUser(t, s, "outro@t.com"))

	if err := s.boards.RemovePlace(ctx, other, saved.ID); err == nil {
		t.Fatal("apagou o lugar de outra crônica")
	}
	if len(s.boards.Places(ctx, campaign)) != 1 {
		t.Error("o lugar sumiu mesmo com a recusa")
	}
}

/*
NÃO existe aqui um caso "trocar de cena arquiva a que estava na mesa". Ele
existiu e afirmava o OPOSTO do produto — reabrir acrescenta uma ABA, e a cena da
mesa continua onde estava —, e ficava verde porque dirigia uma porta morta
(`ShowPlace`) que ainda respondia. Quem for recriá-lo está recriando a mentira.
*/
// O id do lugar vem do cliente, e o `OpenPlace` é a porta VIVA: sem conferir a
// crônica, um mestre puxaria para a própria mesa a cena de OUTRA campanha.
//
// A regra já esteve presa só no `ShowPlace`, que rota nenhuma chama: o verde era
// sobre a porta MORTA, e apagá-la como código morto levaria junto a única prova.
func TestASceneFromAnotherCampaignCannotReachTheTableThroughOpenPlace(t *testing.T) {
	s, campaign, session := mesaComTaverna(t)
	ctx := context.Background()
	neighbourCampaign := seedCampaign(t, s, seedUser(t, s, "vizinho-openplace@t.com"))
	if err := s.boards.Archive(ctx, neighbourCampaign, &board.BoardState{Version: 1, Place: "Cripta alheia"}); err != nil {
		t.Fatalf("guardar a cena da outra mesa: %v", err)
	}
	theirPlace := s.boards.Places(ctx, neighbourCampaign)[0]

	if _, err := s.boards.OpenPlace(ctx, campaign, session, theirPlace.ID); err == nil {
		t.Fatal("abriu na mesa a cena de outra crônica")
	}
	if onTable := s.boards.Get(ctx, session, defaultTab); onTable == nil || onTable.Place != "Taverna do Javali" {
		t.Errorf("a recusa mexeu na cena que estava na mesa: %+v", onTable)
	}

	// O CONTROLE, e sem ele o guarda não vale nada: um `OpenPlace` que recusasse
	// TODO lugar passaria nas asserções acima. O da própria crônica tem de abrir.
	if err := s.boards.Archive(ctx, campaign, s.boards.Get(ctx, session, defaultTab)); err != nil {
		t.Fatalf("arquivar a taverna: %v", err)
	}
	myPlace := placeNamed(t, s.boards.Places(ctx, campaign), "Taverna do Javali")
	if _, err := s.boards.OpenPlace(ctx, campaign, session, myPlace.ID); err != nil {
		t.Fatalf("o lugar da PRÓPRIA crônica foi recusado: %v", err)
	}
}

func placeNamed(t *testing.T, places []board.Place, name string) board.Place {
	t.Helper()
	for _, place := range places {
		if place.Name == name {
			return place
		}
	}
	t.Fatalf("%q não está no acervo: %+v", name, places)
	return board.Place{}
}

/*
Montar o lugar sem pôr nada na mesa. As duas portas de baixo — ler a cena
guardada e gravá-la — são o que o `EditPlace` usa para aplicar um gesto do
rascunho.
*/

// A cena montada volta inteira na próxima vez que o mestre a abrir — e a peça
// nova, que nasceu sem id no cliente, ganha um.
func TestBuildingThePlaceStoresTheSceneWithAnIdForTheNewToken(t *testing.T) {
	s, campaign, session := mesaComTaverna(t)
	ctx := context.Background()
	if err := s.boards.Archive(ctx, campaign, s.boards.Get(ctx, session, defaultTab)); err != nil {
		t.Fatalf("guardar a taverna: %v", err)
	}
	place := s.boards.Places(ctx, campaign)[0]

	built := &board.BoardState{Place: "nome que o cliente inventou", Tokens: []board.BoardToken{
		{Label: "Necromante", X: 4, Y: 4, Footprint: 2},
	}}
	if err := s.boards.SavePlaceScene(ctx, campaign, place.ID, built); err != nil {
		t.Fatalf("guardar a cena montada: %v", err)
	}

	back, err := s.boards.PlaceScene(ctx, campaign, place.ID)
	if err != nil {
		t.Fatalf("reabrir para montar: %v", err)
	}
	if len(back.Tokens) != 1 || back.Tokens[0].Label != "Necromante" {
		t.Fatalf("a cena montada não voltou: %+v", back.Tokens)
	}
	if back.Tokens[0].ID == "" {
		t.Error("a peça nova voltou sem id: nada consegue selecioná-la depois")
	}
	// O NOME é da coluna: o rascunho não renomeia o lugar por baixo do pano.
	if back.Place != "Taverna do Javali" {
		t.Errorf("o lugar passou a se chamar %q", back.Place)
	}
	// E a MESA não foi tocada: montar é preparação.
	if onTable := s.boards.Get(ctx, session, defaultTab); onTable == nil || len(onTable.Tokens) != 1 {
		t.Errorf("montar o lugar mexeu na cena que está na mesa: %+v", onTable)
	}
}

// O estado chega do cliente, então o que ele afirma é conferido: uma peça em
// table.Coordinate absurda estouraria a serialização e a tela de todo mundo quando a
// cena chegasse à mesa — e o erro tem de dizer o valor ofensor.
func TestASceneBuiltWithAnAbsurdCoordinateIsRefused(t *testing.T) {
	s, campaign, session := mesaComTaverna(t)
	ctx := context.Background()
	if err := s.boards.Archive(ctx, campaign, s.boards.Get(ctx, session, defaultTab)); err != nil {
		t.Fatalf("guardar a taverna: %v", err)
	}
	place := s.boards.Places(ctx, campaign)[0]

	absurd := &board.BoardState{Tokens: []board.BoardToken{{Label: "Fantasma", X: 9_000_000, Y: 0}}}
	err := s.boards.SavePlaceScene(ctx, campaign, place.ID, absurd)

	if err == nil {
		t.Fatal("guardou uma peça fora do limite de sanidade")
	}
	if !strings.Contains(err.Error(), "9000000") {
		t.Errorf("o erro não diz o valor ofensor: %v", err)
	}
	if after := s.boards.Places(ctx, campaign)[0]; after.Tokens != 1 {
		t.Errorf("a recusa mexeu no acervo: o lugar ficou com %d peças", after.Tokens)
	}
}

// A mesma posse do apagar e do mostrar à mesa: o id vem do cliente.
func TestAPlaceFromAnotherCampaignCannotBeBuilt(t *testing.T) {
	s, campaign, _ := mesaComTaverna(t)
	ctx := context.Background()
	other := seedCampaign(t, s, seedUser(t, s, "vizinha@t.com"))
	if err := s.boards.Archive(ctx, other, &board.BoardState{Version: 1, Place: "Cripta alheia"}); err != nil {
		t.Fatalf("guardar a cena da outra mesa: %v", err)
	}
	foreign := s.boards.Places(ctx, other)[0]

	if _, err := s.boards.PlaceScene(ctx, campaign, foreign.ID); err == nil {
		t.Error("leu a cena de outra crônica")
	}
	if err := s.boards.SavePlaceScene(ctx, campaign, foreign.ID, &board.BoardState{}); err == nil {
		t.Error("escreveu na cena de outra crônica")
	}
}

/*
O RASCUNHO DE LUGAR: montar a próxima cena FORA da sessão. Ele convive com a
CORTINA e não a substitui — a cortina é durante a sessão, com a mesa presente; o
rascunho é na quinta-feira, sem ninguém conectado.

O gesto do rascunho é o MESMO gesto do tabuleiro vivo, apontado para o acervo em
vez de para a mesa. Por isso ele não ganhou protocolo próprio: cada gesto lê a
cena guardada, aplica a MESMA função pura que a mesa aplica, e grava de volta.
*/

// O gesto do rascunho muda o acervo e NÃO toca na mesa — que é a issue inteira.
//
// A taverna fica ABERTA na mesa durante o caso todo, e o que se monta é a
// cripta: é o cenário da issue, não um arranjo. O mestre prepara sábado enquanto
// a mesa de hoje continua onde está.
func TestEditingThePlaceDraftChangesTheArchiveAndNotTheTable(t *testing.T) {
	s, campaign, session := mesaComTaverna(t)
	ctx := context.Background()
	crypt, err := s.boards.NewPlace(ctx, campaign, "Cripta de Thwor", "crypt")
	if err != nil {
		t.Fatalf("criar a cripta: %v", err)
	}

	scene, err := s.boards.EditPlace(ctx, campaign, crypt.ID, func(b *board.BoardState) error {
		return board.AddToken(b, board.BoardToken{
			Label: "Necromante", X: 4, Y: 4, Footprint: 2,
		}, s.boards.NewID)
	})
	if err != nil {
		t.Fatalf("montar o rascunho: %v", err)
	}
	if len(scene.Tokens) != 1 {
		t.Fatalf("o rascunho voltou com %d peças, esperado 1: %+v", len(scene.Tokens), scene.Tokens)
	}

	// GRAVOU: a próxima abertura acha a peça, e não só a resposta deste gesto.
	back, err := s.boards.PlaceScene(ctx, campaign, crypt.ID)
	if err != nil {
		t.Fatalf("reabrir o rascunho: %v", err)
	}
	if len(back.Tokens) != 1 || back.Tokens[0].Label != "Necromante" {
		t.Errorf("o gesto não ficou gravado: %+v", back.Tokens)
	}
	// E a MESA continua com a peça que ela tinha: preparar não é jogar.
	if onTable := s.boards.Get(ctx, session, defaultTab); onTable == nil || len(onTable.Tokens) != 1 {
		t.Errorf("montar o rascunho mexeu na cena que está na mesa: %+v", onTable)
	}
	if onTable := s.boards.Get(ctx, session, defaultTab); onTable != nil && onTable.Place != "Taverna do Javali" {
		t.Errorf("a cena da mesa virou outra: %q", onTable.Place)
	}
}

// A conferência do `SavePlaceScene` continua valendo pelo caminho do gesto: uma
// mutação que produzisse coordenada absurda é recusada, e o acervo não muda.
func TestAPlaceDraftGestureThatProducesAnAbsurdCoordinateIsRefused(t *testing.T) {
	s, campaign, _ := mesaComTaverna(t)
	ctx := context.Background()
	crypt, err := s.boards.NewPlace(ctx, campaign, "Cripta de Thwor", "crypt")
	if err != nil {
		t.Fatalf("criar a cripta: %v", err)
	}
	if _, err := s.boards.EditPlace(ctx, campaign, crypt.ID, func(b *board.BoardState) error {
		return board.AddToken(b, board.BoardToken{Label: "Porta", X: 3, Y: 3}, s.boards.NewID)
	}); err != nil {
		t.Fatalf("semear a peça: %v", err)
	}

	_, err = s.boards.EditPlace(ctx, campaign, crypt.ID, func(b *board.BoardState) error {
		b.Tokens[0].X = 9_000_000
		return nil
	})

	if err == nil {
		t.Fatal("o gesto do rascunho gravou uma peça fora do limite de sanidade")
	}
	if !strings.Contains(err.Error(), "9000000") {
		t.Errorf("o erro não diz o valor ofensor: %v", err)
	}
	if back, _ := s.boards.PlaceScene(ctx, campaign, crypt.ID); back.Tokens[0].X != 3 {
		t.Errorf("a recusa deixou o acervo mexido: a peça está em x=%d", back.Tokens[0].X)
	}
}

// O lugar ABERTO numa mesa NÃO se monta: seriam duas verdades sobre onde as
// peças estão, e o `Archive` da aba que encerrasse apagaria o rascunho em
// silêncio. Mesma decisão que o acervo já toma com o apagar.
func TestThePlaceOpenOnALiveTableRefusesTheDraft(t *testing.T) {
	s, campaign, session := mesaComTaverna(t)
	ctx := context.Background()
	// A taverna vai para o acervo E CONTINUA na mesa — que é o estado normal de
	// uma sessão em andamento, não um arranjo do teste.
	if err := s.boards.Archive(ctx, campaign, s.boards.Get(ctx, session, defaultTab)); err != nil {
		t.Fatalf("guardar a taverna: %v", err)
	}
	place := s.boards.Places(ctx, campaign)[0]

	_, err := s.boards.EditPlace(ctx, campaign, place.ID, func(b *board.BoardState) error {
		b.Tokens = nil
		return nil
	})

	if err == nil {
		t.Fatal("montou o rascunho de um lugar que está numa mesa")
	}
	if !strings.Contains(err.Error(), "Taverna do Javali") {
		t.Errorf("o erro não diz QUAL lugar está na mesa: %v", err)
	}
	// CONTROLE: outro lugar, na mesma campanha, é montado sem reclamação. Sem
	// ele, uma recusa por qualquer outro motivo — id errado, campanha errada —
	// passaria por "o guarda funcionou".
	crypt, err := s.boards.NewPlace(ctx, campaign, "Cripta de Thwor", "crypt")
	if err != nil {
		t.Fatalf("criar a cripta: %v", err)
	}
	if _, err := s.boards.EditPlace(ctx, campaign, crypt.ID, func(b *board.BoardState) error {
		return nil
	}); err != nil {
		t.Fatalf("o lugar que NÃO está na mesa foi recusado: %v", err)
	}
}

// A trava pega o tabuleiro opened num processo ANTERIOR, pelo disco.
//
// O caso de cima passa pela MEMÓRIA — a taverna está no mapa deste store. Este
// prova a outra fonte, e ela não é redundância: depois de um reinício o mapa
// nasce vazio, e uma trava que só olhasse a memória deixaria montar tudo. A
// segunda `boards.Store` sobre as MESMAS consultas é literalmente o processo que
// subiu de novo e não sabe de nada.
func TestThePlaceOpenBeforeARestartStillRefusesTheDraft(t *testing.T) {
	s, campaign, session := mesaComTaverna(t)
	ctx := context.Background()
	if err := s.boards.Archive(ctx, campaign, s.boards.Get(ctx, session, defaultTab)); err != nil {
		t.Fatalf("guardar a taverna: %v", err)
	}
	place := s.boards.Places(ctx, campaign)[0]
	// A GRAVAÇÃO é o que o servidor de verdade faz depois de todo gesto
	// (`persistBoardAndWarn`), e sem ela o disco não sabe da taverna.
	if dirty, _ := s.boards.Persist(ctx, session, defaultTab); dirty {
		t.Fatal("a gravação do tabuleiro falhou")
	}

	afterRestart := boards.NewStore(s.queries, s.boards.NewID, &events.Bus{})
	_, err := afterRestart.EditPlace(ctx, campaign, place.ID, func(b *board.BoardState) error {
		b.Tokens = nil
		return nil
	})

	if err == nil {
		t.Fatal("depois do reinício, montou o rascunho de um lugar que está numa mesa")
	}
	if !strings.Contains(err.Error(), "Taverna do Javali") {
		t.Errorf("o erro não diz QUAL lugar está na mesa: %v", err)
	}
	// O CONTROLE do controle: o store novo está mesmo vazio de memória, então
	// quem pegou só pode ter sido o disco.
	if len(afterRestart.OpenBoards(ctx, session)) == 0 {
		t.Fatal("o store novo não hidratou nada — o caso mediu outra coisa")
	}
}

// O lugar NOVO nasce vazio, com o chão escolhido, e já aparece no acervo — é o
// caso da issue: preparar a cripta sem ter jogado ela antes.
func TestANewPlaceIsBornEmptyWithTheChosenGround(t *testing.T) {
	s, campaign, _ := mesaComTaverna(t)
	ctx := context.Background()

	place, err := s.boards.NewPlace(ctx, campaign, "Cripta de Thwor", "crypt")
	if err != nil {
		t.Fatalf("criar o lugar: %v", err)
	}
	if place.Tokens != 0 {
		t.Errorf("o lugar novo nasceu com %d peças", place.Tokens)
	}
	scene, err := s.boards.PlaceScene(ctx, campaign, place.ID)
	if err != nil {
		t.Fatalf("abrir o lugar novo: %v", err)
	}
	if scene.Terrain != "crypt" {
		t.Errorf("o chão escolhido não ficou: %q", scene.Terrain)
	}
	if len(s.boards.Places(ctx, campaign)) != 1 {
		t.Errorf("o lugar novo não entrou no acervo: %+v", s.boards.Places(ctx, campaign))
	}
}

// O NOME é a identidade do lugar na campanha (ver o `Archive`): pedir um lugar
// novo com o nome de um que já existe leva ÀQUELE, e não cria um segundo quase
// igual. É a mesma conta que encerrar a taverna duas vezes faz.
func TestANewPlaceWithAnExistingNameOpensThatOne(t *testing.T) {
	s, campaign, session := mesaComTaverna(t)
	ctx := context.Background()
	if err := s.boards.Archive(ctx, campaign, s.boards.Get(ctx, session, defaultTab)); err != nil {
		t.Fatalf("guardar a taverna: %v", err)
	}
	saved := s.boards.Places(ctx, campaign)[0]

	place, err := s.boards.NewPlace(ctx, campaign, "Taverna do Javali", "crypt")
	if err != nil {
		t.Fatalf("pedir o lugar de nome repetido: %v", err)
	}
	if place.ID != saved.ID {
		t.Errorf("nasceu um segundo lugar (%d) com o nome do primeiro (%d)", place.ID, saved.ID)
	}
	if len(s.boards.Places(ctx, campaign)) != 1 {
		t.Errorf("o acervo ficou com dois lugares de mesmo nome: %+v", s.boards.Places(ctx, campaign))
	}
	// E a cena guardada NÃO foi zerada pelo pedido: a peça continua lá.
	scene, _ := s.boards.PlaceScene(ctx, campaign, place.ID)
	if len(scene.Tokens) != 1 {
		t.Errorf("pedir o nome repetido apagou a cena guardada: %+v", scene.Tokens)
	}
}

// Este caso nasceu VERMELHO: o `NewPlace` gravava o `terrain` que chegasse do
// formulário sem passar pelo catálogo, e a cena — que filtra — só corrigia o
// que ELA desenhava. Um cliente velho postando `pedra` gravava `pedra`, e a
// classe `ground-pedra` não existe na folha: o mapa abre sem textura, sem erro
// em lugar nenhum.
func TestAPlaceRefusesAGroundTheStylesheetCannotPaint(t *testing.T) {
	s, campaign, _ := mesaComTaverna(t)
	ctx := context.Background()

	place, err := s.boards.NewPlace(ctx, campaign, "Cripta do guarda", "pedra")
	if err != nil {
		t.Fatalf("criar o lugar: %v", err)
	}
	scene, err := s.boards.PlaceScene(ctx, campaign, place.ID)
	if err != nil {
		t.Fatalf("abrir o lugar: %v", err)
	}
	if scene.Terrain != board.DefaultGround() {
		t.Errorf("o chão gravado foi %q, e a folha não sabe pintá-lo", scene.Terrain)
	}
}
