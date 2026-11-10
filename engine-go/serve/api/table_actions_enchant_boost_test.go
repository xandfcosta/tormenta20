package api

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// O GASTO DE 2 PM DO ENCANTO NA LINHA DE ATAQUE (p335-336). A tira "Com" da superfície Ações oferece o
// gasto só na linha da arma que tem o encanto — a outra mão não o leva —, e o "Rolar" daquela linha o leva
// até o provisório da mesa, com o PM que a proposta cobra. O que o golpe pago soma é do motor
// (`enchant_boost_rules_test.go`); aqui se prende a LIGAÇÃO.
func TestTheEnchantBoostIsOfferedOnlyOnItsWeaponAndTravelsInTheStrike(t *testing.T) {
	f, barbaro, goblin := barbarianOnTurn(t)
	sword := itemSemeia(t, f, barbaro, "espada-longa", "Espada longa", "wielded")
	if err := f.s.characterPlays().EnchantItem(context.Background(), sword, []string{"encanto-venenosa"}, nil); err != nil {
		t.Fatalf("encantar a espada: %v", err)
	}
	venomous := engine.EnchantBoostID(sword, "encanto-venenosa")

	screen := html.UnescapeString(f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/atacar/0?embutida=1", barbaro), ""))
	offered := regexp.MustCompile(`\$boost_weapon = (\d+); \$boost_power = "`+regexp.QuoteMeta(venomous)+`"`).
		FindAllStringSubmatch(screen, -1)
	rows := map[string]bool{}
	for _, found := range offered {
		rows[found[1]] = true
	}
	if len(rows) != 1 {
		t.Fatalf("o gasto da Venenosa está em %d linhas de ataque (%v); quer só na da espada:\n%s", len(rows), rows, recorte(screen))
	}
	// O AVISO DA RODADA vai na tira: fora da iniciativa nada recusa o segundo gasto, e é a linha que lembra
	// a regra (decisão do dono).
	if !strings.Contains(screen, "uma vez por rodada") {
		t.Error("a tira do gasto da Venenosa não avisa que ele é uma vez por rodada")
	}
	// O QUE O GOLPE PAGO PREVÊ vai na tira: quem não sabe o encanto de cor não precisa abrir o livro.
	if !strings.Contains(screen, "no acerto: Envenenado (1d12 PV por rodada, por 3 rodadas)") {
		t.Error("a tira do gasto da Venenosa não diz o que o golpe pago prevê")
	}
	var weapon int
	for row := range rows {
		weapon, _ = strconv.Atoi(row)
	}

	chosen := fmt.Sprintf(`{"turn_target":%q,"boost_weapon":%d,"boost_power":%q,"boost_steps":0,"boost_split":""}`,
		goblin, weapon, venomous)
	if refusal := sceneRefusal(f.posts(t, f.player,
		fmt.Sprintf("/personagens/%d/acoes/atacar/%d?embutida=1", barbaro, weapon), chosen)); refusal != "" {
		t.Fatalf("o golpe com o gasto da Venenosa foi recusado: %q", refusal)
	}
	pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pending == nil || pending.BoostPowerID != venomous || pending.BoostPM != 2 {
		t.Errorf("o provisório levou %+v; quer o gasto da Venenosa por 2 PM (p336)", pending)
	}
}

// O PV DA TUMULAR SAI NA PROPOSTA, junto dos 2 PM, e o cancelamento do mestre devolve os dois (p336,
// decisão do dono): um provisório cancelado não custa nada.
func TestTheTomblikeBoostTakesHitPointsAndTheCancelGivesThemBack(t *testing.T) {
	f, barbaro, goblin := barbarianOnTurn(t)
	sword := itemSemeia(t, f, barbaro, "espada-longa", "Espada longa", "wielded")
	if err := f.s.characterPlays().EnchantItem(context.Background(), sword, []string{"encanto-tumular"}, nil); err != nil {
		t.Fatalf("encantar a espada: %v", err)
	}
	tomblike := engine.EnchantBoostID(sword, "encanto-tumular")
	screen := html.UnescapeString(f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/atacar/0?embutida=1", barbaro), ""))
	found := regexp.MustCompile(`\$boost_weapon = (\d+); \$boost_power = "` + regexp.QuoteMeta(tomblike) + `"`).FindStringSubmatch(screen)
	if found == nil {
		t.Fatalf("a linha da espada não oferece o gasto da Tumular:\n%s", recorte(screen))
	}
	before := poolsOf(t, f.s, barbaro)

	chosen := fmt.Sprintf(`{"turn_target":%q,"boost_weapon":%s,"boost_power":%q,"boost_steps":0,"boost_split":""}`,
		goblin, found[1], tomblike)
	if refusal := sceneRefusal(f.posts(t, f.player,
		fmt.Sprintf("/personagens/%d/acoes/atacar/%s?embutida=1", barbaro, found[1]), chosen)); refusal != "" {
		t.Fatalf("o golpe com o gasto da Tumular foi recusado: %q", refusal)
	}
	pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pending == nil || pending.BoostHitPoints < 1 || pending.BoostHitPoints > 8 {
		t.Fatalf("o provisório leva %+v; quer o 1d8 de PV rolado", pending)
	}
	paid := poolsOf(t, f.s, barbaro)
	if paid.HpCurrent != before.HpCurrent-int64(pending.BoostHitPoints) {
		t.Errorf("a proposta deixou %d PV, want %d − %d", paid.HpCurrent, before.HpCurrent, pending.BoostHitPoints)
	}

	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/ataque/cancelar", ""); rec.Code != 200 {
		t.Fatalf("cancelar o provisório deu %d", rec.Code)
	}
	if back := poolsOf(t, f.s, barbaro); back.HpCurrent != before.HpCurrent || back.MpCurrent != before.MpCurrent {
		t.Errorf("depois do cancelamento a ficha tem %d PV e %d PM, want %d e %d", back.HpCurrent, back.MpCurrent, before.HpCurrent, before.MpCurrent)
	}
}

// "UMA VEZ POR RODADA" (p335-336) é recusado com a iniciativa correndo (decisão do dono), como o ataque
// extra das armas naturais. O caso de mesa é o Disparo Rápido: o disparo adicional é com o MESMO arco e não
// gasta ação, e pagar de novo nele seria o segundo gasto da rodada. O cancelamento do mestre devolve o
// gasto, e a vez seguinte de quem atacou o libera.
func TestTheEnchantBoostIsOncePerRound(t *testing.T) {
	f, id, goblin := styleOnTurn(t, "disparo-rapido", true, "arco-curto")
	var bow int64
	if err := f.s.db.QueryRowContext(context.Background(),
		"SELECT id FROM character_items WHERE characterId = ?", id).Scan(&bow); err != nil {
		t.Fatalf("achar o arco: %v", err)
	}
	if err := f.s.characterPlays().EnchantItem(context.Background(), bow, []string{"encanto-venenosa"}, nil); err != nil {
		t.Fatalf("encantar o arco: %v", err)
	}
	paid := fmt.Sprintf(`{"turn_target":%q,"boost_weapon":0,"boost_power":%q,"boost_steps":0,"boost_split":""}`,
		goblin, engine.EnchantBoostID(bow, "encanto-venenosa"))
	shoot := func(gesture, body string) string {
		t.Helper()
		return sceneRefusal(f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/%s/0?embutida=1", id, gesture), body))
	}

	// O CANCELADO NÃO CONTA: o mestre desfaz o golpe, e o gasto volta com o PM.
	if refusal := shoot("atacar", paid); refusal != "" {
		t.Fatalf("o primeiro disparo pago foi recusado: %q", refusal)
	}
	gmCancels(t, f)
	if refusal := shoot("atacar", paid); refusal != "" {
		t.Fatalf("depois do cancelamento, o disparo pago foi recusado: %q", refusal)
	}
	gmRules(t, f, "errou")
	if refusal := shoot("ataque-extra", paid); !strings.Contains(refusal, "uma vez por rodada") {
		t.Errorf("o adicional pagou o encanto de novo na mesma rodada, ou a recusa não diz a regra: %q", refusal)
	}
	// O CONTROLE: sem o gasto, o adicional passa — a recusa era do encanto, e não do disparo.
	plain := fmt.Sprintf(`{"turn_target":%q}`, goblin)
	if refusal := shoot("ataque-extra", plain); refusal != "" {
		t.Fatalf("o adicional sem o gasto foi recusado: %q", refusal)
	}
	gmRules(t, f, "errou")
	for range 2 { // o Goblin, e de novo o bárbaro
		if _, err := f.s.sessions.NextTurn(context.Background(), f.sessionID); err != nil {
			t.Fatalf("passar a vez: %v", err)
		}
	}
	if refusal := shoot("atacar", paid); refusal != "" {
		t.Errorf("na vez seguinte do bárbaro, o disparo pago foi recusado: %q", refusal)
	}
}

// O PV DA TUMULAR QUE SAIU DA POÇA TEMPORÁRIA VOLTA PARA ELA. A perda gasta os temporários primeiro (p106),
// e o cancelamento desfaz o golpe que não aconteceu: devolver ao PV comum daria PV que não expira no lugar do
// que expiraria no fim do dia — o mesmo motivo do recibo do PM.
func TestTheTomblikeHitPointsComeBackToTheTemporaryPool(t *testing.T) {
	f, barbaro, goblin := barbarianOnTurn(t)
	sword := itemSemeia(t, f, barbaro, "espada-longa", "Espada longa", "wielded")
	if err := f.s.characterPlays().EnchantItem(context.Background(), sword, []string{"encanto-tumular"}, nil); err != nil {
		t.Fatalf("encantar a espada: %v", err)
	}
	seedTempHpPool(t, f.s, barbaro, 20)
	tempHp := func() int {
		t.Helper()
		rows, err := f.s.queries.ListActiveEffectsByCharacter(context.Background(), barbaro)
		if err != nil {
			t.Fatalf("ler as poças: %v", err)
		}
		total := 0
		for _, pool := range sheet.ParseTempHpPools(rows) {
			total += pool.Amount
		}
		return total
	}
	tomblike := engine.EnchantBoostID(sword, "encanto-tumular")
	screen := html.UnescapeString(f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/atacar/0?embutida=1", barbaro), ""))
	found := regexp.MustCompile(`\$boost_weapon = (\d+); \$boost_power = "` + regexp.QuoteMeta(tomblike) + `"`).FindStringSubmatch(screen)
	if found == nil {
		t.Fatalf("a linha da espada não oferece o gasto da Tumular")
	}
	hpBefore := poolsOf(t, f.s, barbaro).HpCurrent

	chosen := fmt.Sprintf(`{"turn_target":%q,"boost_weapon":%s,"boost_power":%q,"boost_steps":0,"boost_split":""}`,
		goblin, found[1], tomblike)
	if refusal := sceneRefusal(f.posts(t, f.player,
		fmt.Sprintf("/personagens/%d/acoes/atacar/%s?embutida=1", barbaro, found[1]), chosen)); refusal != "" {
		t.Fatalf("o golpe com o gasto da Tumular foi recusado: %q", refusal)
	}
	lost := stateOf(t, f.s.sessions, f.sessionID).PendingAttack.BoostHitPoints
	if got := tempHp(); got != 20-lost {
		t.Fatalf("a poça tem %d depois de perder %d, want %d — a perda gasta os temporários primeiro (p106)", got, lost, 20-lost)
	}

	gmCancels(t, f)
	if got := tempHp(); got != 20 {
		t.Errorf("depois do cancelamento a poça tem %d, want 20: o PV que saiu dela volta para ela", got)
	}
	if got := poolsOf(t, f.s, barbaro).HpCurrent; got != hpBefore {
		t.Errorf("depois do cancelamento o PV comum é %d, want %d", got, hpBefore)
	}
}
