package api

import (
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
		v := rastreadorViewOf(c.st, nil, nil, true)
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

	fora := rastreadorViewOf(estadoDe(false, 0, -1, fila...), nil, nil, true)
	if fora.Contador != "Fora de cena" {
		t.Errorf("fora de cena o contador diz %q", fora.Contador)
	}

	montando := rastreadorViewOf(estadoDe(true, 0, -1, fila...), nil, nil, true)
	if montando.Contador != "Rodada 0 · 2 na fila" {
		t.Errorf("montando a ordem o contador diz %q", montando.Contador)
	}
	// Fora de combate o verbo é COMEÇAR, e o contador concorda dizendo que a
	// rodada ainda é 0.
	if montando.Avanco.Rotulo != "Começar: Arwen" {
		t.Errorf("montando a ordem o botão diz %q", montando.Avanco.Rotulo)
	}

	emCombate := rastreadorViewOf(estadoDe(true, 1, 0, fila...), nil, nil, true)
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

	if !rastreadorViewOf(comNPC, nil, nil, true).VeVitais {
		t.Error("o mestre não vê vitais numa fila com NPC")
	}
	if rastreadorViewOf(comNPC, nil, nil, false).VeVitais {
		t.Error("o jogador viu os vitais do NPC")
	}
	if rastreadorViewOf(soPCs, nil, nil, true).VeVitais {
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
	v := rastreadorViewOf(estadoDe(true, 1, 0), membros, []int64{1}, true)
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

	comandos := []string{
		"initiative/next-turn", "initiative/previous-turn",
		"scene/start", "scene/end",
	}
	for _, cmd := range comandos {
		t.Run(cmd, func(t *testing.T) {
			rec := f.pede(t, f.jogador, "POST", f.urlDaMesa()+"/"+cmd, "")
			if rec.Code != http.StatusForbidden {
				t.Errorf("o jogador comandou %q e levou %d, quero 403", cmd, rec.Code)
			}
			if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/"+cmd, ""); rec.Code != http.StatusOK {
				t.Errorf("o mestre foi recusado em %q com %d", cmd, rec.Code)
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
