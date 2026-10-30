package sheetui

import (
	"net/url"
	"sort"
	"strconv"
	"strings"

	"t20engine/app/character"
	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/search"
	"t20engine/domain/sheet"
	"t20engine/serve/web/ds"
)

// A aba PERÍCIAS como dado.
//
// Vinte e nove do livro mais os OFÍCIOS que o jogador inventa, cada uma com o
// total, o atributo que ela usa, o botão de treino e a decomposição por trás do
// número. É o painel mais pesado da ficha, e o único que escreve em quatro
// gestos diferentes.
//
// UMA LINHA POR PERÍCIA, sem a fileira de crachás (½ nível, treino, outros) que
// repetiria palavra por palavra o diálogo de decomposição — decisão do dono. A
// auditoria fica a um toque no número, e as 29 linhas ficam curtas em toda
// largura.
//
// A ORDEM É DO LIVRO com as três resistências à FRENTE: "Teste de Reflexos CD
// 20" é a consulta mais quente da mesa. O resto vem na ordem do catálogo, que é
// alfabética, e os ofícios do jogador fecham a lista.

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
	// Key identifica o diálogo desta linha para o sinal `$detail`, e é o índice
	// e não o nome: nome de ofício é texto livre do jogador, e um apóstrofo nele
	// quebraria a expressão do Datastar que compara a chave.
	Key  string
	Name string
	// Command é o nome ESCAPADO para o caminho dos comandos, e o escape não é
	// zelo: nome de perícia tem acento ("Atuação") e nome de ofício é texto livre
	// do jogador. É a mesma decisão do `stepCommand` com o nome da classe.
	Command   string
	Attribute string
	Total     string
	// Trained é o VEREDITO do motor, e não a coluna da ficha: a perícia que a
	// origem treina (p85) chega treinada sem a coluna marcada (ALE-490).
	Trained bool
	// TrainedBy é quem treinou além da coluna — "Treinada pela Origem: Soldado".
	// Vazio quando só a coluna treina, e aí o interruptor é o gesto que a muda.
	TrainedBy string
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
	// Sum é a conta da perícia: uma linha por parcela.
	Sum []ds.SumTerm
	// TotalValue é o `Total` como número, para a prévia do total com os gastos.
	TotalValue int
	// Spends são os gastos de PM que valem NESTE teste (ALE-489), e
	// SpendsUnlock diz que algum deles destrava a perícia trancada. Ver
	// `expertises_spends.go`.
	Spends       []expertiseSpend
	SpendsUnlock bool
	// Mana é o PM que a ficha tem, para a prévia dizer quanto sobra.
	Mana int
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
func expertisePanelFor(dto sheet.CharacterDTO, computed engine.ComputedSheet, term string) expertisePanel {
	panel := expertisePanel{
		TrainingBonus: book.WithSign(engine.TrainingBonusForLevel(int(dto.Level))),
		HalfLevel:     strconv.FormatInt(dto.Level/2, 10),
		Search:        term,
		Attributes:    attributeOptions(computed),
	}
	spends := character.PrepareSkillSpends(dto, computed)
	for i, entry := range sortedExpertises(dto) {
		if !matchesSearch(entry.Name, term) {
			continue
		}
		row := expertiseRowFor(i, entry, computed)
		row.withSpends(spends.For(entry.Name), int(dto.ManaAvailable()))
		panel.Rows = append(panel.Rows, row)
	}
	return panel
}

// sortedExpertises devolve as perícias na ordem da tela.
func sortedExpertises(dto sheet.CharacterDTO) []sheet.ExpertiseDTO {
	fromBook := map[string]int{}
	for i, p := range book.Expertises() {
		fromBook[p.Name] = i
	}
	sorted := append([]sheet.ExpertiseDTO(nil), dto.Expertises...)
	sort.SliceStable(sorted, func(a, b int) bool {
		return rankOf(sorted[a], fromBook) < rankOf(sorted[b], fromBook)
	})
	return sorted
}

// rankOf é a posição de uma perícia na ordem da tela: resistências, livro,
// ofícios.
//
// Os degraus são largos de propósito — somar o índice do catálogo a um bloco de
// mil mantém a ordem alfabética DENTRO do bloco sem que um bloco invada o
// seguinte.
func rankOf(e sheet.ExpertiseDTO, fromBook map[string]int) int {
	if theSaveNames[e.Name] {
		return fromBook[e.Name]
	}
	if i, ok := fromBook[e.Name]; ok {
		return 1000 + i
	}
	return 2000
}

// matchesSearch compara SEM acento, porque "pericia" tem de achar "Perícia".
//
// A dobra é a do `domain/search`, que é a MESMA que o buscador do livro usa. Ela
// já morou aqui como uma tabela de nove pares, com o argumento de que "o
// alfabeto que a mesa digita é conhecido" — e o argumento era verdadeiro e
// insuficiente: o que a mesa digita inclui o nome que um jogador INVENTA para um
// ofício ou um item, e ali cabe qualquer acento (ALE-352).
//
// O parâmetro se chama `term` e não `search` porque `search` é o PACOTE — duas
// funções deste arquivo o sombreavam, e o compilador só reclama quando alguém
// tenta usar o pacote lá dentro.
func matchesSearch(name, term string) bool {
	if strings.TrimSpace(term) == "" {
		return true
	}
	return strings.Contains(search.Fold(name), search.Fold(term))
}

// attributeOptions são as seis, com o modificador final de cada uma.
func attributeOptions(sheet engine.ComputedSheet) []attributeOption {
	options := make([]attributeOption, 0, len(engine.AttributeKeys))
	for _, key := range engine.AttributeKeys {
		options = append(options, attributeOption{
			Key:   key,
			Label: attributeAbbr[key] + " " + book.WithSign(sheet.Attributes[key].Total),
		})
	}
	return options
}

// expertiseRowFor monta uma linha.
func expertiseRowFor(index int, entry sheet.ExpertiseDTO, sheet engine.ComputedSheet) expertiseRow {
	breaks := expertiseOrZero(sheet, entry.Name, entry.Attribute)
	trainedOnly := trainedOnlyByBook(entry.Name)
	row := expertiseRow{
		Key:         "exp-" + strconv.Itoa(index),
		Name:        entry.Name,
		Command:     url.PathEscape(entry.Name),
		Attribute:   entry.Attribute,
		Total:       book.WithSign(breaks.Total),
		TotalValue:  breaks.Total,
		Trained:     breaks.Trained,
		TrainedBy:   trainedByText(breaks.TrainedBy),
		TrainedOnly: trainedOnly,
		Locked:      trainedOnly && !breaks.Trained,
		AutoFail:    autoFails(sheet, entry.Name),
		Custom:      entry.Custom,
		Sum:         expertiseSumTerms(breaks),
	}
	return row
}

// trainedByText diz quem treinou a perícia fora da coluna da ficha.
//
//	trainedByText([]string{"Origem: Soldado"}) // "Treinada pela Origem: Soldado"
func trainedByText(sources []string) string {
	if len(sources) == 0 {
		return ""
	}
	return "Treinada pela " + strings.Join(sources, " e pela ")
}

// trainedOnlyByBook lê a Tabela 2-1 pelo CATÁLOGO, e não por uma lista aqui.
//
// Um ofício do jogador não está no catálogo e é sempre "só treinada": inventar
// um saber e não tê-lo treinado não é um estado que signifique alguma coisa.
func trainedOnlyByBook(name string) bool {
	for _, p := range book.Expertises() {
		if p.Name == name {
			return p.TrainedOnly
		}
	}
	return true
}

// autoFails diz se o personagem falha automaticamente nesta perícia.
//
// Quem responde é o MOTOR, e não a tela relendo uma condição: a regra de quais
// condições implicam indefeso mora lá (p394).
func autoFails(sheet engine.ComputedSheet, name string) bool {
	for _, fails := range sheet.AutoFailExpertises {
		if fails == name {
			return true
		}
	}
	return false
}
