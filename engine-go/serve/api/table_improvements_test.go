package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/sheet"
	"t20engine/infra/events"
)

// A MELHORIA DA MESA: o mestre a cria, e o jogador a coloca no item, na mesma lista das do livro.

func improvementsURL(f sceneFixture, path string) string {
	return fmt.Sprintf("/campanhas/%d/melhorias-da-mesa%s", f.campaignID, path)
}

// improvementSignals são os sinais do editor para uma melhoria que cabe em qualquer família pedida.
func improvementSignals(name, family, target, skill string, amount int) string {
	return fmt.Sprintf(`{"table_improvement_name":%q,"table_improvement_scope":"mesa","table_improvement_family":%q,
		"table_improvement_target":%q,"table_improvement_skill":%q,"table_improvement_amount":%d}`, name, family, target, skill, amount)
}

func createsImprovement(t *testing.T, f sceneFixture, signals string) string {
	t.Helper()
	rec := f.requests(t, f.gm, http.MethodPost, improvementsURL(f, "/gravar"), signals)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), `role="alert"`) {
		t.Fatalf("gravar a melhoria deu %d: %s", rec.Code, rec.Body.String())
	}
	var id string
	if err := f.s.db.QueryRowContext(context.Background(),
		"SELECT id FROM campaign_improvements WHERE campaignId = ? ORDER BY id DESC LIMIT 1", f.campaignID).Scan(&id); err != nil {
		t.Fatalf("ler a melhoria: %v", err)
	}
	return id
}

// placesImprovements é o gesto do jogador: a lista inteira de melhorias do item, no diálogo dele.
func placesImprovements(t *testing.T, f sceneFixture, who, id, item int64, ids ...string) string {
	t.Helper()
	list := "[]"
	if len(ids) > 0 {
		list = `["` + strings.Join(ids, `","`) + `"]`
	}
	rec := f.requests(t, who, http.MethodPost, fmt.Sprintf("/personagens/%d/itens/%d/melhorias", id, item),
		fmt.Sprintf(`{"item_improvements":%s,"item_material":""}`, list))
	return rec.Body.String()
}

func bagPage(t *testing.T, f sceneFixture, who, id int64) string {
	t.Helper()
	return f.requests(t, who, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=bag", id), "").Body.String()
}

// improvementDialog é o diálogo de melhorias daquele item: a página traz o de cada item.
func improvementDialog(t *testing.T, f sceneFixture, id, item int64) string {
	t.Helper()
	page := bagPage(t, f, f.player, id)
	start := strings.Index(page, fmt.Sprintf(`id="item-improvements-%d"`, item))
	if start < 0 {
		t.Fatalf("a página não traz as melhorias do item %d", item)
	}
	end := strings.Index(page[start:], "</dialog>")
	return page[start : start+end]
}

func damageByWeapon(t *testing.T, f sceneFixture, id int64) map[string]int {
	t.Helper()
	dto, ec := loadedCharacter(t, f, id)
	out := map[string]int{}
	for _, card := range dto.Ruleset.ComputeWeaponCards(ec, sheet.ToStringSet(dto.Conditionals)) {
		out[card.Name] = card.DamageAll.Total + card.DamageOwn.Total
	}
	return out
}

func improvementsOnItem(t *testing.T, f sceneFixture, item int64) string {
	t.Helper()
	var stored string
	if err := f.s.db.QueryRowContext(context.Background(), "SELECT improvements FROM character_items WHERE id = ?", item).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	return stored
}

func improvementCount(t *testing.T, f sceneFixture) (total int) {
	t.Helper()
	if err := f.s.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM campaign_improvements WHERE campaignId = ?", f.campaignID).Scan(&total); err != nil {
		t.Fatal(err)
	}
	return total
}

// O caso que abriu a fatia: um medalhão com +1 em Luta. Na mochila não conta, e em uso conta.
func TestAnImprovementOnTheItemCountsWhileItIsInUse(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	before := lutaOf(t, f.s, id)
	item := itemSemeia(t, f, id, "medalhao-de-prata", "Medalhão de prata", "")
	improvement := createsImprovement(t, f, improvementSignals("Insígnia de Luta", "apparel", "expertise", "Luta", 1))

	placesImprovements(t, f, f.player, id, item, improvement)
	if !strings.Contains(improvementsOnItem(t, f, item), improvement) {
		t.Fatalf("a melhoria não ficou no item: %s", improvementsOnItem(t, f, item))
	}
	if got := lutaOf(t, f.s, id); got != before {
		t.Errorf("com o item na mochila, Luta foi de %+d a %+d", before, got)
	}
	for _, place := range []string{"wielded", "vested"} {
		if _, err := f.s.db.ExecContext(context.Background(), "UPDATE character_items SET equipped = ? WHERE id = ?", place, item); err != nil {
			t.Fatal(err)
		}
		if got := lutaOf(t, f.s, id); got != before+1 {
			t.Errorf("com o item em %s, Luta foi de %+d a %+d, e a melhoria dá +1", place, before, got)
		}
	}
	placesImprovements(t, f, f.player, id, item)
	if got := lutaOf(t, f.s, id); got != before {
		t.Errorf("tirada a melhoria, Luta ficou em %+d, e era %+d", got, before)
	}
}

// Cada alvo chega ao número dele, com o item em uso.
func TestAnImprovementAddsToEveryTarget(t *testing.T) {
	cases := []struct {
		name, target string
		read         func(t *testing.T, f sceneFixture, id int64) int
	}{
		{"Defesa", "defense", func(t *testing.T, f sceneFixture, id int64) int { return ruleSheet(t, f, id).Defense.Total }},
		{"PV", "maxPv", func(t *testing.T, f sceneFixture, id int64) int { return int(poolsOf(t, f.s, id).HpMax) }},
		{"PM", "maxPm", func(t *testing.T, f sceneFixture, id int64) int { return int(poolsOf(t, f.s, id).MpMax) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, id := fighterFixture(t)
			inCampaign(t, f, id)
			item := itemSemeia(t, f, id, "algemas", "Algemas", "vested")
			before := c.read(t, f, id)
			improvement := createsImprovement(t, f, improvementSignals("Forte", "", c.target, "", 3))
			placesImprovements(t, f, f.player, id, item, improvement)
			if got := c.read(t, f, id); got != before+3 {
				t.Errorf("foi de %d a %d, e a melhoria dá +3", before, got)
			}
			if _, err := f.s.db.ExecContext(context.Background(), "UPDATE character_items SET equipped = NULL WHERE id = ?", item); err != nil {
				t.Fatal(err)
			}
			if got := c.read(t, f, id); got != before {
				t.Errorf("com o item na mochila, foi de %d a %d, e ela só vale em uso", before, got)
			}
		})
	}
}

// O número de golpe é da arma em que a melhoria está: a outra mão não sobe.
func TestAnImprovementOnAWeaponStaysOnIt(t *testing.T) {
	cases := []struct {
		name, target string
		read         func(t *testing.T, f sceneFixture, id int64) map[string]int
		adds         int
	}{
		{"ataque", "attack", attackByWeapon, 2},
		{"dano", "damage", func(t *testing.T, f sceneFixture, id int64) map[string]int { return damageByWeapon(t, f, id) }, 2},
		{"margem", "critRange", marginByWeapon, -2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, id := fighterFixture(t)
			inCampaign(t, f, id)
			saveHand(t, f, id, `["armas-simples","armas-marciais"]`)
			sword := itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
			itemSemeia(t, f, id, "adaga", "Adaga", "wielded2")
			before := c.read(t, f, id)
			improvement := createsImprovement(t, f, improvementSignals("Da casa", "weapon", c.target, "", 2))
			placesImprovements(t, f, f.player, id, sword, improvement)
			after := c.read(t, f, id)
			if after["Espada longa"] != before["Espada longa"]+c.adds {
				t.Errorf("a espada foi de %d a %d, e devia andar %d", before["Espada longa"], after["Espada longa"], c.adds)
			}
			if after["Adaga"] != before["Adaga"] {
				t.Errorf("a adaga foi de %d a %d, e a melhoria está na espada", before["Adaga"], after["Adaga"])
			}
		})
	}
}

// A lista do item oferece a melhoria que cabe na família dele, e o servidor recusa a que não cabe.
func TestTheItemOffersOnlyTheImprovementThatFits(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	medal := itemSemeia(t, f, id, "medalhao-de-prata", "Medalhão de prata", "vested")
	weaponOnly := createsImprovement(t, f, improvementSignals("Fio da Casa", "weapon", "damage", "", 1))
	createsImprovement(t, f, improvementSignals("Insígnia", "apparel", "expertise", "Luta", 1))

	dialog := improvementDialog(t, f, id, medal)
	if !strings.Contains(dialog, "Insígnia") {
		t.Error("a lista do medalhão não oferece a melhoria de vestuário")
	}
	if strings.Contains(dialog, "Fio da Casa") {
		t.Error("a lista do medalhão oferece a melhoria de arma")
	}
	placesImprovements(t, f, f.player, id, medal, weaponOnly)
	if strings.Contains(improvementsOnItem(t, f, medal), weaponOnly) {
		t.Error("o servidor gravou a melhoria de arma no medalhão")
	}
}

// "Um personagem" é só dele: os outros não a veem, e o servidor recusa o pedido montado à mão.
func TestOneCharacterImprovementIsOnlyTheirs(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	item := itemSemeia(t, f, id, "medalhao-de-prata", "Medalhão de prata", "vested")
	if _, err := f.s.db.ExecContext(context.Background(), "UPDATE characters SET campaignId = ? WHERE id = ?", f.campaignID, f.charID); err != nil {
		t.Fatal(err)
	}
	signals := strings.Replace(improvementSignals("Só do outro", "apparel", "expertise", "Luta", 1),
		`"table_improvement_scope":"mesa"`, fmt.Sprintf(`"table_improvement_scope":"um","table_improvement_character":"%d"`, f.charID), 1)
	improvement := createsImprovement(t, f, signals)
	if strings.Contains(bagPage(t, f, f.player, id), "Só do outro") {
		t.Error("o outro personagem vê a melhoria que é só de um")
	}
	placesImprovements(t, f, f.player, id, item, improvement)
	if strings.Contains(improvementsOnItem(t, f, item), improvement) {
		t.Error("o servidor gravou a melhoria no personagem errado")
	}
}

// A melhoria de outra mesa não entra pelo pedido montado à mão, e fora da mesa ela soma zero.
func TestAnImprovementStaysInItsCampaign(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	before := lutaOf(t, f.s, id)
	item := itemSemeia(t, f, id, "medalhao-de-prata", "Medalhão de prata", "vested")
	improvement := createsImprovement(t, f, improvementSignals("Insígnia", "apparel", "expertise", "Luta", 1))
	placesImprovements(t, f, f.player, id, item, improvement)
	if got := lutaOf(t, f.s, id); got != before+1 {
		t.Fatalf("o controle: na mesa dela, Luta sobe 1, e foi de %+d a %+d", before, got)
	}
	other := seedCampaign(t, f.s, f.gm)
	if _, err := f.s.db.ExecContext(context.Background(), "UPDATE campaign_improvements SET campaignId = ? WHERE id = ?", other, improvement); err != nil {
		t.Fatal(err)
	}
	if got := lutaOf(t, f.s, id); got != before {
		t.Errorf("a melhoria de outra mesa somou nesta ficha: %+d, e era %+d", got, before)
	}
	placesImprovements(t, f, f.player, id, item)
	placesImprovements(t, f, f.player, id, item, improvement)
	if strings.Contains(improvementsOnItem(t, f, item), improvement) {
		t.Error("o servidor gravou a melhoria de outra mesa")
	}
}

// Mudar quem pode usar não prende a melhoria que já está no item: ela segue na lista, para sair.
func TestAnImprovementAlreadyOnTheItemCanStillComeOff(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	item := itemSemeia(t, f, id, "medalhao-de-prata", "Medalhão de prata", "vested")
	improvement := createsImprovement(t, f, improvementSignals("Insígnia", "apparel", "expertise", "Luta", 1))
	placesImprovements(t, f, f.player, id, item, improvement)
	if _, err := f.s.db.ExecContext(context.Background(), "UPDATE campaign_improvements SET characterId = ? WHERE id = ?", f.charID, improvement); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(improvementDialog(t, f, id, item), "Insígnia") {
		t.Error("a melhoria que está no item sumiu da lista, e ninguém consegue tirá-la")
	}
	// O gesto regrava a lista inteira: ela não pode travar a outra melhoria do mesmo item.
	placesImprovements(t, f, f.player, id, item, improvement, "melhoria-discreto")
	if stored := improvementsOnItem(t, f, item); !strings.Contains(stored, "melhoria-discreto") {
		t.Errorf("a melhoria que já estava no item travou a outra: %s", stored)
	}
	placesImprovements(t, f, f.player, id, item)
	if strings.Contains(improvementsOnItem(t, f, item), improvement) {
		t.Error("não deu para tirar a melhoria")
	}
}

func TestOnlyTheGMWritesImprovements(t *testing.T) {
	f := newSceneFixture(t)
	signals := improvementSignals("Insígnia", "apparel", "expertise", "Luta", 1)
	for _, path := range []string{"/gravar", "/rascunho", "/editor", "/qualquer/excluir"} {
		if rec := f.requests(t, f.player, http.MethodPost, improvementsURL(f, path), signals); rec.Code != http.StatusForbidden {
			t.Errorf("%s respondeu %d ao jogador", path, rec.Code)
		}
	}
	if total := improvementCount(t, f); total != 0 {
		t.Errorf("o jogador gravou %d melhorias", total)
	}
}

// O editor de outra mesa não alcança esta melhoria.
func TestAnImprovementOfAnotherCampaignIsOutOfReach(t *testing.T) {
	f := newSceneFixture(t)
	improvement := createsImprovement(t, f, improvementSignals("Insígnia", "apparel", "expertise", "Luta", 1))
	other := seedCampaign(t, f.s, f.gm)
	f.requests(t, f.gm, http.MethodPost, fmt.Sprintf("/campanhas/%d/melhorias-da-mesa/%s/excluir", other, improvement), "")
	stolen := strings.Replace(improvementSignals("Roubada", "apparel", "expertise", "Luta", 9), `{`, fmt.Sprintf(`{"table_improvement_id":%q,`, improvement), 1)
	f.requests(t, f.gm, http.MethodPost, fmt.Sprintf("/campanhas/%d/melhorias-da-mesa/gravar", other), stolen)
	if total := improvementCount(t, f); total != 1 {
		t.Errorf("o endereço de outra campanha apagou a melhoria: %d", total)
	}
	if strings.Contains(rulesTab(t, f, f.gm), "Roubada") {
		t.Error("o endereço de outra campanha reescreveu a melhoria")
	}
}

func TestTheImprovementEditorRefusesWhatIsWrong(t *testing.T) {
	cases := []struct{ name, signals, says string }{
		{"sem nome", improvementSignals(" ", "apparel", "expertise", "Luta", 1), "nome"},
		{"soma zero", improvementSignals("A", "apparel", "expertise", "Luta", 0), "zero"},
		{"perícia que não existe", improvementSignals("A", "apparel", "expertise", "Sorte", 1), "Sorte"},
		{"família que não existe", improvementSignals("A", "carroça", "expertise", "Luta", 1), "carroça"},
		{"alvo que não existe", improvementSignals("A", "apparel", "sorte", "", 1), "sorte"},
		{"golpe fora de arma", improvementSignals("A", "apparel", "attack", "", 1), "arma"},
		{"golpe em qualquer item", improvementSignals("A", "", "damage", "", 1), "arma"},
		{"sem escolher o personagem", strings.Replace(improvementSignals("A", "apparel", "expertise", "Luta", 1), `"table_improvement_scope":"mesa"`, `"table_improvement_scope":"um"`, 1), "Escolha o personagem"},
		{"personagem de fora da mesa", strings.Replace(improvementSignals("A", "apparel", "expertise", "Luta", 1), `"table_improvement_scope":"mesa"`, `"table_improvement_scope":"um","table_improvement_character":"99999"`, 1), "99999"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newSceneFixture(t)
			body := f.requests(t, f.gm, http.MethodPost, improvementsURL(f, "/gravar"), c.signals).Body.String()
			if !strings.Contains(body, c.says) || !strings.Contains(body, `role="alert"`) {
				t.Errorf("a recusa devia dizer %q num alerta, e veio %q", c.says, body[:min(300, len(body))])
			}
			if total := improvementCount(t, f); total != 0 {
				t.Errorf("a recusa gravou %d melhorias", total)
			}
		})
	}
}

// Excluir tira a melhoria de todo item da mesa, e o diálogo diz de quantos antes.
func TestRemovingAnImprovementTakesItOffEveryItem(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	before := lutaOf(t, f.s, id)
	item := itemSemeia(t, f, id, "medalhao-de-prata", "Medalhão de prata", "vested")
	improvement := createsImprovement(t, f, improvementSignals("Insígnia", "apparel", "expertise", "Luta", 1))
	keeps := createsImprovement(t, f, improvementSignals("Outra", "apparel", "expertise", "Luta", 1))
	placesImprovements(t, f, f.player, id, item, improvement, keeps)

	body := f.requests(t, f.gm, http.MethodPost, improvementsURL(f, "/editor?melhoria="+improvement), "").Body.String()
	if !strings.Contains(body, "em 1 item") {
		t.Errorf("a confirmação devia dizer em quantos itens ela está: %q", body[:min(400, len(body))])
	}
	f.requests(t, f.gm, http.MethodPost, improvementsURL(f, "/"+improvement+"/excluir"), "")
	stored := improvementsOnItem(t, f, item)
	if strings.Contains(stored, improvement) {
		t.Errorf("a melhoria excluída ficou no item: %s", stored)
	}
	if !strings.Contains(stored, keeps) {
		t.Errorf("excluir levou a outra melhoria junto: %s", stored)
	}
	if got := lutaOf(t, f.s, id); got != before+1 {
		t.Errorf("sobra uma melhoria de +1: Luta está em %+d, e era %+d", got, before)
	}
	if total := improvementCount(t, f); total != 1 {
		t.Errorf("restam %d melhorias, e devia restar 1", total)
	}
}

// Editar ou excluir muda a conta de quem a usa, e a ficha aberta na mesa acompanha. Criar não muda nada.
func TestAChangedImprovementWakesTheSheets(t *testing.T) {
	f := newSceneFixture(t)
	subscription, stop := f.s.bus.Subscribe(events.OfCharacter(f.charID))
	defer stop()
	woke := func() bool {
		select {
		case <-subscription.C:
			return true
		default:
			return false
		}
	}
	improvement := createsImprovement(t, f, improvementSignals("Insígnia", "apparel", "expertise", "Luta", 1))
	if woke() {
		t.Error("criar não muda ficha nenhuma, e acordou a ficha")
	}
	edited := strings.Replace(improvementSignals("Insígnia", "apparel", "expertise", "Luta", 2), `{`, fmt.Sprintf(`{"table_improvement_id":%q,`, improvement), 1)
	createsImprovement(t, f, edited)
	if !woke() {
		t.Error("editar não avisou a ficha aberta")
	}
	f.requests(t, f.gm, http.MethodPost, improvementsURL(f, "/"+improvement+"/excluir"), "")
	if !woke() {
		t.Error("excluir não avisou a ficha aberta")
	}
}

// A aba mostra as melhorias: ao mestre, todas e como editar; ao jogador, as da mesa e as dele.
func TestTheTabListsTheImprovements(t *testing.T) {
	f := newSceneFixture(t)
	someoneElse := seedCharacterAtLevel(t, f.s, f.gm, "Heroína do mestre", "Guerreiro", 3, 0, 0)
	if _, err := f.s.db.ExecContext(context.Background(), "INSERT INTO campaign_members (campaignId, characterId, addedAt) VALUES (?, ?, '2026-11-05T00:00:00.000Z')", f.campaignID, someoneElse); err != nil {
		t.Fatal(err)
	}
	createsImprovement(t, f, improvementSignals("Da mesa", "apparel", "expertise", "Luta", 1))
	createsImprovement(t, f, strings.Replace(improvementSignals("Alheia", "apparel", "expertise", "Luta", 1),
		`"table_improvement_scope":"mesa"`, fmt.Sprintf(`"table_improvement_scope":"um","table_improvement_character":"%d"`, someoneElse), 1))

	gm := rulesTab(t, f, f.gm)
	for _, want := range []string{"Da mesa", "Alheia", "+1 em Luta", "Nova melhoria"} {
		if !strings.Contains(gm, want) {
			t.Errorf("o mestre não lê %q", want)
		}
	}
	player := rulesTab(t, f, f.player)
	if !strings.Contains(player, "Da mesa") {
		t.Error("o jogador não lê a melhoria da mesa")
	}
	for _, hidden := range []string{"Alheia", "Nova melhoria", "table_improvement_name"} {
		if strings.Contains(player, hidden) {
			t.Errorf("o jogador recebe %q", hidden)
		}
	}
}

// A prévia diz onde a melhoria cabe, o que ela soma e quem a vê.
func TestTheImprovementPreviewSaysWhatItDoes(t *testing.T) {
	f := newSceneFixture(t)
	body := f.requests(t, f.gm, http.MethodPost, improvementsURL(f, "/rascunho"), improvementSignals("A", "apparel", "expertise", "Luta", 1)).Body.String()
	for _, want := range []string{"vestuário", "+1 em Luta", "enquanto o item está em uso", "Todos da mesa"} {
		if !strings.Contains(body, want) {
			t.Errorf("a prévia não diz %q", want)
		}
	}
}

// Quem abre a conta sabe de onde veio o número.
func TestTheSheetNamesTheImprovementInItsSum(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	saveHand(t, f, id, `["armas-simples","armas-marciais"]`)
	sword := itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	placesImprovements(t, f, f.player, id, sword, createsImprovement(t, f, improvementSignals("Fio da Casa", "weapon", "attack", "", 1)))
	page := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=combat", id), "").Body.String()
	if !strings.Contains(page, "Fio da Casa") {
		t.Error("a aba Combate não diz o nome da melhoria")
	}
	if bag := bagPage(t, f, f.player, id); !strings.Contains(bag, "Fio da Casa") {
		t.Error("a mochila não mostra a melhoria no item")
	}
}

// Duas melhorias da mesa diferentes somam, no PV como na Defesa. A mesma, repetida, conta uma vez.
func TestTwoTableImprovementsAddUpAndTheSameOneCountsOnce(t *testing.T) {
	reads := map[string]func(t *testing.T, f sceneFixture, id int64) int{
		"defense": func(t *testing.T, f sceneFixture, id int64) int { return ruleSheet(t, f, id).Defense.Total },
		"maxPv":   func(t *testing.T, f sceneFixture, id int64) int { return int(poolsOf(t, f.s, id).HpMax) },
	}
	for target, read := range reads {
		t.Run(target, func(t *testing.T) {
			f, id := fighterFixture(t)
			inCampaign(t, f, id)
			cloak := itemSemeia(t, f, id, "algemas", "Algemas", "vested")
			ring := itemSemeia(t, f, id, "anel-da-liberdade", "Anel da liberdade", "vested")
			before := read(t, f, id)
			three := createsImprovement(t, f, improvementSignals("Três", "", target, "", 3))
			minusFour := createsImprovement(t, f, improvementSignals("Menos quatro", "", target, "", -4))
			placesImprovements(t, f, f.player, id, cloak, three, minusFour)
			if got := read(t, f, id); got != before-1 {
				t.Errorf("+3 e -4 no mesmo item foram de %d a %d, e somam -1", before, got)
			}
			placesImprovements(t, f, f.player, id, cloak, three)
			placesImprovements(t, f, f.player, id, ring, three)
			if got := read(t, f, id); got != before+3 {
				t.Errorf("a mesma melhoria em dois itens foi de %d a %d, e conta uma vez: +3", before, got)
			}
		})
	}
}

// Duas adagas na mão dividem o escopo da arma. A mesma melhoria nas duas dá +2 em cada, e não +4.
func TestTheSameImprovementOnTwoEqualWeaponsCountsOnce(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	saveHand(t, f, id, `["armas-simples","armas-marciais"]`)
	first := itemSemeia(t, f, id, "adaga", "Adaga", "wielded")
	second := itemSemeia(t, f, id, "adaga", "Adaga", "wielded2")
	before := attackByWeapon(t, f, id)["Adaga"]
	improvement := createsImprovement(t, f, improvementSignals("Da casa", "weapon", "attack", "", 2))
	placesImprovements(t, f, f.player, id, first, improvement)
	placesImprovements(t, f, f.player, id, second, improvement)
	if got := attackByWeapon(t, f, id)["Adaga"]; got != before+2 {
		t.Errorf("a adaga foi de %+d a %+d, e a melhoria dá +2", before, got)
	}
}

// O pedido montado à mão não repete a melhoria, nem a põe em item que não recebe melhoria.
func TestTheBagRefusesWhatTheDialogNeverOffers(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	medal := itemSemeia(t, f, id, "medalhao-de-prata", "Medalhão de prata", "vested")
	berry := itemSemeia(t, f, id, "baga-de-fogo", "Baga-de-fogo", "wielded")
	improvement := createsImprovement(t, f, improvementSignals("Insígnia", "", "expertise", "Luta", 1))
	if refusal := sceneRefusal(placesImprovements(t, f, f.player, id, medal, improvement, improvement)); refusal == "" {
		t.Error("a mesma melhoria duas vezes devia ser recusada")
	}
	if strings.Contains(improvementsOnItem(t, f, medal), improvement) {
		t.Errorf("gravou a melhoria repetida: %s", improvementsOnItem(t, f, medal))
	}
	if refusal := sceneRefusal(placesImprovements(t, f, f.player, id, berry, improvement)); refusal == "" {
		t.Error("o catalisador não recebe melhoria, e a da mesa entrou")
	}
	if strings.Contains(improvementsOnItem(t, f, berry), improvement) {
		t.Error("gravou a melhoria no catalisador")
	}
}

// Tirar uma melhoria de PV com pouca vida liga o que o PV pede, como guardar o item liga.
func TestTakingOffAnHPImprovementFollowsTheMaximum(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	ring := itemSemeia(t, f, id, "anel-da-liberdade", "Anel da liberdade", "vested")
	improvement := createsImprovement(t, f, improvementSignals("Vigor", "", "maxPv", "", 50))
	placesImprovements(t, f, f.player, id, ring, improvement)
	arrangePools(t, f.s, id, func(pools sheet.Pools) (sheet.Pools, error) {
		pools.HpCurrent = 3
		return pools, nil
	})
	placesImprovements(t, f, f.player, id, ring)
	if got := conditionsOf(t, f.s, id); !strings.Contains(strings.Join(got, ","), "inconsciente") {
		t.Errorf("o PV caiu abaixo de zero e as condições são %v", got)
	}
}

// Um item com a lista de melhorias ilegível não derruba a aba do mestre.
func TestATwistedItemDoesNotTakeTheTabDown(t *testing.T) {
	f := newSceneFixture(t)
	createsImprovement(t, f, improvementSignals("Insígnia", "apparel", "expertise", "Luta", 1))
	item := itemSemeia(t, f, f.charID, "medalhao-de-prata", "Medalhão de prata", "vested")
	if _, err := f.s.db.ExecContext(context.Background(), "UPDATE character_items SET improvements = 'torto' WHERE id = ?", item); err != nil {
		t.Fatal(err)
	}
	if page := rulesTab(t, f, f.gm); !strings.Contains(page, "Insígnia") {
		t.Errorf("a aba caiu com um item ilegível: %q", page[:min(200, len(page))])
	}
}
