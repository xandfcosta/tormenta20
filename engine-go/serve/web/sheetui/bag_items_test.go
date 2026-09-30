package sheetui

import (
	"html"
	"strings"
)

// Os guardas dos DIÁLOGOS da Mochila (ALE-272, fatia 7).
//
// A ficha de item, o catálogo, o item custom, a dose do consumível e — a parte
// que era só de tela até aqui — a compatibilidade de melhoria e material.

// itemScreenSheet acha um item pelo nome, para o teste não guardar ids.
// A FICHA DO ITEM oferece os lugares ALCANÇÁVEIS, e só eles.
// itemScreenSheet recorta o diálogo de UM item pelo rótulo dele.
// itemScreenSheet recorta o diálogo de UM item pelo rótulo dele.
//
// O CORTE É NO DIÁLOGO SEGUINTE, e não num `</div></div>`. A mochila desenha um
// `role="dialog"` por item, então o vizinho é a fronteira confiável; o par de
// fechamentos casava com o PRIMEIRO bloco interno — o de equipar — e devolvia a
// cabeça do diálogo com cara de diálogo inteiro.
//
// Isso fazia toda asserção de AUSÊNCIA passar de graça: o que estivesse depois
// do bloco de equipar não estava no recorte, então "não contém" era verdade por
// truncagem. Medido ao acrescentar o bloco de encantos, que nasce no fim
// (ALE-416).
func itemScreenSheet(screen, name string) string {
	start := strings.Index(screen, `aria-label="`+name+`"`)
	if start < 0 {
		return ""
	}
	rest := screen[start:]
	if next := strings.Index(rest[1:], `role="dialog"`); next >= 0 {
		return rest[:next+1]
	}
	return rest
}

func improvementScreenDialog(screen, name string) string {
	start := strings.Index(screen, `aria-label="Melhorias de `+name+`"`)
	if start < 0 {
		return ""
	}
	end := strings.Index(screen[start:], "Aplicar")
	if end < 0 {
		return screen[start:]
	}
	return screen[start : start+end]
}
func existe(m map[string]bool, key string) bool {
	_, found := m[key]
	return found
}

func sceneRefusal(body string) string {
	found := sceneAlert.FindStringSubmatch(body)
	if found == nil {
		return ""
	}
	return html.UnescapeString(found[1])
}
