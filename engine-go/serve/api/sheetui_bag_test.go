package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
	"testing"
)

func bagScreen(t *testing.T, f sceneFixture, id int64) string {
	t.Helper()
	return f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=bag", id), "").Body.String()
}

// bagCommand dispara um gesto e devolve a recusa, ou "".
func bagCommand(t *testing.T, f sceneFixture, id int64, path string) string {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/%s?tab=bag", id, path)
	return sceneRefusal(f.requests(t, f.player, http.MethodPost, target, "").Body.String())
}

// itemSemeia põe um item na ficha e devolve o id.
func itemSemeia(t *testing.T, f sceneFixture, id int64, catalog, name string, equipped string) int64 {
	t.Helper()
	item, err := f.s.sceneCore().Queries().CreateItem(context.Background(), sqlcgen.CreateItemParams{
		Characterid: id, Catalogid: sql.NullString{String: catalog, Valid: catalog != ""},
		Name: name, Quantity: 1, Slots: 1,
		Equipped:     sql.NullString{String: equipped, Valid: equipped != ""},
		Improvements: "[]", Createdat: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o item %q: %v", name, err)
	}
	return item.ID
}

// equipped lê a coluna direto do banco — o teste não pergunta à tela o que
// aconteceu no banco.
func equipped(t *testing.T, f sceneFixture, itemID int64) string {
	t.Helper()
	item, err := f.s.sceneCore().Queries().GetItem(context.Background(), itemID)
	if err != nil {
		t.Fatalf("ler o item %d: %v", itemID, err)
	}
	return item.Equipped.String
}

// A TIRA desenha os dois tetos do livro, e as posições vazias junto.
func TestTheStripShowsTheTwoCeilingsFromPage141(t *testing.T) {
	f, id := fighterFixture(t)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	itemSemeia(t, f, id, "armadura-couro", "Armadura de couro", "vested")
	itemSemeia(t, f, id, "corda", "Corda", "")

	screen := bagScreen(t, f, id)
	for _, want := range []string{"Mãos", "1/2", "Vestidos", "1/4", "Mão principal", "Mão secundária"} {
		if !strings.Contains(screen, want) {
			t.Errorf("a tira não tem %q", want)
		}
	}
	// AS POSIÇÕES VAZIAS SÃO DESENHADAS, e são contadas: uma mão livre e três
	// vestidos livres. Só afirmar que a palavra "vazio" aparece deixaria passar
	// uma tira que desenha um lugar livre e esconde os outros três — e é
	// justamente a soma que mostra quanto ainda cabe.
	if free := strings.Count(screen, ">vazio</p>"); free != 4 {
		t.Errorf("a tira desenhou %d posições livres, quer 4 (1 mão + 3 vestidos)", free)
	}
	// O que o item CONCEDE aparece no cartão: sem isso a tira vira uma lista de
	// nomes, e a pergunta que ela existe para responder é "o que isto me dá?".
	for _, badge := range []string{"Dano 1d8", "Defesa +2"} {
		if !strings.Contains(screen, badge) {
			t.Errorf("o cartão não mostra %q", badge)
		}
	}
}

// O ITEM ACIMA DO TETO NÃO SOME.
//
// Uma tira que desenha `wielded[0]` e `wielded[1]` e para por aí esconde o
// terceiro item empunhado — que o banco tem — de TODA a tela: nem na tira, nem
// na grade do guardado, porque ele está equipado.
func TestAnItemAboveTheCeilingDoesNotVanishFromTheScreen(t *testing.T) {
	f, id := fighterFixture(t)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	itemSemeia(t, f, id, "adaga", "Adaga", "wielded")
	itemSemeia(t, f, id, "escudo-leve", "Escudo leve", "wielded")

	screen := bagScreen(t, f, id)
	if !strings.Contains(screen, "Escudo leve") {
		t.Error("o terceiro item empunhado sumiu da tela")
	}
	if !strings.Contains(screen, "Acima do limite") {
		t.Error("o item fora do teto não é nomeado como tal")
	}
	// O CONTADOR denuncia junto: 3/2 é o que faz a pessoa entender por que há
	// um cartão a mais.
	if !strings.Contains(screen, "3/2") {
		t.Error("o contador das mãos não mostra que passou do teto")
	}
}

// GUARDAR tira da mão e devolve à grade.
func TestStowingTakesItFromTheHandAndPutsItInTheGrid(t *testing.T) {
	f, id := fighterFixture(t)
	item := itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")

	if refusal := bagCommand(t, f, id, fmt.Sprintf("itens/%d/guarda", item)); refusal != "" {
		t.Fatalf("guardar foi recusado: %q", refusal)
	}
	if got := equipped(t, f, item); got != "" {
		t.Errorf("o item continua equipado como %q", got)
	}
	if !strings.Contains(bagScreen(t, f, id), "0/2") {
		t.Error("a mão não ficou livre na tela")
	}
}

// EQUIPAR respeita o EIXO do item: um escudo não se veste.
func TestEquippingRespectsTheItemAxis(t *testing.T) {
	f, id := fighterFixture(t)
	item := itemSemeia(t, f, id, "escudo-leve", "Escudo leve", "")

	refusal := bagCommand(t, f, id, fmt.Sprintf("itens/%d/equipa/vested", item))
	if !strings.Contains(refusal, "Escudo leve") {
		t.Errorf("a recusa não nomeia o item: %q", refusal)
	}
	if got := equipped(t, f, item); got != "" {
		t.Errorf("a recusa equipou assim mesmo: %q", got)
	}
	// E o caminho que o livro permite continua aberto.
	if refused := bagCommand(t, f, id, fmt.Sprintf("itens/%d/equipa/wielded", item)); refused != "" {
		t.Errorf("empunhar um escudo foi recusado: %q", refused)
	}
	if got := equipped(t, f, item); got != "wielded" {
		t.Errorf("o escudo não foi empunhado: %q", got)
	}
}

// EQUIPAR respeita o TETO de duas mãos (p141), e a recusa chega à TELA.
func TestEquippingRespectsTheTwoHandCeiling(t *testing.T) {
	f, id := fighterFixture(t)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	itemSemeia(t, f, id, "adaga", "Adaga", "wielded")
	third := itemSemeia(t, f, id, "clava", "Clava", "")

	refusal := bagCommand(t, f, id, fmt.Sprintf("itens/%d/equipa/wielded", third))
	if refusal == "" {
		t.Fatal("uma terceira mão foi aceita")
	}
	if !strings.Contains(refusal, "2") {
		t.Errorf("a recusa não diz qual é o teto: %q", refusal)
	}
	if got := equipped(t, f, third); got != "" {
		t.Errorf("a recusa equipou assim mesmo: %q", got)
	}
}

// UM ITEM DE OUTRA FICHA não se mexe por esta.
//
// A posse do PERSONAGEM já é do `sheetCommand`; a do ITEM é o que falta. Sem
// ela o id de outra ficha passaria — o `UPDATE` é por id, e afetar zero linhas
// não é erro que alguém veja.
func TestAnItemFromAnotherSheetCannotBeMoved(t *testing.T) {
	f, mine := fighterFixture(t)
	other := seedCharacterAtLevel(t, f.s, f.player, "Vizinho", "Guerreiro", 1, 0, 0)
	foreign := itemSemeia(t, f, other, "espada-longa", "Espada longa", "wielded")

	if refusal := bagCommand(t, f, mine, fmt.Sprintf("itens/%d/guarda", foreign)); refusal == "" {
		t.Error("guardei o item de outra ficha pela minha")
	}
	if got := equipped(t, f, foreign); got != "wielded" {
		t.Errorf("o item alheio foi mexido assim mesmo: %q", got)
	}
}

// O DINHEIRO: os três modos, o piso e o arredondamento.
func TestMoneyIsReceivedSpentAndCorrected(t *testing.T) {
	f, id := fighterFixture(t)
	if err := f.s.sceneCore().Queries().SetCharacterTibar(context.Background(), sqlcgen.SetCharacterTibarParams{
		Tibar: 35.7, UpdatedAt: dbvalue.NowISO(), ID: id,
	}); err != nil {
		t.Fatalf("semear o dinheiro: %v", err)
	}

	// GASTAR 0,3 sobre 35,7. Este par é escolhido, e não qualquer um: em ponto
	// flutuante ele dá 35,400000000000006, e é ESSE número que iria para o
	// banco e para a tela sem o arredondamento de duas casas. Um par ingênuo
	// não serve — medido, 1200,3 − 80,1 fecha exato em IEEE, e um teste escrito
	// com ele passa verde com o arredondamento removido.
	if refusal := money(t, f, id, "gastar", 0.3); refusal != "" {
		t.Fatalf("gastar foi recusado: %q", refusal)
	}
	if got := tibar(t, f, id); got != 35.4 {
		t.Errorf("o saldo ficou %v, quer 35.4 — a soma binária foi para o banco", got)
	}

	// E RECEBER tem a mesma armadilha do outro lado: 0,1 + 0,2 dá
	// 0,30000000000000004.
	if err := f.s.sceneCore().Queries().SetCharacterTibar(context.Background(), sqlcgen.SetCharacterTibarParams{
		Tibar: 0.1, UpdatedAt: dbvalue.NowISO(), ID: id,
	}); err != nil {
		t.Fatalf("semear o dinheiro: %v", err)
	}
	if refusal := money(t, f, id, "receber", 0.2); refusal != "" {
		t.Fatalf("receber foi recusado: %q", refusal)
	}
	if got := tibar(t, f, id); got != 0.3 {
		t.Errorf("receber deu %v, quer 0.3", got)
	}

	if refusal := money(t, f, id, "corrigir", 42); refusal != "" {
		t.Fatalf("corrigir foi recusado: %q", refusal)
	}
	if got := tibar(t, f, id); got != 42 {
		t.Errorf("corrigir deu %v, quer 42", got)
	}
}

// DÍVIDA NÃO EXISTE na ficha, e a razão é a carga.
//
// Saldo negativo viraria carga de moeda NEGATIVA, que COMPRARIA espaço na
// mochila em vez de ocupar. Por isso o piso é zero, e não um aviso.
func TestMoneyNeverGoesNegative(t *testing.T) {
	f, id := fighterFixture(t)
	if err := f.s.sceneCore().Queries().SetCharacterTibar(context.Background(), sqlcgen.SetCharacterTibarParams{
		Tibar: 50, UpdatedAt: dbvalue.NowISO(), ID: id,
	}); err != nil {
		t.Fatalf("semear o dinheiro: %v", err)
	}

	refusal := money(t, f, id, "gastar", 80)
	if !strings.Contains(refusal, "50") {
		t.Errorf("a recusa não diz quanto a pessoa tem: %q", refusal)
	}
	if got := tibar(t, f, id); got != 50 {
		t.Errorf("a recusa gastou assim mesmo: %v", got)
	}
}

func money(t *testing.T, f sceneFixture, id int64, mode string, value float64) string {
	t.Helper()
	body := fmt.Sprintf(`{"tibar_mode":%q,"tibar_value":%v}`, mode, value)
	target := fmt.Sprintf("/personagens/%d/dinheiro?tab=bag", id)
	return sceneRefusal(f.requests(t, f.player, http.MethodPost, target, body).Body.String())
}

func tibar(t *testing.T, f sceneFixture, id int64) float64 {
	t.Helper()
	row, err := f.s.sceneCore().Queries().GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	return row.Tibar
}

// A BUSCA da grade ignora acento, e o CHIP filtra por categoria.
func TestTheGridFiltersBySearchAndByCategory(t *testing.T) {
	f, id := fighterFixture(t)
	itemSemeia(t, f, id, "balsamo-restaurador", "Bálsamo restaurador", "")
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "")

	withSearch := f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=bag&itembusca=balsamo", id), "").Body.String()
	if !strings.Contains(withSearch, "Bálsamo restaurador") {
		t.Error("a busca sem acento não achou o Bálsamo")
	}
	if strings.Contains(screenSaved(withSearch), "Espada longa") {
		t.Error("a busca deixou passar o que não casa")
	}

	withChip := f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=bag&itemcategoria=weapons", id), "").Body.String()
	if !strings.Contains(screenSaved(withChip), "Espada longa") {
		t.Error("o chip de armas escondeu a espada")
	}
	if strings.Contains(screenSaved(withChip), "Bálsamo restaurador") {
		t.Error("o chip de armas deixou passar um consumível")
	}
}
