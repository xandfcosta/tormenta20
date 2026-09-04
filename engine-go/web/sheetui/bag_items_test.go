package sheetui

import (
	"html"
	"strings"
)

// Os guardas dos DIÁLOGOS da Mochila (ALE-272, fatia 7).
//
// A ficha de item, o catálogo, o item custom, a dose do consumível e — a parte
// que era só de tela até aqui — a compatibilidade de melhoria e material.

// oItemDaFichaPorNome acha um item pelo nome, para o teste não guardar ids.
// A FICHA DO ITEM oferece os lugares ALCANÇÁVEIS, e só eles.
// oFichaDoItemNaTela recorta o diálogo de UM item pelo rótulo dele.
func itemScreenSheet(tela, nome string) string {
	inicio := strings.Index(tela, `aria-label="`+nome+`"`)
	if inicio < 0 {
		return ""
	}
	fim := strings.Index(tela[inicio:], "</div></div>")
	if fim < 0 {
		return tela[inicio:]
	}
	return tela[inicio : inicio+fim]
}

func improvementScreenDialog(tela, nome string) string {
	inicio := strings.Index(tela, `aria-label="Melhorias de `+nome+`"`)
	if inicio < 0 {
		return ""
	}
	fim := strings.Index(tela[inicio:], "Aplicar")
	if fim < 0 {
		return tela[inicio:]
	}
	return tela[inicio : inicio+fim]
}
func existe(m map[string]bool, chave string) bool {
	_, tem := m[chave]
	return tem
}

func sceneRefusal(corpo string) string {
	achado := sceneAlert.FindStringSubmatch(corpo)
	if achado == nil {
		return ""
	}
	return html.UnescapeString(achado[1])
}
