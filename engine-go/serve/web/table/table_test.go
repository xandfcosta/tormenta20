package table

import (
	"strings"
	"t20engine/domain/live"
	"t20engine/serve/web/ui"
	"testing"
)

// O guarda que justifica o app reusar `stateForRole` em vez de montar a
// própria leitura: a PÁGINA obedece à mesma redação que o socket.
//
// Provado VERMELHO trocando `live.StateForRole(role, ...)` por `stateOf(t, s.deps.Sessions(), ...)`
// no `LoadView` — o HTML passou a carregar "12/130", os PV que o mestre
// escondeu, para dentro da tela do jogador.
// A vez é MINHA quando a linha na vez é de um personagem meu — e é "de outro"
// quando não é. A escada mora num lugar só: duas divergiriam em silêncio.
func TestTableTurnOf(t *testing.T) {
	mine, foreign := int64(7), int64(9)
	queue := []live.InitiativeEntry{
		{Label: "Ogro", Initiative: 19, Type: "npc"},
		{Label: "Arcanista", Initiative: 12, Type: "character", CharacterID: &mine},
	}
	owned := map[int64]bool{mine: true}

	cases := []struct {
		name      string
		turnIndex int
		kind      string
		label     string
	}{
		{"fora de combate ninguém está na vez", -1, "idle", ""},
		{"a vez do ogro é de outro", 0, "other", "Ogro"},
		{"a vez do meu personagem é minha", 1, "mine", ""},
		{"índice além da fila não inventa uma vez", 5, "idle", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := tableTurnOf(&live.SessionRuntimeState{Initiative: queue, TurnIndex: c.turnIndex}, owned)
			if got.Kind != c.kind || got.Label != c.label {
				t.Errorf("veio {%s %q}, queria {%s %q}", got.Kind, got.Label, c.kind, c.label)
			}
		})
	}
	// O personagem alheio não acende a faixa de ninguém.
	other := tableTurnOf(&live.SessionRuntimeState{
		Initiative: []live.InitiativeEntry{{Label: "Colega", Type: "character", CharacterID: &foreign}},
		TurnIndex:  0,
	}, owned)
	if other.Kind != "other" {
		t.Errorf("a vez de um PC alheio virou %q", other.Kind)
	}
}

// Os limiares da cor do PV NÃO se prendem aqui: a escada deixou de ser da Mesa
// — as quatro superfícies que pintam PV leem a mesma —, e quem a guarda é o
// `TestTheHpLadderTurnsAtTheThresholds`, em `web/ui`.
//
// (Esse nome fica numa LINHA SÓ de propósito: quebrado em duas, o guarda de
// citação lê o pedaço de cima como um teste inexistente.)

// O campo vazio não pode virar um total.
//
// MEDIDO no navegador: o `data-bind` do Datastar escreve ZERO no sinal quando
// um `<input type=number>` esvazia — digitar 7, apagar, e o sinal vai a 0. Sem
// guarda, apagar para redigitar mostra "Total previsto 8" com bônus 8 e dado
// nenhum: um total que não existe, lido no instante da decisão.
//
// Este guarda pina a EXPRESSÃO, e não o comportamento. O comportamental
// exigiria a cena EM JOGO com o jogador tendo personagem nela, e montar esse
// estado no e2e mede o banco, não o app.
func TestTheD20PreviewDoesNotLieWithAnEmptyField(t *testing.T) {
	bonus := int64(8)
	html, err := ui.RenderFragment(t.Context(), tableScene(View{
		CampaignID: 7, SessionID: 42, SceneActive: true,
		Turn: tableTurn{Kind: "idle"},
		Eu:   &tableMe{CharacterID: 1, Name: "Samira", Bonus: bonus},
	}))
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	// LITERAL e não escapado: o atributo é CONSTANTE, e o templ só escapa os
	// dinâmicos. Procurar a forma ESCAPADA aqui reprova com o guarda certo.
	const strip = "$d20 >= 1 && $d20 <= 20"
	if !strings.Contains(html, strip) {
		t.Errorf("a prévia não é condicionada à faixa do dado — campo vazio vira um total inventado")
	}
	if !strings.Contains(html, "informe o dado") {
		t.Error("sem dado, a linha não diz o que falta")
	}
	if !strings.Contains(html, "$registering || !(") {
		t.Error("o botão continua oferecendo uma ação que o servidor vai recusar")
	}
}

// A FAIXA DE QUEM VEM DEPOIS.
//
// Estes casos prendem a TRADUÇÃO do `live.UpcomingTurns` para a tela; a regra
// da ordem circular continua presa lá, e reafirmá-la aqui seria a mesma
// fronteira duas vezes.
//
// O que é desta camada, e só desta: onde a RODADA VIRA, e qual das três é
// minha. As duas são fatos de apresentação — a regra não sabe quem está olhando
// nem desenha setas.
func TestTheTurnStripSaysWhoIsNextAndWhereTheRoundTurns(t *testing.T) {
	mine := int64(7)
	queue := []live.InitiativeEntry{
		{Label: "Ogro", Initiative: 20, Type: "npc"},
		{Label: "Arwen", Initiative: 15, Type: "character", CharacterID: &mine},
		{Label: "Zumbi 1", Initiative: 10, Type: "npc"},
		{Label: "Zumbi 2", Initiative: 5, Type: "npc"},
	}
	owned := map[int64]bool{mine: true}

	cases := []struct {
		name      string
		turnIndex int
		labels    []string
		mine      int // a posição que é minha, ou -1
		becomesAt int // a posição onde a rodada vira, ou -1
	}{
		{
			name:      "no começo da rodada ela olha para a frente",
			turnIndex: 0, labels: []string{"Ogro", "Arwen", "Zumbi 1"}, mine: 1, becomesAt: -1,
		},
		{
			// O CASO QUE IMPORTA: no último da rodada, "quem vem depois" está no
			// TOPO da lista, e é justamente quando a pergunta mais pesa.
			name:      "no último da rodada ela DÁ A VOLTA",
			turnIndex: 3, labels: []string{"Zumbi 2", "Ogro", "Arwen"}, mine: 2, becomesAt: 1,
		},
		{
			name:      "a volta acontece na terceira posição quando falta um",
			turnIndex: 2, labels: []string{"Zumbi 1", "Zumbi 2", "Ogro"}, mine: -1, becomesAt: 2,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			strip := turnStripOf(&live.SessionRuntimeState{
				Initiative: queue, TurnIndex: c.turnIndex, Scene: &live.Scene{Kind: live.SceneAction, Number: 1}, ScenesSoFar: 1,
			}, owned)

			if len(strip) != len(c.labels) {
				t.Fatalf("a faixa veio com %d nomes, queria %d: %+v", len(strip), len(c.labels), strip)
			}
			for i, want := range c.labels {
				if strip[i].Label != want {
					t.Errorf("posição %d é %q, queria %q", i, strip[i].Label, want)
				}
				if strip[i].Mine != (i == c.mine) {
					t.Errorf("posição %d (%s): Meu=%v, queria %v", i, want, strip[i].Mine, i == c.mine)
				}
				if strip[i].WrapsRound != (i == c.becomesAt) {
					t.Errorf("posição %d (%s): ViraARodada=%v, queria %v", i, want, strip[i].WrapsRound, i == c.becomesAt)
				}
			}
			// A PRIMEIRA é sempre a que está na vez, e as outras nunca são.
			if !strip[0].Now {
				t.Error("a primeira posição não está marcada como a da vez")
			}
			for i := 1; i < len(strip); i++ {
				if strip[i].Now {
					t.Errorf("a posição %d também se diz na vez", i)
				}
			}
		})
	}
}

// FORA DE COMBATE não há faixa, e a fila CURTA não inventa nomes.
//
// Os dois casos juntos porque prendem a mesma decisão: a faixa mostra o que
// EXISTE, e nunca preenche as três posições com repetição. Numa mesa de dois, a
// volta traria o Ogro duas vezes se ela pedisse três da fila de dois — e é o
// `UpcomingTurns` que corta, com o guarda dele.
func TestTheTurnStripShowsNothingOutOfCombatAndNeverRepeats(t *testing.T) {
	mine := map[int64]bool{}

	if strip := turnStripOf(&live.SessionRuntimeState{
		Initiative: []live.InitiativeEntry{{Label: "Ogro"}}, TurnIndex: -1,
	}, mine); len(strip) != 0 {
		t.Errorf("fora de combate a faixa desenhou %d nomes: %+v", len(strip), strip)
	}

	two := []live.InitiativeEntry{{Label: "Ogro"}, {Label: "Arwen"}}
	band := turnStripOf(&live.SessionRuntimeState{
		Initiative: two, TurnIndex: 1, Scene: &live.Scene{Kind: live.SceneAction, Number: 1}, ScenesSoFar: 1,
	}, mine)
	if len(band) != 2 {
		t.Fatalf("uma fila de dois virou uma faixa de %d: %+v", len(band), band)
	}
	if band[0].Label != "Arwen" || band[1].Label != "Ogro" {
		t.Errorf("a faixa da fila de dois é %q,%q", band[0].Label, band[1].Label)
	}
	if !band[1].WrapsRound {
		t.Error("a volta na fila de dois não foi marcada")
	}
}

// "VOCÊ" SÓ QUANDO É UM (decisão do dono).
//
// Dizer "você" em toda linha de quem olha sai como **"você › Tanque Placas Nv10
// › ⟲ você"**: a pessoa com dois personagens na fila não sabe qual é qual, e a
// faixa existe justamente para responder "quanto falta para MIM".
//
// A regra é desambiguar quando precisa: com UM meu na faixa, "você" responde
// direto e não custa largura; com dois ou mais, o nome volta. A cor dourada
// marca os dois casos, e por isso o caso PRECISA de dois personagens meus —
// uma bancada que semeie um por pessoa não enfrenta o ramo.
func TestTheStripSaysYourNameWhenMoreThanOneIsYours(t *testing.T) {
	mine, otherMine := int64(7), int64(8)
	owned := map[int64]bool{mine: true, otherMine: true}

	umSo := turnStripOf(&live.SessionRuntimeState{Scene: &live.Scene{Kind: live.SceneAction, Number: 1}, ScenesSoFar: 1, TurnIndex: 0, Initiative: []live.InitiativeEntry{
		{Label: "Ogro"},
		{Label: "Arwen", Type: "character", CharacterID: &mine},
		{Label: "Zumbi 1"},
	}}, owned)
	if got := turnStripName(umSo[1]); got != "você" {
		t.Errorf("com UM personagem meu a faixa escreveu %q, queria \"você\"", got)
	}

	two := turnStripOf(&live.SessionRuntimeState{Scene: &live.Scene{Kind: live.SceneAction, Number: 1}, ScenesSoFar: 1, TurnIndex: 0, Initiative: []live.InitiativeEntry{
		{Label: "Recruta", Type: "character", CharacterID: &mine},
		{Label: "Tanque"},
		{Label: "Arcanista", Type: "character", CharacterID: &otherMine},
	}}, owned)
	for _, i := range []int{0, 2} {
		if got := turnStripName(two[i]); got == "você" {
			t.Errorf("posição %d escreveu \"você\" com DOIS personagens meus na faixa — qual deles?", i)
		}
	}
	if turnStripName(two[0]) != "Recruta" || turnStripName(two[2]) != "Arcanista" {
		t.Errorf("os nomes não voltaram: %q e %q", turnStripName(two[0]), turnStripName(two[2]))
	}
	// A COR continua marcando os dois: o que se perdeu foi a palavra, não o
	// destaque.
	if !two[0].Mine || !two[2].Mine {
		t.Error("as linhas deixaram de ser marcadas como minhas")
	}
}
