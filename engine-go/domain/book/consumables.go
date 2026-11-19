package book

// Consumable é o que um item faz quando se usa: o ganho imediato, a poção, a dose, o veneno.
type Consumable struct {
	Scope   string         `json:"scope"`
	Instant *ImmediateGain `json:"instant"`
	// Spell, Kind e Augment são a poção (p341) — a magia que o frasco contém,
	// como ela se ativa, e o aprimoramento que aquela linha da tabela compra.
	// A regra mora no `domain/catalog`, que é quem o motor lê.
	Spell   string `json:"spell"`
	Kind    string `json:"kind"`
	Augment string `json:"augment"`
	// Dose é o que o item faz na Injeção alquímica (p165), nula se ele não carrega.
	Dose *DoseEffect `json:"dose"`
	// Poison é o veneno (p161-162): a inoculação, o efeito e a CD; nulo no que não é veneno.
	Poison *PoisonEffect `json:"poison"`
}

// ImmediateGain é o PV/PM que um consumível devolve na hora. O `Dice` é a
// rolagem que a MESA faz — a ficha não rola por ninguém.
type ImmediateGain struct {
	HP *GainRoll `json:"hp"`
	MP *GainRoll `json:"mp"`
}

type GainRoll struct {
	Dice  string `json:"dice"`
	Bonus int    `json:"bonus"`
}

// PoisonEffect é o veneno (p161-162). A CD é de quem aplica, com a Inteligência (p161), e quem a
// escreve é a `sheet.WrittenPoison`; DcBonus é o que o próprio veneno soma a ela (a Beladona, +5).
type PoisonEffect struct {
	Inoculation string `json:"inoculation"`
	Effect      string `json:"effect"`
	OnSave      string `json:"onSave"`
	DcBonus     int    `json:"dcBonus"`
}
