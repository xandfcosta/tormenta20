package sheetui

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
	"t20engine/serve/web/characters"
	"t20engine/serve/web/ui"
)

// A FICHA como dado: a casca, as abas e o crachá.
//
// Ela chega COMPUTADA DO SERVIDOR, pela mesma `ComputeSheet` que a Mesa e a
// cena de personagens usam — sem requisito de offline, não há por que a conta
// acontecer no navegador.

// View é a ficha de um personagem pronta para desenhar.
type View struct {
	ID   int64
	Name string
	// Version é o `updatedAt` do personagem, e existe só para a ficha EMBUTIDA: o
	// ouvinte que repede a ficha compara o carimbo do stream com o que já está
	// na tela, e não pede nada quando são o mesmo. Sem isso, um gesto do próprio
	// jogador produz DOIS pedidos — o dele e o do aviso que a escrita dele
	// acabou de provocar.
	Version string
	// Draft diz que esta ficha é um rascunho, e é só um nome: ela aceita os mesmos gestos da cópia.
	Draft bool
	// Embedded diz que esta ficha está sendo desenhada DENTRO da sessão.
	//
	// Ela muda duas coisas na tela, e as duas são sobre NAVEGAÇÃO: a barra com
	// o "‹ Voltar" some (a sessão tem cabeçalho próprio, e voltar dali tiraria
	// o jogador da mesa), e as abas deixam de ser links para virar comandos que
	// remendam a ficha no lugar. Um `<a href>` ali levaria embora da sessão
	// quem só queria trocar de seção.
	Embedded bool
	// ColorMode é o modo de cor em vigor, para o seletor do ⋯ marcar o certo.
	ColorMode ui.ColorMode
	// Initials e Gradiente são o retrato derivado do nome, como no palco e no
	// cartão da campanha: o app não guarda imagem de personagem.
	Initials string
	Gradient string
	// Role é "Guerreiro 10 · Anão" — a mesma linha do cartão do elenco.
	Role    string
	Summary string
	Level   int64
	// Defense vem do MOTOR e é travessão quando não há catálogo primado: a tela
	// inteira não pode cair por causa de um número, e um zero seria pior — 0 é
	// um valor plausível, e o jogador agiria sobre ele.
	Defense string
	PV      sheetVital
	PM      sheetVital
	NoMana  bool
	// Classes é "Guerreiro 3 / Ladino 2" — a mesma linha do palco, e string e
	// não lista porque é assim que a mesa lê e é assim que o cartão já a monta.
	Classes string
	// AsClasses são as classes com o nível de cada uma, e elas existem para o
	// DEGRAU DE NÍVEL: o nível de um personagem é a SOMA dos níveis de classe, e
	// subir um nível é escolher QUAL classe o recebe.
	AsClasses []sheetClass
	// OtherClasses são as classes do livro em que ele ainda pode entrar (p35).
	OtherClasses []string
	// ByLevel é a vista "Por nível" da aba Poderes, LevelHistory o que ela mostra, e SwapDialog a
	// troca aberta. LastLevel é o que o `−` tira.
	ByLevel      bool
	LevelHistory []sheet.LevelEntry
	SwapDialog   *swapDialogView
	LastLevel    *sheet.LastLevelLoss
	// Identity é quem o personagem é, linha a linha, para o ⋯ da ficha.
	Identity []identityLine
	// Tabs são as sete da ficha, com a ativa marcada. Ver `Tabs`.
	Tabs []Tab
	// ActiveTab é o valor resolvido — nunca o que veio na URL cru, que pode ser
	// um endereço antigo ou lixo digitado.
	ActiveTab string
	// Proficiencies são os dois blocos do painel homônimo. Elas são computadas
	// SEMPRE, e não só quando a aba está aberta: são sete linhas derivadas de
	// dado que a ficha já carregou, e um `if` aqui trocaria microssegundos por um
	// ramo a mais para um guarda cobrir.
	Proficiencies []proficiencyGroup
	// Combat é a aba homônima, computada SEMPRE pela mesma razão: o motor já roda
	// uma vez por carga da ficha para a Defesa do crachá, e repartir esse
	// resultado custa menos que um `if` a mais.
	Combat panelCombat
	// Expertises é a aba homônima. Ela é a única que depende do que a pessoa
	// DIGITOU — o filtro da busca —, e por isso a `Load` recebe o termo em vez de
	// o painel ir buscá-lo.
	Expertises expertisePanel
	// Effects é a aba homônima — tudo que está mexendo nos números AGORA, em
	// quatro blocos que diferem por quem é dono do estado.
	Effects effectsPanel
	// Spells é a aba homônima — o grimório, as concedidas por poder e o catálogo
	// inteiro do Capítulo 4 para aprender.
	Spells spellbookPanel
	// GrimoireSearch, GrimoireCircle e SpellCasting são da aba Magias: a busca e o círculo que filtram o grimório, e a CD e o atributo
	// que o topo diz uma vez.
	GrimoireSearch string
	GrimoireCircle string
	SpellCasting   spellCasting
	// EffectPicks são as buscas dos escolhedores da aba Efeitos.
	EffectPicks effectPickFilters
	// Bag é a aba homônima — a tira de equipados, a carga da p141, o dinheiro e a
	// grade do que está guardado.
	Bag bagPanel
	// Powers é a aba homônima — a lista de jogo: o que se ativa em cima, o que é
	// passivo recolhido embaixo.
	Powers powersPanel
	// Actions é a superfície AÇÕES, que a MESA desenha e a ficha não: ela é o que
	// o Combate, os Poderes e as Magias já computaram, reagrupado pelo que a vez
	// custa. Vem daqui e não do pacote da mesa porque o que ela lê é tudo sheet —
	// o único pedaço que é da sessão é o orçamento do turno, e quem o tem é lá.
	Actions actionsPanel
	// Choices é o diálogo de escolher poderes — a administração da ficha.
	Choices choicesPanel
	// Refusal é a frase de uma regra que barrou o gesto — o teto de duas mãos, o
	// PM que falta, a magia que não está preparada. Vazia no caminho normal.
	// Ela vem com a cena INTEIRA redesenhada, que é o que mostra que nada mudou.
	Refusal string
}

// sheetClass é uma classe do personagem, com o que o degrau precisa saber.
type sheetClass struct {
	Name  string
	Level int64
	// CanRaise e CanLower são a elegibilidade do livro, e elas são POR CLASSE
	// e não do personagem: a primeira classe não desce do 1º nível, e subir com
	// o total em 20 (p32) passaria do teto.
	CanRaise bool
	CanLower bool
	// Leaves diz que descer tira a classe da ficha: ela não é a primeira e está no 1º
	// nível. O gesto pergunta antes, porque um toque não apaga uma classe.
	Leaves bool
}

// Tab é uma das sete seções da ficha.
type Tab struct {
	// Value é o que vai na URL — ver `Tabs`.
	Value  string
	Label  string
	Icon   string
	Active bool
}

// Tabs são as sete seções, na ordem em que aparecem.
//
// OS VALORES SÃO ENDEREÇO GUARDADO, e não se "arrumam": `?tab=abilities`
// continua sendo Poderes apesar do renome Habilidades→Poderes, porque link
// compartilhado e favorito apontam para ele. O mesmo vale para a chave `tab`
// estar em inglês enquanto a tela fala português — ela é FRONTEIRA (GLOSSARY
// §F), e trocá-la quebraria endereços para ganhar estética.
//
// A primeira é o padrão de quem chega sem `?tab=`.
func Tabs() []Tab {
	return []Tab{
		{Value: "expertises", Label: "Perícias", Icon: "Scroll"},
		{Value: "combat", Label: "Combate", Icon: "Swords"},
		{Value: "bag", Label: "Mochila", Icon: "Backpack"},
		{Value: "proficiencies", Label: "Proficiências", Icon: "ShieldCheck"},
		{Value: "conditionals", Label: "Efeitos", Icon: "Zap"},
		{Value: "abilities", Label: "Poderes", Icon: "Star"},
		{Value: "spells", Label: "Magias", Icon: "BookMarked"},
	}
}

// AskedTab resolve o que veio na URL contra as sete que existem. Valor
// desconhecido cai na PRIMEIRA, em vez de dar 404: `?tab=` é endereço, e alguém
// o digita errado.
func AskedTab(raw string) string {
	for _, aba := range Tabs() {
		if aba.Value == raw {
			return raw
		}
	}
	return Tabs()[0].Value
}

// Load monta a ficha de um personagem para desenhar.
//
// A POSSE é conferida aqui, como em toda rota de personagem: a cena não ganha
// uma segunda regra sobre quem pode ver a ficha de quem.
//
// Ela é exportada porque a MESA a chama: o jogador vê a própria ficha dentro da
// sessão, e o painel é o MESMO desenho, parametrizado por `View.Embutida`. A
// cena diz como montar a si mesma, e quem compõe é o hospedeiro; a alternativa
// era um segundo desenho da ficha mantido em dois lugares.
//
// O status é o do HTTP porque quem chama responde por HTTP; a cena não escreve
// resposta nenhuma.
func (s Scene) Load(
	ctx context.Context, userID int64, id int64, aba, search string, signals Signals,
) (View, int, error) {
	row, err := s.deps.Queries().GetCharacter(ctx, id)
	if err != nil {
		return View{}, 404, fmt.Errorf("personagem %d não existe", id)
	}
	if row.Ownerid != userID {
		return View{}, 403, fmt.Errorf("esta ficha não é sua")
	}
	dto, err := s.deps.LoadCharacter(ctx, row)
	if err != nil {
		return View{}, 500, err
	}
	card := characters.HeroCardOf(s.deps.Catalogs(), dto)
	v := View{
		ID:           dto.ID,
		Draft:        !dto.CampaignID.Valid,
		Name:         dto.Name,
		Version:      row.Updatedat,
		Initials:     card.Monogram,
		Gradient:     card.Gradient,
		Role:         card.Role,
		Summary:      card.Summary,
		Level:        dto.Level,
		Defense:      card.DefenseVs,
		PV:           withTempPool(downed(vital(dto.HpCurrent, dto.HpMax), dto.ActiveConditions), sheet.TempHpTarget, dto.ActiveEffects),
		PM:           withTempPool(vital(dto.MpCurrent, dto.MpMax), sheet.TempMpTarget, dto.ActiveEffects),
		NoMana:       dto.MpMax == 0,
		Classes:      card.Classes,
		AsClasses:    stepClasses(dto),
		OtherClasses: otherClasses(dto),
		ByLevel:      signals.ByLevel,
		LevelHistory: sheet.LevelHistory(dto),
		SwapDialog:   swapDialogOf(dto, signals.SwapRow),
		LastLevel:    lastLevelOf(dto),
		Identity:     identityOf(dto, card.Classes),
		ActiveTab:    aba,

		Proficiencies: proficiencyGroupsOf(dto),
	}
	v.Effects = s.effectsPanelOf(dto)
	v.Spells = s.spellbookPanelOf(dto, signals.SpellSearch, signals.SpellCircle, signals.SpellSchool)
	v.GrimoireSearch, v.GrimoireCircle = signals.GrimoireSearch, signals.GrimoireCircle
	v.EffectPicks = effectPickFilters{ConditionSearch: signals.ConditionSearch, BuffSearch: signals.BuffSearch, BuffCircle: signals.BuffCircle}
	v.Powers = s.powersPanelOf(dto, signals.PowerSearch)
	v.Choices = s.choicesPanelOf(dto, signals.PowerSearch)
	v.Bag = s.bagPanelOf(dto, bagFilters{
		Search: signals.ItemSearch, Category: signals.ItemCategory,
		CatalogSearch: signals.CatalogSearch, CatalogCategory: signals.CatalogCategory,
	})
	// UMA conta do motor para os DOIS painéis que a leem.
	if sheet, cards, ok := s.sheetForPanels(dto); ok {
		v.Combat = panelForCombat(sheet, cards, isCaster(sheet), dto)
		v.SpellCasting = spellCastingOf(sheet)
		v.Expertises = expertisePanelFor(dto, sheet, search)
		// DEPOIS dos três, e não ao lado: o painel de Ações não computa nada —
		// ele reagrupa o Combate, os Poderes e as Magias, e montá-lo antes
		// deles o deixaria lendo campos vazios.
		v.Actions = actionsPanelFrom(dto.ID, sheet, v.Combat, v.Powers, v.Spells, v.Effects,
			attackBoostsFor(dto, cards))
	}
	for _, item := range Tabs() {
		item.Active = item.Value == aba
		v.Tabs = append(v.Tabs, item)
	}
	return v, 200, nil
}

// sheetRoute é PARA ONDE se abre uma ficha no app — uma função e não um
// `Sprintf` espalhado, para haver um lugar só a ler quando o endereço mudar.
//
// @example sheetRoute(7, "bag") // "/personagens/7?tab=bag"
func sheetRoute(id int64, aba string) string {
	if aba == "" {
		return fmt.Sprintf("/personagens/%d", id)
	}
	return fmt.Sprintf("/personagens/%d?tab=%s", id, aba)
}

// stepClasses monta as classes com a elegibilidade de cada uma.
//
// Quem tem uma classe só sobe direto; quem tem duas é PERGUNTADO qual recebe o
// nível. Adivinhar é um defeito silencioso: a ficha fecha certo no total e
// errado na classe, e só aparece quando alguém for usar um poder que não veio.
//
// SUBIR exige que o TOTAL caiba em 20 (p32) — o teto é do personagem, não da
// classe. DESCER do 1º nível tira a classe da ficha, e a primeira nunca sai.
func stepClasses(dto sheet.CharacterDTO) []sheetClass {
	var total int64
	for _, cl := range dto.Classes {
		total += cl.Level
	}
	last := lastLevelOf(dto)
	classes := make([]sheetClass, 0, len(dto.Classes))
	for i, cl := range dto.Classes {
		leaves := i > 0 && cl.Level == 1
		classes = append(classes, sheetClass{
			Name:     cl.ClassName,
			Level:    cl.Level,
			CanRaise: total < 20,
			CanLower: last != nil && last.ClassName == cl.ClassName && (leaves || cl.Level > 1),
			Leaves:   leaves,
		})
	}
	return classes
}

// otherClasses são as classes do livro que a ficha ainda não tem. Vazio no teto, porque a
// classe nova contaria o 21º nível.
func otherClasses(dto sheet.CharacterDTO) []string {
	var total int64
	owned := map[string]bool{}
	for _, cl := range dto.Classes {
		total += cl.Level
		owned[cl.ClassName] = true
	}
	if total >= 20 {
		return nil
	}
	_, classes, _ := book.CharacterCatalogs()
	names := make([]string, 0, len(classes))
	for _, class := range classes {
		if !owned[class.Name] {
			names = append(names, class.Name)
		}
	}
	return names
}

// thatCan filtra as classes elegíveis para um sentido do degrau.
func thatCan(classes []sheetClass, step int) []sheetClass {
	eligible := make([]sheetClass, 0, len(classes))
	for _, cl := range classes {
		if (step > 0 && cl.CanRaise) || (step < 0 && cl.CanLower) {
			eligible = append(eligible, cl)
		}
	}
	return eligible
}

// lastLevelOf é o que o `−` tira. Nulo quando não há o que descer: a primeira classe não sai do 1º
// nível.
func lastLevelOf(dto sheet.CharacterDTO) *sheet.LastLevelLoss {
	loss, known := sheet.LastLevelOf(dto)
	if !known || loss.Level <= 1 {
		return nil
	}
	return &loss
}

// sheetPost escreve o `@post` de um gesto da ficha, CARREGANDO A ABA ABERTA.
//
// O `?tab=` não é decoração: todo comando da ficha responde redesenhando a cena
// INTEIRA, e a cena precisa saber em que seção a pessoa está. Sem ele o
// `AskedTab` não acha nada na query e cai na primeira aba — mexer no PV com a
// Mochila aberta joga o jogador em Perícias, e a ficha parece ter se fechado
// sozinha. Quem varre é o `TestNoSheetCommandLosesTheTab`.
func sheetPost(v View, path string) string {
	return fmt.Sprintf("@post('%s')", commandRoute(v, path))
}

// commandRoute monta o endereço de um comando da ficha com o estado que a URL
// carrega — a aba e, quando ela existe, a marca de EMBUTIDA.
//
// A marca viaja pelo mesmo motivo que a aba: o handler descobre o que desenhar
// lendo a requisição, e um comando sem ela devolveria a ficha de página inteira
// — com a barra do "‹ Voltar" e as abas navegando — dentro da sessão.
func commandRoute(v View, path string) string {
	route := fmt.Sprintf("/personagens/%d%s?tab=%s", v.ID, path, v.ActiveTab)
	if v.ByLevel {
		route += "&visao=nivel"
	}
	if v.Embedded {
		route += "&embutida=1"
	}
	return route
}

// sheetGet escreve o `@get` que REDESENHA a cena sem mutar nada — hoje só a
// busca das Perícias.
//
// Ele carrega o `?tab=` pela MESMA razão que todo `@post` carrega: sem ele o
// resolvedor não acha a aba na query e devolve a cena desenhada na primeira.
// A varredura `TestNoSheetCommandLosesTheTab` olha só os `@post`, então este
// caminho tem guarda própria — `TestTheSearchGetCarriesTheTab`.
func sheetGet(v View) string {
	return fmt.Sprintf("@get('%s')", commandRoute(v, ""))
}

// tabEmbeddedGet troca de seção SEM sair da sessão: o mesmo endereço da ficha,
// pedido pelo Datastar, remendando o `#sheet-scene` no lugar.
//
// Ele ESCREVE a aba num sinal ANTES de pedir, e o sinal é o que faz a ficha
// sobreviver a um aviso do servidor: quando o mestre mexe no personagem, quem
// repede a ficha é o cliente, e é daqui que ele sabe em que seção a pessoa está.
//
// Dois comandos separados por `;` e NUNCA num ternário: sequência dentro de
// ternário é erro de sintaxe que o Datastar engole, e o gesto inteiro vira nada.
func tabEmbeddedGet(v View, aba string) string {
	return fmt.Sprintf("$sheet_tab = %q; @get('/personagens/%d?tab=%s&embutida=1')",
		aba, v.ID, aba)
}

// SheetRefetch é o comando que o AVISO do stream dispara: a mesma ficha, na
// aba que o sinal guarda.
//
// A aba entra por CONCATENAÇÃO e não como texto fixo, porque quem a escolhe é
// quem está olhando — este comando é escrito uma vez, no servidor, e serve para
// as sete seções.
func SheetRefetch(v View) string {
	// A GUARDA é a comparação com o que já está na tela: sem ela, um gesto do
	// próprio jogador sai como dois pedidos — o comando grava, o stream vê o
	// `updatedAt` novo e manda o aviso, e o aviso repede a ficha que o comando
	// acabou de trazer.
	//
	// A versão vem do DOM e não de um sinal porque quem a atualiza é o próprio
	// remendo da ficha: um sinal teria de ser reescrito por fora, e é exatamente
	// o tipo de segunda escrita que sai de sincronia.
	return fmt.Sprintf(
		"$sheet_version !== document.getElementById('sheet-scene').dataset.versao && "+
			"@get('/personagens/%d?tab=' + $sheet_tab + '&embutida=1')", v.ID)
}

// attributeCommand escreve o `@post` que repõe a perícia noutro atributo.
//
// O valor escolhido entra por CONCATENAÇÃO no meio da expressão, e não como
// texto fixo: são seis atributos por linha e uma linha por perícia, e um comando
// por combinação daria centenas deles numa página que já tem um diálogo por
// linha.
func attributeCommand(v View, command string) string {
	return sheetPostWith(v, "/pericias/atributo/"+command+"/§", "evt.target.value")
}

// sheetPostWith é o `@post` de um comando que leva um valor da TELA no
// caminho: o `§` marca onde o valor entra. O endereço sai do `commandRoute`
// como todo comando, e é isso que mantém a aba e as marcas da ficha.
//
//	sheetPostWith(v, "/vitais/pv/§", "$vital_amount")
//	// "@post('/personagens/3/vitais/pv/' + $vital_amount + '?tab=combat')"
func sheetPostWith(v View, path, value string) string {
	before, after, _ := strings.Cut(commandRoute(v, path), "§")
	return fmt.Sprintf("@post('%s' + %s + '%s')", before, value, after)
}

// stepCommand escreve o `@post` de subir ou descer uma classe.
//
// A CLASSE VAI NO CAMINHO, codificada: nome de classe é do catálogo e não tem
// espaço hoje, mas escrever a rota assumindo isso é o tipo de suposição que
// quebra no dia em que uma classe nova chegar.
func stepCommand(v View, class string, step int) string {
	return sheetPost(v, fmt.Sprintf("/nivel/%s/%d", url.PathEscape(class), step))
}

// attributeDialogID é o diálogo da soma de um atributo.
func attributeDialogID(key string) string {
	return "attribute-" + key
}

// enterClassCommand escreve o `@post` de entrar numa classe nova.
func enterClassCommand(v View, class string) string {
	return sheetPost(v, fmt.Sprintf("/classes/%s/entra", url.PathEscape(class)))
}

// proficiencyCommand escreve o `@post` de ligar ou desligar uma categoria.
func proficiencyCommand(v View, key string) string {
	return sheetPost(v, "/proficiencias/alterna/"+key)
}

// defaultClassCommand escreve o `@post` do "Restaurar padrão".
func defaultClassCommand(v View) string {
	return sheetPost(v, "/proficiencias/padrao")
}

// stepDialog é o id do diálogo de escolher a classe, por sentido.
//
// DOIS diálogos e não um, porque as listas são diferentes: subir oferece as que
// cabem no teto, descer oferece as que têm nível de sobra. Um diálogo só teria
// de ser reescrito no gesto que o abre, que é a armadilha do nó COMPARTILHADO.
func stepDialog(step int) string {
	if step > 0 {
		return "subir-de-nivel"
	}
	return "descer-de-nivel"
}
