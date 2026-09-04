package api

import (
	"context"
	"net/http"
	"strings"
	"t20engine/aovivo"
	"testing"
)

// TestOnlyTheGmCommandsTheTable é o guarda que importa desta fatia.
//
// Esconder o botão do jogador é UX; a trava é o servidor. Este teste posta na
// mão, como quem abre o console — e é exatamente o que a ALE-144 registrou ao
// tirar três asserções de AUSÊNCIA da suíte: botão ausente nunca foi prova de
// trava, e a garantia mora na camada mais barata que a sustenta.
func TestOnlyTheGmCommandsTheTable(t *testing.T) {
	f := novoPiloto(t)

	// O corpo é o dos SINAIS que o Datastar manda junto. Só o descanso de dia lê
	// algum; os outros levam corpo vazio, que é o que o `@post` manda quando a
	// página não tem sinal nenhum a declarar.
	comandos := []struct{ rota, sinais string }{
		{"initiative/next-turn", ""},
		{"initiative/previous-turn", ""},
		{"scene/start", ""},
		{"scene/end", ""},
		{"initiative/populate", ""},
		{"rest/scene", ""},
		{"rest/day", `{"qualidadedodescanso":"normal"}`},
	}
	for _, cmd := range comandos {
		t.Run(cmd.rota, func(t *testing.T) {
			rec := f.pede(t, f.jogador, "POST", f.tableUrl()+"/"+cmd.rota, cmd.sinais)
			if rec.Code != http.StatusForbidden {
				t.Errorf("o jogador comandou %q e levou %d, quero 403", cmd.rota, rec.Code)
			}
			if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/"+cmd.rota, cmd.sinais); rec.Code != http.StatusOK {
				t.Errorf("o mestre foi recusado em %q com %d", cmd.rota, rec.Code)
			}
		})
	}
}

// TestTheCommandPatchesTheSceneRightAway, em vez de esperar o tique do stream.
//
// O avanço é o botão mais clicado da sessão: esperar até 200ms por um tique que
// vai calar (o hash não muda depois do remendo) seria pagar latência por nada.
func TestTheCommandPatchesTheSceneRightAway(t *testing.T) {
	f := novoPiloto(t)

	rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/scene/start", "")
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

// TestTheCommandAnnouncesToTheWholeTable: enquanto as duas telas existirem, uma escrita
// pelo piloto tem de chegar na SPA.
func TestTheCommandAnnouncesToTheWholeTable(t *testing.T) {
	f := novoPiloto(t)
	conn := f.s.sse.Add(f.sessionID, "espia", "gm")
	defer f.s.sse.Remove(f.sessionID, "espia")

	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}

	var viu bool
	for {
		select {
		case frame := <-conn.Frames:
			if strings.Contains(string(frame), "session-state") {
				viu = true
			}
			continue
		default:
		}
		break
	}
	if !viu {
		t.Error("o comando do piloto não avisou o hub — a SPA ficaria com o estado velho")
	}
}

// TestEndingTheSceneFromTheTableExpiresThePartyBlessings — a REGRESSÃO da ALE-220,
// reaberta pelo piloto.
//
// O "Encerrar cena" da API passa pelo `endSceneForTable`, que é o caminho ÚNICO
// desde aquela issue: ele expira a duração "cena" de toda ficha do grupo ANTES
// de desligar a cena. O piloto chamava `sessions.EndScene` direto, que só mexe
// no rastreador — a fila zerava na tela e a bênção de duração "cena" continuava
// viva na ficha, que é a colisão C1 do glossário com outro botão.
//
// O gesto tem de ser o MESMO nos dois transportes, e a forma de garantir isso
// não é repetir a sequência aqui: é chamar o mesmo helper.
func TestEndingTheSceneFromTheTableExpiresThePartyBlessings(t *testing.T) {
	f := novoPiloto(t)
	seedEffect(t, f.s, f.charID, "bencao", "scene")
	seedEffect(t, f.s, f.charID, "heroismo", "day")

	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/scene/end", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar cena deu %d", rec.Code)
	}

	// Os DOIS lados: o de cena sai, o de dia FICA. Limpar demais apagaria a
	// bênção que o grupo comprou para o dia todo, e ninguém veria.
	if got := effectScopes(t, f.s, f.charID); len(got) != 1 || got[0] != "day" {
		t.Errorf("a ficha do grupo ficou com os escopos %v, queria só [day]", got)
	}
}

// TestEndingTheSceneFromTheTableAnnouncesTheSheetsChanged.
//
// O `session-state` não serve para isto: as fichas não estão no estado do
// rastreador. Sem o `session-rest`, a SPA de quem está com a ficha aberta
// continuaria mostrando o efeito morto e o "usado 1/cena" gasto até alguém
// recarregar — a metade invisível do mesmo defeito.
func TestEndingTheSceneFromTheTableAnnouncesTheSheetsChanged(t *testing.T) {
	f := novoPiloto(t)
	conn := f.s.sse.Add(f.sessionID, "espia", "gm")
	defer f.s.sse.Remove(f.sessionID, "espia")

	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/scene/end", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar cena deu %d", rec.Code)
	}

	// O CONTROLE contra ler ausência como evidência: o `session-state` sai
	// sempre, então achá-lo prova que o canal está aberto e que a busca sabe
	// olhar. Sem ele, "não achei o session-rest" e "o canal não existe" seriam
	// a mesma linha no terminal.
	var viuEstado, viuFichas bool
	for {
		select {
		case frame := <-conn.Frames:
			if strings.Contains(string(frame), "session-state") {
				viuEstado = true
			}
			if strings.Contains(string(frame), "session-rest") {
				viuFichas = true
			}
			continue
		default:
		}
		break
	}
	if !viuEstado {
		t.Fatal("nem o session-state chegou — o canal não estava aberto, e a ausência abaixo não seria evidência de nada")
	}
	if !viuFichas {
		t.Error("a mesa não foi avisada de que as fichas mudaram")
	}
}

// TestTheRefusedCommandReachesTheGm.
//
// Os comandos respondiam `http.Error`, e isso era um beco: o Datastar não
// desenha corpo de resposta 4xx, então a recusa não chegava a lugar nenhum e o
// mestre clicava olhando para uma tela que não mudava. É o MESMO defeito que a
// ALE-213 anotou no socket, onde o cliente não escutava o `exception`.
//
// Ele ficou urgente com o conserto da ALE-220 acima: não alcançar as fichas do
// grupo ABORTA o encerrar-cena de propósito e deixa a cena LIGADA. Sem frase, o
// mestre vê a cena aberta depois de mandar encerrá-la e não tem como saber por
// quê.
func TestTheRefusedCommandReachesTheGm(t *testing.T) {
	f := novoPiloto(t)
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	// A mesma sabotagem do `initiative_rules_test.go`: sem o roster não há como
	// alcançar as fichas, e o gesto inteiro tem de recusar.
	if _, err := f.s.db.Exec("DROP TABLE campaign_members"); err != nil {
		t.Fatalf("derrubar a tabela: %v", err)
	}

	rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/scene/end", "")
	corpo := rec.Body.String()
	if !strings.Contains(corpo, "erroDoComando") {
		t.Fatalf("a recusa não chegou à cena do mestre; corpo = %q", corpo)
	}
	if strings.Contains(corpo, `"erroDoComando":""`) {
		t.Error("a cena recebeu a frase VAZIA — o mestre veria a cena ligada e nenhuma explicação")
	}
}

// E o sinal do comando é OUTRO que o `$erro` do registrar (ALE-263).
//
// Um sinal só faria a recusa de "Adicionar grupo" acender a frase vermelha
// dentro da caixa "Registrar iniciativa" do mestre que também joga — a frase
// certa no lugar errado, que é como se lê um defeito. Uma palavra por conceito
// vale para sinal de página como vale para identificador.
func TestTheCommandErrorDoesNotInvadeTheRecordError(t *testing.T) {
	f := novoPiloto(t)
	corpo := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(corpo, "erroDoComando") {
		t.Error("a página do mestre não declarou o sinal do comando")
	}
}

// TestAddPartyBringsTheCharactersAndCanBeClickedAgain.
//
// As duas metades são o gesto: trazer o grupo, e o segundo clique NÃO duplicar.
// A idempotência é o que sustenta o botão continuar clicável — o mestre que
// aceitou um jogador atrasado clica de novo e leva só o que faltava. Sem ela o
// desenho certo seria apagar o botão, e a fila teria Arwen duas vezes até
// alguém notar.
func TestAddPartyBringsTheCharactersAndCanBeClickedAgain(t *testing.T) {
	f := novoPiloto(t)

	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/populate", ""); rec.Code != http.StatusOK {
		t.Fatalf("adicionar grupo deu %d", rec.Code)
	}
	fila := f.s.tableHost().Sessions().GetState(f.sessionID).Initiative
	if len(fila) != 1 || fila[0].CharacterID == nil || *fila[0].CharacterID != f.charID {
		t.Fatalf("a fila ficou %+v, queria só o personagem %d", fila, f.charID)
	}

	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/populate", ""); rec.Code != http.StatusOK {
		t.Fatalf("o segundo clique deu %d", rec.Code)
	}
	if depois := f.s.tableHost().Sessions().GetState(f.sessionID).Initiative; len(depois) != 1 {
		t.Errorf("o segundo clique deixou %d combatentes na fila, queria 1", len(depois))
	}
}

// E o botão só existe para o MESTRE, porque a view do jogador não tem o que
// desenhar. Esconder por classe deixaria o HTML na página para quem abrisse o
// inspetor — e a trava de verdade é o 403 acima, medido em separado.
func TestThePlayerDoesNotGetAddPartyInTheHtml(t *testing.T) {
	f := novoPiloto(t)

	if corpo := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String(); strings.Contains(corpo, "Adicionar grupo") {
		t.Error("o HTML do jogador veio com o Adicionar grupo")
	}
	if corpo := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(corpo, "Adicionar grupo") {
		t.Error("o mestre não recebeu o Adicionar grupo")
	}
}

// ── a recuperação (T20 p105) ─────────────────────────────────────────────────

// TestTheDayRestUsesTheQualityTheGmChose.
//
// Este é o guarda que carrega a REGRA, e ele mira o desfecho mais silencioso
// possível: o `restMultiplier` do motor cai em "normal" quando não reconhece a
// palavra, então uma qualidade que não chegasse ao servidor não daria erro
// nenhum — o grupo descansaria em "normal" enquanto o mestre pediu outra coisa,
// e ninguém veria a diferença.
//
// Por isso a asserção é sobre o NÚMERO e a qualidade escolhida é "ruim", que é a
// única que se distingue do padrão: nível 8 recupera 4 em "ruim" e 8 em
// "normal", então 24 de PV prova que o sinal atravessou e 28 provaria que ele se
// perdeu no caminho.
func TestTheDayRestUsesTheQualityTheGmChose(t *testing.T) {
	f := novoPiloto(t)

	rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/rest/day", `{"qualidadedodescanso":"ruim"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("descanso de dia deu %d: %s", rec.Code, rec.Body.String())
	}

	ficha, err := f.s.queries.GetCharacter(context.Background(), f.charID)
	if err != nil {
		t.Fatalf("reler a ficha: %v", err)
	}
	if ficha.Hpcurrent != 24 {
		t.Errorf("PV = %d; 24 é o descanso RUIM de um nível 8 (20+4), 28 seria o normal que ninguém pediu", ficha.Hpcurrent)
	}
}

// E uma qualidade que não existe é RECUSADA, não rebaixada em silêncio.
//
// O motor cai em "normal" por conta própria, e para o piloto isso não serve: um
// sinal adulterado faria o grupo descansar em "normal" enquanto a tela dizia
// "luxuosa". Um número plausível no lugar do certo é o desfecho que esta
// migração mais paga para evitar — e a frase nomeia o valor ofensivo e a forma
// esperada, como o CLAUDE.md pede.
func TestAnInventedQualityIsRefused(t *testing.T) {
	f := novoPiloto(t)

	rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/rest/day", `{"qualidadedodescanso":"palaciana"}`)
	corpo := rec.Body.String()
	if !strings.Contains(corpo, "palaciana") {
		t.Errorf("a recusa não citou o valor ofensivo; corpo = %q", corpo)
	}
	if !strings.Contains(corpo, "luxuosa") {
		t.Errorf("a recusa não disse a forma esperada; corpo = %q", corpo)
	}

	ficha, err := f.s.queries.GetCharacter(context.Background(), f.charID)
	if err != nil {
		t.Fatalf("reler a ficha: %v", err)
	}
	if ficha.Hpcurrent != 20 {
		t.Errorf("PV = %d — o grupo descansou mesmo com a qualidade recusada", ficha.Hpcurrent)
	}
}

// A recuperação de CENA é o mesmo gesto do encerrar cena sem desligar a cena:
// expira a duração "cena" das fichas do grupo, e avisa que elas mudaram.
//
// É o `expirePartyScene` dos dois lados desde a ALE-220 — o que se prende aqui é
// que o piloto chama ELE, e não uma sequência própria.
func TestTheSceneRestExpiresTheSheetsWithoutTurningTheSceneOff(t *testing.T) {
	f := novoPiloto(t)
	seedEffect(t, f.s, f.charID, "bencao", "scene")
	seedEffect(t, f.s, f.charID, "heroismo", "day")
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}

	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/rest/scene", ""); rec.Code != http.StatusOK {
		t.Fatalf("recuperar a cena deu %d", rec.Code)
	}

	if got := effectScopes(t, f.s, f.charID); len(got) != 1 || got[0] != "day" {
		t.Errorf("a ficha ficou com os escopos %v, queria só [day]", got)
	}
	// A diferença para o "Encerrar cena": a cena continua LIGADA. Recuperar ao
	// fim de uma luta não acaba a cena, e confundir os dois tiraria a fila da
	// mesa no meio do combate.
	if !f.s.tableHost().Sessions().GetState(f.sessionID).SceneActive {
		t.Error("a recuperação de cena desligou a cena")
	}
}

// ── os verbos da LINHA (ALE-263) ─────────────────────────────────────────────

// tracker põe o grupo na fila e devolve o id do combatente do personagem.
func (f pilotoFixture) tracker(t *testing.T) string {
	t.Helper()
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/populate", ""); rec.Code != http.StatusOK {
		t.Fatalf("adicionar grupo deu %d", rec.Code)
	}
	for _, e := range f.s.tableHost().Sessions().GetState(f.sessionID).Initiative {
		if e.CharacterID != nil && *e.CharacterID == f.charID {
			return e.ID
		}
	}
	t.Fatal("o personagem não entrou na fila")
	return ""
}

// TestWoundingARowGoesThroughTheSheet — o guarda de composição desta fatia.
//
// Com personagem atrás da linha, quem manda é a FICHA: o dano é aplicado lá (é
// ela quem sabe drenar PV temporários) e a entrada ESPELHA o resultado — a regra
// que a ALE-122 pagou caro para ter num lugar só, depois de duas telas mostrarem
// 52/95 e 57/95 do mesmo combatente.
//
// Por isso a asserção é sobre a FICHA e não sobre a linha: escrever só na
// entrada compilaria, deixaria a fila com um número plausível, e a ficha do
// jogador continuaria com o PV de antes.
func TestWoundingARowGoesThroughTheSheet(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.tracker(t)

	rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/"+entryID+"/vitals/hp/harm/5", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("ferir deu %d: %s", rec.Code, rec.Body.String())
	}

	ficha, err := f.s.queries.GetCharacter(context.Background(), f.charID)
	if err != nil {
		t.Fatalf("reler a ficha: %v", err)
	}
	if ficha.Hpcurrent != 15 {
		t.Errorf("a FICHA ficou com %d PV; 20-5 = 15 — o dano não chegou nela", ficha.Hpcurrent)
	}
	// E a linha espelha, senão a fila mostraria o número velho ao lado da ficha
	// certa, que é a ALE-122 pelo outro lado.
	for _, e := range f.s.tableHost().Sessions().GetState(f.sessionID).Initiative {
		if e.ID == entryID && (e.HpCurrent == nil || *e.HpCurrent != 15) {
			t.Errorf("a linha não espelhou a ficha: %v", e.HpCurrent)
		}
	}
}

// TestTheVitalStepComesFromThePathAndThereAreOnlyTwo.
//
// O passo não é dado que a página manda: são duas rotas por verbo. Um passo
// inventado não casa rota nenhuma, e a recusa nomeia o valor e a forma esperada.
func TestTheVitalStepComesFromThePathAndThereAreOnlyTwo(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.tracker(t)
	base := f.tableUrl() + "/initiative/" + entryID + "/vitals/hp/"

	// O CONTROLE: os dois passos que existem passam. Sem ele, "o inventado
	// falhou" também seria verdade se a rota inteira estivesse quebrada.
	for _, passo := range []string{"1", "5"} {
		if rec := f.pede(t, f.mestre, "POST", base+"heal/"+passo, ""); rec.Code != http.StatusOK {
			t.Fatalf("curar em %s deu %d", passo, rec.Code)
		}
	}
	rec := f.pede(t, f.mestre, "POST", base+"harm/99", "")
	if corpo := rec.Body.String(); !strings.Contains(corpo, "99") || !strings.Contains(corpo, "Shift") {
		t.Errorf("a recusa do passo 99 não citou o valor e os passos que existem; corpo = %q", corpo)
	}
}

// TestTheEyeInvertsTheStateTheServerKeeps.
//
// Dois cliques voltam ao começo, e é isso que prova que quem decide é o
// SERVIDOR: se a página mandasse o valor desejado, duas abas do mestre com o
// remendo atrasado mandariam "esconder" duas vezes e a segunda desfaria a
// primeira sem ninguém pedir.
func TestTheEyeInvertsTheStateTheServerKeeps(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.tracker(t)
	olho := f.tableUrl() + "/initiative/" + entryID + "/vitals/hp/hidden"

	oculto := func() bool {
		for _, e := range f.s.tableHost().Sessions().GetState(f.sessionID).Initiative {
			if e.ID == entryID {
				return e.HpHidden != nil && *e.HpHidden
			}
		}
		t.Fatal("a linha sumiu")
		return false
	}
	if oculto() {
		t.Fatal("a linha nasceu escondida — o teste mediria o contrário do que quer")
	}
	if rec := f.pede(t, f.mestre, "POST", olho, ""); rec.Code != http.StatusOK {
		t.Fatalf("esconder deu %d", rec.Code)
	}
	if !oculto() {
		t.Fatal("o primeiro clique não escondeu")
	}
	if rec := f.pede(t, f.mestre, "POST", olho, ""); rec.Code != http.StatusOK {
		t.Fatalf("revelar deu %d", rec.Code)
	}
	if oculto() {
		t.Error("o segundo clique não revelou — o olho não é interruptor, é um valor que a página manda")
	}
}

// O MANA passa pelo mesmo caminho do PV, e chega na FICHA (ALE-211).
//
// A fila mandava `nil` no lugar do mana em todo clique, para todo combatente —
// o `DeltaVitals` sempre soube dos dois e ninguém pedia o segundo. O caminho da
// ficha por baixo também já era o mesmo, então a asserção é sobre ELA: escrever
// só na entrada compilaria e deixaria a fila com um número plausível ao lado de
// uma ficha que não gastou mana (ALE-122, pelo outro lado).
func TestSpendingManaGoesThroughTheSheetToo(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.tracker(t)

	antes, err := f.s.queries.GetCharacter(context.Background(), f.charID)
	if err != nil {
		t.Fatalf("ler a ficha: %v", err)
	}
	// O CONTROLE: sem mana para gastar o caso mediria um clampeamento em zero e
	// passaria verde sobre nada.
	if antes.Mpcurrent < 5 {
		t.Fatalf("a ficha semeada tem %d PM: o caso precisa de mana para gastar", antes.Mpcurrent)
	}

	rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/"+entryID+"/vitals/mp/harm/5", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("gastar mana deu %d: %s", rec.Code, rec.Body.String())
	}

	depois, err := f.s.queries.GetCharacter(context.Background(), f.charID)
	if err != nil {
		t.Fatalf("reler a ficha: %v", err)
	}
	if depois.Mpcurrent != antes.Mpcurrent-5 {
		t.Errorf("a ficha ficou com %d PM; %d-5 = %d", depois.Mpcurrent, antes.Mpcurrent, antes.Mpcurrent-5)
	}
	if depois.Hpcurrent != antes.Hpcurrent {
		t.Errorf("gastar mana mexeu no PV: %d virou %d", antes.Hpcurrent, depois.Hpcurrent)
	}
}

// O PRIMEIRO clique do olho num NPC REVELA, porque ele já nasce escondido
// (ALE-211).
//
// É a armadilha que o padrão por pool criou. Enquanto nulo significava
// "visível", alternar a partir do PONTEIRO estava certo; com o PV do NPC
// nascendo oculto, o mesmo código gravaria "esconder" sobre uma linha já
// escondida — o mestre clica e a tela não muda nada, que é o defeito mais
// difícil de reportar porque o botão parece morto em vez de errado.
//
// A asserção é sobre O QUE A MESA VÊ e não sobre a flag, e essa é a diferença
// que importa: com padrão por pool os dois DIVERGEM, e um caso que lesse
// `HpHidden` continuaria verde exatamente no caso que ele veio medir.
func TestTheFirstEyeClickOnAnNpcRevealsInsteadOfHiding(t *testing.T) {
	f := novoPiloto(t)
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/add",
		`{"novonome":"Ogro","novainiciativa":12,"novopv":130,"novotipo":"npc"}`); rec.Code != http.StatusOK {
		t.Fatalf("pôr o ogro na fila deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	fila := f.s.tableHost().Sessions().GetState(f.sessionID).Initiative
	entryID := fila[0].ID

	aMesaVeOPv := func() bool {
		paraMesa := aovivo.StateForRole("player", f.s.tableHost().Sessions().GetState(f.sessionID))
		for _, e := range paraMesa.Initiative {
			if e.ID == entryID {
				return e.HpMax != nil
			}
		}
		t.Fatal("a linha sumiu da cópia da mesa")
		return false
	}

	if aMesaVeOPv() {
		t.Fatal("o PV do ogro nasceu à vista — o caso mediria o contrário do que quer")
	}
	olho := f.tableUrl() + "/initiative/" + entryID + "/vitals/hp/hidden"
	if rec := f.pede(t, f.mestre, "POST", olho, ""); rec.Code != http.StatusOK {
		t.Fatalf("o primeiro clique deu %d", rec.Code)
	}
	if !aMesaVeOPv() {
		t.Error("o primeiro clique não revelou: o olho alternou o ponteiro em vez do que a mesa vê")
	}
	if rec := f.pede(t, f.mestre, "POST", olho, ""); rec.Code != http.StatusOK {
		t.Fatalf("o segundo clique deu %d", rec.Code)
	}
	if aMesaVeOPv() {
		t.Error("o segundo clique não voltou a esconder")
	}
}

// TestTheRowVerbsBelongToTheGm: a trava é o 403, e o HTML do jogador nem os
// tem. As duas coisas são medidas juntas porque uma sem a outra engana — botão
// ausente nunca foi prova de trava (ALE-144).
func TestTheRowVerbsBelongToTheGm(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.tracker(t)
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}

	// Os DOIS pools entram na varredura desde a ALE-211: um verbo novo que
	// nascesse aberto ao jogador seria exatamente o que este guarda existe para
	// impedir, e enumerar só o `hp` deixaria o `mp` nascer sem medição.
	for _, acao := range []string{
		"vitals/hp/harm/1", "vitals/hp/heal/1", "vitals/hp/hidden",
		"vitals/mp/harm/1", "vitals/mp/heal/1", "vitals/mp/hidden",
		"remove",
	} {
		rec := f.pede(t, f.jogador, "POST", f.tableUrl()+"/initiative/"+entryID+"/"+acao, "")
		if rec.Code != http.StatusForbidden {
			t.Errorf("o jogador fez %q e levou %d, quero 403", acao, rec.Code)
		}
	}

	corpo := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	// O CONTROLE de que ele está mesmo vendo a fila: sem a linha na tela, a
	// ausência dos verbos não seria evidência de nada.
	if !strings.Contains(corpo, "Arcanista") {
		t.Fatalf("o jogador não viu a própria linha; a ausência abaixo não provaria nada")
	}
	for _, verbo := range []string{"Remover Arcanista", "Ferir Arcanista", "Ocultar os PV"} {
		if strings.Contains(corpo, verbo) {
			t.Errorf("o HTML do jogador veio com %q", verbo)
		}
	}
}

// TestRemoveTakesTheCombatantOutOfTheTracker.
func TestRemoveTakesTheCombatantOutOfTheTracker(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.tracker(t)
	if n := len(f.s.tableHost().Sessions().GetState(f.sessionID).Initiative); n != 1 {
		t.Fatalf("a fila começou com %d, queria 1", n)
	}

	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/"+entryID+"/remove", ""); rec.Code != http.StatusOK {
		t.Fatalf("remover deu %d", rec.Code)
	}
	if n := len(f.s.tableHost().Sessions().GetState(f.sessionID).Initiative); n != 0 {
		t.Errorf("a fila ficou com %d combatentes", n)
	}
}

// ── a presença no cartão do Grupo (ALE-263) ──────────────────────────────────

// TestTheGmSeesWhoIsAtTheTableAndThePlayerDoesNot.
//
// A regra de QUEM está conectado já tem guarda no `aovivo`; o que se prende aqui
// é a LIGAÇÃO — que o cartão do Grupo é casado com a presença pelo id do
// personagem, e que ela chega à tela do mestre e não à do jogador.
//
// O precedente é da SPA e é deliberado: presença POR PERSONAGEM é do mestre (o
// trilho do elenco vive na `session-gm-view`), enquanto os crachás de nome são
// de todo mundo. Um anel apagado na tela do jogador diria "fora da mesa" sobre
// um colega a quem ele não tem por que vigiar.
func TestTheGmSeesWhoIsAtTheTableAndThePlayerDoesNot(t *testing.T) {
	f := novoPiloto(t)

	// Fora da mesa primeiro, que é o estado de nascença: sem esta metade, "vi
	// 'na mesa'" não distinguiria a ligação certa de uma frase fixa.
	corpo := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(corpo, "fora da mesa") {
		t.Fatalf("o mestre não viu a presença do grupo com ninguém conectado")
	}
	if strings.Contains(corpo, ">na mesa<") {
		t.Error("alguém apareceu na mesa sem ter entrado")
	}

	f.s.tableHost().Presence().Join(f.sessionID, "conn-do-jogador", aovivo.PresenceUser{
		UserID: f.jogador, Name: "Jogador", Role: "player",
	})

	corpo = f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(corpo, "na mesa") || strings.Contains(corpo, "fora da mesa") {
		t.Error("o dono do personagem entrou e o cartão dele não acendeu")
	}

	// E o jogador não recebe presença nenhuma — nem acesa nem apagada.
	doJogador := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(doJogador, "Arcanista") {
		t.Fatal("o jogador não viu o cartão do Grupo; a ausência abaixo não provaria nada")
	}
	for _, frase := range []string{"na mesa", "fora da mesa"} {
		if strings.Contains(doJogador, frase) {
			t.Errorf("o HTML do jogador veio com %q", frase)
		}
	}
}

// ── acrescentar combatente (ALE-263) ─────────────────────────────────────────

// TestAddingACombatantBuildsTheEntryThroughTheHousePath.
//
// O que se prende é a COMPOSIÇÃO: que o piloto chama o `materializeEntry` e a
// validação do `aovivo`, em vez de montar a linha por conta própria. As duas
// metades do PV são o ponto — digitado ele vira pool cheio, e ZERO fica de fora
// em vez de virar 0/0, que é a diferença entre "capanga sem vida rastreada" e
// "capanga que já está morto".
func TestAddingACombatantBuildsTheEntryThroughTheHousePath(t *testing.T) {
	f := novoPiloto(t)

	rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/add",
		`{"novonome":"  Goblin salteador  ","novainiciativa":17,"novopv":12,"novotipo":"npc"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("acrescentar deu %d: %s", rec.Code, rec.Body.String())
	}
	fila := f.s.tableHost().Sessions().GetState(f.sessionID).Initiative
	if len(fila) != 1 {
		t.Fatalf("a fila ficou com %d combatentes", len(fila))
	}
	// O nome vai APARADO: o mestre digitando com espaço sobrando não deve
	// produzir uma linha que ordena diferente do que ele leu.
	if fila[0].Label != "Goblin salteador" {
		t.Errorf("o rótulo ficou %q", fila[0].Label)
	}
	if fila[0].Initiative != 17 || fila[0].Type != "npc" {
		t.Errorf("a linha ficou %+v", fila[0])
	}
	if fila[0].HpMax == nil || *fila[0].HpMax != 12 || fila[0].HpCurrent == nil || *fila[0].HpCurrent != 12 {
		t.Errorf("o PV digitado não virou pool cheio: %v/%v", fila[0].HpCurrent, fila[0].HpMax)
	}

	rec = f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/add",
		`{"novonome":"Figurante","novainiciativa":3,"novopv":0,"novotipo":"npc"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("acrescentar sem PV deu %d", rec.Code)
	}
	for _, e := range f.s.tableHost().Sessions().GetState(f.sessionID).Initiative {
		if e.Label == "Figurante" && e.HpMax != nil {
			t.Errorf("PV 0 virou barra %v — o capanga sem vida rastreada apareceu morto", *e.HpMax)
		}
	}
}

// E a validação do `aovivo` está LIGADA: o piloto não tem uma segunda escada.
//
// Um caso só, e de propósito — as quatro bordas têm guarda no `aovivo`, contra a
// regra. O que falta provar aqui é a LIGAÇÃO, e repetir as quatro seria afirmar
// a mesma coisa em duas camadas.
func TestAddingACombatantUsesTheLiveValidation(t *testing.T) {
	f := novoPiloto(t)

	rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/add",
		`{"novonome":"Ogro","novainiciativa":400,"novopv":0,"novotipo":"npc"}`)
	if corpo := rec.Body.String(); !strings.Contains(corpo, "400") {
		t.Errorf("a recusa não citou a iniciativa ofensiva; corpo = %q", corpo)
	}
	if n := len(f.s.tableHost().Sessions().GetState(f.sessionID).Initiative); n != 0 {
		t.Errorf("o combatente recusado entrou na fila mesmo assim (%d na fila)", n)
	}
}

// E acrescentar é do MESTRE, com as duas metades medidas juntas (ALE-144).
func TestAddingACombatantBelongsToTheGm(t *testing.T) {
	f := novoPiloto(t)
	corpo := `{"novonome":"Intruso","novainiciativa":10,"novopv":0,"novotipo":"npc"}`

	if rec := f.pede(t, f.jogador, "POST", f.tableUrl()+"/initiative/add", corpo); rec.Code != http.StatusForbidden {
		t.Errorf("o jogador acrescentou e levou %d, quero 403", rec.Code)
	}
	html := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(html, "Iniciativa") {
		t.Fatal("o jogador não viu a seção da fila; a ausência abaixo não provaria nada")
	}
	if strings.Contains(html, "+ Combatente") {
		t.Error("o HTML do jogador veio com o + Combatente")
	}
}

// TestTheFormOnlyClearsWhenTheServerAccepts.
//
// As duas metades são o gesto, e a segunda é a que importa: limpar no clique
// custaria o que a pessoa digitou toda vez que a validação recusasse, e a recusa
// mais comum é sobre o nome — o campo mais caro de redigitar no meio de um
// combate.
//
// Sem a primeira metade, o nome fica no campo e o clique seguinte acrescenta o
// MESMO capanga de novo; ninguém confere a fila antes de clicar durante uma
// luta.
func TestTheFormOnlyClearsWhenTheServerAccepts(t *testing.T) {
	f := novoPiloto(t)

	aceito := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/add",
		`{"novonome":"Goblin","novainiciativa":17,"novopv":12,"novotipo":"character"}`).Body.String()
	if !strings.Contains(aceito, `"novonome":""`) {
		t.Errorf("o formulário não se limpou depois do aceite; corpo = %s", trechoDeSinais(aceito))
	}
	if !strings.Contains(aceito, `"novotipo":"npc"`) {
		t.Errorf("o tipo não voltou para npc; corpo = %s", trechoDeSinais(aceito))
	}

	recusado := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/add",
		`{"novonome":"Ogro","novainiciativa":400,"novopv":0,"novotipo":"npc"}`).Body.String()
	// O CONTROLE: a recusa TEM de ter acontecido, senão "não limpou" seria só
	// "não houve resposta nenhuma".
	if !strings.Contains(recusado, "400") {
		t.Fatalf("a recusa não chegou; corpo = %s", trechoDeSinais(recusado))
	}
	// Quem garante isto é a ORDEM — a mutação só escreve nos sinais depois de a
	// própria escrita ter dado certo —, e não um descarte na resposta. O teste
	// prende a ordem: quem mover a limpeza para antes da mutação cai aqui.
	if strings.Contains(recusado, `"novonome"`) {
		t.Error("o formulário foi limpo numa RECUSA — a pessoa perdeu o que digitou e precisa corrigir")
	}
}

// TestEditingFixesInitiativeAndHpAtOnce.
//
// A iniciativa é o gesto que a ALE-122 nomeou e deixou sem saída: "Adicionar
// grupo" entra com 0 e não havia como consertar, então a única saída era remover
// e acrescentar de novo — perdendo PV e condições no caminho. Por isso o teste
// confere que a linha continua sendo A MESMA depois da edição.
func TestEditingFixesInitiativeAndHpAtOnce(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.tracker(t)
	// O CONTROLE do que a issue descreve: o grupo entra com iniciativa ZERO.
	if fila := f.s.tableHost().Sessions().GetState(f.sessionID).Initiative; fila[0].Initiative != 0 {
		t.Fatalf("o grupo entrou com iniciativa %d; o teste mede o conserto do zero", fila[0].Initiative)
	}

	rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/"+entryID+"/edit",
		`{"edicaoiniciativa":21,"edicaopv":7}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("editar deu %d: %s", rec.Code, trechoDeSinais(rec.Body.String()))
	}

	fila := f.s.tableHost().Sessions().GetState(f.sessionID).Initiative
	if len(fila) != 1 || fila[0].ID != entryID {
		t.Fatalf("a linha não sobreviveu à edição: %+v", fila)
	}
	if fila[0].Initiative != 21 {
		t.Errorf("a iniciativa ficou %d", fila[0].Initiative)
	}
	if fila[0].HpCurrent == nil || *fila[0].HpCurrent != 7 {
		t.Errorf("o PV da linha ficou %v", fila[0].HpCurrent)
	}
	// E o PV passou pela FICHA, como o ferir/curar: a linha espelha, ela não é a
	// autoridade (ALE-122).
	ficha, err := f.s.queries.GetCharacter(context.Background(), f.charID)
	if err != nil {
		t.Fatalf("reler a ficha: %v", err)
	}
	if ficha.Hpcurrent != 7 {
		t.Errorf("a FICHA ficou com %d PV — a edição não chegou nela", ficha.Hpcurrent)
	}
}

// A validação da iniciativa é a MESMA de acrescentar, e agora ela tem um dono só.
//
// Na SPA eram duas constantes copiadas em dois componentes, com um comentário em
// cada dizendo "a mesma do formulário de adicionar" — duas cópias que só um
// comentário mantinha juntas.
func TestEditingUsesTheSameInitiativeRangeAsAdding(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.tracker(t)

	rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/"+entryID+"/edit",
		`{"edicaoiniciativa":41,"edicaopv":10}`)
	if corpo := trechoDeSinais(rec.Body.String()); !strings.Contains(corpo, "41") {
		t.Errorf("a recusa não citou a iniciativa ofensiva; sinais = %s", corpo)
	}
	if fila := f.s.tableHost().Sessions().GetState(f.sessionID).Initiative; fila[0].Initiative != 0 {
		t.Errorf("a iniciativa recusada foi gravada mesmo assim (%d)", fila[0].Initiative)
	}
}

// TestEditingInventsNoPoolOnALifelessEntry.
//
// Quem decide se há PV para editar é o SERVIDOR olhando a linha, e não um sinal
// que a página mande junto: uma tela defasada diria "tem" sobre um combatente
// que acabou de perder a barra, e a escrita inventaria um pool onde não havia.
func TestEditingInventsNoPoolOnALifelessEntry(t *testing.T) {
	f := novoPiloto(t)
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/add",
		`{"novonome":"Figurante","novainiciativa":5,"novopv":0,"novotipo":"npc"}`); rec.Code != http.StatusOK {
		t.Fatalf("acrescentar deu %d", rec.Code)
	}
	entryID := f.s.tableHost().Sessions().GetState(f.sessionID).Initiative[0].ID

	// A página manda PV, como mandaria se estivesse defasada.
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/"+entryID+"/edit",
		`{"edicaoiniciativa":9,"edicaopv":50}`); rec.Code != http.StatusOK {
		t.Fatalf("editar deu %d", rec.Code)
	}

	linha := f.s.tableHost().Sessions().GetState(f.sessionID).Initiative[0]
	if linha.Initiative != 9 {
		t.Errorf("a iniciativa não foi corrigida: %d", linha.Initiative)
	}
	if linha.HpMax != nil || linha.HpCurrent != nil {
		t.Errorf("o capanga sem vida rastreada ganhou um pool inventado: %v/%v", linha.HpCurrent, linha.HpMax)
	}
}

// TestTheRowVerbsBelongToTheGm já cobre o 403 dos outros; editar entra aqui.
func TestEditingBelongsToTheGm(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.tracker(t)
	rec := f.pede(t, f.jogador, "POST", f.tableUrl()+"/initiative/"+entryID+"/edit",
		`{"edicaoiniciativa":21,"edicaopv":7}`)
	if rec.Code != http.StatusForbidden {
		t.Errorf("o jogador editou e levou %d, quero 403", rec.Code)
	}
}

// TestACombatantWithAQuoteInTheNameDoesNotBreakTheExpression.
//
// O rótulo é digitado pelo MESTRE e vai parar DENTRO de uma expressão do
// Datastar, que é JavaScript. Um combatente chamado `O'Brien` fecharia a aspa e
// o resto viraria sintaxe — o `templ` escapa o ATRIBUTO (a aspa vira `&#39;`),
// mas o navegador a desescapa antes de o Datastar compilar. Escape de HTML não é
// escape de JS, e confundir os dois é como se escreve uma injeção sem querer.
func TestACombatantWithAQuoteInTheNameDoesNotBreakTheExpression(t *testing.T) {
	f := novoPiloto(t)
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/initiative/add",
		`{"novonome":"O'Brien, o \"Justo\"","novainiciativa":5,"novopv":0,"novotipo":"npc"}`); rec.Code != http.StatusOK {
		t.Fatalf("acrescentar deu %d", rec.Code)
	}

	corpo := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	// O literal tem de sair como JSON: aspas ESCAPADAS dentro dele, e não uma
	// aspa crua fechando a string no meio do nome.
	if !strings.Contains(corpo, `$edicaonome = &#34;O&#39;Brien, o \&#34;Justo\&#34;&#34;`) {
		t.Errorf("o nome com aspas não virou literal seguro; a semeadura saiu como: %s", trechoDaSemeadura(corpo))
	}
}

// TestTheTrackerBadgeSaysSheetAndNeverPc.
//
// Um teste sobre uma PALAVRA, e ele se justifica por uma lacuna medida: a sessão
// irmã trocou este mesmo crachá na SPA e 260 testes passaram sem piscar, porque
// nada afirmava o texto — o e2e cobre o LEIAUTE do selo, não a palavra.
//
// A palavra carrega regra: o GLOSSARIO bane `PC` sem qualificador de escopo, e o
// canônico aqui é `ficha` e não "personagem" porque a tabela de colisões diz
// qual pergunta o `type == "character"` responde — "esta linha é ficha ou é
// NPC?". As duas metades ficam juntas de propósito: afirmar só a nova deixaria
// passar uma tela que diz as duas coisas.
func TestTheTrackerBadgeSaysSheetAndNeverPc(t *testing.T) {
	f := novoPiloto(t)
	f.tracker(t)

	corpo := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	// O CONTROLE: a linha do personagem está na tela. Sem ele, "não achei PC"
	// seria verdade também numa fila vazia.
	if !strings.Contains(corpo, "Arcanista") {
		t.Fatal("a fila não tem a linha do personagem; as asserções abaixo não provariam nada")
	}
	if !strings.Contains(corpo, ">Ficha<") {
		t.Error("o crachá da linha de personagem não diz Ficha")
	}
	if strings.Contains(corpo, ">PC<") {
		t.Error("o crachá voltou a dizer PC, que o GLOSSARIO proíbe")
	}
}
