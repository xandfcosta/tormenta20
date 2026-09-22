package table

import (
	"strings"
	"testing"

	"t20engine/domain/live"
)

// Os guardas do RASTREADOR DO MESTRE.
//
// As regras têm teste próprio no `live`, contra as bordas que cada issue
// nomeia. O que se prende aqui é a COMPOSIÇÃO — que a cena pergunta a coisa
// certa a cada regra, que é onde um argumento trocado passa por dado plausível.

func estadoDe(activeScene bool, round, turn int, queue ...live.InitiativeEntry) *live.SessionRuntimeState {
	return &live.SessionRuntimeState{
		Scene: anActionScene(activeScene), Round: round, TurnIndex: turn, Initiative: queue,
	}
}

// Separar "não há para onde ir" de "o botão está quebrado" é o ponto: um botão
// aceso que recusa é pior que um apagado que explica. E são DOIS motivos
// diferentes de estar apagado — sem cena, e em cena sem ninguém na fila —, que
// é o que o contador diz enquanto o botão fica quieto.
func TestAdvanceOnlyLightsUpWithASceneAndATracker(t *testing.T) {
	arwen := live.InitiativeEntry{Label: "Arwen"}

	cases := []struct {
		name string
		st   *live.SessionRuntimeState
		want bool
	}{
		{"fora de cena, com fila", estadoDe(false, 0, -1, arwen), false},
		{"em cena, sem fila", estadoDe(true, 0, -1), false},
		{"em cena, com fila", estadoDe(true, 0, -1, arwen), true},
		{"em combate", estadoDe(true, 2, 0, arwen), true},
	}
	for _, c := range cases {
		v := ofViewGm(c.st, nil, nil, true)
		if v.CanAdvance != c.want {
			t.Errorf("%s: PodeAvancar = %v, quero %v", c.name, v.CanAdvance, c.want)
		}
	}
}

// O guarda da COMPOSIÇÃO: as duas regras recebem os
// mesmos argumentos e é fácil trocar um: passar `Round` onde vai `TurnIndex`
// compila, e a tela mente com números plausíveis. Aqui se afirma que as duas
// concordam sobre o estado.
func TestTheCounterAndTheAdvanceTellTheSameStory(t *testing.T) {
	queue := []live.InitiativeEntry{{Label: "Arwen"}, {Label: "Ogro"}}

	outside := ofViewGm(estadoDe(false, 0, -1, queue...), nil, nil, true)
	if outside.Counter != "Fora de cena" {
		t.Errorf("fora de cena o contador diz %q", outside.Counter)
	}

	building := ofViewGm(estadoDe(true, 0, -1, queue...), nil, nil, true)
	if building.Counter != "Rodada 0 · 2 na fila" {
		t.Errorf("montando a ordem o contador diz %q", building.Counter)
	}
	// Fora de combate o verbo é COMEÇAR, e o contador concorda dizendo que a
	// rodada ainda é 0.
	if building.Advance.Label != "Começar: Arwen" {
		t.Errorf("montando a ordem o botão diz %q", building.Advance.Label)
	}

	inCombat := ofViewGm(estadoDe(true, 1, 0, queue...), nil, nil, true)
	if inCombat.Counter != "Rodada 1 · Turno 1/2 · padrão e movimento" {
		t.Errorf("em combate o contador diz %q", inCombat.Counter)
	}
	if inCombat.Advance.Label != "Próximo: Ogro" {
		t.Errorf("em combate o botão diz %q", inCombat.Advance.Label)
	}
}

// As duas condições, e a da fila é a que costuma ser esquecida.
func TestVitalsFollowTheTrackerAndTheRole(t *testing.T) {
	pv := int64(30)
	withNPC := estadoDe(true, 1, 0, live.InitiativeEntry{Label: "Ogro", HpMax: &pv})
	soPCs := estadoDe(true, 1, 0, live.InitiativeEntry{Label: "Arwen"})

	if !ofViewGm(withNPC, nil, nil, true).SeesVitals {
		t.Error("o mestre não vê vitais numa fila com NPC")
	}
	if ofViewGm(withNPC, nil, nil, false).SeesVitals {
		t.Error("o jogador viu os vitais do NPC")
	}
	if ofViewGm(soPCs, nil, nil, true).SeesVitals {
		t.Error("numa fila só de PCs a tela mudou de forma sem ter o que reservar")
	}
}

// Quem está com a aba aberta aparece marcado, e quem não tem personagem ligado
// não vira "personagem 0 online".
func TestPresenceReachesTheScene(t *testing.T) {
	members := []live.TableMember{
		{CharacterID: 10, OwnerID: 1},
		{CharacterID: 11, OwnerID: 2},
		{CharacterID: 12, OwnerID: 0},
	}
	v := ofViewGm(estadoDe(true, 1, 0), members, []int64{1}, true)
	if len(v.Connected) != 1 || !v.Connected[10] {
		t.Errorf("conectados = %v, quero só o 10", v.Connected)
	}
	if v.Connected[0] {
		t.Error("o personagem 0 entrou na presença")
	}
}

// ── os comandos, pelo fio ────────────────────────────────────────────────────

// trechoDeSinais tira só a linha dos sinais da resposta SSE, porque o quadro
// inteiro traz a cena e enterra a asserção em 8 KB de HTML.
func trechoDeSinais(body string) string {
	for _, row := range strings.Split(body, "\n") {
		if strings.HasPrefix(row, "data: signals ") {
			return row
		}
	}
	return "(nenhuma linha de sinais na resposta)"
}

// ── editar o combatente ──────────────────────────────────────────────────────

// trechoDaSemeadura tira só o pedaço da expressão que semeia o nome, porque a
// página inteira enterra a asserção em vários KB de HTML.
func trechoDaSemeadura(body string) string {
	i := strings.Index(body, "$edit_name = ")
	if i < 0 {
		return "(a semeadura do nome não está na página)"
	}
	end := i + 120
	if end > len(body) {
		end = len(body)
	}
	return body[i:end]
}

// anActionScene monta a cena que o caso quer, ou nenhuma. Os casos deste arquivo
// são todos sobre o COMBATE, que é a cena de ação.
func anActionScene(on bool) *live.Scene {
	if !on {
		return nil
	}
	return &live.Scene{Kind: live.SceneAction, Number: 1, StandardLeft: true, MovementLeft: true}
}
