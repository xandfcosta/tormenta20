package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
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

func (f rulesFixture) putRules(t *testing.T, caller, campaignID int64, body string) (int, string) {
	t.Helper()
	path := "/campaigns/" + strconv.FormatInt(campaignID, 10) + "/rules"
	res := authed(t, f.s, caller, http.MethodPut, path, body)
	return res.Code, res.Body.String()
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

		code, body := f.putRules(t, f.owner, f.campaign, `{"ignoredRules":["carga"]}`)

		if code != http.StatusOK {
			t.Fatalf("code=%d body=%s, queria 200", code, body)
		}
		if !f.cargaIgnorada(t) {
			t.Error("a campanha desligou a carga e a ficha continuou aplicando")
		}
	})

	// Ligar de volta é o mesmo gesto com a lista vazia: substituição, não delta.
	t.Run("mandar a lista vazia religa tudo", func(t *testing.T) {
		f := newRulesFixture(t)
		f.Join(t, f.campaign)
		f.putRules(t, f.owner, f.campaign, `{"ignoredRules":["carga"]}`)

		if code, body := f.putRules(t, f.owner, f.campaign, `{"ignoredRules":[]}`); code != http.StatusOK {
			t.Fatalf("code=%d body=%s, queria 200", code, body)
		}
		if f.cargaIgnorada(t) {
			t.Error("a lista vazia não religou a carga")
		}
	})

	// A chave é do MESTRE. Um jogador que pudesse desligar a carga tiraria a
	// penalidade da própria ficha no meio da sessão.
	t.Run("o jogador não desliga regra nenhuma", func(t *testing.T) {
		f := newRulesFixture(t)
		f.Join(t, f.campaign)

		code, _ := f.putRules(t, f.player, f.campaign, `{"ignoredRules":["carga"]}`)

		if code == http.StatusOK {
			t.Fatal("o jogador conseguiu mexer nas regras da mesa")
		}
		if f.cargaIgnorada(t) {
			t.Errorf("a recusa devolveu %d mas GRAVOU mesmo assim", code)
		}
	})

	// Identificador que o motor não implementa não entra no banco: ele ficaria lá
	// sem interruptor na tela que o desfizesse. A mensagem nomeia o valor e o que
	// se esperava, que é a regra da casa para exceção.
	t.Run("regra desconhecida é recusada nomeando o valor", func(t *testing.T) {
		f := newRulesFixture(t)

		code, body := f.putRules(t, f.owner, f.campaign, `{"ignoredRules":["munição"]}`)

		if code != http.StatusBadRequest {
			t.Fatalf("code=%d, queria 400 — body=%s", code, body)
		}
		if !strings.Contains(body, "munição") || !strings.Contains(body, "carga") {
			t.Errorf("a mensagem não nomeia o valor recusado nem o esperado: %s", body)
		}
		if got := f.s.ignoredRulesOf(context.Background(), f.campaign); len(got) != 0 {
			t.Errorf("gravou %v apesar do 400", got)
		}
	})
}

// A ficha pode pertencer a mais de uma campanha, e as duas podem discordar. A
// mais ESTRITA vence: a regra só sai da ficha quando toda mesa dela dispensou.
// Escolher "a primeira campanha" seria arbitrário e mudaria com a ordem das
// linhas — e afrouxar uma regra por ordenação de banco é o pior desfecho dos
// dois.
func TestFichaEmDuasCampanhasSegueAMaisEstrita(t *testing.T) {
	f := newRulesFixture(t)
	f.Join(t, f.campaign)
	f.Join(t, f.otherCamp)

	f.putRules(t, f.owner, f.campaign, `{"ignoredRules":["carga"]}`)
	if f.cargaIgnorada(t) {
		t.Error("uma campanha desligou e a ficha parou de aplicar — a outra mesa ainda usa a regra")
	}

	f.putRules(t, f.otherOwner, f.otherCamp, `{"ignoredRules":["carga"]}`)
	if !f.cargaIgnorada(t) {
		t.Error("as DUAS campanhas desligaram e a ficha continuou aplicando")
	}
}

// A ficha que não está em campanha nenhuma calcula com tudo em vigor. É o caso
// da ficha avulsa, e a conta que o resolve (`COUNT(DISTINCT) = COUNT(*)`) é
// justamente a que responderia "0 = 0" e desligaria TUDO se estivesse escrita
// sem o filtro — um zero contra zero é a armadilha desta consulta.
func TestFichaSemCampanhaAplicaTodasAsRegras(t *testing.T) {
	f := newRulesFixture(t)

	if f.cargaIgnorada(t) {
		t.Error("ficha fora de campanha nasceu com a carga desligada")
	}
}

// O detalhe da campanha carrega as regras porque é nele que elas se configuram:
// uma segunda rota faria os interruptores piscarem de "tudo ligado" para o
// estado real na primeira pintura.
func TestDetalheDaCampanhaCarregaAsRegras(t *testing.T) {
	f := newRulesFixture(t)
	f.putRules(t, f.owner, f.campaign, `{"ignoredRules":["carga"]}`)

	res := authed(t, f.s, f.owner, http.MethodGet, "/campaigns/"+strconv.FormatInt(f.campaign, 10), "")

	var got struct {
		IgnoredRules []string `json:"ignoredRules"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatalf("ler o detalhe: %v — body=%s", err, res.Body.String())
	}
	if len(got.IgnoredRules) != 1 || got.IgnoredRules[0] != "carga" {
		t.Errorf("ignoredRules=%v, queria [carga]", got.IgnoredRules)
	}
}
