package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"t20engine/serve/web/campaigns"
)

/*
A ABA "LUGARES" da crônica: de onde se chega ao rascunho. O que se prende aqui
é o CAMINHO — sem uma tela que leve até ele, montar a cena fora da sessão é uma
capacidade do domínio que ninguém alcança.
*/

func placesTabUrl(campaign int64) string {
	return "/campanhas/" + strconv.FormatInt(campaign, 10) + "?tab=lugares"
}

// A aba lista o acervo e oferece MONTAR em cada lugar.
func TestThePlacesTabListsTheArchiveAndOffersToBuild(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	campaign := seedCampanha(t, s, owner, "A Queda de Tauron", "")
	crypt, err := s.boards.NewPlace(context.Background(), campaign, "Cripta de Thwor", "crypt")
	if err != nil {
		t.Fatalf("criar o lugar: %v", err)
	}

	body := pedeNaCronica(t, s, owner, http.MethodGet, placesTabUrl(campaign), "").Body.String()

	if !strings.Contains(body, "Cripta de Thwor") {
		t.Error("o lugar guardado não apareceu na aba")
	}
	destination := "/campanhas/" + strconv.FormatInt(campaign, 10) + "/lugares/" + strconv.FormatInt(crypt.ID, 10)
	if !strings.Contains(body, destination) {
		t.Errorf("a aba não leva ao rascunho (%s)", destination)
	}
	// A CONTAGEM diz o que a linha é: "cena vazia" é o lugar que ainda não foi
	// montado, e é o que o mestre acabou de criar.
	if !strings.Contains(body, "cena vazia") {
		t.Error("a linha não diz que o lugar ainda está vazio")
	}
}

// A aba NÃO EXISTE para o jogador, e o acervo não é lido para ele.
//
// Não desenhar é UX; NÃO CARREGAR é a regra, e é a mesma decisão que o link de
// convite desta cena já toma. O que ele veria é a cripta de sábado, com nome.
func TestThePlacesTabDoesNotExistForThePlayer(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	player := seedUser(t, s, "jogador@t20.local")
	campaign := seedCampanha(t, s, owner, "A Queda de Tauron", "")
	hero := seedCharacterAtLevel(t, s, player, "Guerreiro", "Guerreiro", 1, 0, 5)
	seedMember(t, s, campaign, hero)
	if _, err := s.boards.NewPlace(context.Background(), campaign, "Cripta de Thwor", "crypt"); err != nil {
		t.Fatalf("criar o lugar: %v", err)
	}

	body := pedeNaCronica(t, s, player, http.MethodGet, placesTabUrl(campaign), "").Body.String()

	if strings.Contains(body, "Cripta de Thwor") {
		t.Error("o jogador viu o nome de um lugar do acervo do mestre")
	}
	if strings.Contains(body, "Acervo da campanha") {
		t.Error("a aba de lugares foi desenhada para o jogador")
	}
	// CONTROLE: ele recebeu a CRÔNICA, e não uma recusa. `?tab=lugares` cai para
	// a visão geral, como `?tab=config` já cai — sem isto, uma página de erro
	// passaria por "a aba não apareceu".
	if !strings.Contains(body, "A Queda de Tauron") {
		t.Fatal("o jogador não recebeu a crônica: o guarda mediu outra coisa")
	}
}

// O lugar NOVO nasce e a tela LEVA para ele — quem digitou um nome quer montar.
func TestANewPlaceTakesTheMasterStraightToTheDraft(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	campaign := seedCampanha(t, s, owner, "A Queda de Tauron", "")

	form := url.Values{"name": {"Cripta de Thwor"}, "ground": {"crypt"}}
	resp := pedeNaCronica(t, s, owner, http.MethodPost,
		"/campanhas/"+strconv.FormatInt(campaign, 10)+"/lugares/novo", form.Encode())

	if resp.Code != http.StatusSeeOther {
		t.Fatalf("criar o lugar respondeu %d", resp.Code)
	}
	places := s.boards.Places(context.Background(), campaign)
	if len(places) != 1 || places[0].Name != "Cripta de Thwor" {
		t.Fatalf("o lugar não entrou no acervo: %+v", places)
	}
	destination := "/campanhas/" + strconv.FormatInt(campaign, 10) + "/lugares/" + strconv.FormatInt(places[0].ID, 10)
	if to := resp.Header().Get("Location"); to != destination {
		t.Errorf("levou para %q em vez do rascunho (%s)", to, destination)
	}
	scene, _ := s.boards.PlaceScene(context.Background(), campaign, places[0].ID)
	if scene.Terrain != "crypt" {
		t.Errorf("o chão escolhido não ficou: %q", scene.Terrain)
	}
}

// O lugar SEM NOME é recusado, e a frase volta no campo.
//
// O nome é a identidade do lugar na campanha — é por ele que encerrar o
// tabuleiro decide qual lugar sobrescrever —, e um lugar anônimo no acervo é uma
// linha que ninguém consegue escolher.
func TestAPlaceWithoutANameIsRefusedWithTheReasonInTheField(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	campaign := seedCampanha(t, s, owner, "A Queda de Tauron", "")

	form := url.Values{"name": {"   "}, "ground": {"crypt"}}
	resp := pedeNaCronica(t, s, owner, http.MethodPost,
		"/campanhas/"+strconv.FormatInt(campaign, 10)+"/lugares/novo", form.Encode())

	if resp.Code != http.StatusUnprocessableEntity {
		t.Errorf("o lugar sem nome respondeu %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), "dê um nome ao lugar") {
		t.Error("a recusa não voltou escrita no campo")
	}
	if places := s.boards.Places(context.Background(), campaign); len(places) != 0 {
		t.Errorf("nasceu um lugar sem nome: %+v", places)
	}
}

// O lugar QUE ESTÁ NUMA MESA não oferece montar nem apagar — oferece ir até ele.
//
// As duas travas de verdade são do servidor; isto é a cortesia de não oferecer o
// gesto que ele vai recusar. E o motivo é o mesmo dos dois lados: encerrar a aba
// chama o `Archive`, que desfaz tanto o rascunho quanto o apagar.
func TestThePlaceOnATableOffersGoingToItInstead(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	campaign := seedCampanha(t, s, owner, "A Queda de Tauron", "")
	session := seedSessao(t, s, campaign, 1)
	ctx := context.Background()
	if _, err := s.boards.Open(ctx, session, "Taverna do Javali", "tavern"); err != nil {
		t.Fatalf("abrir a taverna: %v", err)
	}
	if err := s.boards.Archive(ctx, campaign, s.boards.Get(ctx, session, "")); err != nil {
		t.Fatalf("guardar a taverna: %v", err)
	}

	body := pedeNaCronica(t, s, owner, http.MethodGet, placesTabUrl(campaign), "").Body.String()

	if !strings.Contains(body, "nesta mesa agora") {
		t.Error("a linha não diz que a cena está numa mesa")
	}
	if !strings.Contains(body, "Ver na mesa") {
		t.Error("a linha não oferece ir até a mesa que mostra a cena")
	}
	// PELO ENDEREÇO e não pelo rótulo: "Montar" é também o botão de submeter do
	// diálogo do lugar novo, que está sempre na página — a primeira versão desta
	// asserção procurava a PALAVRA e teria passado por cima do defeito. O que
	// distingue os dois é o link para o rascunho DAQUELE lugar.
	tavern := placeNamed(t, s.boards.Places(ctx, campaign), "Taverna do Javali")
	tavernDraft := "/campanhas/" + strconv.FormatInt(campaign, 10) +
		"/lugares/" + strconv.FormatInt(tavern.ID, 10)
	if strings.Contains(body, tavernDraft) {
		t.Error("ofereceu montar o rascunho de um lugar que está na mesa — o `Archive` apagaria o trabalho")
	}
	if strings.Contains(body, "Apagar Taverna do Javali") {
		t.Error("ofereceu apagar um lugar que está na mesa — encerrar a aba o traria de volta")
	}
	// CONTROLE: um lugar FORA da mesa, na mesma tela, ganha os dois gestos. Sem
	// ele, uma aba que falhasse em desenhar as linhas passaria neste caso.
	crypt, err := s.boards.NewPlace(ctx, campaign, "Cripta de Thwor", "crypt")
	if err != nil {
		t.Fatalf("criar a cripta: %v", err)
	}
	withBoth := pedeNaCronica(t, s, owner, http.MethodGet, placesTabUrl(campaign), "").Body.String()
	cryptDraft := "/campanhas/" + strconv.FormatInt(campaign, 10) +
		"/lugares/" + strconv.FormatInt(crypt.ID, 10)
	if !strings.Contains(withBoth, cryptDraft) {
		t.Fatal("o lugar fora da mesa também não leva ao rascunho: o guarda mediu uma tela vazia")
	}
	if !strings.Contains(withBoth, "Apagar Cripta de Thwor") {
		t.Fatal("o lugar fora da mesa também não ganhou a lixeira: o guarda mediu uma tela vazia")
	}
}

// APAGAR tira o lugar do acervo e devolve a lista.
func TestRemovingAPlaceReturnsToTheList(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	campaign := seedCampanha(t, s, owner, "A Queda de Tauron", "")
	crypt, err := s.boards.NewPlace(context.Background(), campaign, "Cripta de Thwor", "crypt")
	if err != nil {
		t.Fatalf("criar o lugar: %v", err)
	}

	resp := pedeNaCronica(t, s, owner, http.MethodPost,
		"/campanhas/"+strconv.FormatInt(campaign, 10)+"/lugares/"+strconv.FormatInt(crypt.ID, 10)+"/excluir", "")

	if resp.Code != http.StatusSeeOther {
		t.Fatalf("apagar respondeu %d", resp.Code)
	}
	if to := resp.Header().Get("Location"); to != placesTabUrl(campaign) {
		t.Errorf("voltou para %q em vez da lista", to)
	}
	if places := s.boards.Places(context.Background(), campaign); len(places) != 0 {
		t.Errorf("o lugar continuou no acervo: %+v", places)
	}
}

// O acervo NÃO é LIDO fora da aba dele.
//
// Este caso mede a VIEW e não o HTML, e a diferença foi provada por sabotagem:
// procurar o nome do lugar na visão geral deixa o guarda VERDE com o `if` que
// segura a leitura removido, porque ele afirma "não é lido" e mede "não é
// desenhado" — nenhuma outra aba desenha lugar de qualquer jeito. Duas frases
// que soam iguais e não são.
//
// Uma crônica de dois anos tem dezenas de lugares, e ler o acervo para desenhar
// três sinetes é o custo que a condição existe para não pagar.
func TestTheArchiveIsNotReadOutsideItsTab(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	campaign := seedCampanha(t, s, owner, "A Queda de Tauron", "")
	if _, err := s.boards.NewPlace(context.Background(), campaign, "Cripta de Thwor", "crypt"); err != nil {
		t.Fatalf("criar o lugar: %v", err)
	}
	scene := campaigns.New(s.campaignsHost(), s.sessionAccess(), s.campaignDirectory(), s.campaignLifecycle(), s.campaignSeating(), s.boards)

	vision, err := scene.LoadOne(context.Background(), owner, s.ehAdmin(t, owner), campaign, "")
	if err != nil {
		t.Fatalf("carregar a visão geral: %v", err)
	}
	if len(vision.Places) != 0 {
		t.Errorf("a visão geral leu %d lugares do acervo", len(vision.Places))
	}

	// O CONTROLE, e ele é o denominador: na aba dos lugares a leitura ACONTECE.
	// Sem ele, uma porta que devolvesse lista vazia sempre passaria por
	// "carregamento sob demanda funcionando".
	aba, err := scene.LoadOne(context.Background(), owner, s.ehAdmin(t, owner), campaign, "lugares")
	if err != nil {
		t.Fatalf("carregar a aba dos lugares: %v", err)
	}
	if len(aba.Places) != 1 {
		t.Fatalf("a aba dos lugares leu %d lugares — o guarda mediu uma porta muda", len(aba.Places))
	}
	if len(aba.Chaos) == 0 {
		t.Error("a aba não recebeu as aparências para o formulário do lugar novo")
	}
}
