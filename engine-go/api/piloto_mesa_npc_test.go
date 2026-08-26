package api

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"t20engine/db/sqlcgen"
)

// OS NPCs DA CAMPANHA (ALE-269, superfície 6b) — os guardas.
//
// O que se prende: quem pode mexer, o que a cópia guarda, e as duas separações
// que a issue existe para manter — elenco não é fila, e campanha não é sessão.

// TestGuardarOVerbeteCriaOBlocoDoMestre é o caminho principal, ponta a ponta.
//
// O bloco tem de nascer COM OS NÚMEROS DO LIVRO: guardar uma cópia vazia
// obrigaria o mestre a redigitar o que ele acabou de ver na tela, que é o
// oposto do gesto.
func TestGuardarOVerbeteCriaOBlocoDoMestre(t *testing.T) {
	f := novoPiloto(t)

	f.posta(t, f.mestre, f.urlDaMesa()+"/elenco/npc/do-verbete",
		`{"criatura":"ogro","nomedonpc":"Ogro Capitão"}`)

	npcs := f.elencoNoBanco(t)
	if len(npcs) != 1 {
		t.Fatalf("o elenco tem %d NPCs, queria 1", len(npcs))
	}
	if npcs[0].Name != "Ogro Capitão" {
		t.Errorf("o nome guardado é %q", npcs[0].Name)
	}
	var bloco CreatureBlock
	if err := json.Unmarshal([]byte(npcs[0].Block), &bloco); err != nil {
		t.Fatalf("o bloco guardado está ilegível: %v", err)
	}
	if bloco.HP <= 0 || bloco.Defesa <= 0 {
		t.Errorf("o bloco nasceu vazio: PV %d, Defesa %d", bloco.HP, bloco.Defesa)
	}
	// A ORIGEM fica gravada, e é ela que deixa a tela dizer "cópia de ogro"
	// depois de o mestre renomear. Sem ela, "Ogro Capitão" perde o fio até o
	// livro no instante em que ganha nome próprio.
	if bloco.SourceMonsterID != "ogro" {
		t.Errorf("a origem não foi guardada: %q", bloco.SourceMonsterID)
	}
}

// TestONomeVazioCaiNoNomeDoLivro — guardar "Ogro" como "Ogro" é o caso comum, e
// obrigar a digitar faria o mestre repetir o que a tela já mostra.
func TestONomeVazioCaiNoNomeDoLivro(t *testing.T) {
	f := novoPiloto(t)

	f.posta(t, f.mestre, f.urlDaMesa()+"/elenco/npc/do-verbete", `{"criatura":"ogro","nomedonpc":"   "}`)

	npcs := f.elencoNoBanco(t)
	if len(npcs) != 1 || npcs[0].Name == "" {
		t.Fatalf("o NPC não nasceu com o nome do livro: %+v", npcs)
	}
	if strings.TrimSpace(npcs[0].Name) != npcs[0].Name {
		t.Errorf("o nome guardado veio com espaços: %q", npcs[0].Name)
	}
}

// TestOElencoEhDaCampanhaENaoDaSessao é a separação que a ALE-212 nomeia.
//
// "Os NPCs voltam semana que vem" só é verdade se eles não morrerem com a
// sessão. Guardado numa sessão, o NPC tem de aparecer na OUTRA da mesma
// campanha — e um guarda que olhasse só a sessão de origem passaria verde sobre
// um elenco que se perde toda noite.
func TestOElencoEhDaCampanhaENaoDaSessao(t *testing.T) {
	f := novoPiloto(t)
	outraSessao := seedSession(t, f.s, f.campaignID)

	f.posta(t, f.mestre, f.urlDaMesa()+"/elenco/npc/do-verbete", `{"criatura":"ogro"}`)

	// A view da OUTRA sessão da mesma campanha tem de enxergar o mesmo NPC.
	user, err := f.s.queries.GetUserByID(t.Context(), f.mestre)
	if err != nil {
		t.Fatalf("mestre: %v", err)
	}
	view, _, err := f.s.loadMesaView(t.Context(), AuthUser{ID: user.ID}, f.campaignID, outraSessao)
	if err != nil {
		t.Fatalf("montar a view da outra sessão: %v", err)
	}
	if len(view.NPCs) != 1 {
		t.Errorf("a outra sessão da mesma campanha vê %d NPCs, queria 1", len(view.NPCs))
	}
}

// TestOMestreNaoAlcancaOElencoDeOutraCampanha — o id vem do CAMINHO.
//
// O elenco guarda a PREPARAÇÃO da campanha, que é o material mais privado que o
// mestre tem: o chefe da semana que vem está ali. Alcançar o de outra mesa é
// pior que ver a fila dela.
func TestOMestreNaoAlcancaOElencoDeOutraCampanha(t *testing.T) {
	f := novoPiloto(t)
	outraCampanha := seedCampaign(t, f.s, f.jogador)
	agora := "2026-01-01T00:00:00.000Z"
	alheio, err := f.s.queries.CreateCampaignCreature(t.Context(), sqlcgen.CreateCampaignCreatureParams{
		Campaignid: outraCampanha, Name: "Segredo alheio", Block: `{"nd":1,"tipo":"humanoide","size":"medio","hp":10}`,
		Createdat: agora, Updatedat: agora,
	})
	if err != nil {
		t.Fatalf("semear o NPC alheio: %v", err)
	}

	corpo := f.posta(t, f.mestre,
		f.urlDaMesa()+"/elenco/npc/"+strconv.FormatInt(alheio.ID, 10)+"/apagar", "{}")

	if !strings.Contains(corpo, "não é desta campanha") {
		t.Errorf("a recusa não veio: %s", primeirasLinhas(corpo, 5))
	}
	// O CONTROLE: recusar DEPOIS de apagar seria pior que não recusar.
	if _, err := f.s.queries.GetCampaignCreature(t.Context(), alheio.ID); err != nil {
		t.Error("o NPC da outra campanha foi apagado apesar da recusa")
	}
}

// TestApagarDoElencoNaoTiraDaFila é a outra separação: elenco não é fila.
//
// Apagar o NPC do elenco e tirar a linha do combate respondem a duas perguntas
// — "ele não volta mais" e "ele saiu desta cena". Juntá-las faria o mestre
// perder o combatente EM CURSO ao arrumar a preparação, no meio da noite.
func TestApagarDoElencoNaoTiraDaFila(t *testing.T) {
	f := novoPiloto(t)
	f.posta(t, f.mestre, f.urlDaMesa()+"/elenco/npc/do-verbete", `{"criatura":"ogro"}`)
	npcs := f.elencoNoBanco(t)
	if len(npcs) != 1 {
		t.Fatalf("o NPC não foi guardado")
	}
	rota := f.urlDaMesa() + "/elenco/npc/" + strconv.FormatInt(npcs[0].ID, 10)
	f.posta(t, f.mestre, rota+"/na-fila", "{}")
	if n := len(f.s.sessions.GetState(f.sessionID).Initiative); n != 1 {
		t.Fatalf("o NPC não entrou na fila (%d linhas) — o resto do teste mediria nada", n)
	}

	f.posta(t, f.mestre, rota+"/apagar", "{}")

	if n := len(f.s.sessions.GetState(f.sessionID).Initiative); n != 1 {
		t.Errorf("apagar do elenco tirou o combatente da cena: a fila tem %d linhas", n)
	}
}

// TestOJogadorNaoMexeNoElencoDaCampanha — o papel, no servidor.
func TestOJogadorNaoMexeNoElencoDaCampanha(t *testing.T) {
	f := novoPiloto(t)

	rec := f.pede(t, f.jogador, "POST", f.urlDaMesa()+"/elenco/npc/do-verbete", `{"criatura":"ogro"}`)

	if rec.Code != 403 {
		t.Errorf("o jogador guardou NPC no elenco do mestre: %d", rec.Code)
	}
	if npcs := f.elencoNoBanco(t); len(npcs) != 0 {
		t.Error("a recusa veio depois da escrita")
	}
}

func (f pilotoFixture) elencoNoBanco(t *testing.T) []sqlcgen.CampaignCreature {
	t.Helper()
	linhas, err := f.s.queries.ListCampaignCreatures(t.Context(), f.campaignID)
	if err != nil {
		t.Fatalf("ler o elenco: %v", err)
	}
	return linhas
}
