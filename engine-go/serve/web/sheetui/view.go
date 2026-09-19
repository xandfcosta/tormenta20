package sheetui

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"t20engine/domain/sheet"
	"t20engine/serve/web/characters"
)

// A FICHA como dado: a casca, as abas e o crachá.
//
// Ela chega COMPUTADA DO SERVIDOR, pela mesma `ComputeSheetV2` que a Mesa e a
// cena de personagens usam — sem requisito de offline, não há por que a conta
// acontecer no navegador.

// View é a ficha de um personagem pronta para desenhar.
type View struct {
	ID   int64
	Nome string
	// Versao é o `updatedAt` do personagem, e existe só para a ficha EMBUTIDA: o
	// ouvinte que repede a ficha compara o carimbo do stream com o que já está
	// na tela, e não pede nada quando são o mesmo. Sem isso, um gesto do próprio
	// jogador produz DOIS pedidos — o dele e o do aviso que a escrita dele
	// acabou de provocar.
	Versao string
	// Embutida diz que esta ficha está sendo desenhada DENTRO da sessão.
	//
	// Ela muda duas coisas na tela, e as duas são sobre NAVEGAÇÃO: a barra com
	// o "‹ Voltar" some (a sessão tem cabeçalho próprio, e voltar dali tiraria
	// o jogador da mesa), e as abas deixam de ser links para virar comandos que
	// remendam a ficha no lugar. Um `<a href>` ali levaria embora da sessão
	// quem só queria trocar de seção.
	Embutida bool
	// Iniciais e Gradiente são o retrato derivado do nome, como no palco e no
	// cartão da campanha: o app não guarda imagem de personagem.
	Iniciais  string
	Gradiente string
	// Papel é "GUERREIRO 10" — a mesma placa do palco de personagens.
	Papel  string
	Resumo string
	Nivel  int64
	// Defesa vem do MOTOR e é travessão quando não há catálogo primado: a tela
	// inteira não pode cair por causa de um número, e um zero seria pior — 0 é
	// um valor plausível, e o jogador agiria sobre ele.
	Defesa  string
	PV      sheetVital
	PM      sheetVital
	SemMana bool
	// Classes é "Guerreiro 3 / Ladino 2" — a mesma linha do palco, e string e
	// não lista porque é assim que a mesa lê e é assim que o cartão já a monta.
	Classes string
	// AsClasses são as classes com o nível de cada uma, e elas existem para o
	// DEGRAU DE NÍVEL: o nível de um personagem é a SOMA dos níveis de classe, e
	// subir um nível é escolher QUAL classe o recebe.
	AsClasses []sheetClass
	// Abas são as sete da ficha, com a ativa marcada. Ver `Tabs`.
	Abas []Tab
	// AbaAtiva é o valor resolvido — nunca o que veio na URL cru, que pode ser
	// um endereço antigo ou lixo digitado.
	AbaAtiva string
	// Proficiencias são os dois blocos do painel homônimo. Elas são computadas
	// SEMPRE, e não só quando a aba está aberta: são sete linhas derivadas de
	// dado que a ficha já carregou, e um `if` aqui trocaria microssegundos por um
	// ramo a mais para um guarda cobrir.
	Proficiencias []proficiencyGroup
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
	// Bag é a aba homônima — a tira de equipados, a carga da p141, o dinheiro e a
	// grade do que está guardado.
	Bag bagPanel
	// Powers é a aba homônima — a lista de jogo: o que se ativa em cima, o que é
	// passivo recolhido embaixo.
	Powers powersPanel
	// Choices é o diálogo de escolher poderes — a administração da ficha.
	Choices choicesPanel
	// Recusa é a frase de uma regra que barrou o gesto — o teto de duas mãos, o
	// PM que falta, a magia que não está preparada. Vazia no caminho normal.
	// Ela vem com a cena INTEIRA redesenhada, que é o que mostra que nada mudou.
	Recusa string
}

// sheetClass é uma classe do personagem, com o que o degrau precisa saber.
type sheetClass struct {
	Nome  string
	Nivel int64
	// PodeSubir e PodeDescer são a elegibilidade do livro, e elas são POR CLASSE
	// e não do personagem: descer uma classe de nível 1 a apagaria, e subir com
	// o total em 20 (p32) passaria do teto.
	PodeSubir  bool
	PodeDescer bool
}

type sheetVital struct {
	Atual int64
	Max   int64
	// Fracao é "12/20", que é como a mesa fala.
	Fracao string
	// Porcento é a largura da barra, entre 0 e 100.
	Porcento int
	// Temp é o PV TEMPORÁRIO, e ele é uma parcela À PARTE e não um somando.
	//
	// A barra continua sendo o PV de verdade: somar o temporário ao atual faria
	// um herói a 50/137 com 70 de reserva desenhar 88% de vida com 36% de
	// carne. O livro autoriza os dois desenhos — *"são somados a seus pontos
	// atuais, mesmo que ultrapassem o máximo"* (p106) —, e o que decide é a
	// pergunta que a barra responde, que é "quanto apanhei".
	//
	// Vazio quer dizer que não há reserva, e aí a fileira fica IGUAL à de antes.
	Temp string
}

// Tab é uma das sete seções da ficha.
type Tab struct {
	// Valor é o que vai na URL — ver `Tabs`.
	Valor  string
	Rotulo string
	Icone  string
	Ativa  bool
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
		{Valor: "expertises", Rotulo: "Perícias", Icone: "Scroll"},
		{Valor: "combat", Rotulo: "Combate", Icone: "Swords"},
		{Valor: "bag", Rotulo: "Mochila", Icone: "Backpack"},
		{Valor: "proficiencies", Rotulo: "Proficiências", Icone: "ShieldCheck"},
		{Valor: "conditionals", Rotulo: "Efeitos", Icone: "Zap"},
		{Valor: "abilities", Rotulo: "Poderes", Icone: "Star"},
		{Valor: "spells", Rotulo: "Magias", Icone: "BookMarked"},
	}
}

// AskedTab resolve o que veio na URL contra as sete que existem. Valor
// desconhecido cai na PRIMEIRA, em vez de dar 404: `?tab=` é endereço, e alguém
// o digita errado.
func AskedTab(bruto string) string {
	for _, aba := range Tabs() {
		if aba.Valor == bruto {
			return bruto
		}
	}
	return Tabs()[0].Valor
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
	ctx context.Context, userID int64, id int64, aba, busca string, sinais Signals,
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
	cartao := characters.HeroCardOf(s.deps.Catalogs(), dto)
	v := View{
		ID:        dto.ID,
		Nome:      dto.Name,
		Versao:    row.Updatedat,
		Iniciais:  cartao.Monogram,
		Gradiente: cartao.Gradient,
		Papel:     cartao.Role,
		Resumo:    cartao.Summary,
		Nivel:     dto.Level,
		Defesa:    cartao.DefenseVs,
		PV:        withTempHp(vital(dto.HpCurrent, dto.HpMax), dto.ActiveEffects),
		PM:        vital(dto.MpCurrent, dto.MpMax),
		SemMana:   dto.MpMax == 0,
		Classes:   cartao.Classes,
		AsClasses: stepClasses(dto),
		AbaAtiva:  aba,

		Proficiencias: proficiencyGroupsOf(dto),
	}
	v.Effects = s.effectsPanelOf(dto)
	v.Spells = s.spellbookPanelOf(dto, sinais.MagiaBusca, sinais.MagiaCirculo, sinais.MagiaEscola)
	v.Powers = s.powersPanelOf(dto, sinais.PoderBusca)
	v.Choices = s.choicesPanelOf(dto, sinais.PoderBusca)
	v.Bag = s.bagPanelOf(dto, bagFilters{
		Busca: sinais.ItemBusca, Categoria: sinais.ItemCategoria,
		BuscaNoCatalogo: sinais.CatalogoBusca, CategoriaNoCatalogo: sinais.CatalogoCategoria,
	})
	// UMA conta do motor para os DOIS painéis que a leem.
	if sheet, cards, ok := s.sheetForPanels(dto); ok {
		v.Combat = panelForCombat(sheet, cards, isCaster(sheet))
		v.Expertises = expertisePanelFor(dto, sheet, busca)
	}
	for _, item := range Tabs() {
		item.Ativa = item.Valor == aba
		v.Abas = append(v.Abas, item)
	}
	return v, 200, nil
}

// vital monta a barra de PV ou PM.
//
// A FRAÇÃO é o que a mesa fala em voz alta ("doze de vinte"), e a porcentagem é
// só a largura da barra. Máximo ZERO não vira divisão por zero nem barra cheia:
// quem não tem mana tem a barra vazia e apagada.
func vital(atual, max int64) sheetVital {
	v := sheetVital{Atual: atual, Max: max, Fracao: strconv.FormatInt(atual, 10) + "/" + strconv.FormatInt(max, 10)}
	if max <= 0 {
		return v
	}
	pct := int(atual * 100 / max)
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	v.Porcento = pct
	return v
}

// withTempHp acrescenta ao PV a reserva que os efeitos ativos carregam.
//
// A conta é do `sheet` (`TempHpTotal`), e a leitura NÃO custa consulta: o
// agregado já traz os efeitos, porque a aba Efeitos os desenha.
//
// PM não ganha o mesmo: o livro tem pontos de mana temporários (p106) e este
// app ainda não os modela — o motor só conhece o alvo `tempMp` como
// modificador, e nada os gasta. Desenhar um número que nada consome seria pior
// que não desenhá-lo.
func withTempHp(v sheetVital, efeitos []sheet.EffectDTO) sheetVital {
	blobs := make([]string, 0, len(efeitos))
	for _, e := range efeitos {
		blobs = append(blobs, e.Modifiers)
	}
	if total := sheet.TempHpTotal(blobs); total > 0 {
		v.Temp = "+" + strconv.Itoa(total)
	}
	return v
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

// routeVital traduz o rótulo da tela para o pedaço da URL.
//
// Duas palavras para a mesma coisa é o que o GLOSSARY chama de colisão, e aqui
// ela é deliberada e contida: a TELA diz "PV" porque é o que a mesa fala, e a
// ROTA diz "pv" porque endereço é minúsculo. Esta função é a única costura.
func routeVital(rotulo string) string {
	if rotulo == "PM" {
		return "pm"
	}
	return "pv"
}

// stepSignal escreve o rótulo do botão: "+5", "−1".
//
// O MENOS É O SINAL TIPOGRÁFICO (U+2212) e não o hífen: no mesmo tamanho de
// fonte o hífen fica mais curto e mais alto que o traço do "+", e a fileira dos
// quatro botões desalinha.
func stepSignal(passo int) string {
	if passo < 0 {
		return "−" + strconv.Itoa(-passo)
	}
	return "+" + strconv.Itoa(passo)
}

// stepLabel é o nome acessível: "Curar 5 de PV", "Ferir 1 de PV".
//
// O VERBO muda com o sinal em vez de "mais 5 PV", porque é o verbo que a mesa
// usa — e um leitor de tela lendo "menos cinco pê vê" obriga quem ouve a
// traduzir de volta para "levou cinco".
func stepLabel(rotulo string, passo int) string {
	verbo := "Curar"
	if passo < 0 {
		verbo = "Ferir"
		passo = -passo
	}
	return fmt.Sprintf("%s %d de %s", verbo, passo, rotulo)
}

// stepClasses monta as classes com a elegibilidade de cada uma.
//
// Quem tem uma classe só sobe direto; quem tem duas é PERGUNTADO qual recebe o
// nível. Adivinhar é um defeito silencioso: a ficha fecha certo no total e
// errado na classe, e só aparece quando alguém for usar um poder que não veio.
//
// SUBIR exige que o TOTAL caiba em 20 (p32) — o teto é do personagem, não da
// classe. DESCER exige que a classe tenha mais de um nível: levá-la a zero
// apagaria a classe, que é outra coisa e não tem gesto nesta tela.
func stepClasses(dto sheet.CharacterDTO) []sheetClass {
	var total int64
	for _, cl := range dto.Classes {
		total += cl.Level
	}
	classes := make([]sheetClass, 0, len(dto.Classes))
	for _, cl := range dto.Classes {
		classes = append(classes, sheetClass{
			Nome:       cl.ClassName,
			Nivel:      cl.Level,
			PodeSubir:  total < 20,
			PodeDescer: cl.Level > 1,
		})
	}
	return classes
}

// thatCan filtra as classes elegíveis para um sentido do degrau.
func thatCan(classes []sheetClass, passo int) []sheetClass {
	elegiveis := make([]sheetClass, 0, len(classes))
	for _, cl := range classes {
		if (passo > 0 && cl.PodeSubir) || (passo < 0 && cl.PodeDescer) {
			elegiveis = append(elegiveis, cl)
		}
	}
	return elegiveis
}

// directStep é o comando de quem só tem UMA classe elegível — o caso comum.
//
// Vazio quando há mais de uma: aí o gesto abre a escolha, porque adivinhar qual
// classe recebe o nível é o defeito que o `stepClasses` descreve.
func directStep(v View, passo int) string {
	elegiveis := thatCan(v.AsClasses, passo)
	if len(elegiveis) != 1 {
		return ""
	}
	return stepCommand(v, elegiveis[0].Nome, passo)
}

// sheetPost escreve o `@post` de um gesto da ficha, CARREGANDO A ABA ABERTA.
//
// O `?tab=` não é decoração: todo comando da ficha responde redesenhando a cena
// INTEIRA, e a cena precisa saber em que seção a pessoa está. Sem ele o
// `AskedTab` não acha nada na query e cai na primeira aba — mexer no PV com a
// Mochila aberta joga o jogador em Perícias, e a ficha parece ter se fechado
// sozinha. Quem varre é o `TestNoSheetCommandLosesTheTab`.
func sheetPost(v View, caminho string) string {
	return fmt.Sprintf("@post('%s')", commandRoute(v, caminho))
}

// commandRoute monta o endereço de um comando da ficha com o estado que a URL
// carrega — a aba e, quando ela existe, a marca de EMBUTIDA.
//
// A marca viaja pelo mesmo motivo que a aba: o handler descobre o que desenhar
// lendo a requisição, e um comando sem ela devolveria a ficha de página inteira
// — com a barra do "‹ Voltar" e as abas navegando — dentro da sessão.
func commandRoute(v View, caminho string) string {
	rota := fmt.Sprintf("/personagens/%d%s?tab=%s", v.ID, caminho, v.AbaAtiva)
	if v.Embutida {
		rota += "&embutida=1"
	}
	return rota
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
func attributeCommand(v View, comando string) string {
	return fmt.Sprintf(
		"@post('/personagens/%d/pericias/atributo/%s/' + evt.target.value + '?tab=%s')",
		v.ID, comando, v.AbaAtiva)
}

// totalLabel é o nome acessível do número de uma perícia.
//
// A falha automática não diz um número, porque não há um: ela diz o que
// aconteceu. Um botão chamado "—" não informa nada a quem usa leitor de tela.
func totalLabel(linha expertiseRow) string {
	if linha.AutoFail {
		return "Falha automática em " + linha.Name + " — detalhar"
	}
	return "Detalhar " + linha.Name + " " + linha.Total
}

// stepCommand escreve o `@post` de subir ou descer uma classe.
//
// A CLASSE VAI NO CAMINHO, codificada: nome de classe é do catálogo e não tem
// espaço hoje, mas escrever a rota assumindo isso é o tipo de suposição que
// quebra no dia em que uma classe nova chegar.
func stepCommand(v View, classe string, passo int) string {
	return sheetPost(v, fmt.Sprintf("/nivel/%s/%d", url.PathEscape(classe), passo))
}

// vitalCommand escreve o `@post` de um passo de PV ou PM.
func vitalCommand(v View, rotulo string, passo int) string {
	return sheetPost(v, fmt.Sprintf("/vitais/%s/%d", routeVital(rotulo), passo))
}

// proficiencyCommand escreve o `@post` de ligar ou desligar uma categoria.
func proficiencyCommand(v View, chave string) string {
	return sheetPost(v, "/proficiencias/alterna/"+chave)
}

// defaultClassCommand escreve o `@post` do "Restaurar padrão de classe".
func defaultClassCommand(v View) string {
	return sheetPost(v, "/proficiencias/padrao")
}

// stepDialog é o id do diálogo de escolher a classe, por sentido.
//
// DOIS diálogos e não um, porque as listas são diferentes: subir oferece as que
// cabem no teto, descer oferece as que têm nível de sobra. Um diálogo só teria
// de ser reescrito no gesto que o abre, que é a armadilha do nó COMPARTILHADO.
func stepDialog(passo int) string {
	if passo > 0 {
		return "subir-de-nivel"
	}
	return "descer-de-nivel"
}
