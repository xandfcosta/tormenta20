package book

import (
	"encoding/json"
	"slices"
	"sync"

	"t20engine/domain/catalog"
	"t20engine/domain/engine"
)

// O REGISTRO DE ATIVAÇÕES: o que um poder custa para ser usado.
//
// A tabela vem do `transcript.Activations`, e é o que
// responde "este poder se ativa, e com quê". Ela mora aqui e não na cena porque
// **o destino de uma função é a DEPENDÊNCIA dela**: quem lê o catálogo é do
// livro.
//
// A DECISÃO a partir dela — se o poder pode ser usado agora, quanto custa um
// degrau de postura, e a frase da recusa — mudou de lado na ALE-351: ela morava
// na cena, e hoje é o `activation_rules.go`, ao lado. A razão é a mesma frase
// acima: quem DECIDE é o caso de uso, e ele não alcança o `serve/web`.
//
// O que continua sendo da tela é o CRACHÁ ("1/cena", "3/dia") — a mesma entrada
// responde duas coisas, e só uma delas é cobrada.

// Activation é a entrada do registro de ativações: id, nome, tipo, PM e página,
// mais a AÇÃO que o uso consome, o limite, a flag que o gatilho exige e a escala
// da postura.
type Activation struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Action string `json:"action"`
	// PmCost é CRU porque o catálogo escreve duas coisas nele: um número, ou a
	// palavra "variavel" — 33 das 411 ativações são variáveis. Tipar como `int`
	// NÃO estoura: o `json.Unmarshal` de uma lista guarda o erro de tipo daquele
	// campo, segue em frente e deixa ZERO no lugar. O poder de custo variável
	// apareceria como "0 PM" com o botão ATIVO, e usá-lo não cobraria nada.
	PmCost json.RawMessage `json:"pmCost"`
	// Uses é `null`, "cena", "dia" ou um número — por isso ele é cru: os três
	// significam coisas diferentes e só dois são cobrados.
	Uses json.RawMessage `json:"uses"`
	// Flag é a que ESTA postura acende, e ela é DECLARADA.
	//
	// Antes ela era derivada dos modificadores do poder de mesmo id, e a razão
	// era boa: a flag do catálogo e a que os modificadores usam têm de ser a
	// MESMA, e derivar tornava a divergência impossível. O que quebrou a
	// derivação foram as Posturas de Combate do cavaleiro (p54): três das seis
	// não têm modificador NENHUM — o que elas fazem é do mestre —, então não há
	// de onde derivar, e sem flag elas nem seriam reconhecidas como postura.
	//
	// A segurança que a derivação dava não se perdeu: ela desceu para o
	// `TestEveryDeclaredStanceFlagMatchesItsModifiers`, que prende as duas ao
	// mesmo valor quando há modificador.
	//
	// No instantâneo que concede `switch`, ela é a flag que USAR liga: a
	// declaração do Estilo de Duas Armas (p125).
	Flag string `json:"flag"`
	// StanceGroup é o conjunto de posturas MUTUAMENTE EXCLUSIVAS a que esta
	// pertence — *"você só pode manter uma postura por vez"* (p54).
	//
	// Vazio quer dizer que ela não exclui ninguém, que é o caso da Fúria e da
	// Inspiração: elas são de classes diferentes, e um bárbaro/bardo mantém as
	// duas. A exclusão é do GRUPO e não de toda postura, porque é isso que o
	// livro diz — a regra está no quadro do cavaleiro, não nas regras gerais.
	StanceGroup  string           `json:"stanceGroup"`
	RequiresFlag string           `json:"requiresFlag"`
	Scaling      *ActivationScale `json:"scaling"`
	// SkillSpend é o PM que se paga NUM TESTE DE PERÍCIA para mexer nele —
	// Audácia, Orgulho, Eclético… (ALE-489). Ver `skill_spend.go`.
	SkillSpend *SkillSpend `json:"skillSpend,omitempty"`
	Grant      *PowerGrant `json:"grant"`
	// Cumulative é o bônus que SOBE A CADA GATILHO — ver `CumulativeBonus`.
	Cumulative *CumulativeBonus `json:"cumulative"`
	// Recovers é o PM que um evento da mesa devolve ao pool, até o máximo (o Panache,
	// p48). O gatilho é o mesmo do bônus cumulativo (`CumulativeTriggerIsKnown`).
	Recovers *Recovery `json:"recovers"`
	// Rider é o que este poder ACRESCENTA ao uso de outro — ver
	// `ActivationRider`.
	Rider      *ActivationRider `json:"rider,omitempty"`
	AllyTarget bool             `json:"allyTarget,omitempty"`
	BookPage   int              `json:"bookPage"`
}

// CumulativeBonus é o bônus que sobe a cada gatilho enquanto a postura dura —
// o *"bônus cumulativo de +1"* da Sangue dos Inimigos (p42).
//
// ELE NÃO É UM `engine.Modifier`, e não poderia ser: o `Modifier` descreve um
// bônus de valor FIXO, estático ou condicionado por flag, e o catálogo é dado
// transcrito do livro — não há onde escrever "o valor de agora". O que o
// catálogo declara aqui é a REGRA de crescimento; o valor corrente é estado de
// jogo, e mora num efeito ativo de escopo `scene` como qualquer outro.
//
// Por isso ele também é irmão do `Grant` e não um campo dele: o `Grant` é
// aplicado UMA vez, ao entrar na postura, e este é reaplicado a cada gatilho.
// Recovery é o PM que volta ao pool quando o gatilho `On` acontece.
type Recovery struct {
	On     string `json:"on"`
	Amount int    `json:"amount"`
}

type CumulativeBonus struct {
	// On é o gatilho, e o vocabulário é FECHADO: um valor que o motor não
	// conhece não acumula nada, e o `CumulativeTriggerIsKnown` é quem recusa.
	On string `json:"on"`
	// Amount é quanto cada gatilho acrescenta.
	Amount int `json:"amount"`
	// CapPer é de onde sai o TETO — *"limitado pelo seu nível"* (p42).
	CapPer string `json:"capPer"`
	// Targets são os números que ele move, no mesmo vocabulário de alvo que
	// todo modificador do catálogo usa. Reusado e não inventado: um segundo
	// vocabulário de alvo divergiria do primeiro no dia seguinte.
	Targets []engine.ModifierTarget `json:"targets"`
}

// ActivationRider é o poder que não tem gesto: ele entra no efeito de OUTRO
// quando a ficha o usa — o Espreitar, o Ponto Fraco e o Inimigo de (Criatura)
// na Marca da Presa (p51, ALE-509). Ver `GrantModifiersWithRiders`.
type ActivationRider struct {
	// On é a ativação que ele acompanha.
	On string `json:"on"`
	// Modifiers somam ao efeito, e PerStep acrescenta ao Amount de cada um por
	// degrau pago ("+1 para cada PM adicional gasto na habilidade").
	Modifiers []engine.Modifier `json:"modifiers,omitempty"`
	PerStep   int               `json:"perStep,omitempty"`
	// Doubles dobra o efeito inteiro quando quem usa diz que vale — o Inimigo,
	// contra a presa do tipo escolhido.
	Doubles bool `json:"doubles,omitempty"`
	// StepLabel troca o que cada degrau do gesto compra, e Note é o aviso que
	// ele deixa na linha do gesto: o que muda e não é número que o motor guarde
	// — a Compaixão leva a Cura pelas Mãos ao alcance curto e ao 2d6+1 (p83).
	// Ver `ActivationRidden`.
	StepLabel string `json:"stepLabel,omitempty"`
	Note      string `json:"note,omitempty"`
}

// ActivationScale é a postura que sobe de degrau com o nível.
type ActivationScale struct {
	BasePm          int    `json:"basePm"`
	StepPm          int    `json:"stepPm"`
	StepLabel       string `json:"stepLabel"`
	FirstStepLevel  int    `json:"firstStepLevel"`
	StepEveryLevels int    `json:"stepEveryLevels"`
	// StepDice é o dado que cada degrau compra quando ele troca de linha na
	// tabela da classe — ver `ScaledGrantModifiers`.
	StepDice []string `json:"stepDice,omitempty"`
	// CapAttribute e CapPm são o teto do custo VARIÁVEL que o verbete escreve
	// — "limitado pelo seu Carisma" (Gritar Ordens, p80), "até o 4º círculo"
	// (Ladrão Arcano, p74) —, por baixo do teto de nível da p224. A escala que
	// não sobe por nível (`FirstStepLevel` zero) é deste modo (ALE-506).
	CapAttribute string `json:"capAttribute,omitempty"`
	CapPm        int    `json:"capPm,omitempty"`
}

// PowerGrant é o que a ativação APLICA na ficha, em três espécies: a RESERVA de
// PV temporários (`temp-hp`, com o atributo que a dimensiona), o EFEITO ATIVO
// (`active-effect`, com os modificadores que ele liga) e a DECLARAÇÃO
// (`switch`, que liga a `Flag` da ativação até o próximo turno — ver `DeclaredFlag`).
//
// Os `Modifiers` não existiam aqui, e o catálogo os trazia desde sempre: três
// concessões escreviam a lista no JSON e o Go a DESCARTAVA na leitura, sem
// erro nenhum. O En Garde, a Armadura de Allihanna e o Dom da Verdade eram
// dado inerte por isso, antes mesmo de não haver quem os aplicasse (ALE-426).
type PowerGrant struct {
	Kind      string `json:"kind"`
	Scope     string `json:"scope"`
	Attribute string `json:"attribute"`
	// AttributeOnly é a reserva que é SÓ o atributo, mínimo 1 (o Estandarte, p54);
	// sem ele, a reserva é nível + atributo (a Armadura da Honra, p53).
	AttributeOnly bool `json:"attributeOnly"`
	// Modifiers é o que o efeito ativo liga. Vazio na reserva de PV, que se
	// dimensiona pelo `Attribute` e não por lista.
	Modifiers []engine.Modifier `json:"modifiers"`
}

// Activations é o registro, com a página HERDADA do dono: o poder de classe,
// o geral, o item, o poder do deus, achado pela regra única do dono (ver
// `ownerPage`). A ativação escreve página só quando nenhum dono tem.
//
// Ele é montado UMA vez: o registro é embutido e não muda, e decodificá-lo a
// cada pergunta custava a maior parte de cada página da ficha. Quem chama
// recebe uma cópia da lista, para não mexer na de todos.
func Activations() []Activation {
	return slices.Clone(activationsOnce())
}

var activationsOnce = sync.OnceValue(func() []Activation {
	list := transcribedActivations()
	for i, a := range list {
		if page := ownerPage(a); page > 0 {
			list[i].BookPage = page
		}
	}
	return list
})

// transcribedActivations é o registro como foi escrito, sem herança.
func transcribedActivations() []Activation {
	raw, ok := catalog.Resource("activations")
	if !ok {
		return nil
	}
	var list []Activation
	_ = json.Unmarshal(raw, &list)
	return list
}
