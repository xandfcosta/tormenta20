package table

import (
	"strings"
	"testing"

	"t20engine/aovivo"
)

// Os guardas do RASTREADOR DO MESTRE (ALE-265).
//
// As regras têm teste próprio no `aovivo`, contra as bordas que cada issue
// nomeia. O que se prende aqui é a COMPOSIÇÃO — que a cena pergunta a coisa
// certa a cada regra, que é onde um argumento trocado passa por dado plausível.

func estadoDe(cenaAtiva bool, rodada, turno int, fila ...aovivo.InitiativeEntry) *aovivo.SessionRuntimeState {
	return &aovivo.SessionRuntimeState{
		SceneActive: cenaAtiva, Round: rodada, TurnIndex: turno, Initiative: fila,
	}
}

// TestAdvanceOnlyLightsUpWithASceneAndATracker.
//
// Separar "não há para onde ir" de "o botão está quebrado" é o ponto: um botão
// aceso que recusa é pior que um apagado que explica. E são DOIS motivos
// diferentes de estar apagado — sem cena, e em cena sem ninguém na fila —, que
// é o que o contador diz enquanto o botão fica quieto.
func TestAdvanceOnlyLightsUpWithASceneAndATracker(t *testing.T) {
	arwen := aovivo.InitiativeEntry{Label: "Arwen"}

	casos := []struct {
		nome  string
		st    *aovivo.SessionRuntimeState
		quero bool
	}{
		{"fora de cena, com fila", estadoDe(false, 0, -1, arwen), false},
		{"em cena, sem fila", estadoDe(true, 0, -1), false},
		{"em cena, com fila", estadoDe(true, 0, -1, arwen), true},
		{"em combate", estadoDe(true, 2, 0, arwen), true},
	}
	for _, c := range casos {
		v := ofViewGm(c.st, nil, nil, true, false)
		if v.PodeAvancar != c.quero {
			t.Errorf("%s: PodeAvancar = %v, quero %v", c.nome, v.PodeAvancar, c.quero)
		}
	}
}

// TestTheCounterAndTheAdvanceTellTheSameStory.
//
// Este é o guarda da COMPOSIÇÃO, e ele existe porque as duas regras recebem os
// mesmos argumentos e é fácil trocar um: passar `Round` onde vai `TurnIndex`
// compila, e a tela mente com números plausíveis. Aqui se afirma que as duas
// concordam sobre o estado.
func TestTheCounterAndTheAdvanceTellTheSameStory(t *testing.T) {
	fila := []aovivo.InitiativeEntry{{Label: "Arwen"}, {Label: "Ogro"}}

	fora := ofViewGm(estadoDe(false, 0, -1, fila...), nil, nil, true, false)
	if fora.Contador != "Fora de cena" {
		t.Errorf("fora de cena o contador diz %q", fora.Contador)
	}

	montando := ofViewGm(estadoDe(true, 0, -1, fila...), nil, nil, true, false)
	if montando.Contador != "Rodada 0 · 2 na fila" {
		t.Errorf("montando a ordem o contador diz %q", montando.Contador)
	}
	// Fora de combate o verbo é COMEÇAR, e o contador concorda dizendo que a
	// rodada ainda é 0.
	if montando.Avanco.Label != "Começar: Arwen" {
		t.Errorf("montando a ordem o botão diz %q", montando.Avanco.Label)
	}

	emCombate := ofViewGm(estadoDe(true, 1, 0, fila...), nil, nil, true, false)
	if emCombate.Contador != "Rodada 1 · Turno 1/2" {
		t.Errorf("em combate o contador diz %q", emCombate.Contador)
	}
	if emCombate.Avanco.Label != "Próximo: Ogro" {
		t.Errorf("em combate o botão diz %q", emCombate.Avanco.Label)
	}
}

// TestVitalsFollowTheTrackerAndTheRole — as duas condições, e a da fila é a que
// costuma ser esquecida.
func TestVitalsFollowTheTrackerAndTheRole(t *testing.T) {
	pv := int64(30)
	comNPC := estadoDe(true, 1, 0, aovivo.InitiativeEntry{Label: "Ogro", HpMax: &pv})
	soPCs := estadoDe(true, 1, 0, aovivo.InitiativeEntry{Label: "Arwen"})

	if !ofViewGm(comNPC, nil, nil, true, false).VeVitais {
		t.Error("o mestre não vê vitais numa fila com NPC")
	}
	if ofViewGm(comNPC, nil, nil, false, false).VeVitais {
		t.Error("o jogador viu os vitais do NPC")
	}
	if ofViewGm(soPCs, nil, nil, true, false).VeVitais {
		t.Error("numa fila só de PCs a tela mudou de forma sem ter o que reservar")
	}
}

// TestPresenceReachesTheScene: quem está com a aba aberta aparece marcado, e quem
// não tem personagem ligado não vira "personagem 0 online".
func TestPresenceReachesTheScene(t *testing.T) {
	membros := []aovivo.TableMember{
		{CharacterID: 10, OwnerID: 1},
		{CharacterID: 11, OwnerID: 2},
		{CharacterID: 12, OwnerID: 0},
	}
	v := ofViewGm(estadoDe(true, 1, 0), membros, []int64{1}, true, false)
	if len(v.Conectados) != 1 || !v.Conectados[10] {
		t.Errorf("conectados = %v, quero só o 10", v.Conectados)
	}
	if v.Conectados[0] {
		t.Error("o personagem 0 entrou na presença")
	}
}

// ── os comandos, pelo fio ────────────────────────────────────────────────────

// trechoDeSinais tira só a linha dos sinais da resposta SSE, porque o quadro
// inteiro traz a cena e enterra a asserção em 8 KB de HTML.
func trechoDeSinais(corpo string) string {
	for _, linha := range strings.Split(corpo, "\n") {
		if strings.HasPrefix(linha, "data: signals ") {
			return linha
		}
	}
	return "(nenhuma linha de sinais na resposta)"
}

// ── editar o combatente (ALE-263) ────────────────────────────────────────────

// trechoDaSemeadura tira só o pedaço da expressão que semeia o nome, porque a
// página inteira enterra a asserção em vários KB de HTML.
func trechoDaSemeadura(corpo string) string {
	i := strings.Index(corpo, "$edicaonome = ")
	if i < 0 {
		return "(a semeadura do nome não está na página)"
	}
	fim := i + 120
	if fim > len(corpo) {
		fim = len(corpo)
	}
	return corpo[i:fim]
}
