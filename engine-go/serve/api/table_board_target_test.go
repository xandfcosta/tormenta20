package api

import (
	"context"
	"html"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/board"
)

// AGIR NO TABULEIRO (ALE-660, mockup aprovado pelo dono): tocar na peça de outro escolhe o alvo da vez — o mesmo
// `$turn_target` do "Contra" das Ações — e abre o painel com as ações que pedem alvo. O painel é a própria
// superfície Ações, num modo por cima do mapa, e o cabeçalho diz contra quem e a que distância. É só do jogador.

// boardWithTheGoblin é a vez do bárbaro com ele e o Goblin LADO A LADO no mapa.
func boardWithTheGoblin(t *testing.T) (sceneFixture, string, string) {
	t.Helper()
	f, _, goblin := barbarianOnTurn(t)
	f.seedOpenBoard(t, "stone")
	barbarian := entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Furioso")
	for _, token := range []board.BoardToken{
		{Label: "Goblin", Kind: "npc", X: 1, Y: 0, Footprint: 1, EntryID: &goblin},
		{Label: "Furioso", Kind: "character", X: 0, Y: 0, Footprint: 1, EntryID: &barbarian},
	} {
		if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab, token); err != nil {
			t.Fatalf("pôr %q: %v", token.Label, err)
		}
	}
	return f, goblin, barbarian
}

func pageOf(t *testing.T, f sceneFixture, who int64) string {
	t.Helper()
	return html.UnescapeString(f.requests(t, who, http.MethodGet, f.tableUrl(), "").Body.String())
}

func TestTappingAnotherTokenPicksTheTargetOfTheTurn(t *testing.T) {
	f, goblin, barbarian := boardWithTheGoblin(t)
	player := pageOf(t, f, f.player)
	if !strings.Contains(player, "$turn_target = '"+goblin+"'; $board_panel = true") {
		t.Error("tocar na peça do Goblin não escolhe o alvo da vez nem abre o painel")
	}
	if !strings.Contains(player, "if ($tool === '') { $turn_target = '"+goblin+"'") {
		t.Error("o toque na peça escolhe o alvo também com a Régua na mão, e quem mede não pediu o painel")
	}
	if strings.Contains(player, "$turn_target = '"+barbarian+"'") {
		t.Error("a peça do próprio jogador virou alvo dele")
	}
	if !strings.Contains(player, "'board-token-chosen': $turn_target === '"+goblin+"'") {
		t.Error("a peça escolhida como alvo não ganha a marca do alvo")
	}
	if strings.Contains(pageOf(t, f, f.gm), "$board_panel = true") {
		t.Error("o mestre ganhou o painel, e a épica é só do jogador")
	}
}

func TestTheBoardPanelSaysWhoAndHowFar(t *testing.T) {
	f, goblin, _ := boardWithTheGoblin(t)
	player := pageOf(t, f, f.player)
	at := strings.Index(player, `data-show="$turn_target === '`+goblin+`'"`)
	if at < 0 {
		t.Fatal("o cabeçalho do painel não tem o bloco do Goblin")
	}
	head := player[at:min(len(player), at+400)]
	for _, want := range []string{"Goblin", "a 1,5m · corpo a corpo"} {
		if !strings.Contains(head, want) {
			t.Errorf("o cabeçalho do painel para o Goblin ao lado não diz %q: %s", want, head)
		}
	}
	if !strings.Contains(player, `data-on:click="$board_panel = false"`) {
		t.Error("o painel não tem o ✕ que só o fecha")
	}
}

func TestTheActionsSurfaceBecomesTheBoardPanel(t *testing.T) {
	f, _, _ := boardWithTheGoblin(t)
	player := pageOf(t, f, f.player)
	if !strings.Contains(player, `data-show="$surface === "acoes" || ($surface === "tabuleiro" && $board_panel)"`) {
		t.Error("a superfície Ações não aparece por cima do tabuleiro com o painel aberto")
	}
	if !strings.Contains(player, `'ds-board-panel': $surface === "tabuleiro"`) {
		t.Error("a superfície Ações não entra no modo de painel no tabuleiro")
	}
	surface := superficieDeAcoes(t, player)
	attack := rowNamed(t, surface, "Atacar · Machado de batalha")
	move := rowNamed(t, surface, "Mover")
	if !strings.Contains(attack, "data-targets") {
		t.Error("a linha do golpe não se marca como gesto que pede alvo")
	}
	if strings.Contains(move, "data-targets") {
		t.Error("o Mover não pede alvo, e se marcou como se pedisse")
	}
}

// rowNamed é a linha das Ações com este nome, do começo dela até a próxima.
func rowNamed(t *testing.T, surface, name string) string {
	t.Helper()
	at := strings.Index(surface, name)
	if at < 0 {
		t.Fatalf("as Ações não têm a linha %q", name)
	}
	start := strings.LastIndex(surface[:at], `<div class="ds-row ds-turn-action"`)
	end := strings.Index(surface[at:], `<div class="ds-row ds-turn-action"`)
	if end < 0 {
		end = len(surface) - at
	}
	return surface[start : at+end]
}
