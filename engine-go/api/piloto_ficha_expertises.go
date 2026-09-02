package api

import (
	"net/url"
	"sort"
	"strconv"
	"strings"

	"t20engine/book"
	"t20engine/engine"
	"t20engine/sheet"
)

// A aba PERÍCIAS como dado (ALE-272, fatia 4).
//
// Vinte e nove do livro mais os OFÍCIOS que o jogador inventa, cada uma com o
// total, o atributo que ela usa, o botão de treino e a decomposição por trás do
// número. É o painel mais pesado da ficha, e o único que escreve em quatro
// gestos diferentes.
//
// # UMA LINHA POR PERÍCIA, EM TODA LARGURA
//
// A SPA desenha uma segunda fileira de crachás (½lvl, treino, outros) acima de
// 640px, e o comentário dela diz que eles "REPETEM, palavra por palavra, o
// diálogo de decomposição". Aqui não existem: decisão do dono nesta fatia. A
// auditoria continua a um toque no número, que é onde ela sempre esteve — e as
// 29 linhas ficam mais curtas em todo tamanho de tela.
//
// # A ORDEM É DO LIVRO, com as três resistências à frente
//
// "Teste de Reflexos CD 20" é a consulta mais quente da mesa, então Fortitude,
// Reflexos e Vontade sobem para o topo; o resto vem na ordem do catálogo, que é
// alfabética; e os ofícios do jogador fecham a lista. É a ordem da SPA.

// expertisePanel é a aba Perícias pronta para desenhar.
type expertisePanel struct {
	// TrainingBonus e HalfLevel são o cabeçalho ("treino +4 • ½ nível 5"): as
	// duas parcelas que valem para TODAS as linhas, ditas uma vez em vez de 29.
	TrainingBonus string
	HalfLevel     string
	// Search é o que está na caixa de busca, devolvido pelo servidor para a
	// filtragem sobreviver ao remendo.
	Search string
	Rows   []expertiseRow
	// Attributes são as seis opções do seletor, com o modificador FINAL de cada
	// uma. Elas são iguais em toda linha, então moram aqui e não na linha.
	Attributes []attributeOption
}

// expertiseRow é uma perícia na lista.
type expertiseRow struct {
	// Key identifica o diálogo desta linha para o sinal `$detalhe`, e é o índice
	// e não o nome: nome de ofício é texto livre do jogador, e um apóstrofo nele
	// quebraria a expressão do Datastar que compara a chave.
	Key  string
	Name string
	// Command é o nome ESCAPADO para o caminho dos comandos, e o escape não é
	// zelo: nome de perícia tem acento ("Atuação") e nome de ofício é texto livre
	// do jogador. É a mesma decisão do `oComandoDoDegrau` com o nome da classe.
	Command   string
	Attribute string
	Total     string
	Trained   bool
	// TrainedOnly é a perícia que o livro marca como "só treinada" (p115): sem
	// treinamento, nem se rola.
	TrainedOnly bool
	// Locked é `TrainedOnly` sem treino — o número existe e não vale.
	Locked bool
	// AutoFail é a perícia em que o personagem falha AUTOMATICAMENTE, hoje só
	// Reflexos com o Indefeso (p394). Ela não mostra número, porque não há um.
	AutoFail bool
	// Custom é o ofício do jogador, o único que pode ser removido.
	Custom bool
	Rows   []breakdownRow
}

// attributeOption é uma entrada do seletor de atributo.
type attributeOption struct {
	Key string
	// Label é "FOR +4": a abreviação com o modificador FINAL, com raça e itens já
	// dentro. Mostrar o valor cru faria a linha discordar do próprio total.
	Label string
}

// theSaveNames são as três que sobem para o topo da lista.
var theSaveNames = map[string]bool{"Fortitude": true, "Reflexos": true, "Vontade": true}

// expertisePanelFor monta a aba inteira.
func expertisePanelFor(dto sheet.CharacterDTO, sheet engine.ComputedSheetV2, search string) expertisePanel {
	panel := expertisePanel{
		TrainingBonus: comSinalInt(trainingBonusFor(dto.Level)),
		HalfLevel:     strconv.FormatInt(dto.Level/2, 10),
		Search:        search,
		Attributes:    attributeOptions(sheet),
	}
	for i, entry := range sortedExpertises(dto) {
		if !matchesSearch(entry.Name, search) {
			continue
		}
		panel.Rows = append(panel.Rows, expertiseRowFor(i, entry, sheet))
	}
	return panel
}

// trainingBonusFor é o treino por nível — +2, +4 no 7º, +6 no 15º.
//
// Ele é reescrito aqui porque o motor o guarda minúsculo e o cabeçalho precisa
// do número para DIZER a regra antes de a pessoa abrir um diálogo. A regra em si
// tem teste no `engine`, com a página; aqui é o mesmo degrau, e
// `TestTheHeaderSaysTheTrainingForTheLevel` prende os três.
func trainingBonusFor(level int64) int {
	switch {
	case level >= 15:
		return 6
	case level >= 7:
		return 4
	default:
		return 2
	}
}

// sortedExpertises devolve as perícias na ordem da tela.
func sortedExpertises(dto sheet.CharacterDTO) []sheet.ExpertiseDTO {
	doLivro := map[string]int{}
	for i, p := range book.Expertises() {
		doLivro[p.Name] = i
	}
	ordenadas := append([]sheet.ExpertiseDTO(nil), dto.Expertises...)
	sort.SliceStable(ordenadas, func(a, b int) bool {
		return rankOf(ordenadas[a], doLivro) < rankOf(ordenadas[b], doLivro)
	})
	return ordenadas
}

// rankOf é a posição de uma perícia na ordem da tela: resistências, livro,
// ofícios.
//
// Os degraus são largos de propósito — somar o índice do catálogo a um bloco de
// mil mantém a ordem alfabética DENTRO do bloco sem que um bloco invada o
// seguinte.
func rankOf(e sheet.ExpertiseDTO, doLivro map[string]int) int {
	if theSaveNames[e.Name] {
		return doLivro[e.Name]
	}
	if i, ok := doLivro[e.Name]; ok {
		return 1000 + i
	}
	return 2000
}

// matchesSearch compara SEM acento, porque "pericia" tem de achar "Perícia".
func matchesSearch(name, search string) bool {
	if strings.TrimSpace(search) == "" {
		return true
	}
	return strings.Contains(foldAccents(name), foldAccents(search))
}

// foldAccents baixa a caixa e tira os acentos do português.
//
// É uma tabela e não `unicode/norm` porque o alfabeto que a mesa digita é
// conhecido e cabe em nove pares — trazer uma dependência de normalização
// Unicode para isto seria pagar caro por generalidade que ninguém usa.
var accentFolder = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a",
	"é", "e", "ê", "e",
	"í", "i",
	"ó", "o", "ô", "o", "õ", "o",
	"ú", "u", "ü", "u",
	"ç", "c",
)

func foldAccents(s string) string {
	return accentFolder.Replace(strings.ToLower(s))
}

// attributeOptions são as seis, com o modificador final de cada uma.
func attributeOptions(sheet engine.ComputedSheetV2) []attributeOption {
	opcoes := make([]attributeOption, 0, len(engine.AttributeKeys))
	for _, key := range engine.AttributeKeys {
		opcoes = append(opcoes, attributeOption{
			Key:   key,
			Label: attributeAbbr[key] + " " + comSinalInt(sheet.Attributes[key].Total),
		})
	}
	return opcoes
}

// expertiseRowFor monta uma linha.
func expertiseRowFor(index int, entry sheet.ExpertiseDTO, sheet engine.ComputedSheetV2) expertiseRow {
	quebra := expertiseOrZero(sheet, entry.Name, entry.Attribute)
	soTreinada := trainedOnlyByBook(entry.Name)
	linha := expertiseRow{
		Key:         "exp-" + strconv.Itoa(index),
		Name:        entry.Name,
		Command:     url.PathEscape(entry.Name),
		Attribute:   entry.Attribute,
		Total:       comSinalInt(quebra.Total),
		Trained:     entry.Trained,
		TrainedOnly: soTreinada,
		Locked:      soTreinada && !entry.Trained,
		AutoFail:    autoFails(sheet, entry.Name),
		Custom:      entry.Custom,
		Rows:        expertiseBreakdownRows(quebra),
	}
	return linha
}

// trainedOnlyByBook lê a Tabela 2-1 pelo CATÁLOGO, e não por uma lista aqui.
//
// Um ofício do jogador não está no catálogo e é sempre "só treinada": inventar
// um saber e não tê-lo treinado não é um estado que signifique alguma coisa.
func trainedOnlyByBook(name string) bool {
	for _, p := range book.Expertises() {
		if p.Name == name {
			return p.SoTreinada
		}
	}
	return true
}

// autoFails diz se o personagem falha automaticamente nesta perícia.
//
// Quem responde é o MOTOR, e não a tela relendo uma condição: a regra de quais
// condições implicam indefeso mora lá (p394).
func autoFails(sheet engine.ComputedSheetV2, name string) bool {
	for _, falha := range sheet.AutoFailExpertises {
		if falha == name {
			return true
		}
	}
	return false
}

// expertiseBreakdownRows é a decomposição de uma perícia.
//
// A forma é a da SPA e difere da do Combate de propósito: aqui "Outros" é a
// SOMA dos itens, e as contribuições vêm indentadas por baixo dela. Quem abre
// uma perícia quer primeiro as quatro parcelas do livro, e só depois de onde
// saiu a quarta.
func expertiseBreakdownRows(ex engine.ExpertiseBreakdown) []breakdownRow {
	linhas := []breakdownRow{
		{Label: "½ nível", Value: comSinalInt(ex.HalfLevel)},
		{Label: "Atributo (" + attributeAbbr[ex.Attribute] + ")", Value: comSinalInt(ex.AttrValue)},
		{Label: "Treino", Value: comSinalInt(ex.Training)},
		{Label: "Outros", Value: comSinalInt(ex.ItemBonus)},
	}
	for _, c := range ex.ItemContributions {
		linhas = append(linhas, breakdownRow{
			Label: c.Source, Value: comSinalInt(c.Amount), Note: c.Note, Indented: true,
		})
	}
	return linhas
}
