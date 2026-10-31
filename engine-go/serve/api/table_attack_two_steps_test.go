package api

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"t20engine/domain/live"
)

// O ATAQUE EM DOIS TEMPOS (ALE-536, decisão do dono): o jogador rola o teste,
// o MESTRE decide errou, acertou ou crítico, e só então o jogador rola o dano —
// com os dados que a decisão pede. O crítico muda quantos dados se rolam
// (p231), e quem diz se houve crítico é o mestre.
//
// O predicado é QUEM pode cada passo e EM QUE ORDEM, e por isso o caso tenta
// cada passo fora de hora e por quem não pode, antes do caminho certo.

func pendingOf(t *testing.T, f sceneFixture) *live.PendingAttack {
	t.Helper()
	return stateOf(t, f.s.sessions, f.sessionID).PendingAttack
}

func TestTheDamageIsRolledOnlyAfterTheGMRules(t *testing.T) {
	f, goblin := attackOnTurn(t)
	rec := f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/iniciativa/"+goblin+"/atacar", "")
	pa := pendingOf(t, f)
	if pa == nil {
		t.Fatalf("o ataque não virou provisório: %q", tableRefusal(t, rec.Body.String()))
	}
	if len(pa.Dice) != 0 || pa.Damage != 0 {
		t.Fatalf("o teste de ataque já veio com dano (%v, %d): ele é rolado depois da decisão do mestre", pa.Dice, pa.Damage)
	}

	// FORA DE ORDEM: o dano antes da decisão, e o ✓ antes do dano.
	f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/ataque/dano/na-mao", `{"damage_die_1":5}`)
	if pa := pendingOf(t, f); len(pa.Dice) != 0 {
		t.Errorf("o jogador rolou o dano antes de o mestre decidir: %v", pa.Dice)
	}
	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", "")
	if pendingOf(t, f) == nil {
		t.Fatal("o mestre confirmou um ataque sem dano rolado, e o provisório sumiu")
	}

	// QUEM NÃO PODE: a decisão é do mestre.
	f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/ataque/julgar/critico", "")
	if pa := pendingOf(t, f); pa.Ruling != "" {
		t.Errorf("o jogador decidiu o próprio ataque: %q", pa.Ruling)
	}

	// O CAMINHO: o mestre diz crítico, e a espada longa (×2) pede 2d8.
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/julgar/critico", ""); rec.Code != http.StatusOK {
		t.Fatalf("decidir o ataque deu %d", rec.Code)
	}
	pa = pendingOf(t, f)
	if pa.Ruling != "critical" || pa.DamageToRoll == nil ||
		len(pa.DamageToRoll.Dice) != 1 || pa.DamageToRoll.Dice[0].Count != 2 || pa.DamageToRoll.Dice[0].Faces != 8 {
		t.Fatalf("o crítico da espada longa tinha de pedir 2d8, e o provisório ficou %q com %+v", pa.Ruling, pa.DamageToRoll)
	}

	// O JOGADOR ROLA NA MÃO: um campo por dado, na ordem da fórmula.
	if rec := f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/ataque/dano/na-mao",
		`{"damage_die_1":5,"damage_die_2":3}`); rec.Code != http.StatusOK {
		t.Fatalf("rolar o dano deu %d", rec.Code)
	}
	pa = pendingOf(t, f)
	if !slices.Equal(pa.Dice, []int{5, 3}) || !pa.Critical {
		t.Fatalf("o dano da mão saiu %v (crítico=%v), e o digitado foi 5 e 3 num crítico", pa.Dice, pa.Critical)
	}
	if pa.RawDamage != 5+3+pa.DamageToRoll.Bonus {
		t.Errorf("o dano cru saiu %d, e 2d8 (5, 3) + %d = %d", pa.RawDamage, pa.DamageToRoll.Bonus, 5+3+pa.DamageToRoll.Bonus)
	}

	// O ✓ AGORA POUSA O DANO.
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", ""); rec.Code != http.StatusOK {
		t.Fatalf("confirmar deu %d", rec.Code)
	}
	if pendingOf(t, f) != nil {
		t.Error("o ataque confirmado continuou na mesa")
	}
}

func TestTheGMsMissEndsTheAttackWithoutDamage(t *testing.T) {
	f, goblin := attackOnTurn(t)
	f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/iniciativa/"+goblin+"/atacar", "")
	if pendingOf(t, f) == nil {
		t.Fatal("o ataque não virou provisório")
	}
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/julgar/errou", ""); rec.Code != http.StatusOK {
		t.Fatalf("decidir errou deu %d", rec.Code)
	}
	// ERRAR É UMA COISA QUE ACONTECEU: a ação padrão foi gasta (p233).
	if stateOf(t, f.s.sessions, f.sessionID).Scene.StandardLeft {
		t.Error("o ataque que o mestre disse errado não gastou a ação padrão")
	}
	f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/ataque/dano/na-mao", `{"damage_die_1":5}`)
	if pa := pendingOf(t, f); pa != nil && len(pa.Dice) != 0 {
		t.Errorf("o jogador rolou dano num ataque que errou: %v", pa.Dice)
	}

	// O JOGADOR LÊ A DECISÃO; o mestre não tem mais o que fazer com o cartão.
	player := attackSection(t, f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String())
	if !strings.Contains(player, "errou") {
		t.Errorf("o jogador não lê que o mestre disse errou:\n%s", player)
	}
	gm := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(gm, `<section id="table-attack" hidden>`) {
		t.Errorf("o cartão do mestre continuou de pé depois do errou:\n%s", attackSection(t, gm))
	}
}

// settlesAsTheBookSuggests é o mestre que concorda com o livro: decide o que a
// sugestão diz, rola o dano por quem atacou e confirma — ou, no erro, encerra.
// É o caminho dos casos que prendem o que acontece DEPOIS do golpe; a ordem dos
// dois tempos tem os casos dela acima.
func settlesAsTheBookSuggests(t *testing.T, f sceneFixture) *httptest.ResponseRecorder {
	t.Helper()
	pa := pendingOf(t, f)
	if pa == nil || pa.Maneuver != nil {
		return f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", "")
	}
	if pa.Ruling == "" {
		decision := "errou"
		switch {
		case pa.Critical:
			decision = "critico"
		case pa.Hit:
			decision = "acertou"
		}
		rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/julgar/"+decision, "")
		if refusal := tableRefusal(t, rec.Body.String()); refusal != "" || decision == "errou" {
			if refusal != "" {
				t.Fatalf("o mestre dizer %s foi recusado: %s", decision, refusal)
			}
			return rec
		}
	}
	if pa := pendingOf(t, f); pa != nil && !pa.DamageRolled {
		rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/dano", "")
		if refusal := tableRefusal(t, rec.Body.String()); refusal != "" {
			t.Fatalf("rolar o dano foi recusado: %s", refusal)
		}
	}
	return f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", "")
}

// O "ERROU" SAI QUANDO A VEZ PASSA: ele fica na mesa para o jogador ler a
// decisão, e a vez seguinte já é outra conversa. O controle é o ataque AINDA
// SEM DECISÃO, que a vez passando não pode apagar — o mestre ainda não falou.
func TestTheGMsMissLeavesTheTableWhenTheTurnPasses(t *testing.T) {
	f, goblin := attackOnTurn(t)
	f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/iniciativa/"+goblin+"/atacar", "")
	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/proxima-vez", "")
	if pendingOf(t, f) == nil {
		t.Fatal("a vez passou e levou um ataque que o mestre ainda não decidiu")
	}
	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/vez-anterior", "")

	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/julgar/errou", "")
	if pa := pendingOf(t, f); pa == nil || pa.Ruling != "miss" {
		t.Fatalf("o errou do mestre não ficou na mesa para o jogador ler: %+v", pa)
	}
	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/proxima-vez", "")
	if pa := pendingOf(t, f); pa != nil {
		t.Errorf("a vez passou e o errou continuou na mesa: %+v", pa)
	}
}
