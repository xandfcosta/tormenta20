package api

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"

	"t20engine/app/combat"
	"t20engine/domain/live"
	"t20engine/infra/events"
)

// A Injeção alquímica (p165): "um ataque que acerte causa seu dano normal e libera
// uma carga de um preparado (como ácido ou fogo alquímico) ou de água benta, que
// atinge o alvo automaticamente. A melhoria tem espaço para 2 doses."

const injectionID = "melhoria-injecao-alquimica"

// doseAttackOnTurn põe a sessão em curso, porque a superfície Ações acha a mesa do
// personagem varrendo as sessões `active`.
func doseAttackOnTurn(t *testing.T) (sceneFixture, string) {
	t.Helper()
	f, goblin := attackOnTurn(t)
	if rec := f.requests(t, f.gm, http.MethodPatch, f.tableUrl(), `{"status":"active"}`); rec.Code != http.StatusOK {
		t.Fatalf("iniciar a sessão deu %d", rec.Code)
	}
	return f, goblin
}

// weaponWithInjection é a espada longa que o `attackOnTurn` empunha, com a melhoria.
func weaponWithInjection(t *testing.T, f sceneFixture) int64 {
	t.Helper()
	lines := bagLinesOf(t, f, f.charID, "Espada longa")
	if len(lines) != 1 {
		t.Fatalf("a ficha tem %d linhas de Espada longa", len(lines))
	}
	setImprovements(t, f, f.charID, lines[0].ID, `{"item_improvements":["`+injectionID+`"]}`)
	return lines[0].ID
}

func setImprovements(t *testing.T, f sceneFixture, character, item int64, signals string) {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/itens/%d/melhorias?tab=bag", character, item)
	if refusal := bagRefusal(t, f.requests(t, f.player, http.MethodPost, target, signals).Body.String()); refusal != "" {
		t.Fatalf("gravar as melhorias foi recusado: %q", refusal)
	}
}

// loadsDose carrega o item na arma pelo gesto da mochila, e devolve a recusa.
func loadsDose(t *testing.T, f sceneFixture, character, doseItem, weapon int64) string {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/itens/%d/carrega/%d?tab=bag", character, doseItem, weapon)
	return bagRefusal(t, f.requests(t, f.player, http.MethodPost, target, "").Body.String())
}

// loaded põe uma dose na arma, sem ser o que o teste mede.
func loaded(t *testing.T, f sceneFixture, weapon int64, catalog, name string) {
	t.Helper()
	item := itemSemeia(t, f, f.charID, catalog, name, "")
	if refusal := loadsDose(t, f, f.charID, item, weapon); refusal != "" {
		t.Fatalf("carregar %s foi recusado: %q", name, refusal)
	}
}

// dosesIn são as doses gravadas na arma, da mais antiga para a mais nova.
func dosesIn(t *testing.T, f sceneFixture, weapon int64) []string {
	t.Helper()
	rows, err := f.s.db.QueryContext(context.Background(),
		"SELECT catalogId FROM character_item_doses WHERE itemId = ? ORDER BY id", weapon)
	if err != nil {
		t.Fatalf("ler as doses da arma %d: %v", weapon, err)
	}
	defer func() { _ = rows.Close() }()
	doses := []string{}
	for rows.Next() {
		var dose string
		if err := rows.Scan(&dose); err != nil {
			t.Fatalf("ler uma dose: %v", err)
		}
		doses = append(doses, dose)
	}
	return doses
}

func expectDoses(t *testing.T, f sceneFixture, weapon int64, want ...string) {
	t.Helper()
	got := dosesIn(t, f, weapon)
	if !slices.Equal(got, append([]string{}, want...)) {
		t.Errorf("a arma tem as doses %v, e devia ter %v", got, want)
	}
}

// attacks rola o golpe da vez contra o Goblin, pela superfície Ações.
func attacks(t *testing.T, f sceneFixture, goblin, extraSignals string) string {
	t.Helper()
	signals := fmt.Sprintf(`{"turn_target":%q%s}`, goblin, extraSignals)
	return sceneRefusal(f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/atacar/0?embutida=1", f.charID), signals))
}

func gmRules(t *testing.T, f sceneFixture, ruling string) {
	t.Helper()
	rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/julgar/"+ruling, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("decidir %q deu %d", ruling, rec.Code)
	}
}

func TestLoadingADoseMovesItFromTheBagToTheWeapon(t *testing.T) {
	f, _ := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	for range 2 {
		if refusal := addToBag(t, f, f.charID, "fogo-alquimico"); refusal != "" {
			t.Fatalf("adicionar foi recusado: %q", refusal)
		}
	}
	bottle := bagLinesOf(t, f, f.charID, "Fogo alquímico")[0].ID

	if refusal := loadsDose(t, f, f.charID, bottle, weapon); refusal != "" {
		t.Fatalf("carregar foi recusado: %q", refusal)
	}
	expectDoses(t, f, weapon, "fogo-alquimico")
	lines := bagLinesOf(t, f, f.charID, "Fogo alquímico")
	if len(lines) != 1 || lines[0].Quantity != 1 {
		t.Fatalf("depois de carregar um de dois, a mochila ficou com %+v", lines)
	}

	if refusal := loadsDose(t, f, f.charID, bottle, weapon); refusal != "" {
		t.Fatalf("carregar o segundo foi recusado: %q", refusal)
	}
	expectDoses(t, f, weapon, "fogo-alquimico", "fogo-alquimico")
	if lines := bagLinesOf(t, f, f.charID, "Fogo alquímico"); len(lines) != 0 {
		t.Errorf("o último frasco foi carregado e continua na mochila: %+v", lines)
	}
}

func TestTheInjectionHoldsTwoDoses(t *testing.T) {
	f, _ := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	loaded(t, f, weapon, "agua-benta", "Água benta")

	bottle := itemSemeia(t, f, f.charID, "fogo-alquimico", "Fogo alquímico", "")
	if refusal := loadsDose(t, f, f.charID, bottle, weapon); !strings.Contains(refusal, "2 doses") {
		t.Errorf("a terceira dose devia ser recusada dizendo o limite, e veio %q", refusal)
	}
	expectDoses(t, f, weapon, "acido", "agua-benta")
	if lines := bagLinesOf(t, f, f.charID, "Fogo alquímico"); len(lines) != 1 {
		t.Errorf("a dose recusada saiu da mochila: %+v", lines)
	}
}

func TestEachOfTheFourItemsLoads(t *testing.T) {
	for catalog, name := range map[string]string{
		"acido": "Ácido", "fogo-alquimico": "Fogo alquímico", "bomba": "Bomba", "agua-benta": "Água benta",
	} {
		t.Run(catalog, func(t *testing.T) {
			f, _ := doseAttackOnTurn(t)
			weapon := weaponWithInjection(t, f)
			loaded(t, f, weapon, catalog, name)
			expectDoses(t, f, weapon, catalog)
		})
	}
}

func TestOnlyAChosenItemLoadsAndOnlyIntoTheInjection(t *testing.T) {
	f, _ := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)

	balm := itemSemeia(t, f, f.charID, "balsamo-restaurador", "Bálsamo restaurador", "")
	if refusal := loadsDose(t, f, f.charID, balm, weapon); !strings.Contains(refusal, "Bálsamo restaurador") {
		t.Errorf("o bálsamo não carrega, e a recusa veio %q", refusal)
	}

	plain := itemSemeia(t, f, f.charID, "espada-curta", "Espada curta", "")
	bottle := itemSemeia(t, f, f.charID, "fogo-alquimico", "Fogo alquímico", "")
	if refusal := loadsDose(t, f, f.charID, bottle, plain); !strings.Contains(refusal, "Injeção alquímica") {
		t.Errorf("a arma sem a melhoria não recebe dose, e a recusa veio %q", refusal)
	}

	other := seedCharacter(t, f.s, f.player, "Outro")
	foreign := itemSemeia(t, f, other, "espada-longa", "Espada longa", "")
	setImprovements(t, f, other, foreign, `{"item_improvements":["`+injectionID+`"]}`)
	if refusal := loadsDose(t, f, f.charID, bottle, foreign); refusal == "" {
		t.Error("carregou a arma de outra ficha")
	}
	expectDoses(t, f, foreign)
	expectDoses(t, f, weapon)
	expectDoses(t, f, plain)
}

// Decisão do dono: quem quiser gastar o frasco na interpretação remove da mochila.
func TestUsingADoseItemDoesNotSpendIt(t *testing.T) {
	f, _ := doseAttackOnTurn(t)
	bottle := itemSemeia(t, f, f.charID, "fogo-alquimico", "Fogo alquímico", "")

	target := fmt.Sprintf("/personagens/%d/itens/%d/usa?tab=bag", f.charID, bottle)
	refusal := bagRefusal(t, f.requests(t, f.player, http.MethodPost, target, "").Body.String())
	if !strings.Contains(refusal, "remova") {
		t.Errorf("a recusa devia dizer o que fazer, e veio %q", refusal)
	}
	if lines := bagLinesOf(t, f, f.charID, "Fogo alquímico"); len(lines) != 1 {
		t.Errorf("usar gastou o frasco: %+v", lines)
	}
}

func TestRemovingTheInjectionTakesTheDosesAlong(t *testing.T) {
	f, _ := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")

	setImprovements(t, f, f.charID, weapon, `{"item_improvements":["melhoria-certeira"]}`)
	expectDoses(t, f, weapon)
}

func TestAnotherImprovementKeepsTheDoses(t *testing.T) {
	f, _ := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")

	setImprovements(t, f, f.charID, weapon, `{"item_improvements":["`+injectionID+`","melhoria-certeira"]}`)
	expectDoses(t, f, weapon, "acido")
}

func TestAHitReleasesTheDoseAndAMissDoesNot(t *testing.T) {
	for ruling, spends := range map[string]bool{"acertou": true, "critico": true, "errou": false} {
		t.Run(ruling, func(t *testing.T) {
			f, goblin := doseAttackOnTurn(t)
			weapon := weaponWithInjection(t, f)
			loaded(t, f, weapon, "acido", "Ácido")

			if refusal := attacks(t, f, goblin, ""); refusal != "" {
				t.Fatalf("o golpe foi recusado: %q", refusal)
			}
			expectDoses(t, f, weapon, "acido")
			gmRules(t, f, ruling)
			if spends {
				expectDoses(t, f, weapon)
			} else {
				expectDoses(t, f, weapon, "acido")
			}
		})
	}
}

func TestCancellingAfterTheHitGivesTheDoseBack(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	attacks(t, f, goblin, "")
	gmRules(t, f, "acertou")
	expectDoses(t, f, weapon)

	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/cancelar", "")
	if pendingOf(t, f) != nil {
		t.Fatal("o ataque cancelado continuou na mesa")
	}
	expectDoses(t, f, weapon, "acido")
}

func TestCancellingBeforeTheHitDoesNotCreateADose(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	attacks(t, f, goblin, "")

	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/cancelar", "")
	expectDoses(t, f, weapon, "acido")
}

func TestClosingTheMissDoesNotCreateADose(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	attacks(t, f, goblin, "")
	gmRules(t, f, "errou")

	f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/ataque/cancelar", "")
	expectDoses(t, f, weapon, "acido")
}

func TestTheDoseDoesNotComeBackAfterTheCheck(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	attacks(t, f, goblin, "")
	gmRules(t, f, "acertou")
	f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/ataque/dano/na-mao", `{"damage_die_1":5}`)
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", ""); rec.Code != http.StatusOK {
		t.Fatalf("confirmar deu %d", rec.Code)
	}
	if pendingOf(t, f) != nil {
		t.Fatal("o ataque confirmado continuou na mesa")
	}
	expectDoses(t, f, weapon)
}

// Um golpe novo tira da mesa o acerto que ninguém confirmou, e ele não aconteceu.
func TestANewAttackOverAnUnconfirmedHitGivesTheDoseBack(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	attacks(t, f, goblin, "")
	gmRules(t, f, "acertou")
	expectDoses(t, f, weapon)

	if refusal := attacks(t, f, goblin, ""); refusal != "" {
		t.Fatalf("o golpe novo foi recusado: %q", refusal)
	}
	expectDoses(t, f, weapon, "acido")
	// O golpe novo leva a dose que voltou: a arma não ficou sem ela.
	if pa := pendingOf(t, f); pa == nil || pa.Dose == nil || pa.Dose.CatalogID != "acido" {
		t.Fatalf("o golpe novo devia levar o ácido que voltou, e levou %+v", pa)
	}
	gmRules(t, f, "acertou")
	expectDoses(t, f, weapon)
}

func TestTheInvestidaAlsoClearsTheChosenDose(t *testing.T) {
	f, _ := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	loaded(t, f, weapon, "fogo-alquimico", "Fogo alquímico")
	charge := findNodeWithText(actionsOf(t, f), "ds-turn-action", "Investida")
	if charge == nil {
		t.Fatal("a superfície não tem a linha da Investida")
	}
	if !strings.Contains(html.UnescapeString(renderNode(t, charge)), "$dose_choice = ''") {
		t.Error("a Investida leva a dose escolhida, e sem limpar a escolha o próximo golpe é recusado")
	}
}

func TestTheChargeReleasesTheChosenDose(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	loaded(t, f, weapon, "fogo-alquimico", "Fogo alquímico")

	path := fmt.Sprintf("/personagens/%d/acoes/investida/0?embutida=1", f.charID)
	signals := fmt.Sprintf(`{"turn_target":%q,"dose_weapon":0,"dose_choice":"fogo-alquimico"}`, goblin)
	if refusal := sceneRefusal(f.posts(t, f.player, path, signals)); refusal != "" {
		t.Fatalf("a investida foi recusada: %q", refusal)
	}
	gmRules(t, f, "acertou")
	expectDoses(t, f, weapon, "acido")
}

func TestASecondRulingDoesNotSpendAnotherDose(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	loaded(t, f, weapon, "acido", "Ácido")
	attacks(t, f, goblin, "")
	gmRules(t, f, "acertou")
	gmRules(t, f, "critico")
	expectDoses(t, f, weapon, "acido")
}

func TestCancellingAfterTheDamageWasRolledGivesTheDoseBack(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	attacks(t, f, goblin, "")
	gmRules(t, f, "acertou")
	f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/ataque/dano/na-mao", `{"damage_die_1":5}`)
	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/cancelar", "")
	expectDoses(t, f, weapon, "acido")
}

func TestRemovingTheWeaponTakesItsDosesAlong(t *testing.T) {
	f, _ := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")

	remove := fmt.Sprintf("/personagens/%d/itens/%d/remover?tab=bag", f.charID, weapon)
	if refusal := bagRefusal(t, f.requests(t, f.player, http.MethodPost, remove, "").Body.String()); refusal != "" {
		t.Fatalf("remover a arma foi recusado: %q", refusal)
	}
	expectDoses(t, f, weapon)
}

// O efeito da dose é dela, e os outros jogadores só leem quem ataca quem.
func TestOnlyTheAttackerAndTheGMReadTheDose(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	attacks(t, f, goblin, "")

	for who, sees := range map[int64]bool{f.gm: true, f.player: true} {
		body := f.requests(t, who, http.MethodGet, f.tableUrl(), "").Body.String()
		if strings.Contains(body, "2d4 de ácido") != sees {
			t.Errorf("o usuário %d ler a dose devia ser %v", who, sees)
		}
	}
	other := seedUser(t, f.s, "outro@t.com")
	character := seedCharacter(t, f.s, other, "Outro")
	seedMember(t, f.s, f.campaignID, character)
	if strings.Contains(f.requests(t, other, http.MethodGet, f.tableUrl(), "").Body.String(), "2d4 de ácido") {
		t.Error("outro jogador leu a dose de quem atacou")
	}
}

func TestCancellingWithTheWeaponGoneStillCancels(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	attacks(t, f, goblin, "")
	gmRules(t, f, "acertou")

	remove := fmt.Sprintf("/personagens/%d/itens/%d/remover?tab=bag", f.charID, weapon)
	if refusal := bagRefusal(t, f.requests(t, f.player, http.MethodPost, remove, "").Body.String()); refusal != "" {
		t.Fatalf("remover a arma foi recusado: %q", refusal)
	}
	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/cancelar", "")
	if pendingOf(t, f) != nil {
		t.Fatal("a arma sumiu, e o ataque não pôde mais ser cancelado")
	}
	expectDoses(t, f, weapon)
}

func TestTheAttackerChoosesWhichDoseLeaves(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	loaded(t, f, weapon, "fogo-alquimico", "Fogo alquímico")

	if refusal := attacks(t, f, goblin, `,"dose_weapon":0,"dose_choice":"fogo-alquimico"`); refusal != "" {
		t.Fatalf("o golpe foi recusado: %q", refusal)
	}
	gmRules(t, f, "acertou")
	expectDoses(t, f, weapon, "acido")
}

func TestWithoutAChoiceTheOldestDoseLeaves(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	loaded(t, f, weapon, "fogo-alquimico", "Fogo alquímico")

	attacks(t, f, goblin, "")
	gmRules(t, f, "acertou")
	expectDoses(t, f, weapon, "fogo-alquimico")
}

func TestAChoiceOfADoseThatIsNotLoadedIsRefused(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")

	refusal := attacks(t, f, goblin, `,"dose_weapon":0,"dose_choice":"fogo-alquimico"`)
	if !strings.Contains(refusal, "Espada longa") {
		t.Errorf("a recusa devia nomear a arma, e veio %q", refusal)
	}
	if pendingOf(t, f) != nil {
		t.Error("o golpe com dose que a arma não tem virou provisório")
	}
}

// A manobra não causa dano (p234): não há "ataque que acerte" para liberar a dose.
func TestTheManeuverLeavesTheDoseAlone(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")

	path := fmt.Sprintf("/personagens/%d/acoes/manobra/derrubar?embutida=1", f.charID)
	if refusal := sceneRefusal(f.posts(t, f.player, path, fmt.Sprintf(`{"turn_target":%q}`, goblin))); refusal != "" {
		t.Fatalf("a manobra foi recusada: %q", refusal)
	}
	if pa := pendingOf(t, f); pa == nil || pa.Dose != nil {
		t.Errorf("a manobra levou dose: %+v", pa)
	}
}

// O efeito vem da p155 e da p160, e a CD da p227: 10 + metade do nível + atributo.
// A ficha é de nível 8 e o atributo foi posto em 3, então a CD é 17.
func TestTheStripShowsWhatTheDoseDoes(t *testing.T) {
	cases := []struct {
		catalog, name string
		attribute     string
		want          []string
	}{
		{"acido", "Ácido", "dexterity", []string{"Ácido", "2d4 de ácido", "Reflexos CD 17 reduz à metade"}},
		{"fogo-alquimico", "Fogo alquímico", "dexterity", []string{"1d6 de fogo", "em chamas", "Reflexos CD 17 reduz à metade e evita as chamas"}},
		{"bomba", "Bomba", "dexterity", []string{"6d6 de impacto", "3m", "Reflexos CD 17 reduz à metade"}},
		{"agua-benta", "Água benta", "wisdom", []string{"2d10 de luz", "morto-vivo ou abissal", "Reflexos CD 17 reduz à metade"}},
	}
	for _, c := range cases {
		t.Run(c.catalog, func(t *testing.T) {
			f, goblin := doseAttackOnTurn(t)
			if _, err := f.s.db.ExecContext(context.Background(),
				"UPDATE characters SET "+c.attribute+" = 3 WHERE id = ?", f.charID); err != nil {
				t.Fatalf("pôr o atributo: %v", err)
			}
			weapon := weaponWithInjection(t, f)
			loaded(t, f, weapon, c.catalog, c.name)
			attacks(t, f, goblin, "")

			gm := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
			for _, want := range c.want {
				if !strings.Contains(gm, want) {
					t.Errorf("a faixa do mestre não diz %q", want)
				}
			}
		})
	}
}

func TestTheCriticalDoesNotMultiplyTheDose(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	attacks(t, f, goblin, "")
	gmRules(t, f, "critico")

	gm := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(gm, "2d4 de ácido") || !strings.Contains(gm, "não multiplica") {
		t.Errorf("a faixa devia dizer a dose sem multiplicar, e não diz")
	}
}

// Rolar de novo refaz o pedido, e a dose escolhida vai junto.
func TestTheReplayKeepsTheChosenDose(t *testing.T) {
	pending := live.PendingAttack{Dose: &live.AttackDose{ItemID: 7, CatalogID: "acido"}}
	if got := combat.ReplayRequest(1, 1, pending, true, nil).Dose; got != "acido" {
		t.Errorf("o pedido refeito levou a dose %q", got)
	}
}

func TestTheCopyAtTheTableKeepsTheDoses(t *testing.T) {
	f, _ := doseAttackOnTurn(t)
	ctx := context.Background()
	template := seedCharacter(t, f.s, f.gm, "Molde")
	weapon := itemSemeia(t, f, template, "espada-longa", "Espada longa", "wielded")
	if _, err := f.s.db.ExecContext(ctx, "INSERT INTO character_item_doses (itemId, catalogId) VALUES (?, 'acido')", weapon); err != nil {
		t.Fatalf("semear a dose: %v", err)
	}
	if err := f.s.campaignSeating().Seat(ctx, f.gm, f.campaignID, template, ""); err != nil {
		t.Fatalf("sentar à mesa: %v", err)
	}
	var doses int
	if err := f.s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM character_item_doses d
		JOIN character_items i ON i.id = d.itemId
		JOIN characters c ON c.id = i.characterId
		WHERE c.sourceCharacterId = ? AND d.catalogId = 'acido'`, template).Scan(&doses); err != nil {
		t.Fatalf("contar as doses da cópia: %v", err)
	}
	if doses != 1 {
		t.Errorf("a cópia à mesa ficou com %d doses, e o molde tem 1", doses)
	}
}

// findNodeWithText acha o primeiro nó de uma classe que contém o texto.
func findNodeWithText(n *html.Node, token, text string) *html.Node {
	for _, node := range nodesWithClassToken(n, token) {
		if strings.Contains(textOf(node), text) {
			return node
		}
	}
	return nil
}

// actionsOf é o recorte das Ações na mesa do jogador.
func actionsOf(t *testing.T, f sceneFixture) *html.Node {
	t.Helper()
	page := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	actions := findNode(doc, "id", "actions-scene")
	if actions == nil {
		t.Fatal("a sessão do jogador não desenhou as Ações")
	}
	return actions
}

func TestTheAttackRowAsksWhichDoseOnlyWhenTheyDiffer(t *testing.T) {
	f, _ := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	if findNode(actionsOf(t, f), "aria-label", "Dose do ataque") != nil {
		t.Error("uma dose só não tem o que perguntar")
	}
	if !strings.Contains(textOf(actionsOf(t, f)), "Doses: Ácido") {
		t.Error("a linha da arma não diz o que está carregado")
	}

	loaded(t, f, weapon, "acido", "Ácido")
	if findNode(actionsOf(t, f), "aria-label", "Dose do ataque") != nil {
		t.Error("duas doses iguais não têm o que perguntar")
	}

	f2, _ := doseAttackOnTurn(t)
	other := weaponWithInjection(t, f2)
	loaded(t, f2, other, "acido", "Ácido")
	loaded(t, f2, other, "fogo-alquimico", "Fogo alquímico")
	strip := findNode(actionsOf(t, f2), "aria-label", "Dose do ataque")
	if strip == nil {
		t.Fatal("duas doses diferentes pedem a pergunta")
	}
	surface := html.UnescapeString(renderNode(t, strip))
	for _, want := range []string{"Ácido", "Fogo alquímico", "$dose_choice"} {
		if !strings.Contains(surface, want) {
			t.Errorf("a pergunta não tem %q", want)
		}
	}
}

func doseBagScreen(t *testing.T, f sceneFixture) string {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d?tab=bag", f.charID)
	return html.UnescapeString(f.requests(t, f.player, http.MethodGet, target, "").Body.String())
}

func TestTheBagOffersLoadingOnlyWhileAWeaponHasRoom(t *testing.T) {
	f, _ := doseAttackOnTurn(t)
	bottle := itemSemeia(t, f, f.charID, "fogo-alquimico", "Fogo alquímico", "")
	address := fmt.Sprintf("/itens/%d/carrega/", bottle)

	if screen := doseBagScreen(t, f); strings.Contains(screen, address) ||
		strings.Contains(screen, `aria-label="Usar Fogo alquímico"`) || !strings.Contains(screen, "remova da mochila") {
		t.Error("sem arma com a melhoria, o frasco não tem Usar, e o diálogo diz o que fazer")
	}

	weapon := weaponWithInjection(t, f)
	screen := doseBagScreen(t, f)
	if !strings.Contains(screen, fmt.Sprintf("%s%d", address, weapon)) || !strings.Contains(screen, "Espada longa · 0 de 2") {
		t.Error("a arma com espaço devia aparecer para receber o frasco")
	}
	if !strings.Contains(screen, `aria-label="Usar Fogo alquímico"`) {
		t.Error("com arma que cabe, o frasco tem Usar")
	}

	loaded(t, f, weapon, "acido", "Ácido")
	loaded(t, f, weapon, "acido", "Ácido")
	if strings.Contains(doseBagScreen(t, f), fmt.Sprintf("%s%d", address, weapon)) {
		t.Error("a arma cheia não recebe mais nada")
	}
}

func TestTheWeaponSheetListsItsDoses(t *testing.T) {
	f, _ := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	if !strings.Contains(doseBagScreen(t, f), "Nenhuma dose carregada") {
		t.Error("a arma vazia devia dizer que não tem dose")
	}
	loaded(t, f, weapon, "acido", "Ácido")
	screen := doseBagScreen(t, f)
	for _, want := range []string{"Doses da Injeção alquímica", "1 de 2", "2d4 de ácido", "guarda 2 doses"} {
		if !strings.Contains(screen, want) {
			t.Errorf("o diálogo da arma não diz %q", want)
		}
	}
}

func TestAChoiceOnAnotherWeaponDoesNotTravel(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")

	if refusal := attacks(t, f, goblin, `,"dose_weapon":1,"dose_choice":"fogo-alquimico"`); refusal != "" {
		t.Fatalf("a escolha feita na outra arma recusou este golpe: %q", refusal)
	}
	if pa := pendingOf(t, f); pa == nil || pa.Dose == nil || pa.Dose.CatalogID != "acido" {
		t.Errorf("o golpe devia levar o ácido desta arma, e levou %+v", pa)
	}
}

// A dose muda a ficha de quem atacou, e a superfície Ações dele é desenhada com ela.
// Sem o aviso, a linha da arma segue mostrando a dose que já saiu.
func TestMovingTheDoseWakesTheAttackersSheet(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	attacks(t, f, goblin, "")

	notice, stop := f.s.bus.Subscribe(events.OfCharacter(f.charID))
	defer stop()
	expectNotice := func(gesture string) {
		t.Helper()
		select {
		case ev := <-notice.C:
			if _, ok := ev.(events.CharacterChanged); !ok {
				t.Fatalf("%s: chegou %T, e quem escuta a ficha espera um CharacterChanged", gesture, ev)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s mexeu na dose e não avisou a ficha", gesture)
		}
	}
	// A mesa do jogador repede a ficha quando o carimbo dela muda (`sheetVersion`).
	const old = "2000-01-01T00:00:00.000Z"
	stamped := func(gesture string) {
		t.Helper()
		var stamp string
		if err := f.s.db.QueryRowContext(context.Background(),
			"SELECT updatedAt FROM characters WHERE id = ?", f.charID).Scan(&stamp); err != nil {
			t.Fatalf("ler o carimbo: %v", err)
		}
		if stamp == old {
			t.Errorf("%s mexeu na dose e não carimbou a ficha", gesture)
		}
		if _, err := f.s.db.ExecContext(context.Background(),
			"UPDATE characters SET updatedAt = ? WHERE id = ?", old, f.charID); err != nil {
			t.Fatalf("envelhecer o carimbo: %v", err)
		}
	}
	if _, err := f.s.db.ExecContext(context.Background(),
		"UPDATE characters SET updatedAt = ? WHERE id = ?", old, f.charID); err != nil {
		t.Fatalf("envelhecer o carimbo: %v", err)
	}
	gmRules(t, f, "acertou")
	expectNotice("o acerto")
	stamped("o acerto")
	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/cancelar", "")
	expectNotice("o cancelamento")
	stamped("o cancelamento")
}

// O aviso do stream repede a ficha embutida, e as Ações são um nó irmão dela: sem vir
// junto, a linha da arma mostra a dose que o mestre acabou de gastar.
func TestTheEmbeddedRefetchBringsTheActionsAlong(t *testing.T) {
	f, _ := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")

	target := fmt.Sprintf("/personagens/%d?tab=expertises&embutida=1", f.charID)
	body := f.requests(t, f.player, http.MethodGet, target, "").Body.String()
	if !strings.Contains(body, `id="actions-scene"`) || !strings.Contains(body, "Doses: Ácido") {
		t.Error("repedir a ficha embutida não trouxe as Ações")
	}
}

// Decisão do dono: RPG é jogo de turno, e quem propôs um ataque não faz outra coisa até
// ele acabar. É também o que impede a dose devolvida de encontrar a arma cheia.
func TestLoadingWaitsForTheAttackOnTheTable(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	bottle := itemSemeia(t, f, f.charID, "fogo-alquimico", "Fogo alquímico", "")

	expectRefused := func(moment string) {
		t.Helper()
		if refusal := loadsDose(t, f, f.charID, bottle, weapon); !strings.Contains(refusal, "ataque") {
			t.Errorf("%s, carregar devia ser recusado dizendo por quê, e veio %q", moment, refusal)
		}
		if lines := bagLinesOf(t, f, f.charID, "Fogo alquímico"); len(lines) != 1 {
			t.Fatalf("%s, o item recusado saiu da mochila: %+v", moment, lines)
		}
	}

	attacks(t, f, goblin, "")
	expectRefused("com o ataque esperando a decisão")
	gmRules(t, f, "acertou")
	expectRefused("com o ataque esperando o dano")

	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/cancelar", "")
	if refusal := loadsDose(t, f, f.charID, bottle, weapon); refusal != "" {
		t.Errorf("sem ataque na mesa, carregar foi recusado: %q", refusal)
	}
	expectDoses(t, f, weapon, "acido", "fogo-alquimico")
}

// O erro encerra o golpe, e o provisório só fica para o jogador ler a decisão.
func TestLoadingIsFreeAfterTheMiss(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	weapon := weaponWithInjection(t, f)
	bottle := itemSemeia(t, f, f.charID, "fogo-alquimico", "Fogo alquímico", "")
	attacks(t, f, goblin, "")
	gmRules(t, f, "errou")

	if refusal := loadsDose(t, f, f.charID, bottle, weapon); refusal != "" {
		t.Errorf("depois do erro, carregar foi recusado: %q", refusal)
	}
}

// O ataque de outra ficha na mesma mesa não prende esta.
func TestAnotherCharactersAttackDoesNotHoldTheLoad(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	attacks(t, f, goblin, "")
	other := seedCharacter(t, f.s, f.player, "Outro")
	seedMember(t, f.s, f.campaignID, other)
	weapon := itemSemeia(t, f, other, "espada-longa", "Espada longa", "")
	setImprovements(t, f, other, weapon, `{"item_improvements":["`+injectionID+`"]}`)
	bottle := itemSemeia(t, f, other, "fogo-alquimico", "Fogo alquímico", "")

	if refusal := loadsDose(t, f, other, bottle, weapon); refusal != "" {
		t.Errorf("o ataque de outra ficha recusou esta: %q", refusal)
	}
}
