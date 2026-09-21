package live

import (
	"strings"
	"testing"
)

// As regras do rastreador. Os casos são as BORDAS, e não uma transcrição do
// comportamento.

func fila(names ...string) []InitiativeEntry {
	outside := make([]InitiativeEntry, 0, len(names))
	for _, n := range names {
		outside = append(outside, InitiativeEntry{Label: n})
	}
	return outside
}

func rotulos(rows []InitiativeEntry) []string {
	outside := make([]string, 0, len(rows))
	for _, l := range rows {
		outside = append(outside, l.Label)
	}
	return outside
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

// A borda é o ÚLTIMO da fila.
//
// Cortar no fim deixaria a tira vazia justamente no turno em que saber "quem vem
// depois" mais importa — o último antes de virar a rodada.
func TestTheTurnStripIsCircular(t *testing.T) {
	f := fila("Arwen", "Ogro", "Goblin")

	if got := rotulos(UpcomingTurns(f, 0, 3)); !iguais(got, []string{"Arwen", "Ogro", "Goblin"}) {
		t.Errorf("do começo veio %v", got)
	}
	// A borda: no ÚLTIMO turno, a tira volta ao primeiro.
	if got := rotulos(UpcomingTurns(f, 2, 3)); !iguais(got, []string{"Goblin", "Arwen", "Ogro"}) {
		t.Errorf("do último veio %v, quero a volta circular", got)
	}
	// Pedir mais que a fila não repete ninguém: a janela para no tamanho dela.
	if got := UpcomingTurns(f, 0, 10); len(got) != 3 {
		t.Errorf("pedindo 10 de 3 vieram %d", len(got))
	}
	// Fora de combate não há vez de ninguém.
	if got := UpcomingTurns(f, -1, 3); len(got) != 0 {
		t.Errorf("fora de combate vieram %d linhas", len(got))
	}
	if got := UpcomingTurns(nil, 0, 3); len(got) != 0 {
		t.Errorf("fila vazia deu %d linhas", len(got))
	}
}

// O botão diz PARA ONDE vai: com "▶", o mestre tem de contar a lista.
func TestTheButtonSaysWhereItGoes(t *testing.T) {
	f := fila("Arwen", "Ogro")

	if got := NextTurnButton(f, 0); got.Label != "Próximo: Ogro" {
		t.Errorf("no turno 0 o botão diz %q", got.Label)
	}
	// A volta é circular como na tira.
	if got := NextTurnButton(f, 1); got.Label != "Próximo: Arwen" {
		t.Errorf("no último turno o botão diz %q", got.Label)
	}
	// FORA de combate o verbo muda: quem clica ali está COMEÇANDO, e
	// "Próximo: Arwen" mentiria sobre uma rodada que ainda não existe.
	if got := NextTurnButton(f, -1); got.Label != "Começar: Arwen" {
		t.Errorf("fora de combate o botão diz %q, quero o verbo de começar", got.Label)
	}
	// Fila vazia diz o MOTIVO de estar desligado, não o verbo que não acontece.
	empty := NextTurnButton(nil, -1)
	if empty.Label != "Ninguém na fila" {
		t.Errorf("fila vazia diz %q", empty.Label)
	}
	if empty.Entry != nil {
		t.Error("fila vazia prometeu uma linha — seria inventá-la")
	}
}

// O contador tem CINCO estados, e o de combate diz o que SOBROU do turno, e a ORDEM entre eles é regra: a cena existe
// antes da fila, e a fila existe antes do turno. O quinto entrou com a cena
// tipada — fora da de AÇÃO não há rodada para contar, e o contador diz qual
// cena é (p252).
func TestTheCounterHasFiveStates(t *testing.T) {
	action := &Scene{Kind: SceneAction, Number: 1, StandardLeft: true, MovementLeft: true}
	cases := []struct {
		name    string
		scene   *Scene
		round   int
		turn    int
		inQueue int
		want    string
	}{
		{"fora de cena vence tudo", nil, 3, 2, 5, "Fora de cena"},
		{"numa conversa não há rodada", &Scene{Kind: SceneRoleplay, Number: 2}, 0, -1, 0,
			"Interpretação · cena 2"},
		{"numa exploração também não", &Scene{Kind: SceneExploration, Number: 3}, 0, -1, 4,
			"Exploração · cena 3"},
		{"em cena sem fila", action, 0, -1, 0, "Em cena · ninguém na fila"},
		// "Rodada 0" é de propósito: a rodada só vira 1 no primeiro avanço.
		{"fila montada, combate não começou", action, 0, -1, 4, "Rodada 0 · 4 na fila"},
		{"em combate", action, 2, 1, 4, "Rodada 2 · Turno 2/4 · padrão e movimento"},
		// O QUE SOBROU entra na frase porque quem vai clicar precisa saber ANTES
		// (p233). A TROCA aparece: com a padrão de pé e o movimento gasto, ainda
		// dá para mover.
		{"gastou o movimento", &Scene{Kind: SceneAction, Number: 1, StandardLeft: true}, 2, 1, 4,
			"Rodada 2 · Turno 2/4 · padrão (dá para mover)"},
		{"gastou a padrão", &Scene{Kind: SceneAction, Number: 1, MovementLeft: true}, 2, 1, 4,
			"Rodada 2 · Turno 2/4 · movimento"},
		{"turno inteiro gasto", &Scene{Kind: SceneAction, Number: 1}, 2, 1, 4,
			"Rodada 2 · Turno 2/4 · sem ação"},
		{"o turno é 1-indexado na tela", action, 1, 0, 3, "Rodada 1 · Turno 1/3 · padrão e movimento"},
	}
	for _, c := range cases {
		got := TurnCounter(c.scene, c.round, c.turn, c.inQueue)
		if got != c.want {
			t.Errorf("%s: %q, quero %q", c.name, got, c.want)
		}
	}
}

// Não é que ele esteja offline: é que não há personagem para marcar, e um zero
// na lista viraria "o personagem 0 está online" na tela.
func TestAMemberWithoutACharacterDoesNotEnterPresence(t *testing.T) {
	members := []TableMember{
		{CharacterID: 10, OwnerID: 1},
		{CharacterID: 11, OwnerID: 2},
		{CharacterID: 13, OwnerID: 0},
	}
	connected := ConnectedCharacters(members, []int64{1, 3})
	if len(connected) != 1 || !connected[10] {
		t.Errorf("conectados vieram %v, quero só o 10", connected)
	}
	if connected[0] {
		t.Error("o personagem 0 entrou na presença")
	}
	// Ninguém online é lista vazia, não lista inteira.
	if got := ConnectedCharacters(members, nil); len(got) != 0 {
		t.Errorf("sem ninguém online vieram %d", len(got))
	}
}

// Numa fila só de PCs não há vitais para reservar, e a tela não deve mudar de
// forma por causa de um papel que ali não muda nada.
func TestTheGmEyeWatchesTheTrackerNotTheRole(t *testing.T) {
	pv := int64(30)
	withNPC := []InitiativeEntry{{Label: "Arwen"}, {Label: "Ogro", HpMax: &pv}}
	soPCs := []InitiativeEntry{{Label: "Arwen"}, {Label: "Bruna"}}

	if !GmSeesVitals(withNPC, true) {
		t.Error("o mestre não vê os vitais numa fila que tem NPC")
	}
	if GmSeesVitals(withNPC, false) {
		t.Error("o jogador viu os vitais do NPC")
	}
	if GmSeesVitals(soPCs, true) {
		t.Error("numa fila só de PCs não há o que reservar, e a tela mudou de forma")
	}
}

// Uma borda por campo, e cada uma é a que a tela sozinha não segura: limite que
// vive como atributo de campo de formulário é UI, e quem posta na mão passa por
// cima dos quatro.
func TestValidatingANewCombatantPinsTheFourEdges(t *testing.T) {
	good := CombatantDraft{Label: "Ogro", Initiative: 12, HP: 45, Kind: "npc"}
	if err := ValidateCombatantDraft(good); err != nil {
		t.Fatalf("o combatente bom foi recusado: %v — sem isto as recusas abaixo não provariam nada", err)
	}
	// PV zero é ESTADO VÁLIDO e não ausência: é "sem vida registrada", e o
	// capanga anônimo depende dele.
	if err := ValidateCombatantDraft(CombatantDraft{Label: "Figurante", Initiative: 0, HP: 0, Kind: "npc"}); err != nil {
		t.Errorf("PV 0 foi recusado, e ele é o capanga sem vida rastreada: %v", err)
	}

	cases := []struct {
		name  string
		c     CombatantDraft
		cites string
	}{
		{"sem nome", CombatantDraft{Label: "   ", Initiative: 10, Kind: "npc"}, "nome"},
		{"nome comprido", CombatantDraft{Label: strings.Repeat("a", 61), Initiative: 10, Kind: "npc"}, "61"},
		{"iniciativa alta", CombatantDraft{Label: "Ogro", Initiative: 400, Kind: "npc"}, "400"},
		{"iniciativa baixa", CombatantDraft{Label: "Ogro", Initiative: -6, Kind: "npc"}, "-6"},
		{"PV demais", CombatantDraft{Label: "Ogro", Initiative: 10, HP: 1000, Kind: "npc"}, "1000"},
		{"tipo inventado", CombatantDraft{Label: "Ogro", Initiative: 10, Kind: "dragão"}, "dragão"},
	}
	for _, c := range cases {
		err := ValidateCombatantDraft(c.c)
		if err == nil {
			t.Errorf("%s: passou", c.name)
			continue
		}
		// A mensagem tem de nomear o VALOR ofensivo: quem a lê está no meio de
		// um combate e precisa consertar sem sair da tela.
		if !strings.Contains(err.Error(), c.cites) {
			t.Errorf("%s: a recusa %q não cita %q", c.name, err, c.cites)
		}
	}
}

// O limite do nome conta RUNAS e não bytes: um nome de 60 letras acentuadas tem
// mais de 60 bytes, e contar bytes recusaria um nome mais curto do que o que
// deixa passar em ASCII.
func TestTheNameLimitCountsLettersNotBytes(t *testing.T) {
	accented := strings.Repeat("ã", MaxLabelLetters) // 60 letras, 120 bytes
	if err := ValidateCombatantDraft(CombatantDraft{Label: accented, Initiative: 10, Kind: "npc"}); err != nil {
		t.Errorf("60 letras acentuadas foram recusadas: %v", err)
	}
}

// O EXTRATO DA MANUTENÇÃO é a segunda linha da faixa: quem entrou na vez pagou
// o quê, e o que caiu por falta de mana (p227).
func TestTheStripTellsWhatSustainingCostThisTurn(t *testing.T) {
	cases := []struct {
		name      string
		statement *TurnUpkeep
		want      string
	}{
		{name: "sem sustentada a linha não existe"},
		// O SALDO vem junto do gasto: "−1 PM" diz o preço e não diz se dá para
		// pagar de novo, que é a decisão de quem sustenta.
		{name: "uma paga diz o nome, o custo e o que sobrou",
			statement: &TurnUpkeep{Paid: []string{"Velocidade"}, Cost: 1, MpBefore: 12, MpAfter: 11},
			want:      "Velocidade · −1 PM (12 → 11)"},
		{name: "duas pagas somam o custo numa linha só",
			statement: &TurnUpkeep{Paid: []string{"Velocidade", "Oração"}, Cost: 2, MpBefore: 58, MpAfter: 56},
			want:      "Velocidade e Oração · −2 PM (58 → 56)"},
		// A QUE CAIU é a notícia, e ela vem por último porque é o que muda a
		// ficha de quem está jogando.
		{name: "a que caiu é nomeada",
			statement: &TurnUpkeep{Dropped: []string{"Velocidade"}, Cost: 0},
			want:      "Velocidade acabou: sem PM para sustentar"},
		// A RAZÃO muda a frase: no chão não é falta de mana.
		{name: "quem caiu a 0 PV não sustenta, e a faixa diz isso",
			statement: &TurnUpkeep{Dropped: []string{"Velocidade", "Oração"}, Unconscious: true},
			want:      "Velocidade e Oração acabaram: inconsciente não sustenta"},
		{name: "paga e caída convivem",
			statement: &TurnUpkeep{Paid: []string{"Oração"}, Dropped: []string{"Velocidade"}, Cost: 1, MpBefore: 1},
			want:      "Oração · −1 PM (1 → 0) · Velocidade acabou: sem PM para sustentar"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := UpkeepLine(c.statement); got != c.want {
				t.Errorf("a faixa diz %q, quero %q", got, c.want)
			}
		})
	}
}
