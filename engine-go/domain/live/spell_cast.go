package live

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
	Parts  []SpellCastPart `json:"parts,omitempty"`
	ByText bool            `json:"byText,omitempty"`
	// Save é a resistência escrita ("Reflexos reduz à metade"), vazia na magia sem ela, e DC a CD de quem
	// conjurou. Range é o alcance da magia.
	Save  string `json:"save,omitempty"`
	DC    int    `json:"dc,omitempty"`
	Range string `json:"range"`
	// When é o momento do dado da vez ("início do turno"), vazio na conjuração (ver GLOSSARY, **dado da vez**).
	When        string `json:"when,omitempty"`
	CharacterID int64  `json:"characterId"`
	ByUserID    int64  `json:"byUserId"`
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
}

// recordIn guarda a magia conjurada no lugar da anterior.
func (cast SpellCastRoll) recordIn(st *SessionRuntimeState) error {
	st.LastSpellCast = &cast
	return nil
}
