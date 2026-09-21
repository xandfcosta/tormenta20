package sheetui

import (
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// O QUE A ABA EFEITOS ESCREVE.
//
// Traduzir é a responsabilidade PRÓPRIA deste arquivo, e ela muda por outra
// razão que a montagem do painel: um rótulo se mexe quando a mesa lê mal, e o
// painel se mexe quando uma regra chega. Junto, o `effects.go` chegou a uma
// linha do teto de 500 (ALE-365).
//
// A costura PT/EN passa toda por aqui: o catálogo e o motor falam inglês na
// fronteira, e quem lê a ficha lê português. Palavra sem tradução VOLTA CRUA de
// propósito — um efeito sem rótulo some da linha —, e quem cobra a tradução é o
// `TestEveryDurationAnEffectCanCarryIsNamedOnTheSheet`.

// effectDisplayName troca o id do catálogo pelo nome que a mesa lê.
func effectDisplayName(catalogID string) string {
	for _, m := range book.Catalogs().Spells {
		if m.ID == catalogID {
			return m.Name
		}
	}
	// E os PODERES, que é de onde vêm as concessões de postura: a reserva de PV
	// temporários da Alma de Bronze saía na tela escrita
	// `class.barbaro.alma-de-bronze` (ALE-351).
	if spec := book.ActivationOf(catalogID, ""); spec != nil {
		return spec.Name
	}
	// O recuo para o ID é silencioso por construção — ele devolve algo que
	// parece um nome para quem lê o código, e não para quem lê a tela. Ele fica
	// porque a alternativa é a linha sem texto nenhum, mas toda procedência
	// nova de efeito precisa entrar numa das buscas acima.
	return catalogID
}

// scopeLabel diz até quando o efeito vale.
//
// A coluna `scope` grava em INGLÊS porque é fronteira, e a tela lê em
// português; as duas grafias entram aqui porque o dado gravado antes da
// ALE-365 e o dado de hoje convivem. Palavra sem tradução VOLTA CRUA de
// propósito — um efeito sem rótulo some da linha —, e é por isso que cada
// duração nova precisa passar por aqui: a sustentada nasceu sem, e a aba
// mostrou "sustained" para quem lê a ficha.
func scopeLabel(scope string) string {
	switch scope {
	case "day", "dia":
		return "até o fim do dia"
	case "scene", "cena":
		return "até o fim da cena"
	case "sustained", "sustentada":
		return "enquanto for sustentada"
	case "permanent", "permanente":
		return "para sempre"
	case "discharge", "descarregar":
		return "até descarregar"
	case "turn":
		// A VEZ, e não "o turno": a duração de "1 turno" do livro (p192) acaba
		// com a vez EM CURSO, que pode não ser a de quem carrega o efeito —
		// ela chega por reação, e reação acontece fora da sua vez (p233).
		return "até o fim da vez"
	}
	return scope
}

// targetLabel nomeia o que o modificador toca, em português.
//
// O motor fala `{k: "attack", scope: "all"}`, que é fronteira; quem lê a ficha
// lê "ataque". Alvo sem tradução assentada cai no próprio `k`, que é feio e
// honesto — melhor que inventar um nome que não é o do livro.
//
// A tabela cobre também o que a Mochila desenha de um item CONCEDIDO
// (`inventorySlots`, `spellDC`, `maneuver`, `critRange`), e é o único lugar do
// repositório que a tem.
func targetLabel(t engine.ModifierTarget) string {
	names := map[string]string{
		"attack": "Ataque", "damage": "Dano", "defense": "Defesa",
		"expertise": "Perícia", "expertiseAll": "Todas as perícias",
		"expertiseRemovePenalty": "Remove penalidade em", "expertiseByAttribute": "Perícias de",
		"attribute": "Atributo", "maxPv": "PV máximo", "maxPm": "PM máximo",
		"displacement": "Deslocamento", "damageReduction": "Redução de dano",
		"defenseDexCap": "Limite de Des na Defesa", "resistance": "Resistências",
		"fearResistance": "Resistência a medo", "critRange": "Margem de ameaça",
		"critMult": "Multiplicador crítico", "pmLimit": "Limite de PM por magia",
		"pmCost": "Custo em PM", "catalyst": "Catalisador", "spellDC": "CD de magias",
		"inventorySlots": "Espaços de carga", "flySpeed": "Voo",
		"armorPenalty": "Penalidade de armadura", "armorPenaltyExpertises": "Penalidade em perícias",
		"tempHp": "PV temporários", "tempMp": "PM temporários", "maneuver": "Manobra",
	}
	// FLAG é booleana e o rótulo dela é uma frase inteira ("Fadiga ao dormir"),
	// não um alvo com complemento — por isso ela sai antes do resto.
	if t.K == "flag" {
		if label, known := itemFlagLabel[t.Name]; known {
			return label
		}
		return t.Name
	}
	name, found := names[t.K]
	if !found {
		name = t.K
	}
	if complement := targetComplement(t); complement != "" {
		return name + " (" + complement + ")"
	}
	return name
}

// targetComplement é o que vem entre parênteses depois do alvo.
//
// O escopo `this` NÃO vira texto: o crachá está desenhado no próprio item, e
// "Ataque (deste item)" repete em palavras o que a posição já diz. Os outros
// escopos aparecem traduzidos — deixá-los crus poria `all` e `melee` na tela de
// alguém que joga em português.
func targetComplement(t engine.ModifierTarget) string {
	if t.Name != "" {
		return t.Name
	}
	if t.Attribute != "" {
		return t.Attribute
	}
	if t.School != "" {
		return t.School
	}
	scopeWords := map[string]string{"all": "todos", "melee": "corpo a corpo", "ranged": "à distância"}
	return scopeWords[t.Scope]
}

func circleLabel(circle int) string {
	return strconv.Itoa(circle) + "º círculo"
}

// itemFlagLabel é o pt-BR de cada flag sempre ativa. Mora ao lado de quem as
// desenha.
//
// Flag desconhecida cai no próprio id, que é feio e HONESTO: inventar uma frase
// para uma flag nova seria pior, porque a tela diria com confiança algo que
// ninguém escreveu.
var itemFlagLabel = map[string]string{
	"lethal-unarmed":              "Ataques desarmados causam dano letal",
	"cannot-apply-dex-to-defense": "Não soma Destreza na Defesa",
	"fatigue-on-sleep":            "Fadiga ao dormir",
	"reach-extends":               "Alcance ampliado",
	"armadura-pesada":             "Conta como armadura pesada",
	"auto-fail-reflexos":          "Falha automática em Reflexos",
}

// conditionalLabel é a frase que descreve QUANDO o modificador vale.
//
// O motor já monta essa nota (`describeCondition`), então aqui não há tradução
// nova — há a queda para o alvo quando a nota vem vazia, que é o caso de um
// modificador caseiro sem texto.
func conditionalLabel(c engine.ConditionalEffect) string {
	if note := strings.TrimSpace(c.Note); note != "" {
		return note
	}
	return targetLabel(c.Target)
}

// activeWrittenEffects é a linha que o leitor de tela ouve no lugar da pílula
// de contagem, com o singular certo. Um "1" solto é lido como "1", e o número
// sozinho não diz de quê.
func activeWrittenEffects(n int) string {
	if n == 1 {
		return "1 efeito ativo"
	}
	return strconv.Itoa(n) + " efeitos ativos"
}
