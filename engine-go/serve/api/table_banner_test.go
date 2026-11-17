package api

import (
	"net/http"
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
