package api

import (
	"net/http"
	"strings"
	"t20engine/domain/live"
	"testing"
)

// Esconder o botão do jogador é UX; a trava é o servidor. Este teste posta na
// mão, como quem abre o console: botão ausente nunca foi prova de trava.
func TestOnlyTheGmCommandsTheTable(t *testing.T) {
	f := newSceneFixture(t)

	// O corpo é o dos SINAIS que o Datastar manda junto. Só o descanso de dia lê
	// algum; os outros levam corpo vazio, que é o que o `@post` manda quando a
	// página não tem sinal nenhum a declarar.
	commands := []struct{ route, signals string }{
		{"iniciativa/proxima-vez", ""},
		{"iniciativa/vez-anterior", ""},
		{"cena/iniciar/acao", ""},
		{"cena/encerrar", ""},
		{"iniciativa/por-no-mapa", ""},
		{"descanso/cena", ""},
		{"descanso/dia", `{"rest_quality":"normal"}`},
	}
	for _, cmd := range commands {
		t.Run(cmd.route, func(t *testing.T) {
			rec := f.requests(t, f.player, "POST", f.tableUrl()+"/"+cmd.route, cmd.signals)
			if rec.Code != http.StatusForbidden {
				t.Errorf("o jogador comandou %q e levou %d, quero 403", cmd.route, rec.Code)
			}
			if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/"+cmd.route, cmd.signals); rec.Code != http.StatusOK {
				t.Errorf("o mestre foi recusado em %q com %d", cmd.route, rec.Code)
			}
		})
	}
}

// O comando remenda a cena na resposta, em vez de esperar o tique do stream: o
// avanço é o botão mais clicado da sessão, e o tique vai calar de todo jeito
// porque o hash não muda depois do remendo.
func TestTheCommandPatchesTheSceneRightAway(t *testing.T) {
	f := newSceneFixture(t)

	rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/cena/iniciar/acao", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type %q — o comando não remendou a cena", ct)
	}
	// A cena tem de voltar já com a cena ABERTA: se ela voltasse com o estado
	// velho, o mestre veria "Iniciar cena" depois de tê-la iniciado.
	if !strings.Contains(rec.Body.String(), "Encerrar cena") {
		t.Error("o remendo veio com o estado anterior")
	}
}

// Uma escrita pela cena tem de chegar ao HUB: quem está ouvindo o stream fica
// com o estado velho se ela não chegar.
func TestTheCommandAnnouncesToTheWholeTable(t *testing.T) {
	f := newSceneFixture(t)
	conn := f.s.sse.Add(f.sessionID, "espia", "gm")
	defer f.s.sse.Remove(f.sessionID, "espia")

	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/cena/iniciar/acao", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}

	var saw bool
	for {
		select {
		case frame := <-conn.Frames:
			if strings.Contains(string(frame), "session-state") {
				saw = true
			}
			continue
		default:
		}
		break
	}
	if !saw {
		t.Error("o comando do app não avisou o hub — a SPA ficaria com o estado velho")
	}
}

// O "Encerrar cena" passa pelo caso de uso, que expira a duração "cena"
// de toda ficha do grupo ANTES de desligar a cena. Chamar `sessions.EndScene`
// direto só mexe no rastreador: a fila zera na tela e a bênção de duração
// "cena" segue viva na ficha. O que se prende é a chamada ao mesmo helper, e
// não a sequência repetida aqui.
func TestEndingTheSceneFromTheTableExpiresThePartyBlessings(t *testing.T) {
	f := newSceneFixture(t)
	seedEffect(t, f.s, f.charID, "bencao", "scene")
	seedEffect(t, f.s, f.charID, "heroismo", "day")

	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/cena/iniciar/acao", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/cena/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar cena deu %d", rec.Code)
	}

	// Os DOIS lados: o de cena sai, o de dia FICA. Limpar demais apagaria a
	// bênção que o grupo comprou para o dia todo, e ninguém veria.
	if got := effectScopes(t, f.s, f.charID); len(got) != 1 || got[0] != "day" {
		t.Errorf("a ficha do grupo ficou com os escopos %v, queria só [day]", got)
	}
}

// O `session-state` não serve para isto: as fichas não estão no estado do
// rastreador. Sem o `session-rest`, quem está com a ficha aberta continuaria
// vendo o efeito morto e o "usado 1/cena" gasto até recarregar.
func TestEndingTheSceneFromTheTableAnnouncesTheSheetsChanged(t *testing.T) {
	f := newSceneFixture(t)
	conn := f.s.sse.Add(f.sessionID, "espia", "gm")
	defer f.s.sse.Remove(f.sessionID, "espia")

	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/cena/iniciar/acao", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/cena/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar cena deu %d", rec.Code)
	}

	// O CONTROLE contra ler ausência como evidência: o `session-state` sai
	// sempre, então achá-lo prova que o canal está aberto. Sem ele, "não achei
	// o session-rest" e "o canal não existe" seriam a mesma linha no terminal.
	var sawState, sawSheets bool
	for {
		select {
		case frame := <-conn.Frames:
			if strings.Contains(string(frame), "session-state") {
				sawState = true
			}
			if strings.Contains(string(frame), "session-rest") {
				sawSheets = true
			}
			continue
		default:
		}
		break
	}
	if !sawState {
		t.Fatal("nem o session-state chegou — o canal não estava aberto, e a ausência abaixo não seria evidência de nada")
	}
	if !sawSheets {
		t.Error("a mesa não foi avisada de que as fichas mudaram")
	}
}

// O Datastar não desenha corpo de resposta 4xx, então recusar por `http.Error`
// é um beco: a frase não chega a lugar nenhum e o mestre clica olhando para uma
// tela que não muda. E não alcançar as fichas do grupo ABORTA o encerrar-cena
// de propósito, deixando a cena LIGADA — sem frase, não há como saber por quê.
func TestTheRefusedCommandReachesTheGm(t *testing.T) {
	f := newSceneFixture(t)
	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/cena/iniciar/acao", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	// A sabotagem: sem o roster não há como alcançar as fichas, e o gesto
	// inteiro tem de recusar.
	if _, err := f.s.db.Exec("DROP TABLE campaign_members"); err != nil {
		t.Fatalf("derrubar a tabela: %v", err)
	}

	rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/cena/encerrar", "")
	body := rec.Body.String()
	if !strings.Contains(body, "command_error") {
		t.Fatalf("a recusa não chegou à cena do mestre; corpo = %q", body)
	}
	if strings.Contains(body, `"command_error":""`) {
		t.Error("a cena recebeu a frase VAZIA — o mestre veria a cena ligada e nenhuma explicação")
	}
}

// O sinal do comando é OUTRO que o `$error` do registrar: um sinal só faria a
// recusa de "Adicionar grupo" acender a frase vermelha dentro da caixa
// "Registrar iniciativa" do mestre que também joga — a frase certa no lugar
// errado.
func TestTheCommandErrorDoesNotInvadeTheRecordError(t *testing.T) {
	f := newSceneFixture(t)
	body := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(body, "command_error") {
		t.Error("a página do mestre não declarou o sinal do comando")
	}
}

// As duas metades são o gesto: trazer o grupo, e o segundo clique NÃO
// duplicar. A idempotência é o que sustenta o botão continuar clicável — o
// mestre que aceitou um jogador atrasado clica de novo e leva só o que faltava.
func TestAddPartyBringsTheCharactersAndCanBeClickedAgain(t *testing.T) {
	f := newSceneFixture(t)

	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/por-no-mapa", ""); rec.Code != http.StatusOK {
		t.Fatalf("adicionar grupo deu %d", rec.Code)
	}
	queue := stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative
	if len(queue) != 1 || queue[0].CharacterID == nil || *queue[0].CharacterID != f.charID {
		t.Fatalf("a fila ficou %+v, queria só o personagem %d", queue, f.charID)
	}

	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/por-no-mapa", ""); rec.Code != http.StatusOK {
		t.Fatalf("o segundo clique deu %d", rec.Code)
	}
	if after := stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative; len(after) != 1 {
		t.Errorf("o segundo clique deixou %d combatentes na fila, queria 1", len(after))
	}
}

// QUEM é o grupo, que é o predicado do gesto — e ele precisa de TRÊS membros
// para ser medido: com um só, "todos os membros" e "só o do jogador" dão a mesma
// fila, e o caso em que o filtro erraria não chega a existir.
//
// Não há filtro por papel, e o personagem do MESTRE entra: a coluna `role` nunca
// teve outro valor além de `player` em produção, e esperar dois aqui seria um
// verde sobre um estado que só a bancada sabe montar.
func TestAddPartyBringsEveryMemberIncludingTheGmsOwnCharacter(t *testing.T) {
	f := newSceneFixture(t)
	otherPlayer := seedUser(t, f.s, "jogador2@t.com")
	others := seedCharacter(t, f.s, otherPlayer, "Arwen")
	forGM := seedCharacterAtLevel(t, f.s, f.gm, "Bardo do mestre", "Bardo", 1, 0, 0)
	seedMember(t, f.s, f.campaignID, others)
	seedMember(t, f.s, f.campaignID, forGM)

	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/por-no-mapa", ""); rec.Code != http.StatusOK {
		t.Fatalf("adicionar grupo deu %d", rec.Code)
	}

	queue := stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative
	inQueue := map[int64]bool{}
	for _, e := range queue {
		if e.CharacterID != nil {
			inQueue[*e.CharacterID] = true
		}
	}
	for _, who := range []struct {
		id   int64
		name string
	}{{f.charID, "o personagem do jogador"}, {others, "o do segundo jogador"}, {forGM, "o do próprio mestre"}} {
		if !inQueue[who.id] {
			t.Errorf("%s (%d) não entrou na fila; ela ficou com %d linhas", who.name, who.id, len(queue))
		}
	}
	if len(queue) != 3 {
		t.Errorf("a fila ficou com %d linhas, queria os 3 membros da campanha", len(queue))
	}
}

// E o botão só existe para o MESTRE, porque a view do jogador não tem o que
// desenhar. Esconder por classe deixaria o HTML na página para quem abrisse o
// inspetor — e a trava de verdade é o 403 acima, medido em separado.
func TestThePlayerDoesNotGetAddPartyInTheHtml(t *testing.T) {
	f := newSceneFixture(t)

	if body := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String(); strings.Contains(body, "Adicionar grupo") {
		t.Error("o HTML do jogador veio com o Adicionar grupo")
	}
	if body := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(body, "Adicionar grupo") {
		t.Error("o mestre não recebeu o Adicionar grupo")
	}
}

// ── a recuperação (T20 p106) ─────────────────────────────────────────────────

// O guarda que carrega a REGRA, e ele mira o desfecho mais silencioso: a conta
// do livro cai em "normal" quando não reconhece a palavra, então uma
// qualidade que não chegasse ao servidor não daria erro nenhum.
//
// Por isso a asserção é sobre o NÚMERO, com a qualidade "ruim": nível 8
// recupera 4 em "ruim" e 8 em "normal", então 24 de PV prova que o sinal
// atravessou e 28 provaria que ele se perdeu no caminho.
func TestTheDayRestUsesTheQualityTheGmChose(t *testing.T) {
	f := newSceneFixture(t)

	// O PV de partida é LIDO e não escrito à mão: o poço vem do livro, e um
	// número fixo aqui afirmaria a tabela de classe em vez do descanso.
	before := poolsOf(t, f.s, f.charID)

	rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/descanso/dia", `{"rest_quality":"ruim"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("descanso de dia deu %d: %s", rec.Code, rec.Body.String())
	}

	sheet := poolsOf(t, f.s, f.charID)
	// Os números do descanso estão escritos à mão de propósito: o RUIM devolve
	// metade do nível (8/2 = 4), e o normal devolveria o nível inteiro. Derivá-los
	// da regra faria a asserção andar junto com o defeito.
	if wanted := before.HpCurrent + 4; sheet.HpCurrent != wanted {
		t.Errorf("PV = %d; %d é o descanso RUIM de um nível 8 (%d+4), %d seria o normal que ninguém pediu",
			sheet.HpCurrent, wanted, before.HpCurrent, before.HpCurrent+8)
	}
}

// QUEM o descanso visita, que é o predicado da varredura — e ele precisa de
// TRÊS membros para ser medido: com um só, "todos os membros" e "só o do
// jogador" curam a mesma ficha, e o caso em que o filtro erraria não existe.
//
// O personagem do MESTRE entra: não há filtro de papel, a coluna `role` foi
// substituída pelo `ownerId` na ALE-287, e um mestre que também joga tem ficha
// no grupo.
func TestTheDayRestHealsEveryMemberIncludingTheGmsOwn(t *testing.T) {
	f := newSceneFixture(t)
	other := seedUser(t, f.s, "jogador2@t.com")
	// Nível 5, feridos: o descanso "normal" devolve 5, e o teto não interfere.
	// Os dois partem de 1 PV, e o dano sai do POÇO de cada classe — eles têm
	// poços diferentes, e um dano igual para os dois os deixaria em pontos
	// diferentes da barra.
	others := seedCharacterAtLevel(t, f.s, other, "Arwen", "Guerreiro", 5,
		bookPools(t, f.s, "Guerreiro", 5).PvMax-1, 0)
	forGM := seedCharacterAtLevel(t, f.s, f.gm, "Bardo do mestre", "Bardo", 5,
		bookPools(t, f.s, "Bardo", 5).PvMax-1, 0)
	seedMember(t, f.s, f.campaignID, others)
	seedMember(t, f.s, f.campaignID, forGM)

	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/descanso/dia", `{"rest_quality":"normal"}`); rec.Code != http.StatusOK {
		t.Fatalf("descanso de dia deu %d: %s", rec.Code, rec.Body.String())
	}

	for _, who := range []struct {
		name string
		id   int64
	}{{"o do segundo jogador", others}, {"o do próprio mestre", forGM}} {
		sheet := poolsOf(t, f.s, who.id)
		// Eles partem de 1 PV, e o descanso normal devolve o NÍVEL inteiro.
		if sheet.HpCurrent != 1+5 {
			t.Errorf("%s ficou com %d PV; o descanso normal de um nível 5 devolve 5 (1+5)",
				who.name, sheet.HpCurrent)
		}
	}
}

// E uma qualidade que não existe é RECUSADA, não rebaixada em silêncio: o
// motor cai em "normal" por conta própria, e um sinal adulterado faria o grupo
// descansar em "normal" enquanto a tela dizia "luxuosa".
func TestAnInventedQualityIsRefused(t *testing.T) {
	f := newSceneFixture(t)

	// O PV de partida é lido ANTES do gesto. Ler depois faria a asserção comparar
	// o número consigo mesmo e passar sempre — foi o que aconteceu na primeira
	// escrita deste caso.
	before := poolsOf(t, f.s, f.charID)

	rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/descanso/dia", `{"rest_quality":"palaciana"}`)
	body := rec.Body.String()
	if !strings.Contains(body, "palaciana") {
		t.Errorf("a recusa não citou o valor ofensivo; corpo = %q", body)
	}
	if !strings.Contains(body, "luxuosa") {
		t.Errorf("a recusa não disse a forma esperada; corpo = %q", body)
	}

	sheet := poolsOf(t, f.s, f.charID)
	if sheet.HpCurrent != before.HpCurrent {
		t.Errorf("PV = %d — o grupo descansou mesmo com a qualidade recusada", sheet.HpCurrent)
	}
}

// A recuperação de CENA é o mesmo gesto do encerrar cena sem desligar a cena:
// expira a duração "cena" das fichas do grupo e avisa que elas mudaram. O que
// se prende é a chamada ao mesmo caso de uso, e não uma sequência própria.
func TestTheSceneRestExpiresTheSheetsWithoutTurningTheSceneOff(t *testing.T) {
	f := newSceneFixture(t)
	seedEffect(t, f.s, f.charID, "bencao", "scene")
	seedEffect(t, f.s, f.charID, "heroismo", "day")
	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/cena/iniciar/acao", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}

	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/descanso/cena", ""); rec.Code != http.StatusOK {
		t.Fatalf("recuperar a cena deu %d", rec.Code)
	}

	if got := effectScopes(t, f.s, f.charID); len(got) != 1 || got[0] != "day" {
		t.Errorf("a ficha ficou com os escopos %v, queria só [day]", got)
	}
	// A diferença para o "Encerrar cena": a cena continua LIGADA. Recuperar ao
	// fim de uma luta não acaba a cena, e confundir os dois tiraria a fila da
	// mesa no meio do combate.
	if !stateOf(t, f.s.tableHost().Sessions(), f.sessionID).InScene() {
		t.Error("a recuperação de cena desligou a cena")
	}
}

// ── os verbos da LINHA ───────────────────────────────────────────────────────

// tracker põe o grupo na fila e devolve o id do combatente do personagem.
func (f sceneFixture) tracker(t *testing.T) string {
	t.Helper()
	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/por-no-mapa", ""); rec.Code != http.StatusOK {
		t.Fatalf("adicionar grupo deu %d", rec.Code)
	}
	for _, e := range stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative {
		if e.CharacterID != nil && *e.CharacterID == f.charID {
			return e.ID
		}
	}
	t.Fatal("o personagem não entrou na fila")
	return ""
}

// Com personagem atrás da linha, quem manda é a FICHA: o dano é aplicado lá (é
// ela quem sabe drenar PV temporários) e a entrada ESPELHA o resultado.
//
// Por isso a asserção é sobre a FICHA e não sobre a linha: escrever só na
// entrada compilaria, deixaria a fila com um número plausível, e a ficha do
// jogador continuaria com o PV de antes.
func TestWoundingARowGoesThroughTheSheet(t *testing.T) {
	f := newSceneFixture(t)
	entryID := f.tracker(t)

	// O PV de partida é LIDO: o poço vem do livro, e um 20 escrito à mão faria
	// este caso afirmar a tabela de classe em vez do caminho do dano.
	before := poolsOf(t, f.s, f.charID)
	wanted := before.HpCurrent - 5

	rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/"+entryID+"/vitais/hp/ferir/5", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("ferir deu %d: %s", rec.Code, rec.Body.String())
	}

	sheet := poolsOf(t, f.s, f.charID)
	if sheet.HpCurrent != wanted {
		t.Errorf("a FICHA ficou com %d PV; %d-5 = %d — o dano não chegou nela",
			sheet.HpCurrent, before.HpCurrent, wanted)
	}
	// E a linha espelha, senão a fila mostraria o número velho ao lado da ficha
	// certa.
	for _, e := range stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative {
		if e.ID == entryID && (e.HpCurrent == nil || *e.HpCurrent != wanted) {
			t.Errorf("a linha não espelhou a ficha: %v", e.HpCurrent)
		}
	}
}

// O passo não é dado que a página manda: são duas rotas por verbo. Um passo
// inventado não casa rota nenhuma, e a recusa nomeia o valor e a forma
// esperada.
func TestTheVitalStepComesFromThePathAndThereAreOnlyTwo(t *testing.T) {
	f := newSceneFixture(t)
	entryID := f.tracker(t)
	base := f.tableUrl() + "/iniciativa/" + entryID + "/vitais/hp/"

	// O CONTROLE: os dois passos que existem passam. Sem ele, "o inventado
	// falhou" também seria verdade se a rota inteira estivesse quebrada.
	for _, step := range []string{"1", "5"} {
		if rec := f.requests(t, f.gm, "POST", base+"curar/"+step, ""); rec.Code != http.StatusOK {
			t.Fatalf("curar em %s deu %d", step, rec.Code)
		}
	}
	rec := f.requests(t, f.gm, "POST", base+"ferir/99", "")
	if body := rec.Body.String(); !strings.Contains(body, "99") || !strings.Contains(body, "Shift") {
		t.Errorf("a recusa do passo 99 não citou o valor e os passos que existem; corpo = %q", body)
	}
}

// Dois cliques voltam ao começo, e é isso que prova que quem decide é o
// SERVIDOR: se a página mandasse o valor desejado, duas abas do mestre com o
// remendo atrasado mandariam "esconder" duas vezes e a segunda desfaria a
// primeira sem ninguém pedir.
func TestTheEyeInvertsTheStateTheServerKeeps(t *testing.T) {
	f := newSceneFixture(t)
	entryID := f.tracker(t)
	eye := f.tableUrl() + "/iniciativa/" + entryID + "/vitais/hp/oculto"

	hidden := func() bool {
		for _, e := range stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative {
			if e.ID == entryID {
				return e.HpHidden != nil && *e.HpHidden
			}
		}
		t.Fatal("a linha sumiu")
		return false
	}
	if hidden() {
		t.Fatal("a linha nasceu escondida — o teste mediria o contrário do que quer")
	}
	if rec := f.requests(t, f.gm, "POST", eye, ""); rec.Code != http.StatusOK {
		t.Fatalf("esconder deu %d", rec.Code)
	}
	if !hidden() {
		t.Fatal("o primeiro clique não escondeu")
	}
	if rec := f.requests(t, f.gm, "POST", eye, ""); rec.Code != http.StatusOK {
		t.Fatalf("revelar deu %d", rec.Code)
	}
	if hidden() {
		t.Error("o segundo clique não revelou — o olho não é interruptor, é um valor que a página manda")
	}
}

// O MANA passa pelo mesmo caminho do PV, e a asserção é sobre a FICHA:
// escrever só na entrada compilaria e deixaria a fila com um número plausível
// ao lado de uma ficha que não gastou mana.
func TestSpendingManaGoesThroughTheSheetToo(t *testing.T) {
	f := newSceneFixture(t)
	entryID := f.tracker(t)

	before := poolsOf(t, f.s, f.charID)
	// O CONTROLE: sem mana para gastar o caso mediria um clampeamento em zero e
	// passaria verde sobre nada.
	if before.MpCurrent < 5 {
		t.Fatalf("a ficha semeada tem %d PM: o caso precisa de mana para gastar", before.MpCurrent)
	}

	rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/"+entryID+"/vitais/mp/ferir/5", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("gastar mana deu %d: %s", rec.Code, rec.Body.String())
	}

	after := poolsOf(t, f.s, f.charID)
	if after.MpCurrent != before.MpCurrent-5 {
		t.Errorf("a ficha ficou com %d PM; %d-5 = %d", after.MpCurrent, before.MpCurrent, before.MpCurrent-5)
	}
	if after.HpCurrent != before.HpCurrent {
		t.Errorf("gastar mana mexeu no PV: %d virou %d", before.HpCurrent, after.HpCurrent)
	}
}

// O PRIMEIRO clique do olho num NPC REVELA, porque ele já nasce escondido.
// Alternar a partir do PONTEIRO gravaria "esconder" sobre uma linha já
// escondida, e o mestre clicaria sem a tela mudar nada.
//
// A asserção é sobre O QUE A MESA VÊ e não sobre a flag: com padrão por pool
// os dois DIVERGEM, e um caso que lesse `HpHidden` continuaria verde
// exatamente no caso que ele veio medir.
func TestTheFirstEyeClickOnAnNpcRevealsInsteadOfHiding(t *testing.T) {
	f := newSceneFixture(t)
	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/adicionar",
		`{"new_name":"Ogro","new_initiative":12,"new_hp":130,"new_type":"npc"}`); rec.Code != http.StatusOK {
		t.Fatalf("pôr o ogro na fila deu %d", rec.Code)
	}
	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/cena/iniciar/acao", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	queue := stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative
	entryID := queue[0].ID

	tableSeesHP := func() bool {
		forTable := live.StateForRole("player", stateOf(t, f.s.tableHost().Sessions(), f.sessionID))
		for _, e := range forTable.Initiative {
			if e.ID == entryID {
				return e.HpMax != nil
			}
		}
		t.Fatal("a linha sumiu da cópia da mesa")
		return false
	}

	if tableSeesHP() {
		t.Fatal("o PV do ogro nasceu à vista — o caso mediria o contrário do que quer")
	}
	eye := f.tableUrl() + "/iniciativa/" + entryID + "/vitais/hp/oculto"
	if rec := f.requests(t, f.gm, "POST", eye, ""); rec.Code != http.StatusOK {
		t.Fatalf("o primeiro clique deu %d", rec.Code)
	}
	if !tableSeesHP() {
		t.Error("o primeiro clique não revelou: o olho alternou o ponteiro em vez do que a mesa vê")
	}
	if rec := f.requests(t, f.gm, "POST", eye, ""); rec.Code != http.StatusOK {
		t.Fatalf("o segundo clique deu %d", rec.Code)
	}
	if tableSeesHP() {
		t.Error("o segundo clique não voltou a esconder")
	}
}

// A trava é o 403, e o HTML do jogador nem tem os verbos. As duas coisas são
// medidas juntas porque uma sem a outra engana: botão ausente nunca foi prova
// de trava.
func TestTheRowVerbsBelongToTheGm(t *testing.T) {
	f := newSceneFixture(t)
	entryID := f.tracker(t)
	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/cena/iniciar/acao", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}

	// Os DOIS pools entram na varredura: enumerar só o `hp` deixaria um verbo
	// novo do `mp` nascer aberto ao jogador e sem medição.
	for _, action := range []string{
		"vitais/hp/ferir/1", "vitais/hp/curar/1", "vitais/hp/oculto",
		"vitais/mp/ferir/1", "vitais/mp/curar/1", "vitais/mp/oculto",
		"remover",
	} {
		rec := f.requests(t, f.player, "POST", f.tableUrl()+"/iniciativa/"+entryID+"/"+action, "")
		if rec.Code != http.StatusForbidden {
			t.Errorf("o jogador fez %q e levou %d, quero 403", action, rec.Code)
		}
	}

	body := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	// O CONTROLE de que ele está mesmo vendo a fila: sem a linha na tela, a
	// ausência dos verbos não seria evidência de nada.
	if !strings.Contains(body, "Arcanista") {
		t.Fatalf("o jogador não viu a própria linha; a ausência abaixo não provaria nada")
	}
	for _, verb := range []string{"Remover Arcanista", "Ferir Arcanista", "Ocultar os PV"} {
		if strings.Contains(body, verb) {
			t.Errorf("o HTML do jogador veio com %q", verb)
		}
	}
}

func TestRemoveTakesTheCombatantOutOfTheTracker(t *testing.T) {
	f := newSceneFixture(t)
	entryID := f.tracker(t)
	if n := len(stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative); n != 1 {
		t.Fatalf("a fila começou com %d, queria 1", n)
	}

	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/"+entryID+"/remover", ""); rec.Code != http.StatusOK {
		t.Fatalf("remover deu %d", rec.Code)
	}
	if n := len(stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative); n != 0 {
		t.Errorf("a fila ficou com %d combatentes", n)
	}
}

// ── a presença no cartão do Grupo ────────────────────────────────────────────

// A regra de QUEM está conectado já tem guarda no `live`; o que se prende aqui
// é a LIGAÇÃO — que o cartão do Grupo é casado com a presença pelo id do
// personagem, e que ela chega às DUAS telas.
//
// A metade do JOGADOR é decisão do dono (ALE-214), e é o contrário do que a
// discrição sugeriria: saber quem caiu é o que faz a mesa ESPERAR em vez de
// continuar sem alguém. Quem inverter vai achar que ela é descuido.
func TestBothTheGmAndThePlayerSeeWhoIsAtTheTable(t *testing.T) {
	f := newSceneFixture(t)

	// Fora da mesa primeiro, que é o estado de nascença: sem esta metade, "vi
	// 'na mesa'" não distinguiria a ligação certa de uma frase fixa.
	body := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(body, "fora da mesa") {
		t.Fatalf("o mestre não viu a presença do grupo com ninguém conectado")
	}
	if strings.Contains(body, ">na mesa<") {
		t.Error("alguém apareceu na mesa sem ter entrado")
	}

	f.s.tableHost().Presence().Join(f.sessionID, "conn-do-jogador", live.PresenceUser{
		UserID: f.player, Name: "Jogador", Role: "player",
	})

	body = f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(body, "na mesa") || strings.Contains(body, "fora da mesa") {
		t.Error("o dono do personagem entrou e o cartão dele não acendeu")
	}

	// E O JOGADOR RECEBE A MESMA COISA. A frase é o que se prende, e não a cor:
	// cor não existe para quem usa leitor de tela.
	forPlayer := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(forPlayer, "Arcanista") {
		t.Fatal("o jogador não viu o cartão do Grupo; o que vem abaixo não provaria nada")
	}
	if !strings.Contains(forPlayer, "na mesa") {
		t.Error("o cartão do jogador não diz quem está na mesa: o `Presenca` chega nil e o ponto é ramo morto")
	}
}

// ── acrescentar combatente ───────────────────────────────────────────────────

// O que se prende é a COMPOSIÇÃO: que a cena chama o `Roster.Entry` e a
// validação do `live`, em vez de montar a linha por conta própria. As duas
// metades do PV são o ponto — digitado ele vira pool cheio, e ZERO fica de fora
// em vez de virar 0/0, que é a diferença entre "capanga sem vida rastreada" e
// "capanga que já está morto".
func TestAddingACombatantBuildsTheEntryThroughTheHousePath(t *testing.T) {
	f := newSceneFixture(t)

	rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/adicionar",
		`{"new_name":"  Goblin salteador  ","new_initiative":17,"new_hp":12,"new_type":"npc"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("acrescentar deu %d: %s", rec.Code, rec.Body.String())
	}
	queue := stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative
	if len(queue) != 1 {
		t.Fatalf("a fila ficou com %d combatentes", len(queue))
	}
	// O nome vai APARADO: o mestre digitando com espaço sobrando não deve
	// produzir uma linha que ordena diferente do que ele leu.
	if queue[0].Label != "Goblin salteador" {
		t.Errorf("o rótulo ficou %q", queue[0].Label)
	}
	if queue[0].Initiative != 17 || queue[0].Type != "npc" {
		t.Errorf("a linha ficou %+v", queue[0])
	}
	if queue[0].HpMax == nil || *queue[0].HpMax != 12 || queue[0].HpCurrent == nil || *queue[0].HpCurrent != 12 {
		t.Errorf("o PV digitado não virou pool cheio: %v/%v", queue[0].HpCurrent, queue[0].HpMax)
	}

	rec = f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/adicionar",
		`{"new_name":"Figurante","new_initiative":3,"new_hp":0,"new_type":"npc"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("acrescentar sem PV deu %d", rec.Code)
	}
	for _, e := range stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative {
		if e.Label == "Figurante" && e.HpMax != nil {
			t.Errorf("PV 0 virou barra %v — o capanga sem vida rastreada apareceu morto", *e.HpMax)
		}
	}
}

// E a validação do `live` está LIGADA: o app não tem uma segunda escada.
//
// Um caso só, e de propósito — as quatro bordas têm guarda no `live`, contra a
// regra. O que falta provar aqui é a LIGAÇÃO, e repetir as quatro seria afirmar
// a mesma coisa em duas camadas.
func TestAddingACombatantUsesTheLiveValidation(t *testing.T) {
	f := newSceneFixture(t)

	rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/adicionar",
		`{"new_name":"Ogro","new_initiative":400,"new_hp":0,"new_type":"npc"}`)
	if body := rec.Body.String(); !strings.Contains(body, "400") {
		t.Errorf("a recusa não citou a iniciativa ofensiva; corpo = %q", body)
	}
	if n := len(stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative); n != 0 {
		t.Errorf("o combatente recusado entrou na fila mesmo assim (%d na fila)", n)
	}
}

// E acrescentar é do MESTRE, com as duas metades medidas juntas.
func TestAddingACombatantBelongsToTheGm(t *testing.T) {
	f := newSceneFixture(t)
	body := `{"new_name":"Intruso","new_initiative":10,"new_hp":0,"new_type":"npc"}`

	if rec := f.requests(t, f.player, "POST", f.tableUrl()+"/iniciativa/adicionar", body); rec.Code != http.StatusForbidden {
		t.Errorf("o jogador acrescentou e levou %d, quero 403", rec.Code)
	}
	html := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(html, "Iniciativa") {
		t.Fatal("o jogador não viu a seção da fila; a ausência abaixo não provaria nada")
	}
	if strings.Contains(html, "+ Combatente") {
		t.Error("o HTML do jogador veio com o + Combatente")
	}
}

// As duas metades são o gesto. Limpar no clique custaria o que a pessoa
// digitou toda vez que a validação recusasse; não limpar no aceite deixa o nome
// no campo, e o clique seguinte acrescenta o MESMO capanga de novo.
func TestTheFormOnlyClearsWhenTheServerAccepts(t *testing.T) {
	f := newSceneFixture(t)

	accepted := f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/adicionar",
		`{"new_name":"Goblin","new_initiative":17,"new_hp":12,"new_type":"character"}`).Body.String()
	if !strings.Contains(accepted, `"new_name":""`) {
		t.Errorf("o formulário não se limpou depois do aceite; corpo = %s", trechoDeSinais(accepted))
	}
	if !strings.Contains(accepted, `"new_type":"npc"`) {
		t.Errorf("o tipo não voltou para npc; corpo = %s", trechoDeSinais(accepted))
	}

	refused := f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/adicionar",
		`{"new_name":"Ogro","new_initiative":400,"new_hp":0,"new_type":"npc"}`).Body.String()
	// O CONTROLE: a recusa TEM de ter acontecido, senão "não limpou" seria só
	// "não houve resposta nenhuma".
	if !strings.Contains(refused, "400") {
		t.Fatalf("a recusa não chegou; corpo = %s", trechoDeSinais(refused))
	}
	// Quem garante isto é a ORDEM — a mutação só escreve nos sinais depois de a
	// própria escrita ter dado certo —, e não um descarte na resposta. O teste
	// prende a ordem: quem mover a limpeza para antes da mutação cai aqui.
	if strings.Contains(refused, `"new_name"`) {
		t.Error("o formulário foi limpo numa RECUSA — a pessoa perdeu o que digitou e precisa corrigir")
	}
}

// "Adicionar grupo" entra com iniciativa 0; sem editar, a única saída seria
// remover e acrescentar de novo, perdendo PV e condições no caminho. Por isso o
// teste confere que a linha continua sendo A MESMA depois da edição.
func TestEditingFixesInitiativeAndHpAtOnce(t *testing.T) {
	f := newSceneFixture(t)
	entryID := f.tracker(t)
	// O CONTROLE do que a issue descreve: o grupo entra com iniciativa ZERO.
	if queue := stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative; queue[0].Initiative != 0 {
		t.Fatalf("o grupo entrou com iniciativa %d; o teste mede o conserto do zero", queue[0].Initiative)
	}

	rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/"+entryID+"/editar",
		`{"edit_initiative":21,"edit_hp":7}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("editar deu %d: %s", rec.Code, trechoDeSinais(rec.Body.String()))
	}

	order := stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative
	if len(order) != 1 || order[0].ID != entryID {
		t.Fatalf("a linha não sobreviveu à edição: %+v", order)
	}
	if order[0].Initiative != 21 {
		t.Errorf("a iniciativa ficou %d", order[0].Initiative)
	}
	if order[0].HpCurrent == nil || *order[0].HpCurrent != 7 {
		t.Errorf("o PV da linha ficou %v", order[0].HpCurrent)
	}
	// E o PV passou pela FICHA, como o ferir/curar: a linha espelha, ela não é
	// a autoridade.
	sheet := poolsOf(t, f.s, f.charID)
	if sheet.HpCurrent != 7 {
		t.Errorf("a FICHA ficou com %d PV — a edição não chegou nela", sheet.HpCurrent)
	}
}

// A validação da iniciativa é a MESMA de acrescentar, e tem um dono só.
func TestEditingUsesTheSameInitiativeRangeAsAdding(t *testing.T) {
	f := newSceneFixture(t)
	entryID := f.tracker(t)

	rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/"+entryID+"/editar",
		`{"edit_initiative":41,"edit_hp":10}`)
	if body := trechoDeSinais(rec.Body.String()); !strings.Contains(body, "41") {
		t.Errorf("a recusa não citou a iniciativa ofensiva; sinais = %s", body)
	}
	if queue := stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative; queue[0].Initiative != 0 {
		t.Errorf("a iniciativa recusada foi gravada mesmo assim (%d)", queue[0].Initiative)
	}
}

// Quem decide se há PV para editar é o SERVIDOR olhando a linha, e não um sinal
// que a página mande junto: uma tela defasada diria "tem" sobre um combatente
// que acabou de perder a barra, e a escrita inventaria um pool onde não havia.
func TestEditingInventsNoPoolOnALifelessEntry(t *testing.T) {
	f := newSceneFixture(t)
	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/adicionar",
		`{"new_name":"Figurante","new_initiative":5,"new_hp":0,"new_type":"npc"}`); rec.Code != http.StatusOK {
		t.Fatalf("acrescentar deu %d", rec.Code)
	}
	entryID := stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative[0].ID

	// A página manda PV, como mandaria se estivesse defasada.
	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/"+entryID+"/editar",
		`{"edit_initiative":9,"edit_hp":50}`); rec.Code != http.StatusOK {
		t.Fatalf("editar deu %d", rec.Code)
	}

	row := stateOf(t, f.s.tableHost().Sessions(), f.sessionID).Initiative[0]
	if row.Initiative != 9 {
		t.Errorf("a iniciativa não foi corrigida: %d", row.Initiative)
	}
	if row.HpMax != nil || row.HpCurrent != nil {
		t.Errorf("o capanga sem vida rastreada ganhou um pool inventado: %v/%v", row.HpCurrent, row.HpMax)
	}
}

// TestTheRowVerbsBelongToTheGm já cobre o 403 dos outros; editar entra aqui.
func TestEditingBelongsToTheGm(t *testing.T) {
	f := newSceneFixture(t)
	entryID := f.tracker(t)
	rec := f.requests(t, f.player, "POST", f.tableUrl()+"/iniciativa/"+entryID+"/editar",
		`{"edit_initiative":21,"edit_hp":7}`)
	if rec.Code != http.StatusForbidden {
		t.Errorf("o jogador editou e levou %d, quero 403", rec.Code)
	}
}

// O rótulo é digitado pelo MESTRE e vai parar DENTRO de uma expressão do
// Datastar, que é JavaScript. Um combatente chamado `O'Brien` fecharia a aspa e
// o resto viraria sintaxe — o `templ` escapa o ATRIBUTO (a aspa vira `&#39;`),
// mas o navegador a desescapa antes de o Datastar compilar. Escape de HTML não é
// escape de JS, e confundir os dois é como se escreve uma injeção sem querer.
func TestACombatantWithAQuoteInTheNameDoesNotBreakTheExpression(t *testing.T) {
	f := newSceneFixture(t)
	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/adicionar",
		`{"new_name":"O'Brien, o \"Justo\"","new_initiative":5,"new_hp":0,"new_type":"npc"}`); rec.Code != http.StatusOK {
		t.Fatalf("acrescentar deu %d", rec.Code)
	}

	body := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	// O literal tem de sair como JSON: aspas ESCAPADAS dentro dele, e não uma
	// aspa crua fechando a string no meio do nome.
	if !strings.Contains(body, `$edit_name = &#34;O&#39;Brien, o \&#34;Justo\&#34;&#34;`) {
		t.Errorf("o nome com aspas não virou literal seguro; a semeadura saiu como: %s", trechoDaSemeadura(body))
	}
}

// Um teste sobre uma PALAVRA, porque nenhuma outra camada afirma o texto do
// crachá — o e2e cobre o LEIAUTE do selo, não a palavra.
//
// A palavra carrega regra: o GLOSSARY bane `PC`, e o canônico é `sheet` porque
// a pergunta que o `type == "character"` responde é "esta linha é ficha ou é
// NPC?". As duas metades ficam juntas de propósito: afirmar só a nova deixaria
// passar uma tela que diz as duas coisas.
func TestTheTrackerBadgeSaysSheetAndNeverPc(t *testing.T) {
	f := newSceneFixture(t)
	f.tracker(t)

	body := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	// O CONTROLE: a linha do personagem está na tela. Sem ele, "não achei PC"
	// seria verdade também numa fila vazia.
	if !strings.Contains(body, "Arcanista") {
		t.Fatal("a fila não tem a linha do personagem; as asserções abaixo não provariam nada")
	}
	if !strings.Contains(body, ">Ficha<") {
		t.Error("o crachá da linha de personagem não diz Ficha")
	}
	if strings.Contains(body, ">PC<") {
		t.Error("o crachá voltou a dizer PC, que o GLOSSARY proíbe")
	}
}

// A LINHA da fila desenha UM POOL POR BARRA, com os verbos de cada um ao lado,
// e cada papel lê o que a redação lhe deixou.
//
// O caso mede os dois lados no mesmo estado, e é isso que o torna honesto:
// afirmar só o do mestre não distingue "o jogador não vê o PM" de "o PM não foi
// desenhado para ninguém", e afirmar só o do jogador passaria verde sobre uma
// fila que não desenha nada.
func TestTheTrackerRowDrawsAPoolPerBarAndEachRoleReadsItsOwn(t *testing.T) {
	f := newSceneFixture(t)
	f.tracker(t)
	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/cena/iniciar/acao", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}

	forGM := f.requests(t, f.gm, "GET", f.tableUrl(), "").Body.String()
	// O verbo mora ao lado da barra que ele mexe, e o nome acessível é o que
	// prova isso: dois olhos na mesma linha, cada um dizendo de qual pool é.
	for _, label := range []string{"Ocultar os PV de ", "Ocultar os PM de "} {
		if !strings.Contains(forGM, label) {
			t.Errorf("o mestre não recebeu o olho %q — o verbo não diz de qual pool fala", label)
		}
	}
	if !strings.Contains(forGM, "/vitais/mp/ferir/") {
		t.Error("o mestre não recebeu o gesto de gastar mana na fila")
	}

	forPlayer := f.requests(t, f.player, "GET", f.tableUrl(), "").Body.String()
	// O PV do GRUPO é o que a mesa vê por padrão, e o personagem desta bancada é
	// justamente do jogador que está pedindo.
	if !strings.Contains(forPlayer, "PV ") {
		t.Error("a mesa perdeu o PV do próprio grupo")
	}
	if !strings.Contains(forPlayer, "PM ocultos pelo mestre") {
		t.Error("a mesa não foi avisada de que existe PM escondido: 'sem barra' e 'escondido' viraram a mesma coisa")
	}
	// E o jogador não recebe verbo nenhum: a trava é o 403, mas o HTML também
	// não os tem, e as duas coisas se medem juntas.
	if strings.Contains(forPlayer, "/vitais/mp/ferir/") || strings.Contains(forPlayer, "Ocultar os PM de ") {
		t.Error("os verbos do mana vazaram para o HTML do jogador")
	}
}
