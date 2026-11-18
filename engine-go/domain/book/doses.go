package book

// DoseEffect é o efeito de uma dose; a frase com o dano e a CD quem escreve é a `sheet.WrittenDose`.
type DoseEffect struct {
	// Damage é o dano da dose, separado do resto da frase (Effect) para o motor somar nele sem ler
	// texto; nil quando a dose não fere.
	Damage *DoseDamage `json:"damage"`
	// Alchemical diz que o item é um PREPARADO alquímico (p160): o dano dele recebe o que o motor dá
	// ao dano de preparado (o Alquimista de Batalha, p68). A água benta fere e não é preparado (p155).
	Alchemical    bool   `json:"alchemical"`
	Effect        string `json:"effect"`
	Save          string `json:"save"`
	SaveAttribute string `json:"saveAttribute"`
	OnSave        string `json:"onSave"`
}

// DoseDamage é o dado e o tipo do dano de uma dose: "2d4" de "ácido".
type DoseDamage struct {
	Dice string `json:"dice"`
	Type string `json:"type"`
}
