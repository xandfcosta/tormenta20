package book

// As duas devoções que não são um deus: quem cultua o Panteão (p57) e o Paladino do Bem (p82).
const (
	DevotionPantheon = "panteao"
	DevotionOfGood   = "bem"
)

// DevotionName é o nome da devoção de id dado.
func DevotionName(id string) string {
	switch id {
	case DevotionPantheon:
		return "Panteão"
	case DevotionOfGood:
		return "Paladino do Bem"
	}
	return GodName(id)
}

// DevotionWritten é a devoção como a ficha a escreve no cabeçalho.
//
//	DevotionWritten("khalmyr") // "Devoto de Khalmyr"
func DevotionWritten(id string) string {
	switch id {
	case DevotionPantheon:
		return "Devoto do Panteão"
	case DevotionOfGood:
		return "Paladino do Bem"
	}
	return "Devoto de " + GodName(id)
}

// GrantedPower é um poder que o deus concede a quem é devoto dele.
type GrantedPower struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	GodID       string `json:"deusId"`
}

// GrantedPowersOf são os poderes que o deus concede, na ordem do livro. O Panteão e o Paladino do Bem
// não concedem nenhum (p57).
func GrantedPowersOf(god string) []GrantedPower {
	out := []GrantedPower{}
	for _, power := range ListOf[GrantedPower]("divine-powers") {
		if power.GodID == god {
			out = append(out, power)
		}
	}
	return out
}
