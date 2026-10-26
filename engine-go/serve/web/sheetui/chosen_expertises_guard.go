package sheetui

import (
	"net/http"

	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// keepsChosenExpertises recusa o gesto que tiraria a ficha da escolha do
// Especialista (p73, ALE-501): a mudança é aplicada numa cópia e conferida
// ANTES de gravar. Ficha sem escolha contada não paga a carga.
//
// Os gestos que passam por aqui são os que mexem no N ou no treino: treino,
// ofício, item vestido e atributo de raça. Usar item, melhoria e material não
// mudam a Inteligência permanente — a poção é efeito aplicado, que já não conta.
func (s Scene) keepsChosenExpertises(r *http.Request, row sqlcgen.Character, change func(*sheet.CharacterDTO)) error {
	if !sheet.HasCountedChoice(row.Powerchoices) &&
		!sheet.HasRaceAbilityPicks(row.Raceattributechoices, row.Secondaryracechoices) {
		return nil
	}
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	return sheet.ChangeKeepsChosenExpertises(dto, change)
}

// withoutItem tira o item da cópia; equippedAs o põe no lugar pedido.
func withoutItem(itemID int64) func(*sheet.CharacterDTO) {
	return func(dto *sheet.CharacterDTO) {
		kept := dto.Items[:0]
		for _, it := range dto.Items {
			if it.ID != itemID {
				kept = append(kept, it)
			}
		}
		dto.Items = kept
	}
}

func equippedAs(itemID int64, place *string) func(*sheet.CharacterDTO) {
	return func(dto *sheet.CharacterDTO) {
		for i := range dto.Items {
			if dto.Items[i].ID == itemID {
				dto.Items[i].Equipped = place
			}
		}
	}
}

// trainingToggled e craftRemoved são o treino e o ofício como o gesto os deixa.
func trainingToggled(name string) func(*sheet.CharacterDTO) {
	return func(dto *sheet.CharacterDTO) {
		for i := range dto.Expertises {
			if dto.Expertises[i].Name == name {
				dto.Expertises[i].Trained = !dto.Expertises[i].Trained
			}
		}
	}
}

func craftRemoved(name string) func(*sheet.CharacterDTO) {
	return func(dto *sheet.CharacterDTO) {
		kept := dto.Expertises[:0]
		for _, e := range dto.Expertises {
			if e.Name != name {
				kept = append(kept, e)
			}
		}
		dto.Expertises = kept
	}
}
