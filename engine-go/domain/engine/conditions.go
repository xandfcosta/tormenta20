package engine

// O DESTINO DE CADA CONDIÇÃO (p394-395, ver GLOSSARY, **condição**).
//
// Os NÚMEROS que uma condição muda moram no `conditionModifierTable` (collect.go), que o
// `cmd/audit/conditions` confere contra o PDF. Esta tabela é o resto: o que a condição faz com a AÇÃO,
// o dado que ela rola no início da vez, o mecanismo de fora que a aplica, e o que fica com o mestre — e
// por quê. Uma condição pode estar nas duas (o Petrificado tem a RD e fica inconsciente), e toda
// condição do livro está em pelo menos uma: é o `TestEveryBookConditionHasADestination`.

// ConditionActs é o que uma condição faz com a ação de quem a tem.
type ConditionActs int

const (
	// ActsNormally é a condição que não mexe na ação.
	ActsNormally ConditionActs = iota
	// NoAction não age, e ainda reage (p233).
	NoAction
	// Unconscious não age nem reage: "sem ações (incluindo reações)" (p395).
	Unconscious
	// OneAction tem uma ação só no turno, padrão ou de movimento (p394).
	OneAction
)

// ConditionOutcome é uma linha da tabela do dado da condição: até que total ela vale, e o que manda.
type ConditionOutcome struct {
	UpTo int
	Says string
}

// ConditionTurnRoll é o dado que a condição rola no início da vez de quem a tem.
type ConditionTurnRoll struct {
	// ID é a condição, e Name o nome dela na fila e na faixa ("Em Chamas").
	ID, Name string
	Dice     string
	// Kind é o tipo do dano, vazio no dado que não é dano (o do Confuso).
	Kind string
	// Outcomes é a tabela do resultado, em ordem de `UpTo`; vazia no dado que é só dano.
	Outcomes []ConditionOutcome
}

// ConditionRule é o que a condição faz além dos números.
type ConditionRule struct {
	Acts     ConditionActs
	TurnRoll *ConditionTurnRoll
	// Elsewhere é o mecanismo de fora que aplica a condição, dito para quem a procurar aqui.
	Elsewhere string
	// ByMaster é o que fica com o mestre por desenho, e por quê.
	ByMaster string
}

// hasDestination diz se a linha diz alguma coisa: uma linha vazia na tabela seria um destino de mentira.
func (r ConditionRule) hasDestination() bool {
	return r.Acts != ActsNormally || r.TurnRoll != nil || r.Elsewhere != "" || r.ByMaster != ""
}

// conditionRules é o que cada condição faz além dos números, citando a frase do livro que a obriga.
var conditionRules = map[string]ConditionRule{
	// "não pode fazer ações" (p394-395). O fascinado entra porque a exceção dele, "observar aquilo que o
	// fascinou", é um teste de Percepção, e o turno não cobra teste de perícia (decisão do dono).
	"atordoado":    {Acts: NoAction},
	"pasmo":        {Acts: NoAction},
	"surpreendido": {Acts: NoAction},
	"fascinado":    {Acts: NoAction},
	// "sem ações (incluindo reações)" (p395); o petrificado "fica inconsciente" (p395), e a ficha grava só o
	// nome dele.
	ConditionUnconscious: {Acts: Unconscious},
	"petrificado":        {Acts: Unconscious},
	// "só pode realizar uma ação padrão ou de movimento (não ambas) por rodada" (p394).
	ConditionSickened: {Acts: OneAction},
	// "No início de seus turnos, sofre 1d6 pontos de dano de fogo" (p394).
	"em-chamas": {TurnRoll: &ConditionTurnRoll{ID: "em-chamas", Name: "Em Chamas", Dice: "1d6", Kind: "fogo"}},
	// "Role 1d6 no início de seus turnos" (p394), e a tabela em paráfrase. Quem aplica é o mestre: o app não
	// tira a ação nem a condição (decisão do dono).
	"confuso": {TurnRoll: &ConditionTurnRoll{ID: "confuso", Name: "Confuso", Dice: "1d6", Outcomes: []ConditionOutcome{
		{UpTo: 1, Says: "anda numa direção sorteada por 1d8"},
		{UpTo: 3, Says: "não age, e balbucia"},
		{UpTo: 5, Says: "ataca a criatura mais próxima com a arma empunhada, ou a si mesmo se estiver sozinho"},
		{UpTo: 6, Says: "a condição termina, e age normalmente"},
	}}},
	ConditionBleeding: {Elsewhere: "o teste de Constituição CD 15 no início da vez (p236): `BleedingCheckPasses`"},
	"sobrecarregado": {Elsewhere: "a carga (`loadBreakdownOf`), que dá a penalidade de armadura e o deslocamento; " +
		"ela não se marca à mão (decisão do dono)"},
	"paralisado": {ByMaster: "as \"ações puramente mentais\" (p395): o app não sabe quais ações são mentais"},
	"enfeiticado": {ByMaster: "o +10 vale nos testes de Diplomacia da FONTE com quem está enfeitiçado (p394), " +
		"e o teste de perícia do app não tem alvo"},
	"doente":     {ByMaster: "o efeito é o da doença, que o livro descreve em cada uma (p394)"},
	"envenenado": {ByMaster: "o efeito é o do veneno — perda de vida ou outra condição (p394)"},
}

// conditionActs é o que a condição faz com a ação, ou `ActsNormally` na que não mexe nela.
func conditionActs(id string) ConditionActs { return conditionRules[id].Acts }

// ConditionTurnRolls são os dados que estas condições rolam no início da vez, na ordem em que estão.
//
// @example ConditionTurnRolls([]string{"em-chamas"}) // [{ID: "em-chamas", Name: "Em Chamas", Dice: "1d6", Kind: "fogo"}]
func ConditionTurnRolls(conditions []string) []ConditionTurnRoll {
	var rolls []ConditionTurnRoll
	for _, id := range conditions {
		if roll := conditionRules[id].TurnRoll; roll != nil {
			rolls = append(rolls, *roll)
		}
	}
	return rolls
}

// ConditionRollSays é o que o total do dado da condição manda fazer, pela tabela dela, ou vazio na condição sem
// tabela.
//
// @example ConditionRollSays("confuso", 3) // "não age, e balbucia"
func ConditionRollSays(id string, total int) string {
	roll := conditionRules[id].TurnRoll
	if roll == nil {
		return ""
	}
	for _, outcome := range roll.Outcomes {
		if total <= outcome.UpTo {
			return outcome.Says
		}
	}
	return ""
}
