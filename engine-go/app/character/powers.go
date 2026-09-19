package character

import (
	"strings"

	"t20engine/domain/search"
	"t20engine/domain/sheet"
)

// O QUE UM PODER SABE DA FICHA — duas leituras que a decisão de ativar precisa,
// e que a TELA também desenha.
//
// Elas moravam na cena e subiram com o gesto (ALE-351): a decisão de usar um
// poder e o botão que a mostra têm de concordar, e duas cópias concordariam só
// até alguém mexer numa.

// ClassPowerLevel é o nível que vale para este poder: o da CLASSE, e não o do
// personagem (T20 p40).
//
// Um bárbaro 5/ladino 5 tem a Fúria de um bárbaro de nível 5, e não a de um
// personagem de nível 10. A classe é reconhecida pelo pedaço do id
// (`class.barbaro.furia`), e a dobra de acento é a do `domain/search` — a mesma
// que a busca do livro usa, para não haver duas tabelas de acento no projeto.
func ClassPowerLevel(dto sheet.CharacterDTO, activationID string) int {
	for _, classe := range dto.Classes {
		if strings.Contains(activationID, "."+search.Fold(classe.ClassName)+".") {
			return int(classe.Level)
		}
	}
	return int(dto.Level)
}

// PowerUse é quanto um poder já foi usado nesta cena e neste dia.
type PowerUse struct{ Cena, Dia int }

// PowerUses agrupa os usos gravados por poder.
func PowerUses(dto sheet.CharacterDTO) map[string]PowerUse {
	fora := map[string]PowerUse{}
	for _, u := range dto.PowerUses {
		conta := fora[u.PowerID]
		if u.Scope == "scene" {
			conta.Cena = int(u.Used)
		} else {
			conta.Dia = int(u.Used)
		}
		fora[u.PowerID] = conta
	}
	return fora
}
