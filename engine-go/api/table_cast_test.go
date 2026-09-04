package api

import (
	"context"
	"net/http"
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

// O ELENCO FERE E CURA QUEM NÃO ESTÁ NA FILA, que é o buraco desta fatia
// (ALE-211).
//
// As rotas de vital da fila são por `entryId`, e o elenco existe justamente
// para quem NÃO tem linha na iniciativa — o mestre curando a Arwen entre duas
// brigas é o caso comum, não a exceção. Antes disto o único caminho era pôr o
// herói na fila só para poder mexer nele, e tirar depois.
//
// A asserção é sobre a FICHA, e não sobre a fila: escrever só na entrada
// compilaria, deixaria o painel com um número plausível, e a ficha do jogador
// continuaria com o PV de antes (ALE-122).
func TestTheCastHealsSomeoneWhoIsNotInTheTracker(t *testing.T) {
	f := novoPiloto(t)
	ctx := context.Background()

	// O CONTROLE: o herói NÃO está na fila. Sem ele o caso mediria o caminho da
	// fila com outra URL, que é o que ele existe para não fazer.
	if fila := f.s.tableHost().Sessions().GetState(f.sessionID).Initiative; len(fila) != 0 {
		t.Fatalf("a bancada já pôs %d na fila — o caso mediria o outro caminho", len(fila))
	}
	antes, err := f.s.queries.GetCharacter(ctx, f.charID)
	if err != nil {
		t.Fatalf("ler a ficha: %v", err)
	}

	// O STATUS NÃO BASTA, e descobri isso sabotando: numa cena servida a recusa
	// é CONTEÚDO e volta 200, com a frase no `erroDoComando` do rodapé. Um
	// caso que olhasse só o código e a ficha passaria verde sobre um gesto que
	// escreveu a ficha e reprovou depois — que é exatamente a forma que a
	// sabotagem produziu.
	base := f.tableUrl() + "/elenco/" + strconv.FormatInt(f.charID, 10) + "/vitals/"
	for _, caminho := range []string{"hp/harm/5", "mp/harm/1"} {
		rec := f.pede(t, f.mestre, "POST", base+caminho, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("%s deu %d: %s", caminho, rec.Code, rec.Body.String())
		}
		if corpo := rec.Body.String(); strings.Contains(corpo, "erroDoComando") &&
			!strings.Contains(corpo, `"erroDoComando":""`) {
			t.Fatalf("%s foi recusado apesar do 200: %s", caminho, firstRows(corpo, 6))
		}
	}

	depois, err := f.s.queries.GetCharacter(ctx, f.charID)
	if err != nil {
		t.Fatalf("reler a ficha: %v", err)
	}
	if depois.Hpcurrent != antes.Hpcurrent-5 {
		t.Errorf("a ficha ficou com %d PV; %d-5 = %d", depois.Hpcurrent, antes.Hpcurrent, antes.Hpcurrent-5)
	}
	if depois.Mpcurrent != antes.Mpcurrent-1 {
		t.Errorf("a ficha ficou com %d PM; %d-1 = %d", depois.Mpcurrent, antes.Mpcurrent, antes.Mpcurrent-1)
	}
}

// A MESMA TRAVA do "pôr na fila" vale para ferir, e é por isso que ela virou
// função (ALE-211).
//
// O id vem do CAMINHO, e o caminho é digitável: sem a conferência contra o
// roster, o mestre de uma mesa feriria o personagem de OUTRA campanha — que é
// pior que pô-lo na fila, porque escreve na ficha de um estranho.
func TestTheCastVitalsRefuseSomeoneOutsideTheRoster(t *testing.T) {
	f := novoPiloto(t)
	ctx := context.Background()
	forasteiro := seedCharacterAtLevel(t, f.s, f.jogador, "Forasteiro", 3, 10, 10, 2, 4)
	antes, err := f.s.queries.GetCharacter(ctx, forasteiro)
	if err != nil {
		t.Fatalf("ler a ficha do forasteiro: %v", err)
	}

	corpo := f.posta(t, f.mestre,
		f.tableUrl()+"/elenco/"+strconv.FormatInt(forasteiro, 10)+"/vitals/hp/harm/5", "")

	if !strings.Contains(corpo, "não é jogador desta campanha") {
		t.Errorf("a recusa não veio; a resposta foi:\n%s", firstRows(corpo, 6))
	}
	// O CONTROLE do erro: uma recusa que já tivesse ESCRITO seria pior que
	// nenhuma, e a frase sozinha não diria.
	depois, err := f.s.queries.GetCharacter(ctx, forasteiro)
	if err != nil {
		t.Fatalf("reler a ficha do forasteiro: %v", err)
	}
	if depois.Hpcurrent != antes.Hpcurrent {
		t.Errorf("a recusa feriu mesmo assim: %d virou %d", antes.Hpcurrent, depois.Hpcurrent)
	}
}

// COM linha na fila, ela ESPELHA o que o elenco fez — senão as duas telas
// mostram números diferentes do mesmo herói, que é a ALE-122 literal.
func TestTheCastVitalsMirrorIntoTheTrackerWhenThereIsALine(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.tracker(t)

	base := f.tableUrl() + "/elenco/" + strconv.FormatInt(f.charID, 10) + "/vitals/"
	if rec := f.pede(t, f.mestre, "POST", base+"hp/harm/5", ""); rec.Code != http.StatusOK {
		t.Fatalf("ferir pelo elenco deu %d", rec.Code)
	}

	ficha, err := f.s.queries.GetCharacter(context.Background(), f.charID)
	if err != nil {
		t.Fatalf("reler a ficha: %v", err)
	}
	for _, e := range f.s.tableHost().Sessions().GetState(f.sessionID).Initiative {
		if e.ID != entryID {
			continue
		}
		if e.HpCurrent == nil || *e.HpCurrent != ficha.Hpcurrent {
			t.Errorf("a fila ficou com %v e a ficha com %d — as duas telas divergiram",
				e.HpCurrent, ficha.Hpcurrent)
		}
	}
}
