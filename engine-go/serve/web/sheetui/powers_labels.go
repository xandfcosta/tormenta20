package sheetui

import (
	"strconv"
	"strings"

	"t20engine/app/character"
	"t20engine/domain/book"
)

// O QUE A ABA PODERES ESCREVE.
//
// Traduzir é a responsabilidade PRÓPRIA deste arquivo, pelo mesmo motivo que
// partiu o `effects.go` (ALE-365): um rótulo se mexe quando a mesa lê mal, e o
// painel se mexe quando uma regra chega. Junto, o `powers.go` bateu no teto de
// 500 linhas ao dar ao crachá de preço a pergunta que faltava — se ESTE
// personagem tem degrau a comprar (ALE-427).
//
// A costura PT/EN passa toda por aqui: o catálogo fala inglês na fronteira, e
// quem lê a ficha lê português.

// writtenActions é a economia de ações do livro (p233), em caixa alta porque
// ela é crachá.
var writtenActions = map[string]string{
	"padrao": "PADRÃO", "movimento": "MOVIMENTO", "livre": "LIVRE", "reacao": "REAÇÃO",
	"gratuita": "GRATUITA", "completa": "COMPLETA", "passivo": "PASSIVA", "varia": "VARIA",
}

// writtenCost é "LIVRE · 1 PM" — a ação que o uso consome e o preço.
//
// A POSTURA escreve outra coisa: "POSTURA · 2+ PM". A economia de ação dela
// importa menos que o fato de ser postura (ela DURA), e o "+" avisa que o preço
// sobe com os degraus antes de a pessoa abrir o contador.
func writtenCost(spec book.Activation, stance *stanceState) string {
	if spec.Kind == "stance" {
		return "POSTURA · " + strconv.Itoa(book.ScaledCost(spec, 0)) + stepsMore(stance) + " PM"
	}
	action, found := writtenActions[spec.Action]
	if !found {
		action = strings.ToUpper(spec.Action)
	}
	if book.CostIsVariable(spec) {
		return action + " · PM variável"
	}
	return action + " · " + strconv.Itoa(book.ActivationPm(spec)) + " PM"
}

// stepsMore é o "+" do preço, e ele pergunta ao PERSONAGEM e não ao catálogo.
//
// A Fúria escala no catálogo desde o 1º nível e só abre o primeiro degrau no 6º
// (Tab. 1-6), então lendo o `spec.Scaling` o crachá prometia "2+ PM" a um
// bárbaro de 3º — um preço que ele não tem como fazer subir. Com o chip da
// superfície de Ações usando a mesma grafia para dizer "toque e eu pergunto"
// (ALE-427), a promessa falsa passou a prometer também um diálogo que não abre.
func stepsMore(stance *stanceState) string {
	if stance != nil && stance.MaxSteps > 0 {
		return "+"
	}
	return ""
}

// writtenSpent é "usado 1/1 cena" — o que já se gastou do limite cobrado.
func writtenSpent(scope string, use character.PowerUse) string {
	spent, word := use.Day, "dia"
	if scope == "scene" {
		spent, word = use.Scene, "cena"
	}
	return "usado " + strconv.Itoa(spent) + "/1 " + word
}

// shortSource encurta a procedência para caber no crachá da linha.
//
// "Classe · Bárbaro" vira "Bárbaro" porque a palavra "Classe" é a que se repete
// em quase toda linha — o que distingue é o nome da classe.
func shortSource(source string) string {
	if name, found := strings.CutPrefix(source, "Classe · "); found {
		return name
	}
	// O DIVINO segue a regra da classe, e pela mesma razão: o que distingue é o
	// DEUS. Sem esta linha ele caía no "Geral" do fim, e o crachá do Dom da
	// Verdade dizia que ele era um poder geral (ALE-426).
	if name, found := strings.CutPrefix(source, "Divino · "); found {
		return name
	}
	if strings.HasPrefix(source, "Raça") {
		return "Raça"
	}
	if strings.HasPrefix(source, "Origem") {
		return "Origem"
	}
	if source == "Poder da Tormenta" {
		return "Tormenta"
	}
	// O ITEM mostra o NOME dele no crachá, como a classe e o deus: o que
	// distingue dois encantos iguais é de qual arma eles vêm.
	if name, found := strings.CutPrefix(source, "Item · "); found {
		return name
	}
	return "Geral"
}

// activationGlyph escolhe o desenho pelo tipo da ativação.
func activationGlyph(kind string) string {
	switch kind {
	case "instant":
		return "Zap"
	case "stance":
		return "Flame"
	case "triggered-passive":
		return "Sparkles"
	}
	return "BookOpen"
}

// noActionsPhrase explica a seção vazia, e ela DEPENDE de quem lê: mandar às
// Magias quem não conjura seria mandá-lo a uma aba vazia.
func noActionsPhrase(casts bool) string {
	if casts {
		return "Nenhuma ação ativável — suas magias estão na aba Magias."
	}
	return "Nenhuma ação ativável. Suas habilidades são passivas."
}

// costStancePreview soma o custo dos degraus na tela.
//
// PRÉVIA e não decisão: quem cobra é o servidor, com o teto de degraus do nível
// e o PM disponível. Escrever a regra aqui daria uma segunda conta do mesmo
// número.
func costStancePreview(row powerRow) string {
	base := strconv.Itoa(row.Stance.BasePm)
	step := strconv.Itoa(row.Stance.StepPm)
	return "(" + base + " + " + step + " * $stance_degrees) + ' PM'"
}

// degreeCounterShows é o que o contador escreve enquanto alguém escolhe: o
// BÔNUS que a postura vai ter, e não quantos degraus foram somados.
//
// O livro nomeia a postura pelo bônus — "Fúria +3" na Tabela 1-6, "Inspiração
// +2" na 1-8 —, e era isso que a mesa procurava no contador. Ele mostrava "1",
// que é um número que não aparece em lugar nenhum da ficha depois: nem na linha
// de ataque, nem na aba Efeitos, nem na boca de quem narra.
//
// Recua para a CONTAGEM quando a postura não tem um bônus só (ver
// `book.StanceBaseBonus`): mostrar o degrau cru é feio e verdadeiro, e escolher
// um dos dois números seria bonito e mentira.
func degreeCounterShows(row powerRow) string {
	if row.Stance.BaseBonus <= 0 {
		return "$stance_degrees"
	}
	return "'+' + (" + strconv.Itoa(row.Stance.BaseBonus) + " + $stance_degrees)"
}

// degreeRateSays é o que cada degrau custa e compra, dito como TAXA.
//
// "cada +1 PM" e não o rótulo cru do catálogo ("+1 no bônus de Fúria"): com o
// contador mostrando o bônus TOTAL, duas linhas de "+N" empilhadas se parecem, e
// quem lê rápido pode tomar a segunda pelo preço da primeira. A palavra "cada"
// separa a taxa do total sem pedir nenhum número novo ao catálogo.
func degreeRateSays(row powerRow) string {
	if row.Stance.StepLabel == "" {
		return ""
	}
	return "cada +" + strconv.Itoa(row.Stance.StepPm) + " PM: " + row.Stance.StepLabel
}
