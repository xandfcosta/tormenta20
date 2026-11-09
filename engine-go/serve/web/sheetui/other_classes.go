package sheetui

import (
	"t20engine/domain/book"
	"t20engine/domain/sheet"
)

// otherClass é uma classe da lista "Outra classe" do `+` do nível. Refusal diz por que a ficha não entra
// nela; vazia, a linha leva o gesto de entrar.
type otherClass struct {
	Name    string
	Refusal string
}

// otherClasses são as classes do livro que a ficha ainda não tem. Vazio no teto, porque a classe nova
// contaria o 21º nível. A que não serve ao deus da ficha (p96) fica na lista com o porquê, e sem o gesto:
// quem lê a lista sabe antes de tocar, e a recusa do `EnterClass` continua valendo para quem chamar a rota.
func otherClasses(dto sheet.CharacterDTO) []otherClass {
	var total int64
	owned := map[string]bool{}
	for _, cl := range dto.Classes {
		total += cl.Level
		owned[cl.ClassName] = true
	}
	if total >= 20 {
		return nil
	}
	_, classes, _ := book.CharacterCatalogs()
	out := make([]otherClass, 0, len(classes))
	for _, class := range classes {
		if owned[class.Name] {
			continue
		}
		entry := otherClass{Name: class.Name}
		if sheet.EnteringClassKeepsTheDevotion(dto, class.Name) != nil {
			entry.Refusal = "Não serve a " + book.DevotionName(*dto.God) + ": troque a devoção antes"
		}
		out = append(out, entry)
	}
	return out
}
