package api

import (
	"sort"
	"strconv"
	"strings"
)

// OS DIÁLOGOS DA MOCHILA como dado (ALE-272, fatia 7).
//
// Cada item da ficha tem UMA ficha de item, e ela é o alvo de todo toque — o
// ladrilho da grade e o cartão da tira abrem a mesma caixa. É a decisão da tela
// antiga e ela se sustenta: as ações de um item são as mesmas onde quer que ele
// esteja, e dois caminhos diferentes para "usar a poção" seriam dois lugares
// para esquecer de consertar.
//
// # O que a ficha do item mostra
//
// O que ele PESA, o que ele CONCEDE, o que o livro diz dele, as melhorias
// aplicadas — e os botões do que dá para fazer: equipar onde cabe, usar (se for
// consumível), pôr melhoria, editar e remover.

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
	Slot   string
	Rotulo string
}

// consumeChoice é o "Usar" de um consumível.
type consumeChoice struct {
	// Escopo é imediato, 1 cena ou 1 dia, na palavra da mesa.
	Escopo string
	// HpDice e MpDice são as rolagens que a MESA faz. A ficha não rola por
	// ninguém: o dado é da pessoa, e o servidor aceita o número que ela mandar.
	HpDice string
	MpDice string
}

// bookInfo é o que o livro diz do item.
type bookInfo struct {
	Categoria string
	Preco     string
	Pagina    int
	Linhas    []string
}

type overlayRow struct {
	Nome   string
	Efeito string
}

// overlayChoice é uma melhoria ou material que CABE neste item.
type overlayChoice struct {
	ID     string
	Nome   string
	Efeito string
	Preco  string
	Ativa  bool
}

// catalogItemRow é uma linha do catálogo no diálogo de adicionar.
type catalogItemRow struct {
	ID        string
	Nome      string
	Categoria string
	Espacos   string
	Preco     string
	Pagina    int
}

// itemSheetsOf monta uma ficha por item da mochila.
func itemSheetsOf(dto CharacterDTO, proficiencias map[string]bool) []itemSheet {
	fichas := make([]itemSheet, 0, len(dto.Items))
	for _, item := range dto.Items {
		fichas = append(fichas, itemSheetOf(item, proficiencias))
	}
	return fichas
}

func itemSheetOf(item ItemDTO, proficiencias map[string]bool) itemSheet {
	catalogo := itemDoCatalogo(item)
	ficha := itemSheet{
		ID: item.ID, Name: item.Name, Quantity: item.Quantity,
		Slots:         comVirgula(item.Slots),
		Total:         comVirgula(float64(item.Quantity) * item.Slots),
		NoProficiency: !ehProficiente(item, proficiencias),
		Equip:         osLugaresAlcancaveis(item, catalogo),
		Overlays:      asMelhoriasAplicadas(item),
		Command:       strconv.FormatInt(item.ID, 10),
	}
	if catalogo == nil {
		return ficha
	}
	ficha.Book = oQueOLivroDiz(*catalogo)
	ficha.Consumable = oUsoDoConsumivel(*catalogo)
	ficha.Overlayable = aceitaMelhoria(*catalogo)
	if ficha.Overlayable {
		aplicadas := asMelhoriasGuardadas(item.Improvements)
		familia := aFamiliaDoItem(*catalogo)
		ficha.Improvements = asSobreposicoesQueCabem("improvement", familia, aplicadas)
		ficha.Materials = asSobreposicoesQueCabem("material", familia, oMaterialAplicado(item))
	}
	return ficha
}

// oMaterialAplicado é o material como lista, para a comparação ser uma só.
func oMaterialAplicado(item ItemDTO) []string {
	if item.Material == nil || *item.Material == "" {
		return nil
	}
	return []string{*item.Material}
}

// osLugaresAlcancaveis são os estados de equipar que fazem sentido, MENOS o
// atual — oferecer "Guardar" a um item já guardado é um botão que não faz nada.
//
// Item custom não tem eixo no catálogo, então ele aceita os três: não há o que
// saber sobre uma coisa que a pessoa inventou, e recusar por precaução tiraria
// dela a única forma de equipar o que ela criou.
func osLugaresAlcancaveis(item ItemDTO, catalogo *itemDoLivro) []equipChoice {
	atual := equippedSlotOf(item)
	fora := []equipChoice{}
	for _, escolha := range osLugaresDoItem(catalogo) {
		if escolha.Slot != atual {
			fora = append(fora, escolha)
		}
	}
	return fora
}

// osLugaresDoItem lê o EIXO do livro — `vested`, `wielded` ou `either` — e
// devolve o que cabe, incluindo o guardar.
//
// As duas mãos só aparecem quando são OBRIGATÓRIAS (`hands: 2`) ou quando
// mudam alguma coisa: uma arma versátil dá mais dano empunhada com as duas
// (p150). Numa arma de uma mão só, ocupar as duas não ganha nada.
func osLugaresDoItem(catalogo *itemDoLivro) []equipChoice {
	guardar := equipChoice{Slot: "", Rotulo: "Guardar"}
	if catalogo == nil {
		return []equipChoice{guardar, {Slot: "vested", Rotulo: "Vestir"},
			{Slot: "wielded", Rotulo: "Empunhar (1 mão)"}, {Slot: "wielded2", Rotulo: "Empunhar (2 mãos)"}}
	}
	if catalogo.Category == "consumable" || catalogo.Category == "meal" {
		return []equipChoice{guardar}
	}
	if catalogo.Equip == "vested" {
		return []equipChoice{guardar, {Slot: "vested", Rotulo: "Vestir"}}
	}
	maos := asMaosDoItem(*catalogo)
	if catalogo.Equip == "wielded" {
		return append([]equipChoice{guardar}, maos...)
	}
	return append([]equipChoice{guardar, {Slot: "vested", Rotulo: "Vestir"}}, maos...)
}

func asMaosDoItem(catalogo itemDoLivro) []equipChoice {
	duas := equipChoice{Slot: "wielded2", Rotulo: "Empunhar (2 mãos)"}
	if catalogo.Hands == 2 {
		return []equipChoice{duas}
	}
	uma := equipChoice{Slot: "wielded", Rotulo: "Empunhar (1 mão)"}
	if catalogo.Weapon != nil && contemTraco(catalogo.Weapon.Traits, "versatil") {
		return []equipChoice{uma, duas}
	}
	return []equipChoice{uma}
}

func contemTraco(tracos []string, alvo string) bool {
	for _, t := range tracos {
		if t == alvo {
			return true
		}
	}
	return false
}

// oUsoDoConsumivel descreve a dose, ou nil quando o item não se usa.
func oUsoDoConsumivel(catalogo itemDoLivro) *consumeChoice {
	if catalogo.Consumable == nil {
		return nil
	}
	uso := &consumeChoice{Escopo: oEscopoEscrito(catalogo.Consumable.Scope)}
	if imediato := catalogo.Consumable.Instant; imediato != nil && catalogo.Consumable.Scope == "instant" {
		uso.HpDice = aRolagemQuePedeNumero(imediato.HP)
		uso.MpDice = aRolagemQuePedeNumero(imediato.MP)
	}
	return uso
}

var osEscoposDoConsumivel = map[string]string{
	"instant": "imediato", "scene": "1 cena", "day": "1 dia",
}

func oEscopoEscrito(escopo string) string {
	if nome, tem := osEscoposDoConsumivel[escopo]; tem {
		return nome
	}
	return escopo
}

// aRolagemQuePedeNumero é o dado que a MESA rola, ou "" quando o ganho é fixo.
//
// Ganho fixo não pergunta nada: perguntar o resultado de um dado que não existe
// é pedir que a pessoa invente um número.
func aRolagemQuePedeNumero(ganho *rolagemDoGanho) string {
	if ganho == nil || ganho.Dice == "" || ganho.Dice == "0" {
		return ""
	}
	return ganho.Dice
}

// oQueOLivroDiz é o bloco de referência da ficha do item.
func oQueOLivroDiz(catalogo itemDoLivro) *bookInfo {
	info := &bookInfo{
		Categoria: aCategoriaEscrita(catalogo.Category),
		Preco:     comVirgula(catalogo.Price),
		Pagina:    catalogo.BookPage,
	}
	if arma := catalogo.Weapon; arma != nil {
		info.Linhas = append(info.Linhas, "dano "+arma.Damage+" · crítico "+
			strconv.Itoa(arma.CritRange)+"/×"+strconv.Itoa(arma.CritMult))
		if arma.Type != "" {
			info.Linhas = append(info.Linhas, "tipo "+oTipoDeDanoEscrito(arma.Type))
		}
	}
	if protecao := catalogo.Armor; protecao != nil {
		info.Linhas = append(info.Linhas, aLinhaDaProtecao(*protecao, true))
	}
	if protecao := catalogo.Shield; protecao != nil {
		info.Linhas = append(info.Linhas, aLinhaDaProtecao(*protecao, false))
	}
	for _, m := range catalogo.Modifiers {
		info.Linhas = append(info.Linhas, oCrachaDoModificador(m))
	}
	info.Linhas = semRepetidos(info.Linhas)
	return info
}

func aLinhaDaProtecao(protecao protecaoDoLivro, ehArmadura bool) string {
	linha := "Defesa " + comSinalInt(protecao.Defense) + " · penalidade " + strconv.Itoa(protecao.Penalty)
	if !ehArmadura {
		return linha
	}
	if protecao.Heavy {
		return linha + " · pesada"
	}
	return linha + " · leve"
}

// osTiposDeDanoEscritos é o pt-BR do tipo de dano da arma.
//
// O catálogo guarda a CHAVE sem acento (`perfuracao`), como todo id deste
// projeto; quem lê a ficha lê a palavra do livro. Sem esta tabela a tela
// mostrava "tipo perfuracao", que é o id cru vazando para o jogador — medido na
// bancada, na ficha da Adaga.
var osTiposDeDanoEscritos = map[string]string{
	"corte": "corte", "perfuracao": "perfuração", "impacto": "impacto",
	"corte-perfuracao": "corte ou perfuração",
}

func oTipoDeDanoEscrito(tipo string) string {
	if nome, tem := osTiposDeDanoEscritos[tipo]; tem {
		return nome
	}
	return tipo
}

// asCategoriasEscritas é o pt-BR de cada categoria do catálogo.
//
// Uma categoria sem tradução cai no próprio id, que é feio e VISÍVEL — melhor
// que sumir da tela, que é o que uma queda para vazio faria.
var asCategoriasEscritas = map[string]string{
	"animal": "Animal", "apparel": "Vestuário", "armor-heavy": "Armadura pesada",
	"armor-light": "Armadura leve", "catalyst": "Catalisador", "consumable": "Consumível",
	"improvement": "Melhoria", "material": "Material", "meal": "Alimentação",
	"shield": "Escudo", "vehicle": "Veículo", "weapon-exotic": "Arma exótica",
	"weapon-firearm": "Arma de fogo", "weapon-martial": "Arma marcial",
	"weapon-simple": "Arma simples",
}

func aCategoriaEscrita(id string) string {
	if nome, tem := asCategoriasEscritas[id]; tem {
		return nome
	}
	return id
}

// asMelhoriasAplicadas são as sobreposições em vigor, com o que elas fazem.
func asMelhoriasAplicadas(item ItemDTO) []overlayRow {
	linhas := []overlayRow{}
	for _, entrada := range asMelhoriasOrdenadas(item) {
		linhas = append(linhas, overlayRow{Nome: entrada.Name, Efeito: oResumoDaSobreposicao(entrada)})
	}
	return linhas
}

// oResumoDaSobreposicao junta as notas do catálogo numa linha, SEM repetir.
//
// A Equilibrada carrega quatro modificadores de manobra que dividem a mesma
// nota "+2 em manobras"; juntá-las cruas escrevia a frase quatro vezes.
func oResumoDaSobreposicao(entrada itemDoLivro) string {
	notas := []string{}
	for _, m := range entrada.Modifiers {
		if m.Note != "" {
			notas = append(notas, m.Note)
		}
	}
	if resumo := strings.Join(semRepetidos(notas), ", "); resumo != "" {
		return resumo
	}
	return "sem efeito mecânico"
}

// asSobreposicoesQueCabem são as melhorias (ou materiais) da FAMÍLIA do item.
//
// O filtro é o `appliesTo` do catálogo, e ele é a mesma regra que o servidor
// cobra ao gravar (`aMelhoriaCabeNoItem`): a lista mostra o que cabe, e quem
// recusa o resto é o servidor.
func asSobreposicoesQueCabem(categoria, familia string, aplicadas []string) []overlayChoice {
	escolhas := []overlayChoice{}
	for _, entrada := range catalogosDoLivro().Itens {
		if entrada.Category != categoria || !aceitaAFamilia(entrada, familia) {
			continue
		}
		escolhas = append(escolhas, overlayChoice{
			ID: entrada.ID, Nome: entrada.Name, Efeito: oResumoDaSobreposicao(entrada),
			Preco: comVirgula(entrada.Price), Ativa: contemTraco(aplicadas, entrada.ID),
		})
	}
	sort.SliceStable(escolhas, func(a, b int) bool { return escolhas[a].Nome < escolhas[b].Nome })
	return escolhas
}

// catalogItemRowsOf são as entradas do catálogo que o diálogo de adicionar
// mostra, já filtradas pela busca e pela categoria.
//
// As melhorias e os materiais ficam de FORA: eles não são itens que se carrega,
// são coisas que se aplicam a um item — e quem as aplica é o diálogo de
// melhorias, que já filtra pela família. Ofertá-las aqui deixaria a pessoa pôr
// um "Aço-rubi" solto na mochila.
func catalogItemRowsOf(busca, categoria string) []catalogItemRow {
	termo := foldAccents(strings.TrimSpace(busca))
	linhas := []catalogItemRow{}
	for _, entrada := range catalogosDoLivro().Itens {
		if entrada.Category == "improvement" || entrada.Category == "material" {
			continue
		}
		if categoria != "" && entrada.Category != categoria {
			continue
		}
		if termo != "" && !strings.Contains(foldAccents(entrada.Name), termo) &&
			!strings.Contains(foldAccents(aCategoriaEscrita(entrada.Category)), termo) {
			continue
		}
		linhas = append(linhas, catalogItemRow{
			ID: entrada.ID, Nome: entrada.Name, Categoria: aCategoriaEscrita(entrada.Category),
			Espacos: comVirgula(entrada.Slots), Preco: comVirgula(entrada.Price), Pagina: entrada.BookPage,
		})
	}
	return linhas
}

// asCategoriasDoCatalogo são as opções do seletor do diálogo de adicionar,
// lidas do próprio catálogo — uma lista escrita à mão envelheceria calada.
func asCategoriasDoCatalogo(ativa string) []filterOption {
	vistas := map[string]bool{}
	opcoes := []filterOption{{Valor: "", Rotulo: "Todas as categorias", Ativo: ativa == ""}}
	ids := []string{}
	for _, entrada := range catalogosDoLivro().Itens {
		if entrada.Category == "improvement" || entrada.Category == "material" || vistas[entrada.Category] {
			continue
		}
		vistas[entrada.Category] = true
		ids = append(ids, entrada.Category)
	}
	sort.Strings(ids)
	for _, id := range ids {
		opcoes = append(opcoes, filterOption{Valor: id, Rotulo: aCategoriaEscrita(id), Ativo: id == ativa})
	}
	return opcoes
}
