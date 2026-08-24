package aovivo

import (
	"strings"
	"testing"
)

// As regras do rastreador (ALE-265). Os casos são as BORDAS que a história de
// cada issue nomeia — não uma transcrição do comportamento.

func fila(nomes ...string) []InitiativeEntry {
	fora := make([]InitiativeEntry, 0, len(nomes))
	for _, n := range nomes {
		fora = append(fora, InitiativeEntry{Label: n})
	}
	return fora
}

func rotulos(linhas []InitiativeEntry) []string {
	fora := make([]string, 0, len(linhas))
	for _, l := range linhas {
		fora = append(fora, l.Label)
	}
	return fora
}

func iguais(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestATiraDaVezEhCircular é a ALE-179, e a borda é o ÚLTIMO da fila.
//
// Cortar no fim deixaria a tira vazia justamente no turno em que saber "quem vem
// depois" mais importa — o último antes de virar a rodada.
func TestATiraDaVezEhCircular(t *testing.T) {
	f := fila("Arwen", "Ogro", "Goblin")

	if got := rotulos(TurnosAVista(f, 0, 3)); !iguais(got, []string{"Arwen", "Ogro", "Goblin"}) {
		t.Errorf("do começo veio %v", got)
	}
	// A borda: no ÚLTIMO turno, a tira volta ao primeiro.
	if got := rotulos(TurnosAVista(f, 2, 3)); !iguais(got, []string{"Goblin", "Arwen", "Ogro"}) {
		t.Errorf("do último veio %v, quero a volta circular", got)
	}
	// Pedir mais que a fila não repete ninguém: a janela para no tamanho dela.
	if got := TurnosAVista(f, 0, 10); len(got) != 3 {
		t.Errorf("pedindo 10 de 3 vieram %d", len(got))
	}
	// Fora de combate não há vez de ninguém.
	if got := TurnosAVista(f, -1, 3); len(got) != 0 {
		t.Errorf("fora de combate vieram %d linhas", len(got))
	}
	if got := TurnosAVista(nil, 0, 3); len(got) != 0 {
		t.Errorf("fila vazia deu %d linhas", len(got))
	}
}

// TestOBotaoDizParaOndeVai é a ALE-184: o mestre lia "▶" e contava a lista.
func TestOBotaoDizParaOndeVai(t *testing.T) {
	f := fila("Arwen", "Ogro")

	if got := ProximoTurno(f, 0); got.Rotulo != "Próximo: Ogro" {
		t.Errorf("no turno 0 o botão diz %q", got.Rotulo)
	}
	// A volta é circular como na tira.
	if got := ProximoTurno(f, 1); got.Rotulo != "Próximo: Arwen" {
		t.Errorf("no último turno o botão diz %q", got.Rotulo)
	}
	// FORA de combate o verbo muda: quem clica ali está COMEÇANDO, e
	// "Próximo: Arwen" mentiria sobre uma rodada que ainda não existe.
	if got := ProximoTurno(f, -1); got.Rotulo != "Começar: Arwen" {
		t.Errorf("fora de combate o botão diz %q, quero o verbo de começar", got.Rotulo)
	}
	// Fila vazia diz o MOTIVO de estar desligado, não o verbo que não acontece.
	vazia := ProximoTurno(nil, -1)
	if vazia.Rotulo != "Ninguém na fila" {
		t.Errorf("fila vazia diz %q", vazia.Rotulo)
	}
	if vazia.Linha != nil {
		t.Error("fila vazia prometeu uma linha — seria inventá-la")
	}
}

// TestOContadorTemQUATROEstados, e a ordem entre eles é regra: a cena existe
// antes da fila, e a fila existe antes do turno (ALE-210).
func TestOContadorTemQuatroEstados(t *testing.T) {
	casos := []struct {
		nome      string
		cenaAtiva bool
		rodada    int
		turno     int
		naFila    int
		quero     string
	}{
		{"fora de cena vence tudo", false, 3, 2, 5, "Fora de cena"},
		{"em cena sem fila", true, 0, -1, 0, "Em cena · ninguém na fila"},
		// "Rodada 0" é de propósito: a rodada só vira 1 no primeiro avanço.
		{"fila montada, combate não começou", true, 0, -1, 4, "Rodada 0 · 4 na fila"},
		{"em combate", true, 2, 1, 4, "Rodada 2 · Turno 2/4"},
		{"o turno é 1-indexado na tela", true, 1, 0, 3, "Rodada 1 · Turno 1/3"},
	}
	for _, c := range casos {
		got := ContadorDoTurno(c.cenaAtiva, c.rodada, c.turno, c.naFila)
		if got != c.quero {
			t.Errorf("%s: %q, quero %q", c.nome, got, c.quero)
		}
	}
}

// TestAPonteAteAPessoaEhODono, e não o id do personagem: a ficha de um membro é
// o SNAPSHOT da campanha (ALE-33).
func TestAPonteAteAPessoaEhODono(t *testing.T) {
	membros := []MembroDaMesa{
		{CharacterID: 10, DonoID: 1},
		{CharacterID: 11, DonoID: 2},
		{CharacterID: 12, DonoID: 1},
		{CharacterID: 13, DonoID: 0}, // sem personagem ligado
	}
	meus := MeusPersonagens(membros, 1)
	if len(meus) != 2 || !meus[10] || !meus[12] {
		t.Errorf("os meus vieram %v", meus)
	}
	// Sem usuário não há "meu" — e devolver tudo seria pior que devolver nada.
	if got := MeusPersonagens(membros, 0); len(got) != 0 {
		t.Errorf("sem usuário vieram %d personagens", len(got))
	}
}

// TestMembroSemPersonagemNaoEntraNaPresenca.
//
// Não é que ele esteja offline: é que não há personagem para marcar, e um zero
// na lista viraria "o personagem 0 está online" na tela.
func TestMembroSemPersonagemNaoEntraNaPresenca(t *testing.T) {
	membros := []MembroDaMesa{
		{CharacterID: 10, DonoID: 1},
		{CharacterID: 11, DonoID: 2},
		{CharacterID: 13, DonoID: 0},
	}
	conectados := PersonagensConectados(membros, []int64{1, 3})
	if len(conectados) != 1 || !conectados[10] {
		t.Errorf("conectados vieram %v, quero só o 10", conectados)
	}
	if conectados[0] {
		t.Error("o personagem 0 entrou na presença")
	}
	// Ninguém online é lista vazia, não lista inteira.
	if got := PersonagensConectados(membros, nil); len(got) != 0 {
		t.Errorf("sem ninguém online vieram %d", len(got))
	}
}

// TestOOlhoDoMestreOlhaAFilaENaoOPapel.
//
// Numa fila só de PCs não há vitais para reservar, e a tela não deve mudar de
// forma por causa de um papel que ali não muda nada.
func TestOOlhoDoMestreOlhaAFilaENaoOPapel(t *testing.T) {
	pv := int64(30)
	comNPC := []InitiativeEntry{{Label: "Arwen"}, {Label: "Ogro", HpMax: &pv}}
	soPCs := []InitiativeEntry{{Label: "Arwen"}, {Label: "Bruna"}}

	if !OMestreVeOsVitais(comNPC, true) {
		t.Error("o mestre não vê os vitais numa fila que tem NPC")
	}
	if OMestreVeOsVitais(comNPC, false) {
		t.Error("o jogador viu os vitais do NPC")
	}
	if OMestreVeOsVitais(soPCs, true) {
		t.Error("numa fila só de PCs não há o que reservar, e a tela mudou de forma")
	}
}

// TestValidaCombatenteNovoPrendeAsQuatroBordas.
//
// Uma por campo, e cada uma é a que a tela sozinha não segurava: os limites
// viviam como atributos dos campos do formulário da SPA, que é UI — quem
// postasse na mão passava por cima dos quatro.
func TestValidaCombatenteNovoPrendeAsQuatroBordas(t *testing.T) {
	bom := CombatenteNovo{Rotulo: "Ogro", Iniciativa: 12, PV: 45, Tipo: "npc"}
	if err := ValidaCombatenteNovo(bom); err != nil {
		t.Fatalf("o combatente bom foi recusado: %v — sem isto as recusas abaixo não provariam nada", err)
	}
	// PV zero é ESTADO VÁLIDO e não ausência: é "sem vida registrada", e o
	// capanga anônimo depende dele.
	if err := ValidaCombatenteNovo(CombatenteNovo{Rotulo: "Figurante", Iniciativa: 0, PV: 0, Tipo: "npc"}); err != nil {
		t.Errorf("PV 0 foi recusado, e ele é o capanga sem vida rastreada: %v", err)
	}

	casos := []struct {
		nome string
		c    CombatenteNovo
		cita string
	}{
		{"sem nome", CombatenteNovo{Rotulo: "   ", Iniciativa: 10, Tipo: "npc"}, "nome"},
		{"nome comprido", CombatenteNovo{Rotulo: strings.Repeat("a", 61), Iniciativa: 10, Tipo: "npc"}, "61"},
		{"iniciativa alta", CombatenteNovo{Rotulo: "Ogro", Iniciativa: 400, Tipo: "npc"}, "400"},
		{"iniciativa baixa", CombatenteNovo{Rotulo: "Ogro", Iniciativa: -6, Tipo: "npc"}, "-6"},
		{"PV demais", CombatenteNovo{Rotulo: "Ogro", Iniciativa: 10, PV: 1000, Tipo: "npc"}, "1000"},
		{"tipo inventado", CombatenteNovo{Rotulo: "Ogro", Iniciativa: 10, Tipo: "dragão"}, "dragão"},
	}
	for _, c := range casos {
		err := ValidaCombatenteNovo(c.c)
		if err == nil {
			t.Errorf("%s: passou", c.nome)
			continue
		}
		// A mensagem tem de nomear o VALOR ofensivo: quem a lê está no meio de
		// um combate e precisa consertar sem sair da tela.
		if !strings.Contains(err.Error(), c.cita) {
			t.Errorf("%s: a recusa %q não cita %q", c.nome, err, c.cita)
		}
	}
}

// O limite do nome conta RUNAS e não bytes: um nome de 60 letras acentuadas tem
// mais de 60 bytes, e contar bytes recusaria um nome mais curto do que o que
// deixa passar em ASCII.
func TestOLimiteDoNomeContaLetrasENaoBytes(t *testing.T) {
	acentuado := strings.Repeat("ã", MaxLetrasDoRotulo) // 60 letras, 120 bytes
	if err := ValidaCombatenteNovo(CombatenteNovo{Rotulo: acentuado, Iniciativa: 10, Tipo: "npc"}); err != nil {
		t.Errorf("60 letras acentuadas foram recusadas: %v", err)
	}
}
