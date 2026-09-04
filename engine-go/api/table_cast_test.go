package api

import (
	"strconv"
	"strings"
	"t20engine/web/table"
	"testing"
)

func TestTheGmDoesNotTrackWhoIsNotInTheCampaign(t *testing.T) {
	f := novoPiloto(t)
	// Um personagem que existe, mas de OUTRO dono e fora do roster desta mesa.
	forasteiro := seedCharacterAtLevel(t, f.s, f.jogador, "Forasteiro", 3, 10, 10, 2, 4)

	corpo := f.posta(t, f.mestre,
		f.tableUrl()+"/elenco/"+strconv.FormatInt(forasteiro, 10)+"/na-fila", "{}")

	if !strings.Contains(corpo, "não é jogador desta campanha") {
		t.Errorf("a recusa não veio; a resposta foi:\n%s", firstRows(corpo, 6))
	}
	// O CONTROLE do erro: uma recusa que já tivesse ESCRITO seria pior que
	// nenhuma, e o status sozinho não diria.
	for _, l := range f.s.tableHost().Sessions().GetState(f.sessionID).Initiative {
		if l.CharacterID != nil && *l.CharacterID == forasteiro {
			t.Fatal("o forasteiro entrou na fila apesar da recusa")
		}
	}
}

// TestTheCastPutsAPlayerInTheTrackerLinkedToTheSheet é o buraco que o "Adicionar grupo" não cobre:
// ele traz o grupo INTEIRO, e a cena em que só um desce na cripta não tinha
// gesto nenhum.
//
// A linha tem de nascer LIGADA À FICHA (`characterId`), que é a diferença entre
// este caminho e o mestre digitar o nome à mão: sem o id ela fica fora do
// descanso, sem PV de verdade e sem o fio de volta até a pessoa.
func TestTheCastPutsAPlayerInTheTrackerLinkedToTheSheet(t *testing.T) {
	f := novoPiloto(t)

	f.posta(t, f.mestre,
		f.tableUrl()+"/elenco/"+strconv.FormatInt(f.charID, 10)+"/na-fila", "{}")

	fila := f.s.tableHost().Sessions().GetState(f.sessionID).Initiative
	if len(fila) != 1 {
		t.Fatalf("a fila tem %d linhas, queria 1", len(fila))
	}
	if fila[0].CharacterID == nil || *fila[0].CharacterID != f.charID {
		t.Error("a linha nasceu SEM `characterId`: ela fica fora do descanso e sem fio até a ficha")
	}
}

// TestAddingItTwiceDoesNotDuplicateTheEntry.
//
// O elenco esconde o botão de quem já está na fila, mas isso é UX: dois cliques
// rápidos, duas abas, ou um remendo atrasado chegam ao servidor do mesmo jeito.
// Quem garante é o `populateParty`, e este guarda é o que afirma que o caminho
// novo passa por ele em vez de escrever direto.
func TestAddingItTwiceDoesNotDuplicateTheEntry(t *testing.T) {
	f := novoPiloto(t)
	rota := f.tableUrl() + "/elenco/" + strconv.FormatInt(f.charID, 10) + "/na-fila"

	f.posta(t, f.mestre, rota, "{}")
	f.posta(t, f.mestre, rota, "{}")

	if n := len(f.s.tableHost().Sessions().GetState(f.sessionID).Initiative); n != 1 {
		t.Errorf("dois cliques deram %d linhas", n)
	}
}

// TestThePlayerPutsNobodyInTheTracker — o papel, no servidor.
func TestThePlayerPutsNobodyInTheTracker(t *testing.T) {
	f := novoPiloto(t)

	rec := f.pede(t, f.jogador, "POST",
		f.tableUrl()+"/elenco/"+strconv.FormatInt(f.charID, 10)+"/na-fila", "{}")

	if rec.Code != 403 {
		t.Errorf("o jogador montou a fila do mestre: %d", rec.Code)
	}
}

// TestTheCastSaysWhoIsAlreadyInTheTracker prende a marca que decide o que a tela OFERECE.
//
// Oferecer "pôr na iniciativa" a quem já está lá é desenhar um gesto que só
// pode não fazer nada — a mesma regra que trava os verbos do ciclo da sessão.
func TestTheCastSaysWhoIsAlreadyInTheTracker(t *testing.T) {
	f := novoPiloto(t)
	rota := f.tableUrl() + "/elenco/" + strconv.FormatInt(f.charID, 10) + "/na-fila"

	antes := f.castMember(t, f.charID)
	if antes.NaFila {
		t.Fatal("o personagem já nasceu marcado como na fila — o teste mediria nada")
	}
	f.posta(t, f.mestre, rota, "{}")

	if depois := f.castMember(t, f.charID); !depois.NaFila {
		t.Error("pôs na fila e o elenco não soube: o botão continuaria oferecendo o gesto")
	}
}

// castMember monta a view pelo caminho de sempre e devolve um cartão do
// Grupo. Perguntar à view e não ao banco é deliberado: é a view que a tela
// desenha, e é nela que a marca precisa chegar.
func (f pilotoFixture) castMember(t *testing.T, characterID int64) table.Member {
	t.Helper()
	view, _, err := f.s.tableScene.LoadView(t.Context(), f.mestre, f.campaignID, f.sessionID)
	if err != nil {
		t.Fatalf("montar a view: %v", err)
	}
	for _, m := range view.Grupo {
		if m.CharacterID == characterID {
			return m
		}
	}
	t.Fatalf("o personagem %d não está no elenco da view", characterID)
	return table.Member{}
}
