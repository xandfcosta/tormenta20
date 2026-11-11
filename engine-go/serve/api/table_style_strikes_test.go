package api

import (
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/engine"
)

// OS ESTILOS QUE DÃO UM ATAQUE A MAIS (p125).
//
// Estilo de Duas Armas: "se estiver empunhando duas armas [...] e fizer a ação agredir, você
// pode fazer dois ataques, um com cada arma. Se fizer isso, sofre -2 em todos os testes de
// ataque até o seu próximo turno."
//
// Disparo Rápido: "se estiver empunhando uma arma de disparo que possa recarregar como ação
// livre e gastar uma ação completa para agredir, pode fazer um ataque adicional com ela."
//
// Usar o poder é a declaração: o -2 vale nos DOIS ataques, então usá-lo depois do primeiro
// não dá o segundo.

// styleOnTurn é o bárbaro na vez, com o poder, as armas na mão e o poder usado, quando pedido.
func styleOnTurn(t *testing.T, power string, on bool, weapons ...string) (sceneFixture, int64, string) {
	t.Helper()
	f, id, goblin := barbarianOnTurn(t)
	seedPowerChoices(t, f.s, id, `["`+power+`"]`, "{}")
	if _, err := f.s.db.ExecContext(t.Context(), "DELETE FROM character_items WHERE characterId = ?", id); err != nil {
		t.Fatalf("limpar as mãos: %v", err)
	}
	place := "wielded"
	for _, weapon := range weapons {
		itemSemeia(t, f, id, weapon, weapon, place)
	}
	if on {
		turnsOnStyle(t, f, id, power)
	}
	return f, id, goblin
}

func turnsOnStyle(t *testing.T, f sceneFixture, id int64, power string) {
	t.Helper()
	if refusal := usesTheStyle(t, f, id, power); refusal != "" {
		t.Fatalf("usar o %s foi recusado: %q", power, refusal)
	}
}

func TestTwoWeaponStyleGivesOneAttackWithTheOtherWeapon(t *testing.T) {
	f, id, goblin := styleOnTurn(t, "estilo-de-duas-armas", true, "espada-longa", "adaga")
	if refusal := extraStrike(t, f, id, 1, goblin); !strings.Contains(refusal, "agredir") {
		t.Fatalf("sem agredir, o segundo ataque devia ser recusado, e veio %q", refusal)
	}
	if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
		t.Fatalf("o primeiro golpe: %q", refusal)
	}
	if refusal := extraStrike(t, f, id, 1, goblin); !strings.Contains(refusal, "mestre") {
		t.Fatalf("com o golpe esperando o mestre, o segundo devia ser recusado, e veio %q", refusal)
	}
	gmRules(t, f, "errou")

	// "Um com cada arma": a mesma espada não ataca de novo.
	if refusal := extraStrike(t, f, id, 0, goblin); !strings.Contains(refusal, "outra arma") {
		t.Errorf("o segundo ataque com a mesma arma devia ser recusado, e veio %q", refusal)
	}
	before, scene := mpOf(t, f.s, id), *sceneOf(t, f)
	if refusal := extraStrike(t, f, id, 1, goblin); refusal != "" {
		t.Fatalf("o ataque com a outra arma foi recusado: %q", refusal)
	}
	pa := pendingOf(t, f)
	if pa == nil || pa.Weapon != "adaga" || pa.Ruling != "" {
		t.Fatalf("o segundo ataque devia virar proposta com a adaga: %+v", pa)
	}
	if got := mpOf(t, f.s, id); got != before {
		t.Errorf("o estilo não cobra PM: %d virou %d", before, got)
	}
	if after := sceneOf(t, f); after.StandardLeft != scene.StandardLeft || after.MovementLeft != scene.MovementLeft {
		t.Errorf("o segundo ataque não gasta ação: %+v virou %+v", scene, *after)
	}

	// São dois, e não três.
	gmRules(t, f, "errou")
	if refusal := extraStrike(t, f, id, 1, goblin); refusal == "" {
		t.Error("um terceiro ataque passou")
	}
}

func TestTheStyleMustBeUsedBeforeTheFirstAttack(t *testing.T) {
	f, id, goblin := styleOnTurn(t, "estilo-de-duas-armas", false, "espada-longa", "adaga")
	if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
		t.Fatalf("o primeiro golpe: %q", refusal)
	}
	gmRules(t, f, "errou")
	turnsOnStyle(t, f, id, "estilo-de-duas-armas")
	if refusal := extraStrike(t, f, id, 1, goblin); !strings.Contains(refusal, "agredir") {
		t.Errorf("o primeiro golpe não pagou o -2, e o segundo devia ser recusado; veio %q", refusal)
	}
}

func TestACancelledStyleAttackGivesItBack(t *testing.T) {
	f, id, goblin := styleOnTurn(t, "estilo-de-duas-armas", true, "espada-longa", "adaga")
	rollsFromTheActions(t, f, id, "atacar/0", goblin)
	gmRules(t, f, "errou")
	if refusal := extraStrike(t, f, id, 1, goblin); refusal != "" {
		t.Fatalf("o segundo ataque: %q", refusal)
	}
	gmCancels(t, f)
	if refusal := extraStrike(t, f, id, 1, goblin); refusal != "" {
		t.Errorf("cancelado, o segundo ataque devia poder voltar, e veio %q", refusal)
	}
}

func TestTheTwoWeaponAttackEndsWithTheTurn(t *testing.T) {
	f, id, goblin := styleOnTurn(t, "estilo-de-duas-armas", true, "espada-longa", "adaga")
	rollsFromTheActions(t, f, id, "atacar/0", goblin)
	gmRules(t, f, "errou")
	if _, err := f.s.sessions.NextTurn(t.Context(), f.sessionID); err != nil {
		t.Fatalf("girar a vez: %v", err)
	}
	if _, err := f.s.sessions.NextTurn(t.Context(), f.sessionID); err != nil {
		t.Fatalf("girar a vez: %v", err)
	}
	if refusal := extraStrike(t, f, id, 1, goblin); refusal == "" {
		t.Error("o segundo ataque sobreviveu à virada do turno")
	}
}

// Com uma arma só não há "um com cada arma", e a linha nem aparece. A falta da arma LEVE não
// tira a linha: o Usar já avisa, e quem decide é o mestre.
func TestWhenTheTwoWeaponLineShowsUp(t *testing.T) {
	for _, c := range []struct {
		name    string
		on      bool
		weapons []string
		shows   bool
	}{
		{"usado, com duas armas", true, []string{"espada-longa", "adaga"}, true},
		{"usado, sem arma leve", true, []string{"espada-longa", "espada-longa"}, true},
		{"sem usar", false, []string{"espada-longa", "adaga"}, false},
		{"usado, com uma arma", true, []string{"espada-longa"}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, _, _ := styleOnTurn(t, "estilo-de-duas-armas", c.on, c.weapons...)
			surface := superficieDeAcoes(t, acoesDaMesa(t, f))
			if !strings.Contains(surface, "Atacar · ") {
				t.Fatal("o controle falhou: as Ações não têm a linha de atacar")
			}
			if got := strings.Contains(surface, "Ataque adicional · "); got != c.shows {
				t.Errorf("a linha do ataque adicional apareceu = %v, want %v", got, c.shows)
			}
		})
	}
}

func TestRapidShotSpendsTheFullActionAndGivesAnAttackWithTheSameWeapon(t *testing.T) {
	f, id, goblin := styleOnTurn(t, "disparo-rapido", true, "arco-curto")
	if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
		t.Fatalf("o primeiro disparo: %q", refusal)
	}
	if scene := sceneOf(t, f); scene.StandardLeft || scene.MovementLeft {
		t.Errorf("o Disparo Rápido gasta a ação completa: %+v", *scene)
	}
	gmRules(t, f, "errou")
	if refusal := extraStrike(t, f, id, 0, goblin); refusal != "" {
		t.Fatalf("o disparo adicional foi recusado: %q", refusal)
	}
	if pa := pendingOf(t, f); pa == nil || pa.Weapon != "arco-curto" {
		t.Fatalf("o adicional devia ser com o arco: %+v", pa)
	}
	gmCancels(t, f)
	if scene := sceneOf(t, f); scene.StandardLeft || scene.MovementLeft {
		t.Errorf("cancelar o adicional não devolve a ação do primeiro disparo: %+v", *scene)
	}
}

func TestRapidShotOffIsAStandardAttackWithNoExtra(t *testing.T) {
	f, id, goblin := styleOnTurn(t, "disparo-rapido", false, "arco-curto")
	if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
		t.Fatalf("o disparo: %q", refusal)
	}
	if scene := sceneOf(t, f); scene.StandardLeft || !scene.MovementLeft {
		t.Errorf("desligado, o disparo é ação padrão: %+v", *scene)
	}
	gmRules(t, f, "errou")
	if refusal := extraStrike(t, f, id, 0, goblin); !strings.Contains(refusal, "arma natural") {
		t.Errorf("desligado, não há adicional, e veio %q", refusal)
	}
}

// A ação completa pede as duas: quem já andou atira do jeito normal, ou encerra o uso.
func TestRapidShotAfterMovingIsRefusedSayingWhy(t *testing.T) {
	f, id, goblin := styleOnTurn(t, "disparo-rapido", true, "arco-curto")
	if _, err := f.s.sessions.SpendAction(t.Context(), f.sessionID, engine.ActionMovement); err != nil {
		t.Fatalf("gastar o movimento: %v", err)
	}
	if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); !strings.Contains(refusal, "Disparo Rápido") {
		t.Errorf("sem o movimento, a recusa devia dizer que é o Disparo Rápido, e veio %q", refusal)
	}
}

// As Ações dizem o que o uso fez: o golpe muda de grupo, e o adicional aparece.
func TestRapidShotMovesTheAttackToTheFullActions(t *testing.T) {
	f, _, _ := styleOnTurn(t, "disparo-rapido", true, "arco-curto")
	surface := superficieDeAcoes(t, acoesDaMesa(t, f))
	full := strings.Index(surface, "Ação completa")
	attack := strings.Index(surface, "Atacar · arco-curto")
	if full < 0 || attack < full {
		t.Errorf("com o Disparo Rápido, o golpe é ação completa: grupo em %d, linha em %d", full, attack)
	}
	if !strings.Contains(surface, "Ataque adicional · arco-curto") {
		t.Error("as Ações não oferecem o disparo adicional")
	}
}

func TestTheCardSaysItIsTheAdditionalAttack(t *testing.T) {
	f, id, goblin := styleOnTurn(t, "estilo-de-duas-armas", true, "espada-longa", "adaga")
	rollsFromTheActions(t, f, id, "atacar/0", goblin)
	gmRules(t, f, "errou")
	extraStrike(t, f, id, 1, goblin)
	if page := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(page, "Ataque adicional") {
		t.Error("o cartão do mestre não diz que é o ataque adicional")
	}
}

// A arma dupla "pode ser usada com Estilo de Duas Armas [...] para fazer ataques adicionais, como se
// fosse uma arma de uma mão e uma arma leve" (p143): uma só na mão já dá os dois ataques, e o
// Usar não avisa de arma leve nenhuma.
func TestADoubleWeaponAloneGivesTheSecondAttack(t *testing.T) {
	f, id, goblin := styleOnTurn(t, "estilo-de-duas-armas", true, "bordao")
	if chips := acoesDaMesa(t, f); strings.Contains(chips, "Nenhuma arma empunhada é leve") {
		t.Error("o bordão ainda avisa que falta arma leve")
	}
	if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
		t.Fatalf("o primeiro golpe: %q", refusal)
	}
	gmRules(t, f, "acertou")
	if refusal := gmRefusal(t, f, "/ataque/dano"); refusal != "" {
		t.Fatalf("o dano do bordão foi recusado: %q", refusal)
	}
	if refusal := gmRefusal(t, f, "/ataque/confirmar"); refusal != "" {
		t.Fatalf("o ✓: %q", refusal)
	}
	if refusal := extraStrike(t, f, id, 0, goblin); refusal != "" {
		t.Errorf("a outra ponta do bordão foi recusada: %q", refusal)
	}
}
