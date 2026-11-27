package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/engine"
)

// O ESTANDARTE (p54): "No início de cada cena, você e todos os aliados que possam ver
// seu estandarte recebem um número de PM temporários igual ao seu Carisma (mínimo 1).
// Esses pontos temporários desaparecem no final da cena."
//
// O cavaleiro recebe sozinho, como a Armadura da Honra. QUEM VÊ o estandarte é o
// mestre que diz (decisão do dono): a mesa dele mostra uma faixa com um botão por
// aliado, e o sistema só faz a conta.

const banner = "class.cavaleiro.estandarte"

func knightWithTheBanner(t *testing.T, charisma int64) (sceneFixture, int64) {
	t.Helper()
	f := newSceneFixture(t)
	knight := seedCharacterAtLevel(t, f.s, f.player, "Sir Aldo", "Cavaleiro", 14, 0, 0)
	seedMember(t, f.s, f.campaignID, knight)
	charismaOf(t, f, knight, charisma)
	choiceCom(t, f, knight, `["class.cavaleiro.titulo","`+banner+`"]`, `[]`)
	return f, knight
}

func startsTheScene(t *testing.T, f sceneFixture) {
	t.Helper()
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/cena/iniciar/acao", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
}

func tempMpOf(t *testing.T, f sceneFixture, id int64) int {
	t.Helper()
	return statOf(t, f, id, engine.ModifierTarget{K: "tempMp"})
}

func TestTheBannerGivesTheKnightHisCharismaInTemporaryPMWhenTheSceneStarts(t *testing.T) {
	t.Run("Carisma 3 dá 3", func(t *testing.T) {
		f, knight := knightWithTheBanner(t, 3)
		if got := tempMpOf(t, f, knight); got != 0 {
			t.Fatalf("o cavaleiro já tinha %d PM temporários antes da cena — o caso não mede o gesto", got)
		}
		startsTheScene(t, f)
		if got := tempMpOf(t, f, knight); got != 3 {
			t.Errorf("o Estandarte deu %d PM temporários ao cavaleiro de Carisma 3, e o livro diz 3 (p54)", got)
		}
	})
	t.Run("Carisma -1 dá o mínimo, 1", func(t *testing.T) {
		f, knight := knightWithTheBanner(t, -1)
		startsTheScene(t, f)
		if got := tempMpOf(t, f, knight); got != 1 {
			t.Errorf("o Estandarte deu %d PM temporários com Carisma -1, e o livro diz mínimo 1 (p54)", got)
		}
	})
}

func TestTheGameMasterGivesTheBannerToTheAllyWhoSeesIt(t *testing.T) {
	f, knight := knightWithTheBanner(t, 3)
	startsTheScene(t, f)

	page := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	give := fmt.Sprintf("%s/estandarte/%d/%d", f.tableUrl(), knight, f.charID)
	if !strings.Contains(page, give) {
		t.Fatalf("a mesa do mestre não oferece o Estandarte de Sir Aldo à Arcanista (%s)", give)
	}
	if strings.Contains(page, fmt.Sprintf("%s/estandarte/%d/%d", f.tableUrl(), knight, knight)) {
		t.Error("a faixa oferece o Estandarte ao próprio cavaleiro, que já o recebe sozinho")
	}
	card := sectionOf(page, "table-banner")
	requireFloatingCard(t, card, "do Estandarte")
	// A CABEÇA diz quantos ainda faltam escolher: a Arcanista, porque o cavaleiro já o recebe sozinho.
	if !strings.Contains(card, "Estandarte · a escolher") || !strings.Contains(card, `class="ds-float-count">1<`) {
		t.Error("a cabeça do Estandarte não diz quantos faltam escolher")
	}
	if player := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String(); strings.Contains(player, "/estandarte/") {
		t.Error("o jogador vê os botões do Estandarte, que são do mestre")
	}

	if got := tempMpOf(t, f, f.charID); got != 0 {
		t.Fatalf("a Arcanista já tinha %d PM temporários — o caso não mede o gesto", got)
	}
	if refusal := tableRefusal(t, f.requests(t, f.gm, http.MethodPost, give, "").Body.String()); refusal != "" {
		t.Fatalf("dar o Estandarte à Arcanista foi recusado: %q", refusal)
	}
	if got := tempMpOf(t, f, f.charID); got != 3 {
		t.Errorf("a Arcanista recebeu %d PM temporários do Estandarte de Carisma 3, e o livro diz 3 (p54)", got)
	}
	if again := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); strings.Contains(again, give) {
		t.Error("a faixa ainda oferece o Estandarte a quem já o recebeu nesta cena")
	}

	// "Esses pontos temporários desaparecem no final da cena" (p54).
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/cena/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar a cena deu %d", rec.Code)
	}
	if got := tempMpOf(t, f, f.charID); got != 0 {
		t.Errorf("a cena acabou e a Arcanista ainda tem %d PM temporários do Estandarte", got)
	}
	if got := tempMpOf(t, f, knight); got != 0 {
		t.Errorf("a cena acabou e o cavaleiro ainda tem %d PM temporários do Estandarte", got)
	}
}

func TestOnlyTheGameMasterGivesTheBanner(t *testing.T) {
	f, knight := knightWithTheBanner(t, 3)
	startsTheScene(t, f)
	give := fmt.Sprintf("%s/estandarte/%d/%d", f.tableUrl(), knight, f.charID)
	f.requests(t, f.player, http.MethodPost, give, "")
	if got := tempMpOf(t, f, f.charID); got != 0 {
		t.Errorf("o jogador deu o Estandarte pelo endereço, e a Arcanista ganhou %d PM temporários", got)
	}
}

// O "PRONTO" RECOLHE A FAIXA ATÉ A PRÓXIMA CENA (decisão do dono). O mestre diz quem vê o
// estandarte e fecha: sem ele, a faixa ficava até o fim da cena enquanto houvesse aliado
// sem a reserva, e no celular empurrava a fila para baixo.
func TestTheDoneButtonPutsTheBannerAwayUntilTheNextScene(t *testing.T) {
	f, knight := knightWithTheBanner(t, 3)
	startsTheScene(t, f)
	done := fmt.Sprintf("%s/estandarte/%d/pronto", f.tableUrl(), knight)
	give := fmt.Sprintf("%s/estandarte/%d/%d", f.tableUrl(), knight, f.charID)

	if page := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(page, done) {
		t.Fatalf("a faixa do Estandarte não oferece o Pronto (%s)", done)
	}
	if refusal := tableRefusal(t, f.requests(t, f.gm, http.MethodPost, done, "").Body.String()); refusal != "" {
		t.Fatalf("o Pronto foi recusado: %q", refusal)
	}
	if page := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); strings.Contains(page, give) {
		t.Error("o mestre tocou em Pronto e a faixa ainda oferece o Estandarte")
	}

	// A PRÓXIMA CENA traz a faixa de volta: o Estandarte vale "no início de cada cena" (p54).
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/cena/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar a cena deu %d", rec.Code)
	}
	startsTheScene(t, f)
	if page := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(page, give) {
		t.Error("a cena nova não trouxe de volta a faixa do Estandarte")
	}
}
