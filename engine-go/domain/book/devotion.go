package book

// As devoções que não são um dos vinte deuses: quem cultua o Panteão (p57), o Paladino do Bem (p82) e
// quem serve a Tauron, que morreu na luta contra a Tormenta (p25) e por isso não concede nada.
const (
	DevotionPantheon = "panteao"
	DevotionOfGood   = "bem"
	DevotionTauron   = "tauron"
)

// DevotionName é o nome da devoção de id dado.
func DevotionName(id string) string {
	switch id {
	case DevotionPantheon:
		return "Panteão"
	case DevotionOfGood:
		return "Paladino do Bem"
	case DevotionTauron:
		return "Tauron"
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
	return "Devoto de " + DevotionName(id)
}

// GrantedPower é um poder que o deus concede a quem é devoto dele.
type GrantedPower struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	GodID       string `json:"deusId"`
}

// GrantedPowersOf são os poderes que o deus concede, na ordem do livro. O Panteão, o Paladino do Bem e
// Tauron não concedem nenhum.
func GrantedPowersOf(god string) []GrantedPower {
	out := []GrantedPower{}
	for _, power := range ListOf[GrantedPower]("divine-powers") {
		if power.GodID == god {
			out = append(out, power)
		}
	}
	return out
}
