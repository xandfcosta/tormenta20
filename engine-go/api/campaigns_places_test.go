package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"t20engine/web/campaigns"
)

/*
A ABA "LUGARES" da crônica (ALE-292): de onde se chega ao rascunho.

Ela é o caminho que faltava. A capacidade de montar a cena fora da sessão existia
no domínio desde a ALE-191, e o que a mantinha inalcançável não era regra
faltando — era não haver tela nenhuma que levasse até ela.
*/

func placesTabUrl(campanha int64) string {
	return "/campanhas/" + strconv.FormatInt(campanha, 10) + "?tab=lugares"
}

// A aba lista o acervo e oferece MONTAR em cada lugar.
func TestThePlacesTabListsTheArchiveAndOffersToBuild(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")
	campanha := seedCampanha(t, s, dono, "A Queda de Tauron", "")
	cripta, err := s.boards.NewPlace(context.Background(), campanha, "Cripta de Thwor", "cripta")
	if err != nil {
		t.Fatalf("criar o lugar: %v", err)
	}

	corpo := pedeNaCronica(t, s, dono, http.MethodGet, placesTabUrl(campanha), "").Body.String()

	if !strings.Contains(corpo, "Cripta de Thwor") {
		t.Error("o lugar guardado não apareceu na aba")
	}
	destino := "/campanhas/" + strconv.FormatInt(campanha, 10) + "/lugares/" + strconv.FormatInt(cripta.ID, 10)
	if !strings.Contains(corpo, destino) {
		t.Errorf("a aba não leva ao rascunho (%s)", destino)
	}
	// A CONTAGEM diz o que a linha é: "cena vazia" é o lugar que ainda não foi
	// montado, e é o que o mestre acabou de criar.
	if !strings.Contains(corpo, "cena vazia") {
		t.Error("a linha não diz que o lugar ainda está vazio")
	}
}

// A aba NÃO EXISTE para o jogador, e o acervo não é lido para ele.
//
// Não desenhar é UX; NÃO CARREGAR é a regra, e é a mesma decisão que o link de
// convite desta cena já toma. O que ele veria é a cripta de sábado, com nome.
func TestThePlacesTabDoesNotExistForThePlayer(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")
	jogador := seedUser(t, s, "jogador@t20.local")
	campanha := seedCampanha(t, s, dono, "A Queda de Tauron", "")
	heroi := seedCharacterAtLevel(t, s, jogador, "Guerreiro", 1, 10, 10, 5, 10)
	seedMember(t, s, campanha, heroi)
	if _, err := s.boards.NewPlace(context.Background(), campanha, "Cripta de Thwor", "cripta"); err != nil {
		t.Fatalf("criar o lugar: %v", err)
	}

	corpo := pedeNaCronica(t, s, jogador, http.MethodGet, placesTabUrl(campanha), "").Body.String()

	if strings.Contains(corpo, "Cripta de Thwor") {
		t.Error("o jogador viu o nome de um lugar do acervo do mestre")
	}
	if strings.Contains(corpo, "Acervo da campanha") {
		t.Error("a aba de lugares foi desenhada para o jogador")
	}
	// CONTROLE: ele recebeu a CRÔNICA, e não uma recusa. `?tab=lugares` cai para
	// a visão geral, como `?tab=config` já cai — sem isto, uma página de erro
	// passaria por "a aba não apareceu".
	if !strings.Contains(corpo, "A Queda de Tauron") {
		t.Fatal("o jogador não recebeu a crônica: o guarda mediu outra coisa")
	}
}

// O lugar NOVO nasce e a tela LEVA para ele — quem digitou um nome quer montar.
func TestANewPlaceTakesTheMasterStraightToTheDraft(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")
	campanha := seedCampanha(t, s, dono, "A Queda de Tauron", "")

	form := url.Values{"name": {"Cripta de Thwor"}, "ground": {"cripta"}}
	resp := pedeNaCronica(t, s, dono, http.MethodPost,
		"/campanhas/"+strconv.FormatInt(campanha, 10)+"/lugares/novo", form.Encode())

	if resp.Code != http.StatusSeeOther {
		t.Fatalf("criar o lugar respondeu %d", resp.Code)
	}
	lugares := s.boards.Places(context.Background(), campanha)
	if len(lugares) != 1 || lugares[0].Name != "Cripta de Thwor" {
		t.Fatalf("o lugar não entrou no acervo: %+v", lugares)
	}
	destino := "/campanhas/" + strconv.FormatInt(campanha, 10) + "/lugares/" + strconv.FormatInt(lugares[0].ID, 10)
	if para := resp.Header().Get("Location"); para != destino {
		t.Errorf("levou para %q em vez do rascunho (%s)", para, destino)
	}
	cena, _ := s.boards.PlaceScene(context.Background(), campanha, lugares[0].ID)
	if cena.Terrain != "cripta" {
		t.Errorf("o chão escolhido não ficou: %q", cena.Terrain)
	}
}

// O lugar SEM NOME é recusado, e a frase volta no campo.
//
// O nome é a identidade do lugar na campanha — é por ele que encerrar o
// tabuleiro decide qual lugar sobrescrever —, e um lugar anônimo no acervo é uma
// linha que ninguém consegue escolher.
func TestAPlaceWithoutANameIsRefusedWithTheReasonInTheField(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")
	campanha := seedCampanha(t, s, dono, "A Queda de Tauron", "")

	form := url.Values{"name": {"   "}, "ground": {"cripta"}}
	resp := pedeNaCronica(t, s, dono, http.MethodPost,
		"/campanhas/"+strconv.FormatInt(campanha, 10)+"/lugares/novo", form.Encode())

	if resp.Code != http.StatusUnprocessableEntity {
		t.Errorf("o lugar sem nome respondeu %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), "dê um nome ao lugar") {
		t.Error("a recusa não voltou escrita no campo")
	}
	if lugares := s.boards.Places(context.Background(), campanha); len(lugares) != 0 {
		t.Errorf("nasceu um lugar sem nome: %+v", lugares)
	}
}

// O lugar QUE ESTÁ NUMA MESA não oferece montar nem apagar — oferece ir até ele.
//
// As duas travas de verdade são do servidor; isto é a cortesia de não oferecer o
// gesto que ele vai recusar. E o motivo é o mesmo dos dois lados: encerrar a aba
// chama o `Archive`, que desfaz tanto o rascunho quanto o apagar.
func TestThePlaceOnATableOffersGoingToItInstead(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")
	campanha := seedCampanha(t, s, dono, "A Queda de Tauron", "")
	sessao := seedSessao(t, s, campanha, 1)
	ctx := context.Background()
	if _, err := s.boards.Open(ctx, sessao, "Taverna do Javali", "taverna"); err != nil {
		t.Fatalf("abrir a taverna: %v", err)
	}
	if err := s.boards.Archive(ctx, campanha, s.boards.Get(ctx, sessao, "")); err != nil {
		t.Fatalf("guardar a taverna: %v", err)
	}

	corpo := pedeNaCronica(t, s, dono, http.MethodGet, placesTabUrl(campanha), "").Body.String()

	if !strings.Contains(corpo, "nesta mesa agora") {
		t.Error("a linha não diz que a cena está numa mesa")
	}
	if !strings.Contains(corpo, "Ver na mesa") {
		t.Error("a linha não oferece ir até a mesa que mostra a cena")
	}
	// PELO ENDEREÇO e não pelo rótulo: "Montar" é também o botão de submeter do
	// diálogo do lugar novo, que está sempre na página — a primeira versão desta
	// asserção procurava a PALAVRA e teria passado por cima do defeito. O que
	// distingue os dois é o link para o rascunho DAQUELE lugar.
	taverna := placeNamed(t, s.boards.Places(ctx, campanha), "Taverna do Javali")
	rascunhoDaTaverna := "/campanhas/" + strconv.FormatInt(campanha, 10) +
		"/lugares/" + strconv.FormatInt(taverna.ID, 10)
	if strings.Contains(corpo, rascunhoDaTaverna) {
		t.Error("ofereceu montar o rascunho de um lugar que está na mesa — o `Archive` apagaria o trabalho")
	}
	if strings.Contains(corpo, "Apagar Taverna do Javali") {
		t.Error("ofereceu apagar um lugar que está na mesa — encerrar a aba o traria de volta")
	}
	// CONTROLE: um lugar FORA da mesa, na mesma tela, ganha os dois gestos. Sem
	// ele, uma aba que falhasse em desenhar as linhas passaria neste caso.
	cripta, err := s.boards.NewPlace(ctx, campanha, "Cripta de Thwor", "cripta")
	if err != nil {
		t.Fatalf("criar a cripta: %v", err)
	}
	comAsDuas := pedeNaCronica(t, s, dono, http.MethodGet, placesTabUrl(campanha), "").Body.String()
	rascunhoDaCripta := "/campanhas/" + strconv.FormatInt(campanha, 10) +
		"/lugares/" + strconv.FormatInt(cripta.ID, 10)
	if !strings.Contains(comAsDuas, rascunhoDaCripta) {
		t.Fatal("o lugar fora da mesa também não leva ao rascunho: o guarda mediu uma tela vazia")
	}
	if !strings.Contains(comAsDuas, "Apagar Cripta de Thwor") {
		t.Fatal("o lugar fora da mesa também não ganhou a lixeira: o guarda mediu uma tela vazia")
	}
}

// APAGAR tira o lugar do acervo e devolve a lista.
func TestRemovingAPlaceReturnsToTheList(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")
	campanha := seedCampanha(t, s, dono, "A Queda de Tauron", "")
	cripta, err := s.boards.NewPlace(context.Background(), campanha, "Cripta de Thwor", "cripta")
	if err != nil {
		t.Fatalf("criar o lugar: %v", err)
	}

	resp := pedeNaCronica(t, s, dono, http.MethodPost,
		"/campanhas/"+strconv.FormatInt(campanha, 10)+"/lugares/"+strconv.FormatInt(cripta.ID, 10)+"/excluir", "")

	if resp.Code != http.StatusSeeOther {
		t.Fatalf("apagar respondeu %d", resp.Code)
	}
	if para := resp.Header().Get("Location"); para != placesTabUrl(campanha) {
		t.Errorf("voltou para %q em vez da lista", para)
	}
	if lugares := s.boards.Places(context.Background(), campanha); len(lugares) != 0 {
		t.Errorf("o lugar continuou no acervo: %+v", lugares)
	}
}

// O acervo NÃO é LIDO fora da aba dele.
//
// Este caso mede a VIEW e não o HTML, e a diferença foi medida: a primeira
// versão procurava o nome do lugar na visão geral, e ele nunca aparece ali de
// qualquer jeito — nenhuma outra aba desenha lugar. Sabotar o `if` que segura a
// leitura deixou o guarda VERDE, porque ele afirmava "não é lido" e media "não é
// desenhado". Duas frases que soam iguais e não são.
//
// Uma crônica de dois anos tem dezenas de lugares, e ler o acervo para desenhar
// três sinetes é o custo que a condição existe para não pagar.
func TestTheArchiveIsNotReadOutsideItsTab(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")
	campanha := seedCampanha(t, s, dono, "A Queda de Tauron", "")
	if _, err := s.boards.NewPlace(context.Background(), campanha, "Cripta de Thwor", "cripta"); err != nil {
		t.Fatalf("criar o lugar: %v", err)
	}
	cena := campaigns.New(s.campaignsHost())

	visao, err := cena.LoadOne(context.Background(), dono, s.ehAdmin(t, dono), campanha, "")
	if err != nil {
		t.Fatalf("carregar a visão geral: %v", err)
	}
	if len(visao.Lugares) != 0 {
		t.Errorf("a visão geral leu %d lugares do acervo", len(visao.Lugares))
	}

	// O CONTROLE, e ele é o denominador: na aba dos lugares a leitura ACONTECE.
	// Sem ele, uma porta que devolvesse lista vazia sempre passaria por
	// "carregamento sob demanda funcionando".
	aba, err := cena.LoadOne(context.Background(), dono, s.ehAdmin(t, dono), campanha, "lugares")
	if err != nil {
		t.Fatalf("carregar a aba dos lugares: %v", err)
	}
	if len(aba.Lugares) != 1 {
		t.Fatalf("a aba dos lugares leu %d lugares — o guarda mediu uma porta muda", len(aba.Lugares))
	}
	if len(aba.Chaos) == 0 {
		t.Error("a aba não recebeu as aparências para o formulário do lugar novo")
	}
}
