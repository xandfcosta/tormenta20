package api

import (
	"context"
	"net/http"
	"strings"
	"t20engine/tabuleiro"
	"t20engine/web/table"
	"testing"
)

func sceneIds(t *testing.T, f pilotoFixture) (ficha, npc string) {
	t.Helper()
	for _, e := range f.s.Sessions().GetState(f.sessionID).Initiative {
		switch e.Type {
		case "character":
			ficha = e.ID
		case "npc":
			npc = e.ID
		}
	}
	if ficha == "" || npc == "" {
		t.Fatalf("a cena não tem os dois lados: ficha=%q npc=%q", ficha, npc)
	}
	return ficha, npc
}

// TestPopulateBringsOnlyWhoWasChosen — o coração da ALE-204.
//
// A fila inteira num clique punha no mapa o vilão montado para aparecer no
// terceiro turno, e desfazer era peça por peça. O guarda tem CONTROLE: ele
// afirma que o NPC estava na fila antes de afirmar que ele não chegou ao mapa —
// senão "o ogro não veio" seria verdade sobre uma fila vazia.
func TestPopulateBringsOnlyWhoWasChosen(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	f.seedOpenBoard(t, "pedra")
	ficha, npc := sceneIds(t, f)

	f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/pecas", `{"escolhidosdomapa":"`+ficha+`"}`)

	b := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab)
	if len(b.Tokens) != 1 {
		t.Fatalf("o mapa ficou com %d peças, esperado 1", len(b.Tokens))
	}
	if id := b.Tokens[0].EntryID; id == nil || *id != ficha {
		t.Errorf("a peça no mapa não é a escolhida: %v", id)
	}
	for _, peca := range b.Tokens {
		if peca.EntryID != nil && *peca.EntryID == npc {
			t.Error("o NPC que ninguém escolheu foi para o mapa — é a emboscada vazando")
		}
	}
}

// TestWithoutAChoiceTheCommandRefusesInsteadOfBringingEveryone.
//
// `EntrySelection` nil significa TODAS no `populateBoard`, e é exatamente o
// padrão inseguro que a ALE-204 tirou do app. Um sinal perdido no caminho não
// pode virar "traz todo mundo": a diferença entre recusar e trazer a fila
// inteira é o vilão do terceiro turno aparecendo na tela da mesa.
func TestWithoutAChoiceTheCommandRefusesInsteadOfBringingEveryone(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	f.seedOpenBoard(t, "pedra")

	corpo := f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/pecas", `{"escolhidosdomapa":""}`)

	if b := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab); len(b.Tokens) != 0 {
		t.Fatalf("escolha vazia trouxe %d peças — nil virou TODAS", len(b.Tokens))
	}
	// E a recusa FALA: um comando que não faz nada e não diz nada é lido como
	// tela travada. O texto vai para o `erroDoComando`, o rodapé do mestre.
	if !strings.Contains(corpo, "escolha ao menos um") {
		t.Errorf("a recusa não chegou ao rodapé do mestre; resposta: %.200s", corpo)
	}
}

// TestTheTokenIsBornWithADisplacement — a metade que se perde num porte apressado.
//
// O `Populate` cria a peça e o `SetSpeeds` grava o orçamento de movimento dela.
// Sem o segundo a peça nasce no mapa sem deslocamento, o alcance não acende e o
// jogador vê uma peça que não anda — um meio-recurso que ninguém reporta porque
// parece regra. Provado VERMELHO tirando o `SetSpeeds` do `poeNoMapa`.
func TestTheTokenIsBornWithADisplacement(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	f.seedOpenBoard(t, "pedra")
	ficha, _ := sceneIds(t, f)

	f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/pecas", `{"escolhidosdomapa":"`+ficha+`"}`)

	b := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab)
	if len(b.Tokens) != 1 {
		t.Fatalf("o mapa ficou com %d peças, esperado 1", len(b.Tokens))
	}
	if b.Tokens[0].SpeedSquares <= 0 {
		t.Errorf("a peça nasceu com deslocamento %d — ela não anda", b.Tokens[0].SpeedSquares)
	}
}

// TestThePopulateDialogDoesNotReachThePlayer.
//
// A lista de candidatos é a fila INTEIRA, inclusive quem o mestre ainda não pôs
// em cena. Ela é montada só quando `v.Mestre`, e este guarda prende isso onde
// dói: no HTML que sai para o jogador. O vazamento não apareceria na tela — o
// diálogo nasce fechado —, só em "ver código-fonte".
//
// O CONTROLE vem primeiro e é obrigatório: sem afirmar que o diálogo EXISTE na
// página do mestre, "não achei no HTML do jogador" seria igualmente verdade se
// eu tivesse errado o seletor, e o guarda passaria verde sobre nada.
func TestThePopulateDialogDoesNotReachThePlayer(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	f.seedOpenBoard(t, "pedra")

	doMestre := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(doMestre, `id="por-no-mapa"`) {
		t.Fatal("o diálogo não está na página do MESTRE — o controle falhou, e sem ele o resto não mede nada")
	}

	doJogador := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	if strings.Contains(doJogador, `id="por-no-mapa"`) {
		t.Error("o diálogo do mestre foi para o HTML do jogador, com a fila inteira dentro")
	}
}

// TestThePlayerDoesNotPopulateTheMap — a trava é do SERVIDOR e não do desenho.
//
// O botão escondido é cortesia; quem postar na mão leva 403. É a mesma regra do
// `gmBoardCommand`, afirmada aqui porque esta rota é nova e a trava
// dela é uma linha de registro que alguém pode trocar sem perceber.
func TestThePlayerDoesNotPopulateTheMap(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	f.seedOpenBoard(t, "pedra")
	ficha, _ := sceneIds(t, f)

	rec := f.pede(t, f.jogador, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas", `{"escolhidosdomapa":"`+ficha+`"}`)

	if rec.Code != http.StatusForbidden {
		t.Errorf("o jogador pôs peça no mapa: %d", rec.Code)
	}
	if b := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab); len(b.Tokens) != 0 {
		t.Errorf("o mapa mudou apesar do 403 (%d peças)", len(b.Tokens))
	}
}

// TestTheCandidatesSayWhoIsAlreadyOnTheMap.
//
// A linha de quem já tem peça continua aparecendo, marcada e travada. Esconder
// faria o mestre procurar um nome que ele acabou de ver na fila; oferecer faria
// um clique que o servidor ignora, que é pior — parece que não funcionou.
func TestTheCandidatesSayWhoIsAlreadyOnTheMap(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	f.seedOpenBoard(t, "pedra")
	ficha, npc := sceneIds(t, f)

	f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/pecas", `{"escolhidosdomapa":"`+ficha+`"}`)

	b := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab)
	candidatos := table.MapCandidates(b, f.s.Sessions().GetState(f.sessionID))
	if len(candidatos) != 2 {
		t.Fatalf("a fila tem 2 combatentes e o diálogo ofereceu %d", len(candidatos))
	}
	for _, c := range candidatos {
		switch c.ID {
		case ficha:
			if !c.NoMapa {
				t.Error("quem acabou de virar peça continua sendo oferecido para trazer")
			}
			if !c.Ficha {
				t.Error("o PC não foi marcado como ficha — ele nasceria do lado errado do mapa")
			}
		case npc:
			if c.NoMapa {
				t.Error("o NPC que não foi escolhido aparece como se já estivesse no mapa")
			}
		}
	}
	// O atalho do clique direito escolhe as FICHAS que faltam, e agora não falta
	// nenhuma: sem isto ele reenviaria a mesma ficha a cada clique.
	if ids := table.MapOutsideSheets(candidatos); len(ids) != 0 {
		t.Errorf("o atalho ainda ofereceria %v, que já está no mapa", ids)
	}
}

// TestPopulateDoesNotPaintTerrain é um controle de vizinhança: este
// arquivo mexe no mesmo `BoardState` que o pincel, e as peças novas nascem em
// posições calculadas. Se pôr no mapa passasse a pintar chão, ninguém veria.
func TestPopulateDoesNotPaintTerrain(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	f.seedOpenBoard(t, "pedra")
	ficha, _ := sceneIds(t, f)

	f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/pecas", `{"escolhidosdomapa":"`+ficha+`"}`)

	b := f.s.Boards().Get(context.Background(), f.sessionID, defaultTab)
	for _, especie := range tabuleiro.TerrainKinds {
		if casas := tabuleiro.SquaresOf(b, especie.ID); len(casas) != 0 {
			t.Errorf("pôr no mapa pintou %s em %v", especie.ID, casas)
		}
	}
}
