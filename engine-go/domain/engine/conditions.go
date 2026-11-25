package engine

import (
	"fmt"

	"t20engine/domain/catalog/transcript"
)

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
	// ActsNot não age, e ainda reage (p233).
	ActsNot
	// ActsUnconscious não age nem reage: "sem ações (incluindo reações)" (p395).
	ActsUnconscious
	// ActsOnce tem uma ação só no turno, padrão ou de movimento (p394).
	ActsOnce
)

// ConditionOutcome é uma linha da tabela do dado da condição: até que total ela vale, e o que manda.
type ConditionOutcome struct {
	UpTo int
	Says string
}

// ConditionTurnRoll é o dado que a condição rola no início da vez de quem a tem.
type ConditionTurnRoll struct {
	// ID é a condição, e Name o nome dela na fila e na faixa ("Em Chamas"). Os dois vêm da chave da tabela e
	// do catálogo, no `ConditionTurnRolls`: escritos na linha, seriam uma segunda grafia do mesmo fato.
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
	// Elsewhere é o mecanismo de fora que aplica a condição, escrito para uma PESSOA: ele sai na recusa de marcar
	// a condição derivada.
	Elsewhere string
	// Derived diz que é o mecanismo de `Elsewhere` que dá a condição, e ninguém a marca à mão: a marca seria um
	// sinal que nada lê.
	Derived bool
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
	"atordoado":    {Acts: ActsNot},
	"pasmo":        {Acts: ActsNot},
	"surpreendido": {Acts: ActsNot},
	"fascinado":    {Acts: ActsNot},
	// "sem ações (incluindo reações)" (p395); o petrificado "fica inconsciente" (p395), e a ficha grava só o
	// nome dele.
	ConditionUnconscious: {Acts: ActsUnconscious},
	"petrificado":        {Acts: ActsUnconscious},
	// "só pode realizar uma ação padrão ou de movimento (não ambas) por rodada" (p394).
	ConditionSickened: {Acts: ActsOnce},
	// "No início de seus turnos, sofre 1d6 pontos de dano de fogo" (p394).
	"em-chamas": {TurnRoll: &ConditionTurnRoll{Dice: "1d6", Kind: "fogo"}},
	// "Role 1d6 no início de seus turnos" (p394), e a tabela em paráfrase. Quem aplica é o mestre: o app não
	// tira a ação nem a condição (decisão do dono).
	"confuso": {TurnRoll: &ConditionTurnRoll{Dice: "1d6", Outcomes: []ConditionOutcome{
		{UpTo: 1, Says: "anda numa direção sorteada por 1d8"},
		{UpTo: 3, Says: "não age, e balbucia"},
		{UpTo: 5, Says: "ataca a criatura mais próxima com a arma empunhada, ou a si mesmo se estiver sozinho"},
		{UpTo: 6, Says: "a condição termina, e age normalmente"},
	}}},
	ConditionBleeding: {Elsewhere: "o teste de Constituição CD 15 no início da vez (p236)"},
	// Só a carga a dá — nenhuma magia ou poder do livro a impõe —, e por isso ela não se marca à mão (decisão do
	// dono).
	"sobrecarregado": {Elsewhere: "a carga que o personagem leva (p141)", Derived: true},
	"paralisado":     {ByMaster: "as \"ações puramente mentais\" (p395): o app não sabe quais ações são mentais"},
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
			named := *roll
			named.ID, named.Name = id, conditionNameOf(id)
			rolls = append(rolls, named)
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

// HandMarkRefusal é nil na condição que se marca à mão; na que o app deriva, é a recusa, dizendo de onde ela vem.
// Ela barra só o que ENTRA: a marca gravada antes desta regra ainda sai.
//
// @example HandMarkRefusal("sobrecarregado") // "Sobrecarregado não se marca à mão: quem a dá é a carga (…)"
func HandMarkRefusal(id string) error {
	rule := conditionRules[id]
	if !rule.Derived {
		return nil
	}
	return fmt.Errorf("%s não se marca à mão: quem a dá é %s", conditionNameOf(id), rule.Elsewhere)
}

// conditionNameOf é o nome da condição no livro ("Em Chamas"), ou o id na que o catálogo não tem.
func conditionNameOf(id string) string {
	for _, c := range transcript.Conditions {
		if c.ID == id {
			return c.Name
		}
	}
	return id
}
