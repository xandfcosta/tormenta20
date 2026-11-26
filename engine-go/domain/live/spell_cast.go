package live

import "fmt"

// SpellCastRoll é a magia conjurada com o dado dela, rolado na ficha (ver GLOSSARY, **dado da magia**).
//
// NÃO É UM PROVISÓRIO, como o disparo da Flamejante (`BlastRoll`) não é: o mestre diz quem passou na resistência
// e aplica com os gestos de PV que já existem, e a faixa só INFORMA — o dado, o tipo, a resistência e a CD. Fica
// no estado da sessão até a próxima, pela razão do teste: quem chega atrasado tem de ver a magia que caiu.
type SpellCastRoll struct {
	// Who é o nome de quem conjurou, pronto, e Spell o nome da magia.
	Who   string `json:"who"`
	Spell string `json:"spell"`
	// Parts são as parcelas rodadas. ByText diz que um aprimoramento escolhido mudou o dado para um que não sai
	// na conjuração: não há parcela, e quem joga rola pelo texto.
	Parts []SpellCastPart `json:"parts,omitempty"`
	// Bonuses são as parcelas fixas que os poderes de quem conjurou somam (o Arcano de Batalha, o Aspecto do
	// Inverno; ver GLOSSARY, **dado da magia**), cada uma com o nome do poder.
	Bonuses []SpellCastBonus `json:"bonuses,omitempty"`
	// VitalShare são os PV que outra criatura recupera pela Comunhão Vital, a metade da cura (p58); zero sem ela.
	VitalShare int  `json:"vitalShare,omitempty"`
	ByText     bool `json:"byText,omitempty"`
	// Save é a resistência escrita ("Reflexos reduz à metade"), vazia na magia sem ela, e DC a CD de quem
	// conjurou. Range é o alcance da magia.
	Save  string `json:"save,omitempty"`
	DC    int    `json:"dc,omitempty"`
	Range string `json:"range"`
	// Kind é o que esta rolagem é (ver GLOSSARY, **rolagem informada**), e a cabeça da faixa sai dele (`Head`).
	// O zero é a conjuração, e é por isso que a faixa gravada antes do campo continua lendo como conjuração.
	Kind SpellBandKind `json:"kind,omitempty"`
	// Moment é "inicio" ou "fim" do turno: o do dado da vez, o da marca e o da condição.
	Moment string `json:"moment,omitempty"`
	// Option é a opção da ação da magia ("Esmagar"), ou, na conjuração que cria cargas, a ação que elas dão
	// ("Dardo de lava") — e Charges quantas.
	Option  string `json:"option,omitempty"`
	Charges int    `json:"charges,omitempty"`
	// Target é de quem é a vez em que a marca ou a condição rolou, e Condition o nome da condição, que NÃO é magia:
	// na condição, `Who` e `Spell` ficam vazios.
	Target    string `json:"target,omitempty"`
	Condition string `json:"condition,omitempty"`
	// Lingering é a marca que esta conjuração deixa em quem foi atingido, nil na que não deixa nada (ver GLOSSARY,
	// **marca na fila**): ela vira a `Markable` da mesa, a que o mestre põe nas linhas de quem falhou.
	Lingering   *QueueMark `json:"lingering,omitempty"`
	CharacterID int64      `json:"characterId"`
	ByUserID    int64      `json:"byUserId"`
}

// SpellCastBonus é uma parcela fixa de um poder: o nome dele, o número, o tipo quando tem, e a nota.
type SpellCastBonus struct {
	Name   string `json:"name"`
	Amount int    `json:"amount"`
	Kind   string `json:"kind,omitempty"`
	Note   string `json:"note,omitempty"`
}

// SpellBandKind é o tipo de uma rolagem da faixa da magia.
type SpellBandKind string

// Os tipos da faixa da magia. O da conjuração é o zero (ver o `SpellCastRoll.Kind`).
const (
	RollOfTheCast      SpellBandKind = ""
	RollOfTheTurn      SpellBandKind = "turn"
	RollOfTheAction    SpellBandKind = "action"
	RollOfTheMark      SpellBandKind = "mark"
	RollOfTheCondition SpellBandKind = "condition"
)

// Head é a cabeça da faixa, escrita pelo tipo da rolagem: "Arwen · Flecha Ácida · início do turno de Ogro".
//
//	live.SpellCastRoll{Kind: live.RollOfTheAction, Who: "Arwen", Spell: "Mão Poderosa de Talude", Option: "Esmagar"}.Head()
func (r SpellCastRoll) Head() string {
	switch r.Kind {
	case RollOfTheTurn:
		return r.Who + " · " + r.Spell + " · " + MomentWritten(r.Moment)
	case RollOfTheAction:
		return r.Who + " · " + r.Spell + " · " + r.Option
	case RollOfTheMark:
		// A marca do PERIGO não tem conjurador, e a cabeça começa pela origem.
		head := r.Spell + " · " + MomentWritten(r.Moment) + " de " + r.Target
		if r.Who == "" {
			return head
		}
		return r.Who + " · " + head
	case RollOfTheCondition:
		return r.Target + " · " + r.Condition + " · " + MomentWritten(r.Moment)
	}
	head := r.Who + " · " + r.Spell
	if r.Charges > 0 {
		head += fmt.Sprintf(" · %s ×%d", r.Option, r.Charges)
	}
	return head
}

// SpellCastPart é uma parcela rolada: o dado, o tipo, a condição do texto, os dados e o total. ByHand: o total
// veio da mesa, e não há dados.
type SpellCastPart struct {
	Dice   string `json:"dice"`
	Kind   string `json:"kind,omitempty"`
	Note   string `json:"note,omitempty"`
	Rolls  []int  `json:"rolls,omitempty"`
	Total  int    `json:"total"`
	ByHand bool   `json:"byHand,omitempty"`
	// Maximized diz que a parcela não rolou: o Espírito dos Solstícios a pôs no máximo (p62).
	Maximized bool `json:"maximized,omitempty"`
}

// recordIn guarda a magia conjurada no lugar da anterior, e a marca dela, se deixa uma, no lugar da última marca
// que o mestre podia pôr.
func (cast SpellCastRoll) recordIn(st *SessionRuntimeState) error {
	st.LastSpellCast = &cast
	if cast.Lingering != nil {
		mark := *cast.Lingering
		mark.Who = cast.Who
		st.Markable = &mark
	}
	return nil
}
