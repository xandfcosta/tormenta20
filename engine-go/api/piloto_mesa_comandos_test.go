package api

import (
	"context"
	"net/http"
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

// TestOAvancoSoAcendeComCenaEFila.
//
// Separar "não há para onde ir" de "o botão está quebrado" é o ponto: um botão
// aceso que recusa é pior que um apagado que explica. E são DOIS motivos
// diferentes de estar apagado — sem cena, e em cena sem ninguém na fila —, que
// é o que o contador diz enquanto o botão fica quieto.
func TestOAvancoSoAcendeComCenaEFila(t *testing.T) {
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
		v := mestreViewOf(c.st, nil, nil, true)
		if v.PodeAvancar != c.quero {
			t.Errorf("%s: PodeAvancar = %v, quero %v", c.nome, v.PodeAvancar, c.quero)
		}
	}
}

// TestOContadorEOAvancoContamAMESMAHistoria.
//
// Este é o guarda da COMPOSIÇÃO, e ele existe porque as duas regras recebem os
// mesmos argumentos e é fácil trocar um: passar `Round` onde vai `TurnIndex`
// compila, e a tela mente com números plausíveis. Aqui se afirma que as duas
// concordam sobre o estado.
func TestOContadorEOAvancoContamAMesmaHistoria(t *testing.T) {
	fila := []aovivo.InitiativeEntry{{Label: "Arwen"}, {Label: "Ogro"}}

	fora := mestreViewOf(estadoDe(false, 0, -1, fila...), nil, nil, true)
	if fora.Contador != "Fora de cena" {
		t.Errorf("fora de cena o contador diz %q", fora.Contador)
	}

	montando := mestreViewOf(estadoDe(true, 0, -1, fila...), nil, nil, true)
	if montando.Contador != "Rodada 0 · 2 na fila" {
		t.Errorf("montando a ordem o contador diz %q", montando.Contador)
	}
	// Fora de combate o verbo é COMEÇAR, e o contador concorda dizendo que a
	// rodada ainda é 0.
	if montando.Avanco.Rotulo != "Começar: Arwen" {
		t.Errorf("montando a ordem o botão diz %q", montando.Avanco.Rotulo)
	}

	emCombate := mestreViewOf(estadoDe(true, 1, 0, fila...), nil, nil, true)
	if emCombate.Contador != "Rodada 1 · Turno 1/2" {
		t.Errorf("em combate o contador diz %q", emCombate.Contador)
	}
	if emCombate.Avanco.Rotulo != "Próximo: Ogro" {
		t.Errorf("em combate o botão diz %q", emCombate.Avanco.Rotulo)
	}
}

// TestOsVitaisSeguemAFilaEOPapel — as duas condições, e a da fila é a que
// costuma ser esquecida.
func TestOsVitaisSeguemAFilaEOPapel(t *testing.T) {
	pv := int64(30)
	comNPC := estadoDe(true, 1, 0, aovivo.InitiativeEntry{Label: "Ogro", HpMax: &pv})
	soPCs := estadoDe(true, 1, 0, aovivo.InitiativeEntry{Label: "Arwen"})

	if !mestreViewOf(comNPC, nil, nil, true).VeVitais {
		t.Error("o mestre não vê vitais numa fila com NPC")
	}
	if mestreViewOf(comNPC, nil, nil, false).VeVitais {
		t.Error("o jogador viu os vitais do NPC")
	}
	if mestreViewOf(soPCs, nil, nil, true).VeVitais {
		t.Error("numa fila só de PCs a tela mudou de forma sem ter o que reservar")
	}
}

// TestAPresencaChegaNaCena: quem está com a aba aberta aparece marcado, e quem
// não tem personagem ligado não vira "personagem 0 online".
func TestAPresencaChegaNaCena(t *testing.T) {
	membros := []aovivo.MembroDaMesa{
		{CharacterID: 10, DonoID: 1},
		{CharacterID: 11, DonoID: 2},
		{CharacterID: 12, DonoID: 0},
	}
	v := mestreViewOf(estadoDe(true, 1, 0), membros, []int64{1}, true)
	if len(v.Conectados) != 1 || !v.Conectados[10] {
		t.Errorf("conectados = %v, quero só o 10", v.Conectados)
	}
	if v.Conectados[0] {
		t.Error("o personagem 0 entrou na presença")
	}
}

// ── os comandos, pelo fio ────────────────────────────────────────────────────

// TestSoOMestreComandaAMesa é o guarda que importa desta fatia.
//
// Esconder o botão do jogador é UX; a trava é o servidor. Este teste posta na
// mão, como quem abre o console — e é exatamente o que a ALE-144 registrou ao
// tirar três asserções de AUSÊNCIA da suíte: botão ausente nunca foi prova de
// trava, e a garantia mora na camada mais barata que a sustenta.
func TestSoOMestreComandaAMesa(t *testing.T) {
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
			rec := f.pede(t, f.jogador, "POST", f.urlDaMesa()+"/"+cmd.rota, cmd.sinais)
			if rec.Code != http.StatusForbidden {
				t.Errorf("o jogador comandou %q e levou %d, quero 403", cmd.rota, rec.Code)
			}
			if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/"+cmd.rota, cmd.sinais); rec.Code != http.StatusOK {
				t.Errorf("o mestre foi recusado em %q com %d", cmd.rota, rec.Code)
			}
		})
	}
}

// TestOComandoRemendaACenaNaHora, em vez de esperar o tique do stream.
//
// O avanço é o botão mais clicado da sessão: esperar até 200ms por um tique que
// vai calar (o hash não muda depois do remendo) seria pagar latência por nada.
func TestOComandoRemendaACenaNaHora(t *testing.T) {
	f := novoPiloto(t)

	rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/scene/start", "")
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

// TestOComandoAvisaAMesaInteira: enquanto as duas telas existirem, uma escrita
// pelo piloto tem de chegar na SPA.
func TestOComandoAvisaAMesaInteira(t *testing.T) {
	f := novoPiloto(t)
	conn := f.s.sse.Add(f.sessionID, "espia", "gm")
	defer f.s.sse.Remove(f.sessionID, "espia")

	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/scene/start", ""); rec.Code != http.StatusOK {
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

// TestEncerrarCenaPeloPilotoExpiraAsBencaosDoGrupo — a REGRESSÃO da ALE-220,
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
func TestEncerrarCenaPeloPilotoExpiraAsBencaosDoGrupo(t *testing.T) {
	f := novoPiloto(t)
	seedEffect(t, f.s, f.charID, "bencao", "scene")
	seedEffect(t, f.s, f.charID, "heroismo", "day")

	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/scene/end", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar cena deu %d", rec.Code)
	}

	// Os DOIS lados: o de cena sai, o de dia FICA. Limpar demais apagaria a
	// bênção que o grupo comprou para o dia todo, e ninguém veria.
	if got := effectScopes(t, f.s, f.charID); len(got) != 1 || got[0] != "day" {
		t.Errorf("a ficha do grupo ficou com os escopos %v, queria só [day]", got)
	}
}

// TestEncerrarCenaPeloPilotoAvisaQueAsFICHASMudaram.
//
// O `session-state` não serve para isto: as fichas não estão no estado do
// rastreador. Sem o `session-rest`, a SPA de quem está com a ficha aberta
// continuaria mostrando o efeito morto e o "usado 1/cena" gasto até alguém
// recarregar — a metade invisível do mesmo defeito.
func TestEncerrarCenaPeloPilotoAvisaQueAsFichasMudaram(t *testing.T) {
	f := novoPiloto(t)
	conn := f.s.sse.Add(f.sessionID, "espia", "gm")
	defer f.s.sse.Remove(f.sessionID, "espia")

	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/scene/end", ""); rec.Code != http.StatusOK {
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

// TestOComandoRecusadoCHEGAaoMestre.
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
func TestOComandoRecusadoChegaAoMestre(t *testing.T) {
	f := novoPiloto(t)
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	// A mesma sabotagem do `initiative_rules_test.go`: sem o roster não há como
	// alcançar as fichas, e o gesto inteiro tem de recusar.
	if _, err := f.s.db.Exec("DROP TABLE campaign_members"); err != nil {
		t.Fatalf("derrubar a tabela: %v", err)
	}

	rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/scene/end", "")
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
func TestOErroDoComandoNaoInvadeOErroDoRegistrar(t *testing.T) {
	f := novoPiloto(t)
	corpo := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(corpo, "erroDoComando") {
		t.Error("a página do mestre não declarou o sinal do comando")
	}
}

// TestAdicionarGrupoTrazOsPersonagensEPodeSerClicadoDeNovo.
//
// As duas metades são o gesto: trazer o grupo, e o segundo clique NÃO duplicar.
// A idempotência é o que sustenta o botão continuar clicável — o mestre que
// aceitou um jogador atrasado clica de novo e leva só o que faltava. Sem ela o
// desenho certo seria apagar o botão, e a fila teria Arwen duas vezes até
// alguém notar.
func TestAdicionarGrupoTrazOsPersonagensEPodeSerClicadoDeNovo(t *testing.T) {
	f := novoPiloto(t)

	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/populate", ""); rec.Code != http.StatusOK {
		t.Fatalf("adicionar grupo deu %d", rec.Code)
	}
	fila := f.s.sessions.GetState(f.sessionID).Initiative
	if len(fila) != 1 || fila[0].CharacterID == nil || *fila[0].CharacterID != f.charID {
		t.Fatalf("a fila ficou %+v, queria só o personagem %d", fila, f.charID)
	}

	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/populate", ""); rec.Code != http.StatusOK {
		t.Fatalf("o segundo clique deu %d", rec.Code)
	}
	if depois := f.s.sessions.GetState(f.sessionID).Initiative; len(depois) != 1 {
		t.Errorf("o segundo clique deixou %d combatentes na fila, queria 1", len(depois))
	}
}

// E o botão só existe para o MESTRE, porque a view do jogador não tem o que
// desenhar. Esconder por classe deixaria o HTML na página para quem abrisse o
// inspetor — e a trava de verdade é o 403 acima, medido em separado.
func TestOJogadorNaoRecebeOAdicionarGrupoNoHTML(t *testing.T) {
	f := novoPiloto(t)

	if corpo := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String(); strings.Contains(corpo, "Adicionar grupo") {
		t.Error("o HTML do jogador veio com o Adicionar grupo")
	}
	if corpo := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String(); !strings.Contains(corpo, "Adicionar grupo") {
		t.Error("o mestre não recebeu o Adicionar grupo")
	}
}

// ── a recuperação (T20 p105) ─────────────────────────────────────────────────

// TestODescansoDeDiaUsaAQualidadeQueOMestreEscolheu.
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
func TestODescansoDeDiaUsaAQualidadeQueOMestreEscolheu(t *testing.T) {
	f := novoPiloto(t)

	rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/rest/day", `{"qualidadedodescanso":"ruim"}`)
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
func TestUmaQualidadeInventadaERecusada(t *testing.T) {
	f := novoPiloto(t)

	rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/rest/day", `{"qualidadedodescanso":"palaciana"}`)
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
func TestARecuperacaoDeCenaExpiraAsFichasESemDesligarACena(t *testing.T) {
	f := novoPiloto(t)
	seedEffect(t, f.s, f.charID, "bencao", "scene")
	seedEffect(t, f.s, f.charID, "heroismo", "day")
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}

	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/rest/scene", ""); rec.Code != http.StatusOK {
		t.Fatalf("recuperar a cena deu %d", rec.Code)
	}

	if got := effectScopes(t, f.s, f.charID); len(got) != 1 || got[0] != "day" {
		t.Errorf("a ficha ficou com os escopos %v, queria só [day]", got)
	}
	// A diferença para o "Encerrar cena": a cena continua LIGADA. Recuperar ao
	// fim de uma luta não acaba a cena, e confundir os dois tiraria a fila da
	// mesa no meio do combate.
	if !f.s.sessions.GetState(f.sessionID).SceneActive {
		t.Error("a recuperação de cena desligou a cena")
	}
}

// ── os verbos da LINHA (ALE-263) ─────────────────────────────────────────────

// naFila põe o grupo na fila e devolve o id do combatente do personagem.
func (f pilotoFixture) naFila(t *testing.T) string {
	t.Helper()
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/populate", ""); rec.Code != http.StatusOK {
		t.Fatalf("adicionar grupo deu %d", rec.Code)
	}
	for _, e := range f.s.sessions.GetState(f.sessionID).Initiative {
		if e.CharacterID != nil && *e.CharacterID == f.charID {
			return e.ID
		}
	}
	t.Fatal("o personagem não entrou na fila")
	return ""
}

// TestFerirUmaLinhaPassaPelaFICHA — o guarda de composição desta fatia.
//
// Com personagem atrás da linha, quem manda é a FICHA: o dano é aplicado lá (é
// ela quem sabe drenar PV temporários) e a entrada ESPELHA o resultado — a regra
// que a ALE-122 pagou caro para ter num lugar só, depois de duas telas mostrarem
// 52/95 e 57/95 do mesmo combatente.
//
// Por isso a asserção é sobre a FICHA e não sobre a linha: escrever só na
// entrada compilaria, deixaria a fila com um número plausível, e a ficha do
// jogador continuaria com o PV de antes.
func TestFerirUmaLinhaPassaPelaFicha(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.naFila(t)

	rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/"+entryID+"/vitals/harm/5", "")
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
	for _, e := range f.s.sessions.GetState(f.sessionID).Initiative {
		if e.ID == entryID && (e.HpCurrent == nil || *e.HpCurrent != 15) {
			t.Errorf("a linha não espelhou a ficha: %v", e.HpCurrent)
		}
	}
}

// TestOPassoDoVitalVemDoCaminhoESoExistemDois.
//
// O passo não é dado que a página manda: são duas rotas por verbo. Um passo
// inventado não casa rota nenhuma, e a recusa nomeia o valor e a forma esperada.
func TestOPassoDoVitalVemDoCaminhoESoExistemDois(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.naFila(t)
	base := f.urlDaMesa() + "/initiative/" + entryID + "/vitals/"

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

// TestOOlhoINVERTEoEstadoQueOServidorGUARDA.
//
// Dois cliques voltam ao começo, e é isso que prova que quem decide é o
// SERVIDOR: se a página mandasse o valor desejado, duas abas do mestre com o
// remendo atrasado mandariam "esconder" duas vezes e a segunda desfaria a
// primeira sem ninguém pedir.
func TestOOlhoInverteOEstadoQueOServidorGuarda(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.naFila(t)
	olho := f.urlDaMesa() + "/initiative/" + entryID + "/vitals/hidden"

	oculto := func() bool {
		for _, e := range f.s.sessions.GetState(f.sessionID).Initiative {
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

// TestOsVerbosDaLinhaSaoDoMestre: a trava é o 403, e o HTML do jogador nem os
// tem. As duas coisas são medidas juntas porque uma sem a outra engana — botão
// ausente nunca foi prova de trava (ALE-144).
func TestOsVerbosDaLinhaSaoDoMestre(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.naFila(t)
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}

	for _, acao := range []string{"vitals/harm/1", "vitals/heal/1", "vitals/hidden", "remove"} {
		rec := f.pede(t, f.jogador, "POST", f.urlDaMesa()+"/initiative/"+entryID+"/"+acao, "")
		if rec.Code != http.StatusForbidden {
			t.Errorf("o jogador fez %q e levou %d, quero 403", acao, rec.Code)
		}
	}

	corpo := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()
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

// TestRemoverTiraOCombatenteDaFila.
func TestRemoverTiraOCombatenteDaFila(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.naFila(t)
	if n := len(f.s.sessions.GetState(f.sessionID).Initiative); n != 1 {
		t.Fatalf("a fila começou com %d, queria 1", n)
	}

	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/"+entryID+"/remove", ""); rec.Code != http.StatusOK {
		t.Fatalf("remover deu %d", rec.Code)
	}
	if n := len(f.s.sessions.GetState(f.sessionID).Initiative); n != 0 {
		t.Errorf("a fila ficou com %d combatentes", n)
	}
}

// ── a presença no cartão do Grupo (ALE-263) ──────────────────────────────────

// TestOMestreVeQuemEstaNaMesaEOJogadorNao.
//
// A regra de QUEM está conectado já tem guarda no `aovivo`; o que se prende aqui
// é a LIGAÇÃO — que o cartão do Grupo é casado com a presença pelo id do
// personagem, e que ela chega à tela do mestre e não à do jogador.
//
// O precedente é da SPA e é deliberado: presença POR PERSONAGEM é do mestre (o
// trilho do elenco vive na `session-gm-view`), enquanto os crachás de nome são
// de todo mundo. Um anel apagado na tela do jogador diria "fora da mesa" sobre
// um colega a quem ele não tem por que vigiar.
func TestOMestreVeQuemEstaNaMesaEOJogadorNao(t *testing.T) {
	f := novoPiloto(t)

	// Fora da mesa primeiro, que é o estado de nascença: sem esta metade, "vi
	// 'na mesa'" não distinguiria a ligação certa de uma frase fixa.
	corpo := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(corpo, "fora da mesa") {
		t.Fatalf("o mestre não viu a presença do grupo com ninguém conectado")
	}
	if strings.Contains(corpo, ">na mesa<") {
		t.Error("alguém apareceu na mesa sem ter entrado")
	}

	f.s.presence.Join(f.sessionID, "conn-do-jogador", aovivo.PresenceUser{
		UserID: f.jogador, Name: "Jogador", Role: "player",
	})

	corpo = f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(corpo, "na mesa") || strings.Contains(corpo, "fora da mesa") {
		t.Error("o dono do personagem entrou e o cartão dele não acendeu")
	}

	// E o jogador não recebe presença nenhuma — nem acesa nem apagada.
	doJogador := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()
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

// TestAcrescentarCombatenteMontaALinhaPELOCaminhoDaCasa.
//
// O que se prende é a COMPOSIÇÃO: que o piloto chama o `materializeEntry` e a
// validação do `aovivo`, em vez de montar a linha por conta própria. As duas
// metades do PV são o ponto — digitado ele vira pool cheio, e ZERO fica de fora
// em vez de virar 0/0, que é a diferença entre "capanga sem vida rastreada" e
// "capanga que já está morto".
func TestAcrescentarCombatenteMontaALinhaPeloCaminhoDaCasa(t *testing.T) {
	f := novoPiloto(t)

	rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/add",
		`{"novonome":"  Goblin salteador  ","novainiciativa":17,"novopv":12,"novotipo":"npc"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("acrescentar deu %d: %s", rec.Code, rec.Body.String())
	}
	fila := f.s.sessions.GetState(f.sessionID).Initiative
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

	rec = f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/add",
		`{"novonome":"Figurante","novainiciativa":3,"novopv":0,"novotipo":"npc"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("acrescentar sem PV deu %d", rec.Code)
	}
	for _, e := range f.s.sessions.GetState(f.sessionID).Initiative {
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
func TestAcrescentarCombatenteUsaAValidacaoDoAovivo(t *testing.T) {
	f := novoPiloto(t)

	rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/add",
		`{"novonome":"Ogro","novainiciativa":400,"novopv":0,"novotipo":"npc"}`)
	if corpo := rec.Body.String(); !strings.Contains(corpo, "400") {
		t.Errorf("a recusa não citou a iniciativa ofensiva; corpo = %q", corpo)
	}
	if n := len(f.s.sessions.GetState(f.sessionID).Initiative); n != 0 {
		t.Errorf("o combatente recusado entrou na fila mesmo assim (%d na fila)", n)
	}
}

// E acrescentar é do MESTRE, com as duas metades medidas juntas (ALE-144).
func TestAcrescentarCombatenteEDoMestre(t *testing.T) {
	f := novoPiloto(t)
	corpo := `{"novonome":"Intruso","novainiciativa":10,"novopv":0,"novotipo":"npc"}`

	if rec := f.pede(t, f.jogador, "POST", f.urlDaMesa()+"/initiative/add", corpo); rec.Code != http.StatusForbidden {
		t.Errorf("o jogador acrescentou e levou %d, quero 403", rec.Code)
	}
	html := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(html, "Iniciativa") {
		t.Fatal("o jogador não viu a seção da fila; a ausência abaixo não provaria nada")
	}
	if strings.Contains(html, "+ Combatente") {
		t.Error("o HTML do jogador veio com o + Combatente")
	}
}

// TestOFormularioSoSeLimpaQuandoOServidorACEITA.
//
// As duas metades são o gesto, e a segunda é a que importa: limpar no clique
// custaria o que a pessoa digitou toda vez que a validação recusasse, e a recusa
// mais comum é sobre o nome — o campo mais caro de redigitar no meio de um
// combate.
//
// Sem a primeira metade, o nome fica no campo e o clique seguinte acrescenta o
// MESMO capanga de novo; ninguém confere a fila antes de clicar durante uma
// luta.
func TestOFormularioSoSeLimpaQuandoOServidorAceita(t *testing.T) {
	f := novoPiloto(t)

	aceito := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/add",
		`{"novonome":"Goblin","novainiciativa":17,"novopv":12,"novotipo":"character"}`).Body.String()
	if !strings.Contains(aceito, `"novonome":""`) {
		t.Errorf("o formulário não se limpou depois do aceite; corpo = %s", trechoDeSinais(aceito))
	}
	if !strings.Contains(aceito, `"novotipo":"npc"`) {
		t.Errorf("o tipo não voltou para npc; corpo = %s", trechoDeSinais(aceito))
	}

	recusado := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/add",
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

// TestEditarCorrigeAIniciativaEOPVDeUmaVez.
//
// A iniciativa é o gesto que a ALE-122 nomeou e deixou sem saída: "Adicionar
// grupo" entra com 0 e não havia como consertar, então a única saída era remover
// e acrescentar de novo — perdendo PV e condições no caminho. Por isso o teste
// confere que a linha continua sendo A MESMA depois da edição.
func TestEditarCorrigeAIniciativaEOPVDeUmaVez(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.naFila(t)
	// O CONTROLE do que a issue descreve: o grupo entra com iniciativa ZERO.
	if fila := f.s.sessions.GetState(f.sessionID).Initiative; fila[0].Initiative != 0 {
		t.Fatalf("o grupo entrou com iniciativa %d; o teste mede o conserto do zero", fila[0].Initiative)
	}

	rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/"+entryID+"/edit",
		`{"edicaoiniciativa":21,"edicaopv":7}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("editar deu %d: %s", rec.Code, trechoDeSinais(rec.Body.String()))
	}

	fila := f.s.sessions.GetState(f.sessionID).Initiative
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
func TestEditarUsaAMesmaFaixaDeIniciativaQueAcrescentar(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.naFila(t)

	rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/"+entryID+"/edit",
		`{"edicaoiniciativa":41,"edicaopv":10}`)
	if corpo := trechoDeSinais(rec.Body.String()); !strings.Contains(corpo, "41") {
		t.Errorf("a recusa não citou a iniciativa ofensiva; sinais = %s", corpo)
	}
	if fila := f.s.sessions.GetState(f.sessionID).Initiative; fila[0].Initiative != 0 {
		t.Errorf("a iniciativa recusada foi gravada mesmo assim (%d)", fila[0].Initiative)
	}
}

// TestEditarNaoInventaPoolEmLinhaSemVida.
//
// Quem decide se há PV para editar é o SERVIDOR olhando a linha, e não um sinal
// que a página mande junto: uma tela defasada diria "tem" sobre um combatente
// que acabou de perder a barra, e a escrita inventaria um pool onde não havia.
func TestEditarNaoInventaPoolEmLinhaSemVida(t *testing.T) {
	f := novoPiloto(t)
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/add",
		`{"novonome":"Figurante","novainiciativa":5,"novopv":0,"novotipo":"npc"}`); rec.Code != http.StatusOK {
		t.Fatalf("acrescentar deu %d", rec.Code)
	}
	entryID := f.s.sessions.GetState(f.sessionID).Initiative[0].ID

	// A página manda PV, como mandaria se estivesse defasada.
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/"+entryID+"/edit",
		`{"edicaoiniciativa":9,"edicaopv":50}`); rec.Code != http.StatusOK {
		t.Fatalf("editar deu %d", rec.Code)
	}

	linha := f.s.sessions.GetState(f.sessionID).Initiative[0]
	if linha.Initiative != 9 {
		t.Errorf("a iniciativa não foi corrigida: %d", linha.Initiative)
	}
	if linha.HpMax != nil || linha.HpCurrent != nil {
		t.Errorf("o capanga sem vida rastreada ganhou um pool inventado: %v/%v", linha.HpCurrent, linha.HpMax)
	}
}

// TestOsVerbosDaLinhaSaoDoMestre já cobre o 403 dos outros; editar entra aqui.
func TestEditarEDoMestre(t *testing.T) {
	f := novoPiloto(t)
	entryID := f.naFila(t)
	rec := f.pede(t, f.jogador, "POST", f.urlDaMesa()+"/initiative/"+entryID+"/edit",
		`{"edicaoiniciativa":21,"edicaopv":7}`)
	if rec.Code != http.StatusForbidden {
		t.Errorf("o jogador editou e levou %d, quero 403", rec.Code)
	}
}

// TestUmCombatenteComASPAnoNomeNaoQuebraAExpressao.
//
// O rótulo é digitado pelo MESTRE e vai parar DENTRO de uma expressão do
// Datastar, que é JavaScript. Um combatente chamado `O'Brien` fecharia a aspa e
// o resto viraria sintaxe — o `templ` escapa o ATRIBUTO (a aspa vira `&#39;`),
// mas o navegador a desescapa antes de o Datastar compilar. Escape de HTML não é
// escape de JS, e confundir os dois é como se escreve uma injeção sem querer.
func TestUmCombatenteComAspaNoNomeNaoQuebraAExpressao(t *testing.T) {
	f := novoPiloto(t)
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/add",
		`{"novonome":"O'Brien, o \"Justo\"","novainiciativa":5,"novopv":0,"novotipo":"npc"}`); rec.Code != http.StatusOK {
		t.Fatalf("acrescentar deu %d", rec.Code)
	}

	corpo := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	// O literal tem de sair como JSON: aspas ESCAPADAS dentro dele, e não uma
	// aspa crua fechando a string no meio do nome.
	if !strings.Contains(corpo, `$edicaonome = &#34;O&#39;Brien, o \&#34;Justo\&#34;&#34;`) {
		t.Errorf("o nome com aspas não virou literal seguro; a semeadura saiu como: %s", trechoDaSemeadura(corpo))
	}
}

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
