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

	"golang.org/x/net/html"
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
	return bagRefusal(t, f.requests(t, f.player, http.MethodPost, target, "").Body.String())
}

// bagRefusal acha a frase da RECUSA do gesto, e só ela.
//
// O `sceneRefusal` pega o PRIMEIRO `role="alert"` da resposta, e na ficha
// Tempestade isso não é mais a recusa: a folha de cada item desenha o aviso
// "Sem proficiência" com o mesmo `role="alert"`, e a carga estourada também.
// Lido assim, todo gesto sobre uma arma sem proficiência saía "recusado" pelo
// aviso. A recusa é a filha DIRETA da coluna da ficha (`ds-sheet-main`), logo
// abaixo das abas; os avisos moram dentro do painel e dos diálogos.
//
// A coluna é o CONTROLE: sem ela, "nenhuma recusa" e "não li a resposta" dariam
// o mesmo "".
func bagRefusal(t *testing.T, body string) string {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("ler a resposta do gesto: %v", err)
	}
	column := findNode(doc, "class", "ds-sheet-main")
	if column == nil {
		t.Fatalf("a resposta do gesto não tem a coluna da ficha (ds-sheet-main), "+
			"então a recusa não teria onde aparecer:\n%.400s", body)
	}
	for c := column.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "p" && attr(c, "role") == "alert" {
			return strings.TrimSpace(textOf(c))
		}
	}
	return ""
}

// inUseOf recorta a metade EM USO da Mochila — as listas de Mãos e de
// Vestidos —, e falha quando não a acha. O fim é a barra de filtros, que abre a
// metade do guardado e está sempre à vista.
func inUseOf(t *testing.T, screen string) string {
	t.Helper()
	start := strings.Index(screen, "<span>Mãos</span>")
	if start < 0 {
		t.Fatal("a Mochila não desenhou a lista das mãos")
	}
	end := strings.Index(screen[start:], "ds-filter-bar")
	if end < 0 {
		t.Fatal("a Mochila não desenhou a barra de filtros do guardado")
	}
	return screen[start : start+end]
}

// stowedOf recorta a lista do GUARDADO, do cabeçalho dela ao fim do painel.
func stowedOf(t *testing.T, screen string) string {
	t.Helper()
	start := strings.Index(screen, "<span>Guardado</span>")
	if start < 0 {
		t.Fatal("a Mochila não desenhou a lista do guardado")
	}
	end := strings.Index(screen[start:], "</section>")
	if end < 0 {
		t.Fatal("a lista do guardado não fecha o painel")
	}
	return screen[start : start+end]
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

// cardInStrip diz se o item tem LINHA nas listas do que está em uso.
//
// Procurar o nome na tela inteira não serve, e isso já custou um caso verde
// sobre nada: o nome aparece de novo na folha daquele item, desenhada depois do
// painel, e na lista do guardado — procurar na tela inteira acha justamente o
// que as listas esconderam. Por isso o recorte é a metade em uso, e o âncora é
// o título da linha.
func cardInStrip(t *testing.T, screen, name string) bool {
	t.Helper()
	return strings.Contains(inUseOf(t, screen), `<span class="ds-row-title">`+name+`</span>`)
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

	inUse := inUseOf(t, bagScreen(t, f, id))
	for _, want := range []string{"Mãos", "1 / 2", "Vestidos", "1 / 4", "Mão principal"} {
		if !strings.Contains(inUse, want) {
			t.Errorf("as listas em uso não têm %q", want)
		}
	}
	// AS POSIÇÕES LIVRES SÃO CONTADAS: uma mão e três vestidos. Só afirmar que
	// "livre" aparece deixaria passar uma lista que conta um lugar e esconde os
	// outros três — e é justamente a soma que mostra quanto ainda cabe.
	for _, free := range []string{"1 mão livre", "3 lugares livres"} {
		if !strings.Contains(inUse, free) {
			t.Errorf("as listas em uso não contam %q", free)
		}
	}
	// O que o item CONCEDE aparece na linha: sem isso a lista vira uma lista de
	// nomes, e a pergunta que ela existe para responder é "o que isto me dá?".
	for _, badge := range []string{"Dano 1d8", "Defesa +2"} {
		if !strings.Contains(inUse, badge) {
			t.Errorf("a linha não mostra %q", badge)
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
	// O CONTROLE vem primeiro: os dois primeiros têm cartão. Sem ele, um
	// âncora que não casasse com nada diria "sumiu" sobre uma tira inteira.
	for _, nome := range []string{"Espada longa", "Adaga"} {
		if !cardInStrip(t, screen, nome) {
			t.Fatalf("o controle já estava errado: %q não tem cartão na tira", nome)
		}
	}
	if !cardInStrip(t, screen, "Escudo leve") {
		t.Error("o terceiro item empunhado sumiu da tira")
	}
	if !strings.Contains(inUseOf(t, screen), "Acima do limite") {
		t.Error("o item fora do teto não é nomeado como tal")
	}
	// O CONTADOR denuncia junto: 3/2 é o que faz a pessoa entender por que há
	// um cartão a mais.
	if !strings.Contains(inUseOf(t, screen), "3 / 2") {
		t.Error("o contador das mãos não mostra que passou do teto")
	}
}

// O ITEM DE UMA MÃO NÃO SOME ATRÁS DO DE DUAS (ALE-450).
//
// É o irmão do caso de cima, e o que o separa dele é o SLOT: `wielded` vale uma
// mão e `wielded2` vale as duas — é o que o diálogo oferece ("Empunhar (1 mão)"
// e "Empunhar (2 mãos)"). Com um item em cada, a tira escolhia UM dos dois e o
// escudo sumia da tela, enquanto a Defesa continuava somando o +2 dele.
//
// O servidor recusa chegar nesse estado (`EquipLimitError`, p141), e é por isso
// que o caso semeia direto no banco: quem está assim veio de linha antiga, e o
// painel é a última coisa que pode denunciá-lo.
func TestTheOneHandedItemDoesNotVanishBehindTheTwoHandedOne(t *testing.T) {
	f, id := fighterFixture(t)
	itemSemeia(t, f, id, "escudo-leve", "Escudo leve", "wielded")
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded2")

	screen := bagScreen(t, f, id)
	for _, nome := range []string{"Escudo leve", "Espada longa"} {
		if !cardInStrip(t, screen, nome) {
			t.Errorf("%q não tem cartão na tira: ela escolheu um dos dois slots", nome)
		}
	}
	if !strings.Contains(inUseOf(t, screen), "Acima do limite") {
		t.Error("o item fora do teto não é nomeado como tal")
	}
	// O CONTADOR denuncia junto, e é ele que explica o cartão a mais: uma mão do
	// escudo e duas da espada são TRÊS, e dizer 2/2 esconde o estouro com a
	// mesma eficácia com que a tira escondia o item.
	if !strings.Contains(inUseOf(t, screen), "3 / 2") {
		t.Error("o contador das mãos não mostra que passou do teto")
	}
}

// E DOIS ITENS DE DUAS MÃOS TAMBÉM NÃO SE ENGOLEM (ALE-450).
//
// O campo que os recebia era um PONTEIRO, então o segundo substituía o
// primeiro — o mesmo defeito do caso de cima com outra aritmética: quatro mãos.
func TestTwoTwoHandedItemsDoNotSwallowEachOther(t *testing.T) {
	f, id := fighterFixture(t)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded2")
	itemSemeia(t, f, id, "machado-batalha", "Machado de batalha", "wielded2")

	screen := bagScreen(t, f, id)
	for _, nome := range []string{"Espada longa", "Machado de batalha"} {
		if !cardInStrip(t, screen, nome) {
			t.Errorf("%q não tem cartão na tira: o segundo item de duas mãos "+
				"substituiu o primeiro", nome)
		}
	}
	if !strings.Contains(inUseOf(t, screen), "4 / 2") {
		t.Error("o contador das mãos não conta as quatro que dois itens de duas mãos pedem")
	}
}

// E O QUINTO VESTIDO TAMBÉM NÃO SOME (ALE-450).
//
// A tira de Vestidos tinha o MESMO corte com outro nome: ela completava a lista
// até quatro posições e devolvia `cards[:4]`, jogando fora o que passasse. O
// contador já dizia 5/4 — ele é contado antes do corte —, então a tela
// denunciava o estouro e escondia o item que o causava.
func TestTheFifthVestedItemDoesNotVanishFromTheScreen(t *testing.T) {
	f, id := fighterFixture(t)
	for _, peca := range [][2]string{
		{"armadura-couro", "Armadura de couro"}, {"bandana", "Bandana"},
		{"coroa-majestosa", "Coroa majestosa"}, {"anel-da-protecao", "Anel da proteção"},
		{"manto-da-noite", "Manto da noite"},
	} {
		itemSemeia(t, f, id, peca[0], peca[1], "vested")
	}

	screen := bagScreen(t, f, id)
	// O CONTROLE: os quatro que cabem têm cartão.
	for _, nome := range []string{"Armadura de couro", "Bandana", "Coroa majestosa", "Anel da proteção"} {
		if !cardInStrip(t, screen, nome) {
			t.Fatalf("o controle já estava errado: %q não tem cartão na tira", nome)
		}
	}
	if !cardInStrip(t, screen, "Manto da noite") {
		t.Error("o quinto vestido sumiu da tira: ela cortou em quatro posições")
	}
	if !strings.Contains(inUseOf(t, screen), "5 / 4") {
		t.Error("o contador dos vestidos não mostra que passou do teto")
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
	if !strings.Contains(inUseOf(t, bagScreen(t, f, id)), "0 / 2") {
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
	return bagRefusal(t, f.requests(t, f.player, http.MethodPost, target, body).Body.String())
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
	// O RECORTE é a lista do guardado: a folha de cada item é desenhada fora
	// dela, filtro ou não, e procurar na tela inteira acharia a espada lá.
	if found := stowedOf(t, withSearch); !strings.Contains(found, "Bálsamo restaurador") {
		t.Error("a busca sem acento não achou o Bálsamo")
	} else if strings.Contains(found, "Espada longa") {
		t.Error("a busca deixou passar o que não casa")
	}

	withChip := f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=bag&itemcategoria=weapons", id), "").Body.String()
	if !strings.Contains(stowedOf(t, withChip), "Espada longa") {
		t.Error("o chip de armas escondeu a espada")
	}
	if strings.Contains(stowedOf(t, withChip), "Bálsamo restaurador") {
		t.Error("o chip de armas deixou passar um consumível")
	}
}
