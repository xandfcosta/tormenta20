package api

import (
	"strings"
	"testing"
)

// A FORMA DO MESTRE (ALE-269) — os guardas do palco.
//
// O que se prende aqui NÃO é o desenho: é o contrato que o desenho pode quebrar
// em SILÊNCIO. A Mesa é remendada por REGIÃO, e o stream acha a região pelo id.
// Um id que some, ou que apareça duas vezes porque alguém pendurou a mesma
// região nas duas formas, faz o remendo escrever no vazio — e o sintoma disso
// não é uma exceção, é uma tela que PARA DE ATUALIZAR sozinha, que é a família
// de defeito mais cara deste repositório.
//
// Estes três casos foram vistos VERMELHOS antes de valerem:
//   - tirar `@mesaTrilhoDaFila` do palco → o primeiro falha
//   - pendurar `@mesaFila` no palco E na gaveta → o segundo falha, dizendo 2
//   - listar `mesa-trilho-fila` sem a guarda de papel → o terceiro falha

// asRegioesDaMesa são os ids que o stream remenda. A lista é a mesma do
// `regioesDaMesa`, escrita aqui de novo DE PROPÓSITO: derivá-la da produção
// faria o teste concordar com o defeito: uma região removida sumiria dos dois
// lados e o guarda ficaria verde sobre nada.
var asRegioesDaMesa = []string{
	"mesa-cabecalho",
	"mesa-registrar",
	"mesa-grupo",
	"mesa-tabuleiro",
	"mesa-por-no-mapa",
	"mesa-acervo",
	"mesa-config-da-sessao",
	"mesa-fila",
	"mesa-comandos",
}

// TestTheGmStageHasEveryRegionExactlyOnce é o guarda central da virada de
// forma: o mestre deixou de receber a coluna e passou a receber o palco, e o que
// não pode mudar é QUE REGIÕES existem no documento dele.
//
// "Uma vez só" é metade do teste e é a metade que pega o erro provável: mover
// uma região para o trilho e esquecer de tirá-la de onde estava deixa DUAS
// raízes com o mesmo id. O `PatchElements` acerta a primeira, a segunda envelhece
// na tela, e as duas discordam sobre a mesma fila — que é exatamente o defeito
// da ALE-122, agora em HTML.
func TestTheGmStageHasEveryRegionExactlyOnce(t *testing.T) {
	f := novoPiloto(t)
	f.cena(t)

	html := f.pede(t, f.mestre, "GET", f.urlDaMesa(), "").Body.String()

	for _, id := range append(asRegioesDaMesa, "mesa-trilho-fila") {
		marca := `id="` + id + `"`
		if n := strings.Count(html, marca); n != 1 {
			t.Errorf("a região %q aparece %d vezes no palco do mestre, e o remendo precisa de exatamente 1", id, n)
		}
	}
}

// TestThePlayerColumnDidNotGetTheGmRail é o outro lado, e o que ele
// afirma sobreviveu à forma do jogador (ALE-269) mesmo com o nome envelhecendo:
// o jogador deixou de estar numa coluna e passou a ter duas SUPERFÍCIES, mas o
// trilho continua não sendo dele.
//
// O trilho de 80px é do mestre, e mandá-lo ao jogador seria mandar a fila
// inteira para quem o `redactForPlayers` acabou de esvaziá-la — a trava da
// ALE-210 furada por leiaute.
func TestThePlayerColumnDidNotGetTheGmRail(t *testing.T) {
	f := novoPiloto(t)
	f.cena(t)

	html := f.pede(t, f.jogador, "GET", f.urlDaMesa(), "").Body.String()

	// O CONTROLE primeiro: sem ele, "não achei o trilho" seria verdade também
	// numa página que voltou vazia, num 403, ou num id que alguém renomeou.
	if !strings.Contains(html, `id="mesa-fila"`) {
		t.Fatal("o jogador não recebeu nem a fila — a página não é o que este teste pensa que é")
	}
	if strings.Contains(html, `id="mesa-trilho-fila"`) {
		t.Error("o jogador recebeu o trilho do mestre")
	}
}

// TestTheStreamOnlySendsTheRegionTheDocumentHas prende as DUAS pontas juntas, que é
// onde a divergência nasce: a lista de regiões do stream e a página são escritas
// em lugares diferentes e têm de concordar sobre quem existe.
//
// Discordar não dá erro em lugar nenhum — o remendo simplesmente não pousa —,
// e por isso a guarda de papel do `regioesDaMesa` e o `if` que desenha o trilho
// leem a MESMA `view`. Este teste é o que afirma que continuam lendo.
func TestTheStreamOnlySendsTheRegionTheDocumentHas(t *testing.T) {
	f := novoPiloto(t)
	f.cena(t)

	paraOMestre := idsDasRegioes(t, f, f.mestre)
	paraOJogador := idsDasRegioes(t, f, f.jogador)

	if !paraOMestre["mesa-trilho-fila"] {
		t.Error("o palco do mestre desenha o trilho e o stream não o remenda: ele nasce e nunca mais muda")
	}
	if paraOJogador["mesa-trilho-fila"] {
		t.Error("o stream manda ao jogador uma região que a coluna dele não tem — o remendo escreve no vazio")
	}
}

// idsDasRegioes pergunta ao MESMO `regioesDaMesa` da produção quais regiões o
// stream mandaria para aquele papel, montando a view pelo caminho de sempre.
func idsDasRegioes(t *testing.T, f pilotoFixture, userID int64) map[string]bool {
	t.Helper()
	user, err := f.s.queries.GetUserByID(t.Context(), userID)
	if err != nil {
		t.Fatalf("usuário %d não existe: %v", userID, err)
	}
	view, _, err := f.s.loadMesaView(t.Context(), AuthUser{ID: user.ID}, f.campaignID, f.sessionID)
	if err != nil {
		t.Fatalf("montar a view: %v", err)
	}
	ids := map[string]bool{}
	for _, r := range regioesDaMesa(view) {
		ids[r.ID] = true
	}
	return ids
}
