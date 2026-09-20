package engine

import (
	"errors"
	"fmt"
)

// A ECONOMIA DE AÇÃO DO TURNO (T20 p233).
//
//	"No seu turno, você pode fazer uma ação padrão e uma ação de movimento, em
//	qualquer ordem. Você pode trocar sua ação padrão por uma ação de movimento,
//	para fazer duas ações de movimento, mas não pode fazer o inverso. Você
//	também pode abrir mão das duas para fazer uma ação completa."
//
// OS CINCO TIPOS JÁ ESTAVAM TRANSCRITOS no catálogo, no campo `action` das 411
// ativações — `passivo` 238, `livre` 72, `padrao` 27, `varia` 27, `reacao` 16,
// `completa` 16, `movimento` 15 —, e nada os consumia: o poder que custa uma
// ação padrão podia ser usado três vezes no mesmo turno (ALE-365).

// ActionCost é o que uma habilidade custa do turno de quem a usa.
//
// O valor sai em PORTUGUÊS porque é o que o catálogo escreve — ele é a
// transcrição do livro, e traduzi-la na leitura criaria duas grafias para um
// conceito, que é a raiz do defeito do `scope`.
type ActionCost string

const (
	ActionStandard ActionCost = "padrao"
	ActionMovement ActionCost = "movimento"
	ActionFull     ActionCost = "completa"
	ActionFree     ActionCost = "livre"
	ActionReaction ActionCost = "reacao"
	// ActionPassive não é uma ação: é a habilidade que vale sem ser acionada.
	// São 238 das 411, e elas passam pelo mesmo caminho porque o catálogo as
	// escreve no mesmo campo.
	ActionPassive ActionCost = "passivo"
	// ActionVaries é o custo NEGOCIADO com a mesa — 27 ativações. O turno não o
	// cobra, e a razão é a mesma do custo variável em PM: cobrar um número que
	// ninguém decidiu deixaria o poder indisponível por uma conta que não
	// existe.
	ActionVaries ActionCost = "varia"
)

// ErrNoActionLeft é a recusa por falta de ação no turno. Ela é reconhecível
// pelo chamador porque a tela precisa dizer "espere o próximo turno", que é
// outra frase de "não pode".
var ErrNoActionLeft = errors.New("não sobrou ação neste turno")

// TurnBudget é o que AINDA está de pé no turno de alguém.
//
// Dois booleanos e não um contador: o livro não diz "duas ações", ele diz UMA
// padrão e UMA de movimento, com uma troca de mão única entre elas. Um contador
// de dois perderia exatamente a assimetria que a troca descreve.
type TurnBudget struct {
	Standard bool `json:"standard"`
	Movement bool `json:"movement"`
}

// FullTurn é o turno de quem acabou de entrar nele.
func FullTurn() TurnBudget { return TurnBudget{Standard: true, Movement: true} }

// Spend devolve o turno DEPOIS de pagar o custo, ou a recusa.
//
// Ele não muta: o chamador decide se guarda o resultado, e é isso que deixa a
// TELA perguntar "daria para usar?" sem gastar nada — a mesma pergunta que o
// `UseDecision` faz para desenhar o botão.
func (b TurnBudget) Spend(custo ActionCost) (TurnBudget, error) {
	switch custo {
	// Não custam nada e cabem sempre: "como ações livres, reações tomam tão
	// pouco tempo que você pode realizar qualquer quantidade delas" (p233). A
	// passiva não é acionada, e a variável é decidida pela mesa.
	case ActionFree, ActionReaction, ActionPassive, ActionVaries:
		return b, nil

	case ActionStandard:
		if !b.Standard {
			return b, fmt.Errorf("%w para uma ação padrão", ErrNoActionLeft)
		}
		b.Standard = false
		return b, nil

	case ActionMovement:
		if b.Movement {
			b.Movement = false
			return b, nil
		}
		// A TROCA, e ela é de mão única: "você pode trocar sua ação padrão por
		// uma ação de movimento… mas não pode fazer o inverso" (p233). Por isso
		// ela mora AQUI e não no ramo da padrão.
		if b.Standard {
			b.Standard = false
			return b, nil
		}
		return b, fmt.Errorf("%w para mover", ErrNoActionLeft)

	case ActionFull:
		// "Você também pode abrir mão das DUAS para fazer uma ação completa": ela
		// exige as duas de pé, e não "o que sobrar".
		if !b.Standard || !b.Movement {
			return b, fmt.Errorf("%w, e uma ação completa exige o turno inteiro", ErrNoActionLeft)
		}
		return TurnBudget{}, nil
	}
	return b, fmt.Errorf("custo de ação %q não é um dos do livro (p233): padrao, movimento, completa, livre, reacao", custo)
}

// Spent diz se algo já foi gasto neste turno.
func (b TurnBudget) Spent() bool { return !b.Standard || !b.Movement }
