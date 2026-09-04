package table

import (
	"strings"
	"t20engine/aovivo"
	"t20engine/web/ui"
	"testing"
)

// O guarda que justifica o piloto reusar `stateForRole` em vez de montar a
// própria leitura: a PÁGINA obedece à mesma redação que o socket.
//
// Provado VERMELHO trocando `aovivo.StateForRole(role, ...)` por `s.deps.Sessions().GetState(...)`
// no `LoadView` — o HTML passou a carregar "12/130", os PV que o mestre
// escondeu, para dentro da tela do jogador.
// A vez é MINHA quando a linha na vez é de um personagem meu — e é "de outro"
// quando não é. Tradução literal do `playerTurnState` da SPA; duas escadas
// divergiriam em silêncio.
func TestTableTurnOf(t *testing.T) {
	meu, alheio := int64(7), int64(9)
	fila := []aovivo.InitiativeEntry{
		{Label: "Ogro", Initiative: 19, Type: "npc"},
		{Label: "Arcanista", Initiative: 12, Type: "character", CharacterID: &meu},
	}
	meus := map[int64]bool{meu: true}

	casos := []struct {
		nome      string
		turnIndex int
		kind      string
		label     string
	}{
		{"fora de combate ninguém está na vez", -1, "idle", ""},
		{"a vez do ogro é de outro", 0, "other", "Ogro"},
		{"a vez do meu personagem é minha", 1, "mine", ""},
		{"índice além da fila não inventa uma vez", 5, "idle", ""},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			got := tableTurnOf(&aovivo.SessionRuntimeState{Initiative: fila, TurnIndex: c.turnIndex}, meus)
			if got.Kind != c.kind || got.Label != c.label {
				t.Errorf("veio {%s %q}, queria {%s %q}", got.Kind, got.Label, c.kind, c.label)
			}
		})
	}
	// O personagem alheio não acende a faixa de ninguém.
	outro := tableTurnOf(&aovivo.SessionRuntimeState{
		Initiative: []aovivo.InitiativeEntry{{Label: "Colega", Type: "character", CharacterID: &alheio}},
		TurnIndex:  0,
	}, meus)
	if outro.Kind != "other" {
		t.Errorf("a vez de um PC alheio virou %q", outro.Kind)
	}
}

// Os LIMIARES da cor, e só eles: a tabela inteira de porcentagens seria a
// implementação reescrita. 25 e 50 são os mesmos do `hpFillVar` da SPA, e é a
// divergência entre os dois que este teste existe para tornar barulhenta.
func TestHpToneAtTheThresholds(t *testing.T) {
	casos := []struct {
		pct int
		tom string
	}{
		{0, "bg-hp-critical"},
		{25, "bg-hp-critical"},
		{26, "bg-hp-hurt"},
		{50, "bg-hp-hurt"},
		{51, "bg-hp-full"},
		{100, "bg-hp-full"},
	}
	for _, c := range casos {
		if got := hpToneOf(c.pct); got != c.tom {
			t.Errorf("hpToneOf(%d) = %q, queria %q", c.pct, got, c.tom)
		}
	}
}

// O campo vazio não pode virar um total (ALE-236).
//
// MEDIDO no navegador: o `data-bind` do Datastar escreve ZERO no sinal quando
// um `<input type=number>` esvazia — digitar 7, apagar, e o sinal vai a 0. Sem
// guarda, apagar para redigitar mostra "Total previsto 8" com bônus 8 e dado
// nenhum: um total que não existe, lido no instante da decisão. Mesma família
// da ALE-224, onde a prévia era o que impedia o erro silencioso.
//
// Este guarda pina a EXPRESSÃO, e digo isso em vez de fingir que ele pina o
// comportamento. O teste comportamental exigiria a cena EM JOGO com o jogador
// tendo personagem nela, e montar esse estado no e2e é entrar exatamente na
// armadilha que a ALE-238 documenta: asserção que depende do estado do combate
// mede o banco, não o app. A prova do comportamento foi a medição no navegador,
// que está descrita acima e é reproduzível em três linhas.
func TestTheD20PreviewDoesNotLieWithAnEmptyField(t *testing.T) {
	bonus := int64(8)
	html, err := ui.RenderFragment(t.Context(), tableScene(View{
		CampaignID: 7, SessionID: 42, SceneActive: true,
		Turn: tableTurn{Kind: "idle"},
		Eu:   &tableMe{CharacterID: 1, Nome: "Samira", Bonus: bonus},
	}))
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	// LITERAL e não escapado: o atributo é CONSTANTE, e o templ só escapa os
	// dinâmicos — foi o que a ALE-227 mediu ao comparar as duas saídas byte a
	// byte. Escrevi a forma escapada primeiro e o guarda nasceu vermelho por
	// isso, o que ao menos provou que ele lê o HTML de verdade.
	const faixa = "$d20 >= 1 && $d20 <= 20"
	if !strings.Contains(html, faixa) {
		t.Errorf("a prévia não é condicionada à faixa do dado — campo vazio vira um total inventado")
	}
	if !strings.Contains(html, "informe o dado") {
		t.Error("sem dado, a linha não diz o que falta")
	}
	if !strings.Contains(html, "$registrando || !(") {
		t.Error("o botão continua oferecendo uma ação que o servidor vai recusar")
	}
}

// A FAIXA DE QUEM VEM DEPOIS (ALE-290).
//
// O `aovivo.UpcomingTurns` existia desde a ALE-179 com cinco guardas e ZERO
// telas — a conta da ordem circular no ar, e ninguém desenhando quem vem depois.
// Estes casos prendem a TRADUÇÃO dela para a tela; a regra circular continua
// presa lá, e reafirmá-la aqui seria a mesma fronteira duas vezes.
//
// O que é desta camada, e só desta: onde a RODADA VIRA, e qual das três é
// minha. As duas são fatos de apresentação — a regra não sabe quem está olhando
// nem desenha setas.
func TestTheTurnStripSaysWhoIsNextAndWhereTheRoundTurns(t *testing.T) {
	meu := int64(7)
	fila := []aovivo.InitiativeEntry{
		{Label: "Ogro", Initiative: 20, Type: "npc"},
		{Label: "Arwen", Initiative: 15, Type: "character", CharacterID: &meu},
		{Label: "Zumbi 1", Initiative: 10, Type: "npc"},
		{Label: "Zumbi 2", Initiative: 5, Type: "npc"},
	}
	meus := map[int64]bool{meu: true}

	casos := []struct {
		nome      string
		turnIndex int
		rotulos   []string
		meu       int // a posição que é minha, ou -1
		viraEm    int // a posição onde a rodada vira, ou -1
	}{
		{
			nome:      "no começo da rodada ela olha para a frente",
			turnIndex: 0, rotulos: []string{"Ogro", "Arwen", "Zumbi 1"}, meu: 1, viraEm: -1,
		},
		{
			// O CASO QUE A ALE-179 NOMEIA: no último da rodada, "quem vem depois"
			// está no TOPO da lista, e é justamente quando a pergunta mais importa.
			nome:      "no último da rodada ela DÁ A VOLTA",
			turnIndex: 3, rotulos: []string{"Zumbi 2", "Ogro", "Arwen"}, meu: 2, viraEm: 1,
		},
		{
			nome:      "a volta acontece na terceira posição quando falta um",
			turnIndex: 2, rotulos: []string{"Zumbi 1", "Zumbi 2", "Ogro"}, meu: -1, viraEm: 2,
		},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			faixa := turnStripOf(&aovivo.SessionRuntimeState{
				Initiative: fila, TurnIndex: c.turnIndex, SceneActive: true,
			}, meus)

			if len(faixa) != len(c.rotulos) {
				t.Fatalf("a faixa veio com %d nomes, queria %d: %+v", len(faixa), len(c.rotulos), faixa)
			}
			for i, esperado := range c.rotulos {
				if faixa[i].Rotulo != esperado {
					t.Errorf("posição %d é %q, queria %q", i, faixa[i].Rotulo, esperado)
				}
				if faixa[i].Meu != (i == c.meu) {
					t.Errorf("posição %d (%s): Meu=%v, queria %v", i, esperado, faixa[i].Meu, i == c.meu)
				}
				if faixa[i].ViraARodada != (i == c.viraEm) {
					t.Errorf("posição %d (%s): ViraARodada=%v, queria %v", i, esperado, faixa[i].ViraARodada, i == c.viraEm)
				}
			}
			// A PRIMEIRA é sempre a que está na vez, e as outras nunca são.
			if !faixa[0].Agora {
				t.Error("a primeira posição não está marcada como a da vez")
			}
			for i := 1; i < len(faixa); i++ {
				if faixa[i].Agora {
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
	meus := map[int64]bool{}

	if faixa := turnStripOf(&aovivo.SessionRuntimeState{
		Initiative: []aovivo.InitiativeEntry{{Label: "Ogro"}}, TurnIndex: -1,
	}, meus); len(faixa) != 0 {
		t.Errorf("fora de combate a faixa desenhou %d nomes: %+v", len(faixa), faixa)
	}

	dois := []aovivo.InitiativeEntry{{Label: "Ogro"}, {Label: "Arwen"}}
	faixa := turnStripOf(&aovivo.SessionRuntimeState{
		Initiative: dois, TurnIndex: 1, SceneActive: true,
	}, meus)
	if len(faixa) != 2 {
		t.Fatalf("uma fila de dois virou uma faixa de %d: %+v", len(faixa), faixa)
	}
	if faixa[0].Rotulo != "Arwen" || faixa[1].Rotulo != "Ogro" {
		t.Errorf("a faixa da fila de dois é %q,%q", faixa[0].Rotulo, faixa[1].Rotulo)
	}
	if !faixa[1].ViraARodada {
		t.Error("a volta na fila de dois não foi marcada")
	}
}

// "VOCÊ" SÓ QUANDO É UM (ALE-290, decisão do dono depois da medição no navegador).
//
// A faixa dizia "você" em toda linha de quem olha, e no navegador isso saiu como
// **"você › Tanque Placas Nv10 › ⟲ você"** — a pessoa com dois personagens na
// fila não sabe qual é qual, e a faixa existe justamente para responder "quanto
// falta para MIM".
//
// A regra é desambiguar quando precisa: com UM meu na faixa, "você" responde
// direto e não custa largura; com dois ou mais, o nome volta. A cor dourada
// marca os dois casos.
//
// O defeito não aparecia em teste nenhum porque a bancada semeia UM personagem
// por pessoa — foi a seed, com o dono levando vários, que o mostrou.
func TestTheStripSaysYourNameWhenMoreThanOneIsYours(t *testing.T) {
	meu, outroMeu := int64(7), int64(8)
	meus := map[int64]bool{meu: true, outroMeu: true}

	umSo := turnStripOf(&aovivo.SessionRuntimeState{SceneActive: true, TurnIndex: 0, Initiative: []aovivo.InitiativeEntry{
		{Label: "Ogro"},
		{Label: "Arwen", Type: "character", CharacterID: &meu},
		{Label: "Zumbi 1"},
	}}, meus)
	if got := turnStripName(umSo[1]); got != "você" {
		t.Errorf("com UM personagem meu a faixa escreveu %q, queria \"você\"", got)
	}

	dois := turnStripOf(&aovivo.SessionRuntimeState{SceneActive: true, TurnIndex: 0, Initiative: []aovivo.InitiativeEntry{
		{Label: "Recruta", Type: "character", CharacterID: &meu},
		{Label: "Tanque"},
		{Label: "Arcanista", Type: "character", CharacterID: &outroMeu},
	}}, meus)
	for _, i := range []int{0, 2} {
		if got := turnStripName(dois[i]); got == "você" {
			t.Errorf("posição %d escreveu \"você\" com DOIS personagens meus na faixa — qual deles?", i)
		}
	}
	if turnStripName(dois[0]) != "Recruta" || turnStripName(dois[2]) != "Arcanista" {
		t.Errorf("os nomes não voltaram: %q e %q", turnStripName(dois[0]), turnStripName(dois[2]))
	}
	// A COR continua marcando os dois: o que se perdeu foi a palavra, não o
	// destaque.
	if !dois[0].Meu || !dois[2].Meu {
		t.Error("as linhas deixaram de ser marcadas como minhas")
	}
}
