package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestSendingToTheTablePutsOneRowPerCopy(t *testing.T) {
	f := newSceneFixture(t)

	rec := f.pede(t, f.gm, "POST", f.tableUrl()+"/bestiario/enviar",
		`{"creature":"goblin-salteador","entry_hp":4,"entry_initiative":13,"entry_copies":3}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("mandar para a mesa deu %d: %s", rec.Code, trechoDeSinais(rec.Body.String()))
	}

	queue := f.s.tableHost().Sessions().GetState(f.sessionID).Initiative
	if len(queue) != 3 {
		t.Fatalf("a fila ficou com %d combatentes, queria 3", len(queue))
	}
	labels := map[string]bool{}
	for _, e := range queue {
		labels[e.Label] = true
		if e.MonsterID == nil || *e.MonsterID != "goblin-salteador" {
			t.Errorf("a linha %q não levou o monsterId: %v", e.Label, e.MonsterID)
		}
		if e.Initiative != 13 {
			t.Errorf("a linha %q entrou com iniciativa %d, e o bando entra junto", e.Label, e.Initiative)
		}
		if e.HpMax == nil || *e.HpMax != 4 {
			t.Errorf("a linha %q não levou o PV do livro: %v", e.Label, e.HpMax)
		}
	}
	// Os TRÊS têm rótulos distintos, e é o servidor que os numera: três linhas
	// com o mesmo nome na fila deixam o mestre sem saber qual ele feriu.
	if len(labels) != 3 {
		t.Errorf("as três cópias ficaram com %d rótulos distintos: %v", len(labels), labels)
	}
}

// O teto de cópias existe para separar "quatro goblins" de um zero a mais.
//
// Ele não é regra do livro — é o que impede a fila de encher e o mestre ter de
// desfazer linha por linha. E o `min`/`max` do campo não é a trava: digitar
// passa direto pelo spinner.
func TestTheCopyCeilingIsEnforcedOnTheServer(t *testing.T) {
	f := newSceneFixture(t)

	rec := f.pede(t, f.gm, "POST", f.tableUrl()+"/bestiario/enviar",
		`{"creature":"goblin-salteador","entry_hp":4,"entry_initiative":13,"entry_copies":99}`)
	if body := trechoDeSinais(rec.Body.String()); !strings.Contains(body, "99") {
		t.Errorf("a recusa não citou o valor ofensivo; sinais = %s", body)
	}
	if n := len(f.s.tableHost().Sessions().GetState(f.sessionID).Initiative); n != 0 {
		t.Errorf("entraram %d combatentes apesar da recusa", n)
	}
}

// Criatura que o livro não tem é recusada com o id na frase: ali um id
// desconhecido só chega por adulteração, e engolir em silêncio poria uma linha
// sem bloco na fila.
func TestAnInventedCreatureIsRefused(t *testing.T) {
	f := newSceneFixture(t)

	rec := f.pede(t, f.gm, "POST", f.tableUrl()+"/bestiario/enviar",
		`{"creature":"grifo-de-neon","entry_hp":10,"entry_initiative":10,"entry_copies":1}`)
	if body := trechoDeSinais(rec.Body.String()); !strings.Contains(body, "grifo-de-neon") {
		t.Errorf("a recusa não citou a criatura; sinais = %s", body)
	}
	if n := len(f.s.tableHost().Sessions().GetState(f.sessionID).Initiative); n != 0 {
		t.Errorf("entraram %d combatentes apesar da recusa", n)
	}
}

// O painel é o DONO do rascunho: PV, iniciativa e quantas nascem do bloco do
// livro a cada criatura ABERTA. Sem isso, o PV que o mestre baixou para um ogro
// reapareceria no próximo bicho e ele não teria como saber que carregou.
//
// A outra metade é a que quase ninguém escreve e é a que importa: FILTRAR não
// pode semear. Se semeasse, cada tecla da busca apagaria o PV que o mestre
// acabou de ajustar.
func TestThePanelSeedsTheDraftOnlyWhenAnotherCreatureOpens(t *testing.T) {
	f := newSceneFixture(t)
	panel := f.tableUrl() + "/bestiario"

	// Primeira abertura: o rascunho na tela não é de ninguém ainda.
	opened := f.pede(t, f.gm, http.MethodGet, panel+signals(`{"creature":"zumbi","draft_of":""}`), "").Body.String()
	if !strings.Contains(trechoDeSinais(opened), `"entry_hp":20`) {
		t.Errorf("abrir o Zumbi não semeou o PV do livro (20); sinais = %s", trechoDeSinais(opened))
	}
	if !strings.Contains(trechoDeSinais(opened), `"draft_of":"zumbi"`) {
		t.Errorf("o rascunho não ficou marcado como do Zumbi; sinais = %s", trechoDeSinais(opened))
	}

	// Segunda visita à MESMA criatura, agora com o rascunho já sendo dela: é o
	// que acontece a cada tecla da busca, e não pode semear nada.
	again := f.pede(t, f.gm, http.MethodGet,
		panel+signals(`{"creature":"zumbi","search":"zu","draft_of":"zumbi"}`), "").Body.String()
	// O CONTROLE: o painel FOI redesenhado, senão "não semeou" seria só "não
	// respondeu".
	if !strings.Contains(again, "table-bestiary") {
		t.Fatalf("o painel não voltou no remendo; a ausência abaixo não provaria nada")
	}
	// A asserção é sobre a LINHA DE SINAIS e não sobre o corpo inteiro: a
	// palavra `pvdoverbete` também aparece no HTML, no `data-bind` do campo de
	// ajuste. Procurá-la no corpo casava com o desenho e acusava o código por um
	// defeito que era do teste.
	if signals := trechoDeSinais(again); strings.Contains(signals, "entry_hp") {
		t.Errorf("filtrar semeou o rascunho de novo e apagaria o ajuste do mestre; sinais = %s", signals)
	}
}

// Nas duas metades: a lista diz o PV e a defesa de cada bicho — é exatamente o que o olho da linha
// esconde da mesa —, então a trava é do painel INTEIRO e não só do enviar.
func TestTheTableBestiaryBelongsToTheGm(t *testing.T) {
	f := newSceneFixture(t)

	routes := []struct{ method, path, body string }{
		{http.MethodGet, "/bestiario", ""},
		{"POST", "/bestiario/tipo/animal", ""},
		{"POST", "/bestiario/enviar", `{"creature":"zumbi","entry_hp":20,"entry_initiative":10,"entry_copies":1}`},
	}
	for _, route := range routes {
		rec := f.pede(t, f.player, route.method, f.tableUrl()+route.path, route.body)
		if rec.Code != http.StatusForbidden {
			t.Errorf("o jogador chamou %q e levou %d, quero 403", route.path, rec.Code)
		}
	}

	html := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(html, "Iniciativa") {
		t.Fatal("o jogador não viu a cena; a ausência abaixo não provaria nada")
	}
	// O painel não é ESCONDIDO na tela do jogador: ele não existe nela. Mandá-lo
	// e esconder por CSS entregaria as 80 criaturas com PV e defesa a quem
	// abrisse o inspetor.
	for _, mark := range []string{"table-bestiary", "Abrir o bestiário"} {
		if strings.Contains(html, mark) {
			t.Errorf("o HTML do jogador veio com %q", mark)
		}
	}
	if forGM := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(forGM, "table-bestiary") {
		t.Error("o mestre não recebeu o painel")
	}
}
