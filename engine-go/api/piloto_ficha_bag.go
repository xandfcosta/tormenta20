package api

import (
	"sort"
	"strconv"
	"strings"

	"t20engine/book"
	"t20engine/engine"
	"t20engine/sheet"
)

// A aba MOCHILA como dado (ALE-272, fatia 7).
//
// Três blocos, e a ordem responde a três perguntas diferentes da mesa:
//
//  1. O QUE ESTÁ NA MÃO E NO CORPO — a tira de Mãos (≤2) e Vestidos (≤4). Não
//     são casas de corpo: o livro não tem slots de armadura e de elmo, ele tem
//     dois TETOS (p141), e a tira desenha exatamente esses dois.
//  2. QUANTO PESA — a carga da p141, que vem inteira do motor.
//  3. O QUE ESTÁ GUARDADO — a grade de ladrilhos, com busca e categoria.
//
// # Nenhum número desta aba é somado aqui
//
// Espaços ocupados, limite, dobro do limite, sobrecarga e as duas penalidades
// são `sheet.Carga` (ALE-215 tirou essa conta da tela). O que este arquivo faz
// é escolher palavras e agrupar linhas.

// bagPanel é a aba Mochila pronta para desenhar.
type bagPanel struct {
	// Hands são as duas mãos. Uma arma de duas mãos ocupa as duas, e por isso
	// ela é um cartão LARGO em vez de aparecer repetida nos dois lugares.
	Hands     handSlots
	HandsUsed int
	// Vested são quatro POSIÇÕES, e uma posição vazia é um cartão pontilhado —
	// ver quantas sobram é metade da informação.
	Vested     []*equippedCard
	VestedUsed int
	// Stowed é o que está guardado, já filtrado pela busca e pela categoria.
	Stowed []stowedTile
	// StowedTotal é o total ANTES do filtro, para a tela distinguir "mochila
	// vazia" de "nenhum item para esse filtro".
	StowedTotal int
	Load        loadMeter
	Money       moneyLine
	Search      string
	Category    string
	Categories  []filterOption
	// Sheets é uma ficha por item — o diálogo que o ladrilho e o cartão abrem.
	Sheets []itemSheet
	// Catalogo é o Capítulo 3 filtrado, para o diálogo de adicionar.
	Catalogo          []catalogItemRow
	CatalogSearch     string
	CatalogCategory   string
	CatalogCategories []filterOption
	CatalogTotal      int
}

// handSlots são as mãos: ou UMA arma de duas mãos, ou as empunhadas em ordem.
//
// `Wielded` é uma lista e não dois campos porque o banco pode ter MAIS de duas
// — a semente tem um personagem com clava, espada e escudo empunhados ao mesmo
// tempo. O servidor recusa a terceira hoje, mas linha antiga não passou por
// essa recusa, e um par de campos jogaria a sobra fora: o item sumiria da tela
// inteira, sem aparecer nem na grade do que está guardado.
type handSlots struct {
	TwoHand *equippedCard
	Wielded []*equippedCard
}

// equippedCard é um item na tira, com o que ele concede.
type equippedCard struct {
	ID    int64
	Label string
	Name  string
	// Chips são as sobreposições e o que o item concede — "Reforçada",
	// "Defesa +2", "Dano 1d8".
	Chips []string
	// NoProficiency marca o item equipado que o personagem não sabe usar. O
	// motor já cobra a penalidade (p142); o crachá é para ela não ser uma
	// surpresa no meio de um teste.
	NoProficiency bool
	Command       string
}

// stowedTile é um ladrilho da grade.
type stowedTile struct {
	ID       int64
	Name     string
	Quantity int64
	// Glyph é o nome do ícone da casa, escolhido pela categoria.
	Glyph    string
	Overlays []string
	Command  string
}

// loadMeter é a carga da p141 traduzida para a tela.
type loadMeter struct {
	Used  string
	Limit int
	Max   int
	// Percent é a largura da barra, presa em 100 — uma barra de 340% desenharia
	// para fora do painel.
	Percent             int
	Coins               string
	CoinSlots           float64
	Overloaded          bool
	OverMax             bool
	Enforced            bool
	ArmorPenalty        string
	DisplacementPenalty string
	// LimitLabel é a conta que produziu o limite, e não a notação: "limite 18 ·
	// 10 + 2×For +4" em vez de "10 + 2×|FOR|".
	LimitLabel string
}

// moneyLine é o dinheiro do personagem.
type moneyLine struct {
	Tibar string
	// Slots são os espaços que as moedas ocupam, escrito só quando há moeda
	// bastante para ocupar algum.
	Slots string
}

// osFiltrosDaMochila é o que a pessoa digitou: os dois filtros da grade e os dois do
// diálogo do catálogo. Uma struct e não quatro parâmetros — quatro strings em
// sequência é a assinatura em que se troca a ordem sem o compilador reclamar.
type osFiltrosDaMochila struct {
	Busca               string
	Categoria           string
	BuscaNoCatalogo     string
	CategoriaNoCatalogo string
}

// bagPanelOf monta a aba.
func (s *Server) bagPanelOf(dto sheet.CharacterDTO, filtros osFiltrosDaMochila) bagPanel {
	busca, categoria := filtros.Busca, filtros.Categoria
	proficiencias := asProficienciasGuardadas(dto.Proficiencies)
	panel := bagPanel{
		Search:            busca,
		Category:          categoria,
		Categories:        bagCategoryOptions(categoria),
		Money:             moneyLineOf(dto),
		CatalogSearch:     filtros.BuscaNoCatalogo,
		CatalogCategory:   filtros.CategoriaNoCatalogo,
		CatalogCategories: asCategoriasDoCatalogo(filtros.CategoriaNoCatalogo),
	}
	panel.Catalogo = catalogItemRowsOf(filtros.BuscaNoCatalogo, filtros.CategoriaNoCatalogo)
	panel.CatalogTotal = len(panel.Catalogo)
	guardados := []sheet.ItemDTO{}
	for _, item := range dto.Items {
		switch equippedSlotOf(item) {
		case "wielded2":
			panel.Hands.TwoHand = equippedCardOf(item, "Duas mãos", proficiencias)
		case "wielded":
			panel.Hands.Wielded = append(panel.Hands.Wielded,
				equippedCardOf(item, oRotuloDaMao(len(panel.Hands.Wielded)), proficiencias))
		case "vested":
			panel.Vested = append(panel.Vested, equippedCardOf(item, "Vestido", proficiencias))
		default:
			guardados = append(guardados, item)
		}
	}
	panel.VestedUsed = len(panel.Vested)
	panel.Vested = comQuatroPosicoes(panel.Vested)
	panel.HandsUsed = handsUsedIn(panel.Hands)
	panel.Hands.Wielded = asDuasMaos(panel.Hands.Wielded)
	panel.StowedTotal = len(guardados)
	panel.Stowed = stowedTilesOf(filtradosNaMochila(guardados, busca, categoria))
	panel.Sheets = itemSheetsOf(dto, proficiencias)
	if sheet, _, ok := s.sheetForPanels(dto); ok {
		panel.Load = loadMeterOf(sheet)
	}
	return panel
}

// equippedSlotOf lê o estado de equipar, tratando nulo como guardado.
func equippedSlotOf(item sheet.ItemDTO) string {
	if item.Equipped == nil {
		return ""
	}
	return *item.Equipped
}

// comQuatroPosicoes completa a tira de Vestidos com as posições vazias.
//
// Quatro sempre, mesmo com um item só: o teto do livro é a informação, e uma
// tira que crescesse com o uso esconderia justamente quanto ainda cabe.
func comQuatroPosicoes(cards []*equippedCard) []*equippedCard {
	for len(cards) < vestedLimit {
		cards = append(cards, nil)
	}
	return cards[:vestedLimit]
}

func handsUsedIn(hands handSlots) int {
	if hands.TwoHand != nil {
		return 2
	}
	return len(hands.Wielded)
}

// oRotuloDaMao nomeia a posição pela ORDEM. A terceira em diante não tem nome
// no livro porque não deveria existir — e é justamente por isso que ela é
// nomeada do jeito que denuncia.
func oRotuloDaMao(indice int) string {
	switch indice {
	case 0:
		return "Mão principal"
	case 1:
		return "Mão secundária"
	}
	return "Acima do limite"
}

// asDuasMaos completa as empunhadas com as posições vazias, sem NUNCA cortar o
// que passou do teto.
func asDuasMaos(cards []*equippedCard) []*equippedCard {
	for len(cards) < 2 {
		cards = append(cards, nil)
	}
	return cards
}

// equippedCardOf traduz um item equipado para o cartão da tira.
func equippedCardOf(item sheet.ItemDTO, rotulo string, proficiencias map[string]bool) *equippedCard {
	return &equippedCard{
		ID:            item.ID,
		Label:         rotulo,
		Name:          item.Name,
		Chips:         append(asSobreposicoesDoItem(item), oQueOItemConcede(item)...),
		NoProficiency: !ehProficiente(item, proficiencias),
		Command:       strconv.FormatInt(item.ID, 10),
	}
}

// ehProficiente diz se o personagem sabe usar o item.
//
// A tabela que decide é a do MOTOR (`engine.RequiredProficiency`), a mesma que
// resolve a penalidade da p142: um segundo mapa aqui daria uma tela que avisa
// sobre um item e um motor que penaliza outro. Item custom e item fora do
// catálogo contam como proficientes — não há categoria de onde tirar exigência,
// e acusar o que não se sabe seria pior que calar.
func ehProficiente(item sheet.ItemDTO, proficiencias map[string]bool) bool {
	catalogo := itemDoCatalogo(item)
	if catalogo == nil {
		return true
	}
	exigida := engine.RequiredProficiency(&engine.CatalogItem{Category: catalogo.Category})
	return exigida == "" || proficiencias[exigida]
}

// stowedTilesOf traduz os itens guardados em ladrilhos.
func stowedTilesOf(itens []sheet.ItemDTO) []stowedTile {
	linhas := make([]stowedTile, 0, len(itens))
	for _, item := range itens {
		linhas = append(linhas, stowedTile{
			ID: item.ID, Name: item.Name, Quantity: item.Quantity,
			Glyph:    oGlifoDoItem(item),
			Overlays: asSobreposicoesDoItem(item),
			Command:  strconv.FormatInt(item.ID, 10),
		})
	}
	return linhas
}

// oGlifoDoItem escolhe o desenho do ladrilho pela categoria.
//
// Item sem catálogo cai no pacote genérico, que é honesto: não há o que
// adivinhar sobre um item que a pessoa inventou.
func oGlifoDoItem(item sheet.ItemDTO) string {
	catalogo := itemDoCatalogo(item)
	if catalogo == nil {
		return "Package"
	}
	switch {
	case strings.HasPrefix(catalogo.Category, "weapon-"):
		return "Sword"
	case strings.HasPrefix(catalogo.Category, "armor-"), catalogo.Category == "shield":
		return "Shield"
	case catalogo.Category == "apparel" && catalogo.Equip == "wielded":
		return "Wand2"
	case catalogo.Category == "apparel":
		return "Shirt"
	case catalogo.Category == "consumable":
		return "FlaskConical"
	case catalogo.Category == "meal":
		return "Utensils"
	}
	return "Package"
}

// filtradosNaMochila cruza a busca com a categoria escolhida.
//
// A busca ignora acento pela mesma razão das Perícias: quem digita "balsamo"
// tem de achar "Bálsamo restaurador".
func filtradosNaMochila(itens []sheet.ItemDTO, busca, categoria string) []sheet.ItemDTO {
	termo := foldAccents(strings.TrimSpace(busca))
	fora := []sheet.ItemDTO{}
	for _, item := range itens {
		if termo != "" && !strings.Contains(foldAccents(item.Name), termo) {
			continue
		}
		if !daCategoriaDaMochila(item, categoria) {
			continue
		}
		fora = append(fora, item)
	}
	return fora
}

// asCategoriasDaMochila são os chips, na ordem em que aparecem.
//
// São CINCO e não as quinze do catálogo: o chip existe para achar uma coisa no
// meio de trinta ladrilhos, e quinze chips numa tela de 390px seriam outra
// lista para procurar dentro.
var asCategoriasDaMochila = []filterOption{
	{Valor: "", Rotulo: "tudo"},
	{Valor: "weapons", Rotulo: "armas"},
	{Valor: "defense", Rotulo: "defesa"},
	{Valor: "consumables", Rotulo: "consumo"},
	{Valor: "other", Rotulo: "outros"},
}

func bagCategoryOptions(ativa string) []filterOption {
	fora := make([]filterOption, 0, len(asCategoriasDaMochila))
	for _, chip := range asCategoriasDaMochila {
		fora = append(fora, filterOption{Valor: chip.Valor, Rotulo: chip.Rotulo, Ativo: chip.Valor == ativa})
	}
	return fora
}

// daCategoriaDaMochila diz se o item aparece sob o chip escolhido.
//
// Item custom não tem categoria de catálogo e conta como equipamento comum —
// assim ele cai em "tudo" e em "outros", e nunca some da mochila inteira.
func daCategoriaDaMochila(item sheet.ItemDTO, chip string) bool {
	if chip == "" {
		return true
	}
	categoria := "gear"
	if catalogo := itemDoCatalogo(item); catalogo != nil {
		categoria = catalogo.Category
	}
	switch chip {
	case "weapons":
		return strings.HasPrefix(categoria, "weapon-")
	case "defense":
		return strings.HasPrefix(categoria, "armor-") || categoria == "shield"
	case "consumables":
		return categoria == "consumable" || categoria == "meal"
	case "other":
		return !strings.HasPrefix(categoria, "weapon-") &&
			!strings.HasPrefix(categoria, "armor-") &&
			categoria != "shield" && categoria != "consumable" && categoria != "meal"
	}
	return true
}

// loadMeterOf traduz a carga do motor.
func loadMeterOf(sheet engine.ComputedSheetV2) loadMeter {
	carga := sheet.Carga
	return loadMeter{
		Used:                comVirgula(carga.Used),
		Limit:               carga.Limit,
		Max:                 carga.Max,
		Percent:             aLarguraDaBarra(carga.Used, carga.Limit),
		Coins:               comVirgula(carga.Coins),
		CoinSlots:           carga.Coins,
		Overloaded:          carga.Overloaded,
		OverMax:             carga.OverMax,
		Enforced:            carga.Enforced,
		ArmorPenalty:        comSinalInt(carga.ArmorPenalty),
		DisplacementPenalty: comSinalInt(carga.DisplacementPenalty),
		LimitLabel:          oRotuloDoLimite(carga.Limit, sheet.Attributes["strength"].Total),
	}
}

// aLarguraDaBarra é a porcentagem já presa em 100.
func aLarguraDaBarra(usado float64, limite int) int {
	if limite <= 0 {
		return 0
	}
	percent := int(usado * 100 / float64(limite))
	if percent > 100 {
		return 100
	}
	return percent
}

// oRotuloDoLimite mostra a CONTA que produziu o limite, com o valor de Força
// resolvido — e não a notação "10 + 2×|FOR|", que manda a pessoa fazer a conta
// de cabeça para conferir o número que já está do lado.
func oRotuloDoLimite(limite, forca int) string {
	return "limite " + strconv.Itoa(limite) + " · 10 + 2×For " + comSinalInt(forca)
}

// moneyLineOf escreve o dinheiro e o espaço que ele ocupa.
func moneyLineOf(dto sheet.CharacterDTO) moneyLine {
	linha := moneyLine{Tibar: comVirgula(dto.Tibar)}
	if espacos := espacosDeMoeda(dto.Tibar); espacos > 0 {
		linha.Slots = comVirgula(espacos) + oPluralDoEspaco(espacos)
	}
	return linha
}

// espacosDeMoeda são os milheiros COMPLETOS: "cada 1.000 moedas ocupam um
// espaço" (p141), e 1.999 T$ ocupam um espaço e não dois.
func espacosDeMoeda(tibar float64) float64 {
	return float64(int(tibar) / int(engine.CoinsPerSlot))
}

func oPluralDoEspaco(espacos float64) string {
	if espacos == 1 {
		return " espaço"
	}
	return " espaços"
}

// comVirgula escreve o número como a mesa escreve: sem casa decimal quando ele
// é inteiro, e com VÍRGULA quando não é.
func comVirgula(valor float64) string {
	if valor == float64(int64(valor)) {
		return strconv.FormatInt(int64(valor), 10)
	}
	return strings.Replace(strconv.FormatFloat(valor, 'f', -1, 64), ".", ",", 1)
}

// asMelhoriasOrdenadas devolve as sobreposições do item ordenadas por nome,
// para a ficha do item listá-las sempre na mesma ordem.
func asMelhoriasOrdenadas(item sheet.ItemDTO) []book.Item {
	entradas := asSobreposicoesDoLivro(item)
	sort.SliceStable(entradas, func(a, b int) bool { return entradas[a].Name < entradas[b].Name })
	return entradas
}

// ── o que a TELA precisa escrever ────────────────────────────────────────────

// osItensEscrito é "3 itens", com o singular certo.
func osItensEscrito(n int) string {
	if n == 1 {
		return "1 item"
	}
	return strconv.Itoa(n) + " itens"
}

// juntoComPonto é a lista de sobreposições numa linha só.
func juntoComPonto(nomes []string) string {
	return strings.Join(nomes, " · ")
}

// oAvisoDaSobrecarga diz o que a sobrecarga CUSTA, com os dois números que o
// motor já aplicou — e não uma frase decorada que pode divergir deles.
//
// A segunda metade só aparece acima do DOBRO do limite, que é o outro teto da
// p141: "você não pode carregar mais do que o dobro do seu limite". O motor não
// recusa a linha, porque o próprio livro deixa a carga a critério do mestre —
// então quem diz que passou é a tela.
func oAvisoDaSobrecarga(load loadMeter) string {
	aviso := "Sobrecarregado (p141): " + load.ArmorPenalty +
		" em Acrobacia, Furtividade e Ladinagem · " + load.DisplacementPenalty + "m de deslocamento"
	if load.OverMax {
		aviso += " · acima de " + strconv.Itoa(load.Max) + " espaços o livro diz que não dá para carregar"
	}
	return aviso
}

// osModosDoDinheiro são as três coisas que se fazem com dinheiro na mesa
// (ALE-224): "achamos 350 no baú", "paguei 80 pela estalagem", e escrever o
// total — que é o gesto da forja (Tabela 3-1, p140) e o de consertar um erro de
// digitação.
var osModosDoDinheiro = []filterOption{
	{Valor: "receber", Rotulo: "Receber"},
	{Valor: "gastar", Rotulo: "Gastar"},
	{Valor: "corrigir", Rotulo: "Corrigir"},
}
