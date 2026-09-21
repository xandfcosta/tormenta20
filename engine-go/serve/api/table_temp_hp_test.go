package api

import (
	"net/http"
	"strings"
	"testing"
)

// A MESA VÊ A RESERVA DE PV TEMPORÁRIO de quem está na fila.
//
// O mestre decide a pancada, e a reserva é o que o dano come ANTES do PV
// (p106): sem ela na tela, ele calcula o golpe contra um número que não é o que
// o servidor vai drenar.
//
// O CONTROLE vem primeiro: a mesa SEM poça não pode ter a frase, senão o caso
// ficaria verde sobre uma tela que a mostra sempre.
func TestTheTableShowsTheTemporaryHpOfWhoIsInTheQueue(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	const sentence = "mais 30 temporários"

	before := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if strings.Contains(before, sentence) {
		t.Fatal("a Mesa SEM poça já fala em temporários — o caso mediria o repouso")
	}

	// O Campo de Força dá 30 PV temporários de cena, e a ficha do Arcanista é a
	// que a fixture põe na fila.
	if rec := effect(t, f, f.charID, "aplica/campo-de-forca"); rec.Code != http.StatusOK {
		t.Fatalf("aplicar o Campo de Força devolveu %d", rec.Code)
	}

	after := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	// O NOME ACESSÍVEL é o canal ÚNICO no trilho da fila, que não tem número
	// nenhum — é ele, e não o filete dourado, que responde para quem não vê.
	if !strings.Contains(after, sentence) {
		t.Error("a reserva não chegou ao nome acessível da barra")
	}
	// E o número sai escrito onde há espaço para ele: o cartão do Grupo.
	if !strings.Contains(after, ">+30</span>") {
		t.Error("a reserva não saiu escrita em lugar nenhum da Mesa")
	}
}

// A REDAÇÃO ALCANÇA A RESERVA: o pool que o mestre escondeu não vaza o colchão.
//
// Dizer "este tem 30 de temporário" sobre uma linha cujo PV está oculto contaria
// à mesa exatamente o que o mestre escolheu não contar — e por uma porta que a
// decisão de esconder não passou. O gargalo é UM (`poolBar`), e este caso é o
// que prova que a reserva entrou nele e não ao lado dele.
//
// A asserção é sobre a VIEW e não sobre o HTML, e isso é medida e não preguiça:
// a frase "mais 30 temporários" sai em DUAS superfícies da mesma página — a
// linha da fila e o cartão do Grupo —, e o cartão do Grupo mostra o PV de todo
// membro independentemente do olho da fila, que é comportamento anterior a esta
// fatia. Procurá-la no corpo inteiro responderia sobre a página, e a pergunta é
// sobre a LINHA.
func TestAHiddenPoolHidesItsTemporaryHpToo(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)
	if rec := effect(t, f, f.charID, "aplica/campo-de-forca"); rec.Code != http.StatusOK {
		t.Fatalf("aplicar o Campo de Força devolveu %d", rec.Code)
	}
	sheet, _ := sceneIds(t, f)

	queuedReserve := func(who int64) int64 {
		t.Helper()
		view, _, err := f.s.tableScene.LoadView(t.Context(), who, f.campaignID, f.sessionID)
		if err != nil {
			t.Fatalf("carregar a Mesa: %v", err)
		}
		for _, l := range view.Queue {
			if l.ID == sheet {
				if l.PV == nil {
					return 0
				}
				return l.PV.Temp
			}
		}
		t.Fatal("a ficha sumiu da fila")
		return 0
	}

	// CONTROLE: com o PV à vista, a linha do JOGADOR carrega a reserva. Sem esta
	// metade, o caso abaixo ficaria verde sobre uma fila que nunca a carrega.
	if open := queuedReserve(f.player); open != 30 {
		t.Fatalf("com o PV à vista a linha diz %d de reserva, e o Campo de Força dá 30", open)
	}

	if rec := f.pede(t, f.gm, http.MethodPost,
		f.tableUrl()+"/iniciativa/"+sheet+"/vitais/hp/oculto", ""); rec.Code != http.StatusOK {
		t.Fatalf("esconder o PV devolveu %d", rec.Code)
	}

	if hidden := queuedReserve(f.player); hidden != 0 {
		t.Errorf("o PV oculto vazou %d de reserva para a mesa", hidden)
	}
	// E O MESTRE continua vendo: esconder é decisão sobre o que a MESA vê, e
	// uma redação que cegasse quem a tomou seria outro defeito.
	if forGM := queuedReserve(f.gm); forGM != 30 {
		t.Errorf("o mestre perdeu a própria reserva de vista: %d", forGM)
	}
}
