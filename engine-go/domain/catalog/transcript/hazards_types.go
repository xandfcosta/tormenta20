package transcript

// Hazard é um perigo do Capítulo 7 (p317-319): a ameaça que não é criatura (ver GLOSSARY, **perigo**).
type Hazard struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Kind é "armadilha", "doenca" ou "ambiente".
	Kind     string `json:"kind"`
	BookPage int    `json:"bookPage"`
	UID      string `json:"uid"`
	// Description é a regra em PARÁFRASE, com os números do livro. A regra que mora noutro lugar diz "veja a página N"
	// (o clima, na p267), e a frase vira elo para o livro.
	Description string         `json:"description"`
	Trap        *HazardTrap    `json:"trap,omitempty"`
	Disease     *HazardDisease `json:"disease,omitempty"`
}

// HazardTrap é a ficha da armadilha (p317): o que ela causa, a resistência, as CDs para achá-la e desarmá-la e o ND.
type HazardTrap struct {
	ND float64 `json:"nd"`
	// Magical é o nome em ITÁLICO do livro: a armadilha é mágica e Dissipar Magia a anula.
	Magical bool           `json:"magical,omitempty"`
	Damage  []HazardDamage `json:"damage,omitempty"`
	// Area é quem ela alcança, quando não é só quem a disparou ("todas as criaturas a até 3m").
	Area string      `json:"area,omitempty"`
	Save *HazardSave `json:"save,omitempty"`
	// Investigation e Thievery são as CDs de Investigação (achar) e de Ladinagem (desarmar).
	Investigation int `json:"investigation"`
	Thievery      int `json:"thievery"`
}

// HazardDamage é uma parcela do dano da armadilha.
type HazardDamage struct {
	Dice string `json:"dice"`
	Kind string `json:"kind,omitempty"`
}

// HazardSave é a resistência da vítima: o teste, a CD, e o que passar faz ("evita", "reduz à metade").
type HazardSave struct {
	Test   string `json:"test"`
	DC     int    `json:"dc"`
	Effect string `json:"effect"`
}

// HazardDisease é a ficha da doença (p318): o contágio, a CD de Fortitude e os efeitos, um por falha seguida.
type HazardDisease struct {
	Contagion string   `json:"contagion"`
	DC        int      `json:"dc"`
	Effects   []string `json:"effects"`
}
