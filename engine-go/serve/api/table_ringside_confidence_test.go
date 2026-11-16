package api

import (
	"context"
	"net/http"
	"testing"

	"t20engine/domain/live"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// A CONFIANÇA DOS RINGUES (p76): *"quando um inimigo erra um ataque corpo a
// corpo contra você, você recebe 2 PM temporários (cumulativos). Você pode ganhar
// um máximo de PM temporários por cena igual ao seu nível"*.
//
// O gatilho é a DECISÃO do mestre: é o "errou" dele que diz que o inimigo errou, e
// por isso o caso passa pela rota do julgamento, e não pelo caso de uso.

// fighterAgainstTheGoblin é a bancada: um lutador de nível 8, com ou sem o poder, e
// um Goblin na fila, com a vez.
func fighterAgainstTheGoblin(t *testing.T, withThePower bool) (sceneFixture, int64, string, string) {
	t.Helper()
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t.com")
	player := seedUser(t, s, "jogador@t.com")
	campaignID := seedCampaign(t, s, gm)
	sessionID := seedSession(t, s, campaignID)
	id, err := s.queries.CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: player, Name: "Pugilista", Origin: "guarda", Level: 8,
		Strength: 4, Dexterity: 2, Constitution: 3, Intelligence: 0, Wisdom: 1, Charisma: 0,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassChoices: "{}", CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o lutador: %v", err)
	}
	seedClasse(t, s, id, "Lutador", 8)
	seedMember(t, s, campaignID, id)
	f := sceneFixture{s: s, gm: gm, player: player,
		campaignID: campaignID, sessionID: sessionID, charID: id}
	if withThePower {
		if refused := powerCommand(t, f, id, "escolhe/class.lutador.confianca-dos-ringues", ""); refused != "" {
			t.Fatalf("escolher a Confiança dos Ringues: %s", refused)
		}
	}
	f.actionSceneWithTheGoblinFirst(t, "Pugilista", f.charID, 2)
	state := stateOf(t, f.s.sessions, f.sessionID)
	return f, id, entryLabeled(t, state, "Pugilista"), entryLabeled(t, state, "Goblin")
}

// theGameMasterSaysItMissed semeia um golpe que o mestre julga ERRADO, corpo a corpo
// ou à distância.
func theGameMasterSaysItMissed(t *testing.T, f sceneFixture, attacker, target string, melee bool) {
	t.Helper()
	if _, err := f.s.sessions.ProposeAttack(context.Background(), f.sessionID, live.PendingAttack{
		AttackerEntryID: attacker, TargetEntryID: target,
		Weapon: "Clava", Roll: 3, Total: 6, Defense: 20, Melee: melee,
	}); err != nil {
		t.Fatalf("semear o provisório: %v", err)
	}
	gmRules(t, f, "errou")
	// O ERRO FICA NA MESA para a leitura, mas decidido.
	if p := pendingOf(t, f); p == nil || p.Ruling != live.RulingMiss {
		t.Fatalf("o \"errou\" do mestre não ficou registrado: %+v", p)
	}
}

func TestTheRingsideConfidenceGivesTwoTemporaryPMWhenAnEnemyMissesInMelee(t *testing.T) {
	t.Run("com o poder, o erro corpo a corpo dá 2", func(t *testing.T) {
		f, lutador, alvo, goblin := fighterAgainstTheGoblin(t, true)
		theGameMasterSaysItMissed(t, f, goblin, alvo, true)
		if _, reserva := manaOnTheSheet(t, f, lutador); reserva != 2 {
			t.Errorf("o Goblin errou corpo a corpo, e a reserva é %d e não 2", reserva)
		}
	})
	// O CONTROLE DO GATILHO: o livro diz "corpo a corpo", e o erro à distância não conta.
	t.Run("o erro à distância não dá nada", func(t *testing.T) {
		f, lutador, alvo, goblin := fighterAgainstTheGoblin(t, true)
		theGameMasterSaysItMissed(t, f, goblin, alvo, false)
		if _, reserva := manaOnTheSheet(t, f, lutador); reserva != 0 {
			t.Errorf("o Goblin errou à distância, e a reserva foi para %d", reserva)
		}
	})
	// O ACERTO, pela rota inteira do mestre: acertou, o dano, o ✓.
	t.Run("o acerto não dá nada", func(t *testing.T) {
		f, lutador, alvo, goblin := fighterAgainstTheGoblin(t, true)
		if _, err := f.s.sessions.ProposeAttack(context.Background(), f.sessionID, live.PendingAttack{
			AttackerEntryID: goblin, TargetEntryID: alvo,
			Weapon: "Clava", Roll: 18, Total: 22, Defense: 14, Melee: true,
			Hit: true, Ruling: live.RulingHit, DamageRolled: true, RawDamage: 1, Damage: 1,
		}); err != nil {
			t.Fatalf("semear o provisório: %v", err)
		}
		if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", ""); rec.Code != http.StatusOK {
			t.Fatalf("confirmar deu %d", rec.Code)
		}
		if pendingOf(t, f) != nil {
			t.Fatal("o ✓ do mestre não encerrou o golpe")
		}
		if _, reserva := manaOnTheSheet(t, f, lutador); reserva != 0 {
			t.Errorf("o Goblin acertou, e a reserva foi para %d", reserva)
		}
	})
	t.Run("sem o poder, o erro não dá nada", func(t *testing.T) {
		f, lutador, alvo, goblin := fighterAgainstTheGoblin(t, false)
		theGameMasterSaysItMissed(t, f, goblin, alvo, true)
		if _, reserva := manaOnTheSheet(t, f, lutador); reserva != 0 {
			t.Errorf("sem a Confiança dos Ringues, a reserva foi para %d", reserva)
		}
	})
	// O LADO: quem erra é o lutador, e o PM é de quem foi atacado.
	t.Run("o lutador que erra não ganha", func(t *testing.T) {
		f, lutador, alvo, goblin := fighterAgainstTheGoblin(t, true)
		theGameMasterSaysItMissed(t, f, alvo, goblin, true)
		if _, reserva := manaOnTheSheet(t, f, lutador); reserva != 0 {
			t.Errorf("o lutador errou o Goblin, e a própria reserva foi para %d", reserva)
		}
	})
}

// O TETO É DO QUE SE GANHOU NA CENA, e é o nível: cinco erros dariam 10, e o lutador de
// nível 8 para em 8. Cada golpe gasta a ação padrão do Goblin naquela vez, e por isso a
// vez gira uma rodada inteira entre um e outro — a cena continua a mesma.
func TestTheRingsideConfidenceStopsAtTheFighterLevelPerScene(t *testing.T) {
	f, lutador, alvo, goblin := fighterAgainstTheGoblin(t, true)
	for range 5 {
		theGameMasterSaysItMissed(t, f, goblin, alvo, true)
		for range 2 {
			if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/proxima-vez", ""); rec.Code != http.StatusOK {
				t.Fatalf("passar a vez deu %d", rec.Code)
			}
		}
	}
	if _, reserva := manaOnTheSheet(t, f, lutador); reserva != 8 {
		t.Errorf("cinco erros num lutador de nível 8 deram reserva %d, e o teto é 8", reserva)
	}
}
