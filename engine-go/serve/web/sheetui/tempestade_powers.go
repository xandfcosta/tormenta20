package sheetui

import (
	"strconv"
	"strings"

	"github.com/a-h/templ"
)

// A ABA PODERES DA FICHA PARALELA (ALE-473). O dado é o da ficha de hoje
// (`powersPanel`); daqui sai só o que a folha e a linha leem. A linha NÃO
// mostra o texto da regra — tocar no nome abre a folha com ele inteiro
// (decisão do dono). O "Escolher" abre o diálogo de `tempestade_choices.go`.

// powerSheetID é o `id` da folha de um poder. O ponto do id do catálogo sai,
// pela razão do `useDialogID`.
func powerSheetID(row powerRow) string {
	return "power-" + strings.ReplaceAll(row.ID, ".", "-")
}

// stanceDialogID e useStepsDialogID são os diálogos de degraus da ficha
// paralela.
func stanceDialogID(row powerRow) string {
	return "stance-" + row.Stance.Flag
}

func useStepsDialogID(row powerRow) string {
	return "use-steps-" + strings.ReplaceAll(row.ID, ".", "-")
}

// powerMeta é a nota da linha: de onde vem, a ação e o custo, e o limite —
// gasto, quando a ficha cobra.
//
//	powerMeta(row) // "Bárbaro · LIVRE · 2 PM · usado 1/1 cena"
func powerMeta(row powerRow) string {
	parts := []string{}
	for _, part := range []string{row.Source, row.Cost} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	switch {
	case row.Spent != "":
		parts = append(parts, row.Spent)
	case row.Limit != "":
		parts = append(parts, row.Limit)
	}
	return strings.Join(parts, " · ")
}

// powerSheetMeta é a linha de cima da folha: a nota da linha e a página.
func powerSheetMeta(row powerRow) string {
	meta := powerMeta(row)
	if row.Page > 0 {
		if meta != "" {
			meta += " · "
		}
		meta += "p" + strconv.Itoa(row.Page)
	}
	return meta
}

// powerSheetRows são os poderes que ganham folha nesta tela: os que a lista
// mostra, e só eles — o resultado da busca, ou as ações e as passivas.
func powerSheetRows(p powersPanel) []powerRow {
	if p.Search != "" {
		return p.Results
	}
	return append(append([]powerRow{}, p.Actions...), p.Passives...)
}

// powerGestureAttrs junta o rótulo acessível e a recusa ao gesto: sem poder
// usar agora, o botão nasce desabilitado e o `title` diz por quê.
func powerGestureAttrs(row powerRow, aria string, attrs templ.Attributes) templ.Attributes {
	attrs["type"] = "button"
	attrs["aria-label"] = aria
	if !row.Can {
		attrs["disabled"] = true
		if row.Why != "" {
			attrs["title"] = row.Why
		}
	}
	return attrs
}

// powerSheetFoot é o pé da folha: o mesmo gesto da linha, quando há um.
func powerSheetFoot(v View, row powerRow) templ.Component {
	if row.Stance == nil && row.Kind != "instant" {
		return nil
	}
	return powerSheetFootBody(v, row)
}

// stepsConfirm é o botão que confirma os degraus e fecha o diálogo.
func stepsConfirm(dialogID, label, aria, post string) templ.Component {
	return stepsConfirmBody(dialogID, label, aria, post)
}
