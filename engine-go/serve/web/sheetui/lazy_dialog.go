package sheetui

import "github.com/a-h/templ"

// opensLazyDialog é o clique que abre um diálogo pesado: marca qual está aberto e pede a cena, que volta
// com o corpo dele (`lazyBody`). O `command="show-modal"` do botão abre o diálogo no mesmo toque.
func opensLazyDialog(v View, id string) string {
	return "$sheet_dialog = '" + id + "'; " + sheetGet(v)
}

// lazyDialogCloses limpa o `$sheet_dialog` quando o diálogo fecha, por qualquer caminho: o X, o Esc, o
// toque no fundo e o `command="close"` disparam o `close` do `<dialog>`.
var lazyDialogCloses = templ.Attributes{"data-on:close": "$sheet_dialog = ''"}
