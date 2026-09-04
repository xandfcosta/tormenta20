package api

import (
	"context"
	"t20engine/plataforma"
	"testing"

	"t20engine/db/sqlcgen"
)

// As regras opcionais pelo ROUTER real (ALE-221).
//
// Duas coisas se provam aqui e em lugar nenhum mais: que a chave é do MESTRE, e
// que a ficha resolve a mesa dela sozinha. A segunda é a que não cabe no motor —
// o motor recebe as regras já resolvidas, e quem as resolve é este carregamento.

type rulesFixture struct {
	s        *Server
	owner    int64
	player   int64
	campaign int64
	// segunda campanha do MESMO jogador, para provar a resolução da ficha que
	// pertence a duas mesas.
	otherOwner int64
	otherCamp  int64
	pc         int64
}

func newRulesFixture(t *testing.T) rulesFixture {
	t.Helper()
	s := newTestServer(t)
	owner := seedUser(t, s, "mestre@t.com")
	player := seedUser(t, s, "jogador@t.com")
	otherOwner := seedUser(t, s, "outro-mestre@t.com")
	campaign := seedCampaign(t, s, owner)
	otherCamp := seedCampaign(t, s, otherOwner)
	pc := seedCharacter(t, s, player, "Herói", 20, 30, 5, 10)
	return rulesFixture{
		s: s, owner: owner, player: player, campaign: campaign,
		otherOwner: otherOwner, otherCamp: otherCamp, pc: pc,
	}
}

func (f rulesFixture) Join(t *testing.T, campaignID int64) {
	t.Helper()
	if _, err := f.s.queries.CreateMember(context.Background(), sqlcgen.CreateMemberParams{
		Campaignid: campaignID, Characterid: f.pc, Role: "player", Addedat: plataforma.NowISO(),
	}); err != nil {
		t.Fatalf("entrar na campanha %d: %v", campaignID, err)
	}
}

// putRules chama a REGRA direto, e não a rota.
//
// Ela batia em `PUT /campaigns/{id}/rules`, que saiu na ALE-277 com as outras
// sem consumidor. O que estes casos prendem nunca foi o transporte: é a mais
// ESTRITA vencendo entre duas mesas, e a ficha avulsa aplicando tudo. A cena das
// campanhas grava pelo mesmo `saveIgnoredRules`, pela porta.
//
// O `caller` sai da assinatura junto com a rota: a AUTORIZAÇÃO era do handler, e
// a cena tem a dela (`RequesterIsAdmin` e o dono da campanha). Uma regra, uma
// camada.
func (f rulesFixture) putRules(t *testing.T, campaignID int64, regras ...string) error {
	t.Helper()
	return f.s.campaignRules().saveIgnoredRules(context.Background(), campaignID, regras)
}

// cargaIgnorada pergunta ao CARREGAMENTO da ficha, e não à tabela: é o que o
// motor recebe, e é onde um defeito de resolução aparece.
func (f rulesFixture) cargaIgnorada(t *testing.T) bool {
	t.Helper()
	row, err := f.s.queries.GetCharacter(context.Background(), f.pc)
	if err != nil {
		t.Fatalf("ler ficha: %v", err)
	}
	dto, err := f.s.LoadCharacter(context.Background(), row)
	if err != nil {
		t.Fatalf("carregar ficha: %v", err)
	}
	return dto.IgnoredRules.Carga
}

func TestReplaceCampaignRules(t *testing.T) {
	t.Run("o mestre desliga a carga e a ficha do jogador para de aplicá-la", func(t *testing.T) {
		f := newRulesFixture(t)
		f.Join(t, f.campaign)
		if f.cargaIgnorada(t) {
			t.Fatal("a carga já nascia desligada — o padrão do livro é ela EM VIGOR")
		}

		if err := f.putRules(t, f.campaign, "carga"); err != nil {
			t.Fatalf("desligar a carga falhou: %v", err)
		}
		if !f.cargaIgnorada(t) {
			t.Error("a campanha desligou a carga e a ficha continuou aplicando")
		}
	})

	// Ligar de volta é o mesmo gesto com a lista vazia: substituição, não delta.
	t.Run("mandar a lista vazia religa tudo", func(t *testing.T) {
		f := newRulesFixture(t)
		f.Join(t, f.campaign)
		f.putRules(t, f.campaign, "carga")

		if err := f.putRules(t, f.campaign); err != nil {
			t.Fatalf("religar tudo falhou: %v", err)
		}
		if f.cargaIgnorada(t) {
			t.Error("a lista vazia não religou a carga")
		}
	})

	// Aqui morava o subcaso "o jogador não desliga regra nenhuma". Ele provava a
	// AUTORIZAÇÃO do handler, que saiu com a rota na ALE-277 — e a garantia
	// continua onde ela é usada: a cena das campanhas só desenha os
	// interruptores para o dono, e o comando dela confere. Uma regra, uma camada.

	// Aqui morava o subcaso "regra desconhecida é recusada nomeando o valor". Ele
	// media o 400 de uma rota JSON que saiu na ALE-277, e a garantia desceu para
	// onde a regra MORA: `campaign.TestAnUnknownRuleIsRefusedNamingTheValue`.
}

// A ficha pode pertencer a mais de uma campanha, e as duas podem discordar. A
// mais ESTRITA vence: a regra só sai da ficha quando toda mesa dela dispensou.
// Escolher "a primeira campanha" seria arbitrário e mudaria com a ordem das
// linhas — e afrouxar uma regra por ordenação de banco é o pior desfecho dos
// dois.
func TestASheetInTwoCampaignsFollowsTheStrictestOne(t *testing.T) {
	f := newRulesFixture(t)
	f.Join(t, f.campaign)
	f.Join(t, f.otherCamp)

	f.putRules(t, f.campaign, "carga")
	if f.cargaIgnorada(t) {
		t.Error("uma campanha desligou e a ficha parou de aplicar — a outra mesa ainda usa a regra")
	}

	f.putRules(t, f.otherCamp, "carga")
	if !f.cargaIgnorada(t) {
		t.Error("as DUAS campanhas desligaram e a ficha continuou aplicando")
	}
}

// A ficha que não está em campanha nenhuma calcula com tudo em vigor. É o caso
// da ficha avulsa, e a conta que o resolve (`COUNT(DISTINCT) = COUNT(*)`) é
// justamente a que responderia "0 = 0" e desligaria TUDO se estivesse escrita
// sem o filtro — um zero contra zero é a armadilha desta consulta.
func TestASheetWithoutACampaignAppliesEveryRule(t *testing.T) {
	f := newRulesFixture(t)

	if f.cargaIgnorada(t) {
		t.Error("ficha fora de campanha nasceu com a carga desligada")
	}
}

// Aqui morava o TestTheCampaignDetailLoadsTheRules, que lia as regras pelo
// `GET /campaigns/{id}` — rota que saiu na ALE-277. A garantia é da cena das
// campanhas, que desenha os interruptores no estado real na primeira pintura, e
// ela tem guarda lá.
