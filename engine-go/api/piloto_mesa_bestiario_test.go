package api

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// Os guardas do BESTIÁRIO DENTRO DA MESA (ALE-263).
//
// O desenho é o mesmo da cena do mestre e tem guarda lá; o que se prende aqui é
// o que só existe por ser DENTRO da mesa — a trava do papel, o envio para a
// fila, e a regra de quem é dono do rascunho.

// TestSendingToTheTablePutsOneRowPerCopy.
//
// Uma entrada por cópia, e quem numera os repetidos é o SERVIDOR (ALE-192): a
// tela não pode adivinhar um número que outro cliente acabou de usar. Todas
// entram com a MESMA iniciativa — é o que a mesa faz com um bando.
func TestSendingToTheTablePutsOneRowPerCopy(t *testing.T) {
	f := novoPiloto(t)

	rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/bestiario/enviar",
		`{"criatura":"goblin-salteador","pvdoverbete":4,"inidoverbete":13,"copiasdoverbete":3}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("mandar para a mesa deu %d: %s", rec.Code, trechoDeSinais(rec.Body.String()))
	}

	fila := f.s.sessions.GetState(f.sessionID).Initiative
	if len(fila) != 3 {
		t.Fatalf("a fila ficou com %d combatentes, queria 3", len(fila))
	}
	rotulos := map[string]bool{}
	for _, e := range fila {
		rotulos[e.Label] = true
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
	if len(rotulos) != 3 {
		t.Errorf("as três cópias ficaram com %d rótulos distintos: %v", len(rotulos), rotulos)
	}
}

// O teto de cópias existe para separar "quatro goblins" de um zero a mais.
//
// Ele não é regra do livro — é o que impede a fila de encher e o mestre ter de
// desfazer linha por linha. E o `min`/`max` do campo não é a trava: digitar
// passa direto pelo spinner (ALE-236).
func TestTheCopyCeilingIsEnforcedOnTheServer(t *testing.T) {
	f := novoPiloto(t)

	rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/bestiario/enviar",
		`{"criatura":"goblin-salteador","pvdoverbete":4,"inidoverbete":13,"copiasdoverbete":99}`)
	if corpo := trechoDeSinais(rec.Body.String()); !strings.Contains(corpo, "99") {
		t.Errorf("a recusa não citou o valor ofensivo; sinais = %s", corpo)
	}
	if n := len(f.s.sessions.GetState(f.sessionID).Initiative); n != 0 {
		t.Errorf("entraram %d combatentes apesar da recusa", n)
	}
}

// Criatura que o livro não tem é recusada com o id na frase: ali um id
// desconhecido só chega por adulteração, e engolir em silêncio poria uma linha
// sem bloco na fila.
func TestAnInventedCreatureIsRefused(t *testing.T) {
	f := novoPiloto(t)

	rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/bestiario/enviar",
		`{"criatura":"grifo-de-neon","pvdoverbete":10,"inidoverbete":10,"copiasdoverbete":1}`)
	if corpo := trechoDeSinais(rec.Body.String()); !strings.Contains(corpo, "grifo-de-neon") {
		t.Errorf("a recusa não citou a criatura; sinais = %s", corpo)
	}
	if n := len(f.s.sessions.GetState(f.sessionID).Initiative); n != 0 {
		t.Errorf("entraram %d combatentes apesar da recusa", n)
	}
}

// TestThePanelSeedsTheDraftOnlyWhenAnotherCreatureOpens.
//
// O painel é o DONO do rascunho: PV, iniciativa e quantas nascem do bloco do
// livro a cada criatura ABERTA. Sem isso, o PV que o mestre baixou para um ogro
// reapareceria no próximo bicho e ele não teria como saber que carregou.
//
// A outra metade é a que quase ninguém escreve e é a que importa: FILTRAR não
// pode semear. Se semeasse, cada tecla da busca apagaria o PV que o mestre
// acabou de ajustar.
func TestThePanelSeedsTheDraftOnlyWhenAnotherCreatureOpens(t *testing.T) {
	f := novoPiloto(t)
	painel := f.urlDaMesa() + "/bestiario"

	// Primeira abertura: o rascunho na tela não é de ninguém ainda.
	abriu := f.pede(t, f.mestre, http.MethodGet, painel+comSinais(`{"criatura":"zumbi","rascunhode":""}`), "").Body.String()
	if !strings.Contains(trechoDeSinais(abriu), `"pvdoverbete":20`) {
		t.Errorf("abrir o Zumbi não semeou o PV do livro (20); sinais = %s", trechoDeSinais(abriu))
	}
	if !strings.Contains(trechoDeSinais(abriu), `"rascunhode":"zumbi"`) {
		t.Errorf("o rascunho não ficou marcado como do Zumbi; sinais = %s", trechoDeSinais(abriu))
	}

	// Segunda visita à MESMA criatura, agora com o rascunho já sendo dela: é o
	// que acontece a cada tecla da busca, e não pode semear nada.
	dinovo := f.pede(t, f.mestre, http.MethodGet,
		painel+comSinais(`{"criatura":"zumbi","busca":"zu","rascunhode":"zumbi"}`), "").Body.String()
	// O CONTROLE: o painel FOI redesenhado, senão "não semeou" seria só "não
	// respondeu".
	if !strings.Contains(dinovo, "bestiario-da-mesa") {
		t.Fatalf("o painel não voltou no remendo; a ausência abaixo não provaria nada")
	}
	// A asserção é sobre a LINHA DE SINAIS e não sobre o corpo inteiro: a
	// palavra `pvdoverbete` também aparece no HTML, no `data-bind` do campo de
	// ajuste. Procurá-la no corpo casava com o desenho e acusava o código por um
	// defeito que era do teste.
	if sinais := trechoDeSinais(dinovo); strings.Contains(sinais, "pvdoverbete") {
		t.Errorf("filtrar semeou o rascunho de novo e apagaria o ajuste do mestre; sinais = %s", sinais)
	}
}

// TestTheTableBestiaryBelongsToTheGm, nas duas metades (ALE-144).
//
// A lista diz o PV e a defesa de cada bicho — é exatamente o que o olho da linha
// esconde da mesa —, então a trava é do painel INTEIRO e não só do enviar.
func TestTheTableBestiaryBelongsToTheGm(t *testing.T) {
	f := novoPiloto(t)

	rotas := []struct{ metodo, caminho, corpo string }{
		{http.MethodGet, "/bestiario", ""},
		{"POST", "/bestiario/tipo/animal", ""},
		{"POST", "/bestiario/enviar", `{"criatura":"zumbi","pvdoverbete":20,"inidoverbete":10,"copiasdoverbete":1}`},
	}
	for _, rota := range rotas {
		rec := f.pede(t, f.jogador, rota.metodo, f.urlDaMesa()+rota.caminho, rota.corpo)
		if rec.Code != http.StatusForbidden {
			t.Errorf("o jogador chamou %q e levou %d, quero 403", rota.caminho, rec.Code)
		}
	}

	html := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(html, "Iniciativa") {
		t.Fatal("o jogador não viu a cena; a ausência abaixo não provaria nada")
	}
	// O painel não é ESCONDIDO na tela do jogador: ele não existe nela. Mandá-lo
	// e esconder por CSS entregaria as 80 criaturas com PV e defesa a quem
	// abrisse o inspetor.
	for _, marca := range []string{"bestiario-da-mesa", "Abrir o bestiário"} {
		if strings.Contains(html, marca) {
			t.Errorf("o HTML do jogador veio com %q", marca)
		}
	}
	if doMestre := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String(); !strings.Contains(doMestre, "bestiario-da-mesa") {
		t.Error("o mestre não recebeu o painel")
	}
}

// comSinais escreve os sinais do jeito que o Datastar os manda num GET: um
// parâmetro `datastar` com o JSON inteiro.
//
// A primeira versão deste teste usava query params soltos (`?criatura=zumbi`), e
// eles NÃO são a mesma coisa: o `criteriosDoPedido` lê os dois, mas o
// `rascunhode` só existe como sinal — então o teste mandava um pedido que o
// navegador nunca manda, e o painel semeava por não achar o rascunho. O teste
// acusou o código por um defeito que era dele.
func comSinais(json string) string {
	return "?datastar=" + url.QueryEscape(json)
}
