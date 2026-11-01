package sheetui

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/go-chi/chi/v5"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// swapChoice é o gesto de trocar a escolha de um nível: o poder novo, e a opção dele quando ele
// pede uma. Treino e proficiência acompanham, como quando se escolhe.
func swapChoice(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	rowID, err := strconv.ParseInt(chi.URLParam(r, "escolha"), 10, 64)
	if err != nil {
		return fmt.Errorf("escolha %q não é um número", chi.URLParam(r, "escolha"))
	}
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	swap, err := sheet.SwapLevelChoice(dto, rowID, chi.URLParam(r, "poder"), pathParam(r, "opcao"))
	if err != nil {
		return err
	}
	if err := s.plays.ApplyLevelSwap(r.Context(), row, swap); err != nil {
		return err
	}
	return s.settlePickEffects(r, row, dto, swap.Removed, swap.Added)
}

// freeChoice libera a escolha que impede o atributo de baixar. A perícia continua treinada: o
// Especialista escolhe entre as treinadas, e não as treina (p73).
func freeChoice(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	rowID, err := strconv.ParseInt(chi.URLParam(r, "escolha"), 10, 64)
	if err != nil {
		return fmt.Errorf("escolha %q não é um número", chi.URLParam(r, "escolha"))
	}
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	freed, err := sheet.FreeLevelChoice(dto, rowID)
	if err != nil {
		return err
	}
	return s.plays.ApplyLevelSwap(r.Context(), row, freed)
}

// settlePickEffects leva ao treino e às proficiências o que entrou e saiu. A proficiência marcada à
// mão fica: só sai a que o poder dava e que nada mais concede (ALE-510).
func (s Scene) settlePickEffects(
	r *http.Request, row sqlcgen.Character, before sheet.CharacterDTO, removed, added []sheet.LevelChoiceDTO,
) error {
	wants := map[string]bool{}
	for _, pick := range removed {
		wants[trainingOf(pick)] = false
	}
	for _, pick := range added {
		wants[trainingOf(pick)] = true
	}
	delete(wants, "")
	for _, expertise := range before.Expertises {
		if want, moved := wants[expertise.Name]; moved && want != expertise.Trained {
			if err := s.plays.ToggleTraining(r.Context(), row.ID, expertise.Name); err != nil {
				return err
			}
		}
	}
	fresh, err := s.deps.Queries().GetCharacter(r.Context(), row.ID)
	if err != nil {
		return err
	}
	after, err := s.deps.LoadCharacter(r.Context(), fresh)
	if err != nil {
		return err
	}
	lost, gained := sheet.PowerProficiencies(before.PowerChoices), sheet.PowerProficiencies(after.PowerChoices)
	if slices.Equal(lost, gained) {
		return nil
	}
	kept := []string{}
	for _, key := range sheet.UnmarshalStrings(after.Proficiencies) {
		if !slices.Contains(lost, key) || len(grantersOf(after, key)) > 0 {
			kept = append(kept, key)
		}
	}
	for _, key := range gained {
		if !slices.Contains(kept, key) {
			kept = append(kept, key)
		}
	}
	return s.plays.SaveProficiencies(r.Context(), row.ID, kept)
}

// trainingOf é a perícia que esta escolha treina, quando treina (p131).
func trainingOf(pick sheet.LevelChoiceDTO) string {
	power, chooses := book.PowerWithChoice(pick.Ref)
	if pick.Kind != sheet.LevelKindPick || !chooses || power.Choice.Kind != "untrained-expertise" {
		return ""
	}
	return pick.Pick
}

// swapDialogView é o diálogo de trocar uma escolha.
type swapDialogView struct {
	RowID   int64
	Level   int64
	Current string
	Options []swapOption
}

type swapOption struct {
	Label  string
	Target string
}

// swapDialogOf monta o diálogo do que se pediu para trocar. As opções são desenhadas só aqui, sob
// pedido, porque cada escolha tem dezenas de candidatas e a ficha tem até dezenove escolhas.
func swapDialogOf(dto sheet.CharacterDTO, rowID int64) *swapDialogView {
	for _, entry := range sheet.LevelHistory(dto) {
		for _, line := range entry.Lines {
			if line.RowID != rowID || rowID == 0 {
				continue
			}
			view := &swapDialogView{RowID: rowID, Level: entry.Level, Current: line.Name}
			row := rowOf(dto, rowID)
			if row.Kind == sheet.LevelKindPick {
				view.Current += ", " + line.Detail
			}
			view.Options = swapOptionsOf(dto, row, entry.ClassName)
			return view
		}
	}
	return nil
}

func rowOf(dto sheet.CharacterDTO, id int64) sheet.LevelChoiceDTO {
	for _, row := range dto.LevelChoices {
		if row.ID == id {
			return row
		}
	}
	return sheet.LevelChoiceDTO{}
}

// swapOptionsOf são as candidatas. A escolha que não ocupa vaga, como o animal do Totem, só troca
// de opção dentro do mesmo poder; a que ocupa troca por qualquer poder daquele nível.
func swapOptionsOf(dto sheet.CharacterDTO, row sheet.LevelChoiceDTO, class string) []swapOption {
	chosen := sheet.UnmarshalStrings(dto.ClassPowers)
	picks := sheet.PowerPicks(dto.PowerChoices)
	owned := ownedOptionsOf(dto)
	out := []swapOption{}
	power, chooses := book.PowerWithChoice(row.Ref)
	if row.Kind == sheet.LevelKindPick && chooses && !power.Choice.Repeatable {
		for _, option := range optionsOf(power, picks[row.Ref], owned) {
			if option.Value != row.Pick {
				out = append(out, swapOption{
					Label: power.Name + ", " + option.Label, Target: row.Ref + "/" + url.PathEscape(option.Value),
				})
			}
		}
		return out
	}
	for _, candidate := range thatChoosePowers(class, chosen, picks, owned, "") {
		if len(candidate.Options) == 0 {
			if !candidate.Chosen {
				out = append(out, swapOption{Label: candidate.Name, Target: candidate.ID})
			}
			continue
		}
		for _, option := range candidate.Options {
			if candidate.ID == row.Ref && option.Value == row.Pick {
				continue
			}
			out = append(out, swapOption{
				Label: candidate.Name + ", " + option.Label, Target: candidate.ID + "/" + url.PathEscape(option.Value),
			})
		}
	}
	return out
}

// swapGet abre o diálogo pedindo a cena de novo, e não a página: dentro da Mesa o gesto não pode
// sair dela. O diálogo da vez anterior sai antes, porque o `data-init` só roda em elemento novo.
func swapGet(v View, rowID int64) string {
	return fmt.Sprintf("document.getElementById('swap-dialog-host')?.remove(); @get('%s&trocar=%d')", commandRoute(v, ""), rowID)
}

func swapPost(v View, rowID int64, target string) string {
	return sheetPost(v, fmt.Sprintf("/escolhas/%d/troca/%s", rowID, target))
}

func freePost(v View, rowID int64) string {
	return sheetPost(v, fmt.Sprintf("/escolhas/%d/libera", rowID))
}

// byLevelGet troca entre as duas vistas da aba.
func byLevelGet(v View, byLevel bool) string {
	v.ByLevel = byLevel
	return fmt.Sprintf("@get('%s')", commandRoute(v, ""))
}
