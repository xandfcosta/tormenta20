package sheetui

import (
	"slices"
	"strconv"
	"strings"

	"t20engine/app/character"
	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// AS DECLARAÇÕES NA FICHA: o Estilo de Duas Armas, a Ambidestria e o Disparo Rápido se usam antes de
// agredir (ver `book.DeclaredFlag`). A linha do poder ganha o Usar nas Ações e nos Poderes, com o aviso
// de arma; o uso em curso aparece em vigor na aba Efeitos, com o encerrar.

// declarationOf completa a linha de um poder que declara: o aviso de arma e, quando a ficha não tem o
// que declarar com ele, a volta a passivo. Devolve false nesse caso, e a linha não ganha preço.
func declarationOf(row *powerRow, dto sheet.CharacterDTO, flag string) bool {
	if !character.OffersTheDeclaration(dto, flag) {
		// Com o Estilo de Duas Armas, a Ambidestria só tira o -2 (p125): um Usar ali seria um segundo
		// botão para a mesma declaração.
		row.Kind, row.Glyph = "passive", activationGlyph("passive")
		return false
	}
	row.Declares = flag
	row.Declared = slices.Contains(dto.Conditionals, engine.FlagGroupID(flag))
	if warning := declarationWarning(flag, dto); warning != "" {
		row.ItemNotes = append(row.ItemNotes, warning)
	}
	return true
}

// declarationWarning avisa o que o livro pede da arma, sem travar: quem decide é o mestre.
func declarationWarning(flag string, dto sheet.CharacterDTO) string {
	switch flag {
	case engine.TwoWeaponStylePowerID, engine.AmbidextrousSwitch:
		return twoWeaponStyleWarning(dto)
	case engine.RapidShotPowerID:
		return rapidShotWarning(dto)
	}
	return ""
}

// twoWeaponStyleWarning avisa que o Estilo de Duas Armas pede uma arma leve: "se
// estiver empunhando duas armas (e pelo menos uma delas for leve)" (p125). O Arma
// Secundária Grande dispensa a leve (p124).
//
// Item feito à mão não diz se é leve, e aí a tela cala em vez de acusar.
func twoWeaponStyleWarning(dto sheet.CharacterDTO) string {
	owned := sheet.OwnedPowers(dto)
	if slices.ContainsFunc(owned, func(p sheet.OwnedPower) bool { return p.ID == "arma-secundaria-grande" }) {
		return ""
	}
	for _, item := range dto.Items {
		if item.Equipped == nil || sheet.HandsFor(*item.Equipped) == 0 {
			continue
		}
		if item.CatalogID == nil {
			return ""
		}
		if fromBook := book.ItemByID(*item.CatalogID); fromBook == nil || fromBook.Weapon == nil {
			continue
		} else if fromBook.Weapon.Hand == "light" || slices.Contains(fromBook.Weapon.Traits, engine.DoubleTrait) {
			return ""
		}
	}
	return "Nenhuma arma empunhada é leve (p125)"
}

// rapidShotWarning avisa que o Disparo Rápido pede arma que recarregue como ação livre (p125), e só
// o arco faz isso: besta e funda recarregam com movimento, e pistola e mosquete, com a padrão
// (p146-151). Não trava pela mesma razão do aviso acima.
func rapidShotWarning(dto sheet.CharacterDTO) string {
	for _, item := range dto.Items {
		if item.Equipped == nil || sheet.HandsFor(*item.Equipped) == 0 {
			continue
		}
		if item.CatalogID == nil || strings.HasPrefix(*item.CatalogID, "arco-") {
			return ""
		}
	}
	return "Só o arco recarrega como ação livre (p125)"
}

// declaredRow é uma declaração em curso, para a aba Efeitos.
type declaredRow struct {
	Flag string
	Name string
	Page int
}

// declaredRowsOf são as declarações usadas e ainda não vencidas.
func declaredRowsOf(dto sheet.CharacterDTO) []declaredRow {
	rows := []declaredRow{}
	for _, key := range dto.Conditionals {
		flag, isFlag := strings.CutPrefix(key, engine.FlagGroupID(""))
		if !isFlag {
			continue
		}
		if spec, declared := book.DeclarationFor(flag); declared {
			rows = append(rows, declaredRow{Flag: flag, Name: spec.Name, Page: spec.BookPage})
		}
	}
	return rows
}

// declaredMeta é "Usado · até o seu próximo turno · p125".
func declaredMeta(d declaredRow) string {
	meta := "Usado · até o seu próximo turno"
	if d.Page > 0 {
		meta += " · p" + strconv.Itoa(d.Page)
	}
	return meta
}
