package api

import (
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"testing"
)

// O kit de p140 mora no capítulo dos itens e serviços mundanos, e as armas dele são as da Tabela 3-3
// (p144-145). Os itens específicos do Capítulo 8 têm a categoria da arma de que são feitos, e não entram.
func TestTheStartingKitOffersOnlyMundaneWeapons(t *testing.T) {
	f := newSceneFixture(t)
	body := postaAForja(t, f, f.player, "/personagens/nova/esboco", url.Values{"classes": {"Guerreiro:1"}}).Body.String()
	for field, want := range map[string][]string{
		"weaponSimple": {
			"adaga", "arco-curto", "azagaia", "besta-leve", "bordao", "clava", "espada-curta",
			"foice", "funda", "lanca", "maca", "pique", "tacape",
		},
		"weaponMartial": {
			"alabarda", "alfange", "arco-longo", "besta-pesada", "cimitarra", "espada-longa", "florete",
			"gadanho", "lanca-montada", "machadinha", "machado-batalha", "machado-guerra", "mangual",
			"marreta", "martelo-guerra", "montante", "picareta", "tridente",
		},
	} {
		radio := regexp.MustCompile(`<input[^>]*name="` + field + `"[^>]*value="([^"]*)"`)
		var got []string
		for _, m := range radio.FindAllStringSubmatch(body, -1) {
			got = append(got, m[1])
		}
		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Errorf("%s oferece %v, e a Tabela 3-3 tem %v", field, got, want)
		}
	}
}

// A folha não oferece, e o servidor também recusa quem manda à mão.
func TestTheForgeRefusesAMagicWeaponInTheKit(t *testing.T) {
	f := newSceneFixture(t)
	for field, id := range map[string]string{"weaponSimple": "cajado-do-poder", "weaponMartial": "espada-deus"} {
		answers := aFolhaPreenchida()
		answers.Set(field, id)
		rec := postaAForja(t, f, f.player, "/personagens/nova", answers)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("a forja deu %s no kit: %d", id, rec.Code)
		}
	}
}

// Os atributos do Suraggel vêm da ascendência (p31), e a carta diz isso em vez de ficar em branco.
func TestTheSuraggelCardSaysItDependsOnTheAscendancy(t *testing.T) {
	f := newSceneFixture(t)
	body := f.requests(t, f.player, http.MethodGet, "/personagens/nova", "").Body.String()
	card := regexp.MustCompile(`value="suraggel"[\s\S]*?</label>`).FindString(body)
	if card == "" {
		t.Fatal("a folha não tem a carta do Suraggel")
	}
	if !regexp.MustCompile(`depende da ascendência`).MatchString(card) {
		t.Errorf("a carta do Suraggel não diz de onde vêm os atributos:\n%s", card)
	}
}
