package sheetui

import (
	"sort"
	"strconv"
	"strings"
	"t20engine/domain/book"
	"t20engine/domain/search"
	"t20engine/domain/sheet"
)

// OS DIÁLOGOS DA MOCHILA como dado.
//
// Cada item da ficha tem UMA ficha de item, e ela é o alvo de todo toque — o
// ladrilho da grade e o cartão da tira abrem a mesma caixa: as ações de um item
// são as mesmas onde quer que ele esteja, e dois caminhos diferentes para "usar
// a poção" seriam dois lugares para esquecer de consertar.

// itemSheet é a ficha de UM item, pronta para desenhar.
type itemSheet struct {
	ID       int64
	Name     string
	Quantity int64
	Slots    string
	// Total é quantidade × espaços, que é o que a carga conta.
	Total         string
	NoProficiency bool
	// Equip são os lugares ALCANÇÁVEIS, sem o atual: um item guardado não
	// oferece "Guardar", e um empunhado não oferece a mesma mão de novo.
	Equip []equipChoice
	// Consumable existe só para o que se usa, e diz se a mesa precisa rolar.
	Consumable *consumeChoice
	Book       *bookInfo
	// Overlays são as melhorias e o material JÁ aplicados, com o que fazem.
	Overlays []overlayRow
	// Overlayable diz se o item ACEITA melhoria: não se forja uma poção em
	// aço-rubi, e um diálogo com duas listas vazias é pior que botão nenhum.
	Overlayable  bool
	Improvements []overlayChoice
	Materials    []overlayChoice
	// Command é o id como texto, para montar o `@post` sem conversão na tela.
	Command string
}

type equipChoice struct {
	Slot  string
	Label string
}

// consumeChoice é o "Usar" de um consumível.
type consumeChoice struct {
	// Scope é imediato, 1 cena ou 1 dia, na palavra da mesa.
	Scope string
	// HpDice e MpDice são as rolagens que a MESA faz. A ficha não rola por
	// ninguém: o dado é da pessoa, e o servidor aceita o número que ela mandar.
	HpDice string
	MpDice string
}

// bookInfo é o que o livro diz do item.
type bookInfo struct {
	Category string
	Price    string
	Page     int
	Rows     []string
}

type overlayRow struct {
	Name   string
	Effect string
}

// overlayChoice é uma melhoria ou material que CABE neste item.
type overlayChoice struct {
	ID     string
	Name   string
	Effect string
	Price  string
	Active bool
}

// catalogItemRow é uma linha do catálogo no diálogo de adicionar.
type catalogItemRow struct {
	ID       string
	Name     string
	Category string
	Slots    string
	Price    string
	Page     int
}

func itemSheetsOf(dto sheet.CharacterDTO, proficiencies map[string]bool) []itemSheet {
	sheets := make([]itemSheet, 0, len(dto.Items))
	for _, item := range dto.Items {
		sheets = append(sheets, itemSheetOf(item, proficiencies))
	}
	return sheets
}

func itemSheetOf(item sheet.ItemDTO, proficiencies map[string]bool) itemSheet {
	catalog := catalogItem(item)
	character := itemSheet{
		ID: item.ID, Name: item.Name, Quantity: item.Quantity,
		Slots:         sheet.WithComma(item.Slots),
		Total:         sheet.WithComma(float64(item.Quantity) * item.Slots),
		NoProficiency: !proficienteEh(item, proficiencies),
		Equip:         reachablePlaces(item, catalog),
		Overlays:      appliedImprovements(item),
		Command:       strconv.FormatInt(item.ID, 10),
	}
	if catalog == nil {
		return character
	}
	character.Book = thatSaysBook(*catalog)
	character.Consumable = consumableUse(*catalog)
	character.Overlayable = aceitaMelhoria(*catalog)
	if character.Overlayable {
		applied := savedImprovements(item.Improvements)
		family := itemFamily(*catalog)
		character.Improvements = thatFitOverlays("improvement", family, applied)
		character.Materials = thatFitOverlays("material", family, appliedMaterial(item))
	}
	return character
}

// appliedMaterial é o material como lista, para a comparação ser uma só.
func appliedMaterial(item sheet.ItemDTO) []string {
	if item.Material == nil || *item.Material == "" {
		return nil
	}
	return []string{*item.Material}
}

// reachablePlaces são os estados de equipar que fazem sentido, MENOS o
// atual — oferecer "Guardar" a um item já guardado é um botão que não faz nada.
//
// Item custom não tem eixo no catálogo, então ele aceita os três: não há o que
// saber sobre uma coisa que a pessoa inventou, e recusar por precaução tiraria
// dela a única forma de equipar o que ela criou.
func reachablePlaces(item sheet.ItemDTO, catalog *book.Item) []equipChoice {
	current := equippedSlotOf(item)
	outside := []equipChoice{}
	for _, choice := range itemPlaces(catalog) {
		if choice.Slot != current {
			outside = append(outside, choice)
		}
	}
	return outside
}

// itemPlaces lê o EIXO do livro — `vested`, `wielded` ou `either` — e
// devolve o que cabe, incluindo o guardar.
//
// As duas mãos só aparecem quando são OBRIGATÓRIAS (`hands: 2`) ou quando
// mudam alguma coisa: uma arma versátil dá mais dano empunhada com as duas
// (p150). Numa arma de uma mão só, ocupar as duas não ganha nada.
func itemPlaces(catalog *book.Item) []equipChoice {
	save := equipChoice{Slot: "", Label: "Guardar"}
	if catalog == nil {
		return []equipChoice{save, {Slot: "vested", Label: "Vestir"},
			{Slot: "wielded", Label: "Empunhar (1 mão)"}, {Slot: "wielded2", Label: "Empunhar (2 mãos)"}}
	}
	if catalog.Category == "consumable" || catalog.Category == "meal" {
		return []equipChoice{save}
	}
	if catalog.Equip == "vested" {
		return []equipChoice{save, {Slot: "vested", Label: "Vestir"}}
	}
	hands := itemHands(*catalog)
	if catalog.Equip == "wielded" {
		return append([]equipChoice{save}, hands...)
	}
	return append([]equipChoice{save, {Slot: "vested", Label: "Vestir"}}, hands...)
}

func itemHands(catalog book.Item) []equipChoice {
	two := equipChoice{Slot: "wielded2", Label: "Empunhar (2 mãos)"}
	if catalog.Hands == 2 {
		return []equipChoice{two}
	}
	one := equipChoice{Slot: "wielded", Label: "Empunhar (1 mão)"}
	if catalog.Weapon != nil && contemTraco(catalog.Weapon.Traits, "versatil") {
		return []equipChoice{one, two}
	}
	return []equipChoice{one}
}

func contemTraco(traits []string, target string) bool {
	for _, t := range traits {
		if t == target {
			return true
		}
	}
	return false
}

// consumableUse descreve a dose, ou nil quando o item não se usa.
func consumableUse(catalog book.Item) *consumeChoice {
	if catalog.Consumable == nil {
		return nil
	}
	use := &consumeChoice{Scope: writtenScope(catalog.Consumable.Scope)}
	if immediate := catalog.Consumable.Instant; immediate != nil && catalog.Consumable.Scope == "instant" {
		use.HpDice = rollThatAsksANumber(immediate.HP)
		use.MpDice = rollThatAsksANumber(immediate.MP)
	}
	return use
}

var consumableScopes = map[string]string{
	"instant": "imediato", "scene": "1 cena", "day": "1 dia",
}

func writtenScope(scope string) string {
	if name, found := consumableScopes[scope]; found {
		return name
	}
	return scope
}

// rollThatAsksANumber é o dado que a MESA rola, ou "" quando o ganho é fixo.
//
// Ganho fixo não pergunta nada: perguntar o resultado de um dado que não existe
// é pedir que a pessoa invente um número.
func rollThatAsksANumber(gain *book.GainRoll) string {
	if gain == nil || gain.Dice == "" || gain.Dice == "0" {
		return ""
	}
	return gain.Dice
}

// thatSaysBook é o bloco de referência da ficha do item.
func thatSaysBook(catalog book.Item) *bookInfo {
	info := &bookInfo{
		Category: writtenCategory(catalog.Category),
		Price:    sheet.WithComma(catalog.Price),
		Page:     catalog.BookPage,
	}
	if weapon := catalog.Weapon; weapon != nil {
		info.Rows = append(info.Rows, "dano "+weapon.Damage+" · crítico "+
			strconv.Itoa(weapon.CritRange)+"/×"+strconv.Itoa(weapon.CritMult))
		if weapon.Type != "" {
			info.Rows = append(info.Rows, "tipo "+damageWrittenKind(weapon.Type))
		}
	}
	if protection := catalog.Armor; protection != nil {
		info.Rows = append(info.Rows, protectionRow(*protection, true))
	}
	if protection := catalog.Shield; protection != nil {
		info.Rows = append(info.Rows, protectionRow(*protection, false))
	}
	for _, m := range catalog.Modifiers {
		info.Rows = append(info.Rows, modifierBadge(m))
	}
	info.Rows = repetidosSem(info.Rows)
	return info
}

func protectionRow(protection book.Armor, isArmor bool) string {
	row := "Defesa " + book.WithSign(protection.Defense) + " · penalidade " + strconv.Itoa(protection.Penalty)
	if !isArmor {
		return row
	}
	if protection.Heavy {
		return row + " · pesada"
	}
	return row + " · leve"
}

// damageWrittenKinds é o pt-BR do tipo de dano da arma.
//
// O catálogo guarda a CHAVE sem acento (`perfuracao`), como todo id deste
// projeto; sem esta tabela o id cru vaza para o jogador ("tipo perfuracao").
var damageWrittenKinds = map[string]string{
	"corte": "corte", "perfuracao": "perfuração", "impacto": "impacto",
	"corte-perfuracao": "corte ou perfuração",
}

func damageWrittenKind(kind string) string {
	if name, found := damageWrittenKinds[kind]; found {
		return name
	}
	return kind
}

// writtenCategories é o pt-BR de cada categoria do catálogo.
//
// Uma categoria sem tradução cai no próprio id, que é feio e VISÍVEL — melhor
// que sumir da tela, que é o que uma queda para vazio faria.
var writtenCategories = map[string]string{
	"animal": "Animal", "apparel": "Vestuário", "armor-heavy": "Armadura pesada",
	"armor-light": "Armadura leve", "catalyst": "Catalisador", "consumable": "Consumível",
	"improvement": "Melhoria", "material": "Material", "meal": "Alimentação",
	"shield": "Escudo", "vehicle": "Veículo", "weapon-exotic": "Arma exótica",
	"weapon-firearm": "Arma de fogo", "weapon-martial": "Arma marcial",
	"weapon-simple": "Arma simples",
}

func writtenCategory(id string) string {
	if name, found := writtenCategories[id]; found {
		return name
	}
	return id
}

// appliedImprovements são as sobreposições em vigor, com o que elas fazem.
func appliedImprovements(item sheet.ItemDTO) []overlayRow {
	rows := []overlayRow{}
	for _, entry := range sortedImprovements(item) {
		rows = append(rows, overlayRow{Name: entry.Name, Effect: overlaySummary(entry)})
	}
	return rows
}

// overlaySummary junta as notas do catálogo numa linha, SEM repetir.
//
// A Equilibrada carrega quatro modificadores de manobra que dividem a mesma
// nota "+2 em manobras"; juntá-las cruas escrevia a frase quatro vezes.
func overlaySummary(entry book.Item) string {
	notes := []string{}
	for _, m := range entry.Modifiers {
		if m.Note != "" {
			notes = append(notes, m.Note)
		}
	}
	if summary := strings.Join(repetidosSem(notes), ", "); summary != "" {
		return summary
	}
	return "sem efeito mecânico"
}

// thatFitOverlays são as melhorias (ou materiais) da FAMÍLIA do item.
//
// O filtro é o `appliesTo` do catálogo, e ele é a mesma regra que o servidor
// cobra ao gravar (`fitsItemImprovement`): a lista mostra o que cabe, e quem
// recusa o resto é o servidor.
func thatFitOverlays(category, family string, applied []string) []overlayChoice {
	choices := []overlayChoice{}
	for _, entry := range book.Catalogs().Items {
		if entry.Category != category || !aceitaAFamilia(entry, family) {
			continue
		}
		choices = append(choices, overlayChoice{
			ID: entry.ID, Name: entry.Name, Effect: overlaySummary(entry),
			Price: sheet.WithComma(entry.Price), Active: contemTraco(applied, entry.ID),
		})
	}
	sort.SliceStable(choices, func(a, b int) bool { return choices[a].Name < choices[b].Name })
	return choices
}

// catalogItemRowsOf são as entradas do catálogo que o diálogo de adicionar
// mostra, já filtradas pela busca e pela categoria.
//
// As melhorias e os materiais ficam de FORA: eles não são itens que se carrega,
// são coisas que se aplicam a um item — e quem as aplica é o diálogo de
// melhorias, que já filtra pela família. Ofertá-las aqui deixaria a pessoa pôr
// um "Aço-rubi" solto na mochila.
func catalogItemRowsOf(query, category string) []catalogItemRow {
	term := search.Fold(strings.TrimSpace(query))
	rows := []catalogItemRow{}
	for _, entry := range book.Catalogs().Items {
		if entry.Category == "improvement" || entry.Category == "material" {
			continue
		}
		if category != "" && entry.Category != category {
			continue
		}
		if term != "" && !strings.Contains(search.Fold(entry.Name), term) &&
			!strings.Contains(search.Fold(writtenCategory(entry.Category)), term) {
			continue
		}
		rows = append(rows, catalogItemRow{
			ID: entry.ID, Name: entry.Name, Category: writtenCategory(entry.Category),
			Slots: sheet.WithComma(entry.Slots), Price: sheet.WithComma(entry.Price), Page: entry.BookPage,
		})
	}
	return rows
}

// catalogCategories são as opções do seletor do diálogo de adicionar,
// lidas do próprio catálogo — uma lista escrita à mão envelheceria calada.
func catalogCategories(active string) []filterOption {
	seen := map[string]bool{}
	options := []filterOption{{Value: "", Label: "Todas as categorias", Active: active == ""}}
	ids := []string{}
	for _, entry := range book.Catalogs().Items {
		if entry.Category == "improvement" || entry.Category == "material" || seen[entry.Category] {
			continue
		}
		seen[entry.Category] = true
		ids = append(ids, entry.Category)
	}
	sort.Strings(ids)
	for _, id := range ids {
		options = append(options, filterOption{Value: id, Label: writtenCategory(id), Active: id == active})
	}
	return options
}
