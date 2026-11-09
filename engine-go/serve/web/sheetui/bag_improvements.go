package sheetui

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// MELHORIA E MATERIAL: a regra de quem cabe em quem.
//
// Ela é do SERVIDOR e não da tela, e a razão é a de sempre: pré-validação de
// tela não é fronteira — um pedido montado à mão põe corda de arco num escudo, e
// o servidor grava.

// categoriesWithoutOverlap são as que nunca recebem melhoria nem material.
//
// Não se forja uma poção em aço-rubi, nem se põe mira telescópica num cavalo.
// Recusar aqui é o que faz o botão SUMIR na tela em vez de abrir um diálogo com
// duas listas vazias.
var categoriesWithoutOverlap = map[string]bool{
	"consumable": true, "meal": true, "catalyst": true,
	"improvement": true, "material": true, "animal": true, "vehicle": true,
}

func aceitaMelhoria(catalog book.Item) bool {
	return !categoriesWithoutOverlap[catalog.Category]
}

// aceitaAFamilia diz se a sobreposição serve à família do item.
//
// Sobreposição SEM `appliesTo` serve a qualquer um: o catálogo usa o campo para
// restringir, e a ausência dele é "não restringe" — não "não serve a ninguém",
// que faria uma melhoria nova nascer inalcançável.
func aceitaAFamilia(overlap book.Item, family string) bool {
	if len(overlap.AppliesTo) == 0 {
		return true
	}
	return contemTraco(overlap.AppliesTo, family)
}

// fitsItemImprovement é a RECUSA do servidor, e ela é a fronteira.
//
// Ela confere três coisas de cada id: que ele existe no catálogo, que ele é da
// categoria certa (melhoria não entra no campo do material e vice-versa), e que
// ele serve ao item (`aceitaAFamilia`, `book.ImprovementFitsItem`).
func fitsItemImprovement(catalog *book.Item, ids []string, category string) error {
	if catalog == nil {
		return fmt.Errorf("um item custom não recebe melhoria: ele não tem família no catálogo")
	}
	if len(ids) > 0 && !aceitaMelhoria(*catalog) {
		return fmt.Errorf("%q não recebe melhoria nem material", catalog.Name)
	}
	family := book.ItemFamily(*catalog)
	for _, id := range ids {
		overlap := book.ItemByID(id)
		if overlap == nil || overlap.Category != category {
			return fmt.Errorf("%q não é uma %s do livro", id, categoryName(category))
		}
		if !aceitaAFamilia(*overlap, family) || !book.ImprovementFitsItem(id, *catalog) {
			return fmt.Errorf("%q não cabe em %q", overlap.Name, catalog.Name)
		}
	}
	return nil
}

// repeatedImprovement é a melhoria que o pedido traz duas vezes. O diálogo nunca a manda assim, e a
// do livro repetida não somaria; a da mesa somaria, então a recusa vale para as duas.
func repeatedImprovement(ids []string) string {
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			return id
		}
		seen[id] = true
	}
	return ""
}

func splitImprovementIDs(ids []string) (fromBook, fromTable []string) {
	for _, id := range ids {
		if strings.HasPrefix(id, engine.TableImprovementPrefix) {
			fromTable = append(fromTable, id)
		} else {
			fromBook = append(fromBook, id)
		}
	}
	return fromBook, fromTable
}

// tableImprovementsFit é a recusa da melhoria da mesa: ela tem de ser desta mesa, de quem pede, e
// caber no item. A que JÁ ESTÁ no item passa: o gesto regrava a lista inteira, e o mestre mudar
// quem pode usar não pode travar as outras melhorias daquele item.
func tableImprovementsFit(s Scene, r *http.Request, row sqlcgen.Character, item sqlcgen.GetItemRow, catalog *book.Item, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if catalog == nil {
		return fmt.Errorf("um item custom não recebe melhoria: ele não tem família no catálogo")
	}
	if !aceitaMelhoria(*catalog) {
		return fmt.Errorf("%q não recebe melhoria nem material", catalog.Name)
	}
	world, err := sheet.RulesetFor(r.Context(), s.deps.Queries(), s.deps.Catalogs(), row.CampaignId)
	if err != nil {
		return err
	}
	applied := savedImprovements(item.Improvements)
	for _, id := range ids {
		if contemTraco(applied, id) {
			continue
		}
		at := slices.IndexFunc(world.TableImprovements(), func(i engine.TableImprovement) bool { return i.ID == id })
		if at < 0 {
			return fmt.Errorf("uma das melhorias da mesa não existe mais, ou é de outra mesa: feche a lista e abra de novo")
		}
		if improvement := world.TableImprovements()[at]; !tableImprovementOffered(improvement, row.ID, *catalog) {
			return fmt.Errorf("%q não cabe em %q, ou não é para este personagem", improvement.Name, catalog.Name)
		}
	}
	return nil
}

// offerTableImprovements põe as melhorias da mesa na folha do item, antes das do livro. A que já está
// no item segue na lista mesmo sem caber mais: o mestre pode tê-la dado a outra pessoa, e ela precisa
// poder sair.
func offerTableImprovements(entry *itemSheet, dto sheet.CharacterDTO, item sheet.ItemDTO) {
	catalog := catalogItem(item)
	if dto.Ruleset == nil || catalog == nil || !entry.Overlayable {
		return
	}
	applied := savedImprovements(item.Improvements)
	choices, rows := []overlayChoice{}, []overlayRow{}
	for _, improvement := range dto.Ruleset.TableImprovements() {
		active := contemTraco(applied, improvement.ID)
		if !active && !tableImprovementOffered(improvement, dto.ID, *catalog) {
			continue
		}
		effect := book.WrittenSum(improvement.Effect)
		choices = append(choices, overlayChoice{ID: improvement.ID, Name: improvement.Name, Effect: effect, Active: active, FromTable: true})
		if active {
			rows = append(rows, overlayRow{Name: improvement.Name, Effect: effect})
		}
	}
	entry.Improvements = append(choices, entry.Improvements...)
	entry.Overlays = append(rows, entry.Overlays...)
}

// tableImprovementNames são as melhorias da mesa que estão no item, para o crachá dele. `itemOverlays`
// só conhece o livro.
func tableImprovementNames(world *engine.Ruleset, item sheet.ItemDTO) []string {
	names := []string{}
	if world == nil || catalogItem(item) == nil {
		return names
	}
	applied := savedImprovements(item.Improvements)
	for _, improvement := range world.TableImprovements() {
		if contemTraco(applied, improvement.ID) {
			names = append(names, improvement.Name)
		}
	}
	return names
}

func tableImprovementOffered(improvement engine.TableImprovement, characterID int64, catalog book.Item) bool {
	return improvement.Applies.Matches(engine.Character{ID: int(characterID)}) && book.TableImprovementFits(improvement, catalog)
}

func categoryName(category string) string {
	if category == "material" {
		return "material"
	}
	return "melhoria"
}
