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
func itemScreenSheet(screen, name string) string {
	start := strings.Index(screen, `aria-label="`+name+`"`)
	if start < 0 {
		return ""
	}
	end := strings.Index(screen[start:], "</div></div>")
	if end < 0 {
		return screen[start:]
	}
	return screen[start : start+end]
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
