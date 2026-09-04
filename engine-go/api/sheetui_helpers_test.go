package api

import (
	"html"
	"regexp"
	"strings"
)

// Os ajudantes de leitura da cena da ficha, COPIADOS de `web/sheetui`
// (ALE-278).
//
// A cópia é deliberada e tem precedente nesta épica: importar o ajudante
// do pacote que está sendo testado faz o teste andar junto com o defeito
// — foi a regra que a fatia da porta deixou escrita, e o que a cena de
// personagens fez com o `corpoDoBotao`. Eles leem HTML servido e não
// dependem de uma linha da cena.

// sceneAlert acha a frase da recusa: a cena responde 200 com a página inteira
// redesenhada, e a única marca do "não deu" é o `role="alert"`. Ver a armadilha
// do Datastar que diz por que a recusa é CONTEÚDO e não status.
var sceneAlert = regexp.MustCompile(`role="alert"[^>]*>([^<]*)</p>`)

func actionsSlice(tela string) string {
	inicio := strings.Index(tela, ">Ações</h3>")
	if inicio < 0 {
		return ""
	}
	fim := strings.Index(tela[inicio:], "Passivas ·")
	if fim < 0 {
		return tela[inicio:]
	}
	return tela[inicio : inicio+fim]
}
func existe(m map[string]bool, chave string) bool {
	_, tem := m[chave]
	return tem
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

var panelTitle = map[string]string{
	"proficiencies": "Proficiências",
	"combat":        "Combate",
	"expertises":    "Perícias",
	"conditionals":  "Efeitos",
	"spells":        "Grimório",
	"bag":           "Mochila",
	"abilities":     "Poderes",
}

func powerPanel(tela string) string {
	// O CORTE é no ABRIR do primeiro diálogo, e não no primeiro `</section>`: as
	// duas seções da lista são `<section>` ANINHADAS, e cortar no primeiro
	// fechamento deixaria de fora justamente as passivas. Os diálogos começam
	// depois do painel, e todos são sobreposições de tela cheia.
	fim := strings.Index(tela, `class="fixed inset-0`)
	if fim < 0 {
		return tela
	}
	return tela[:fim]
}

type responseRecorderLike struct {
	Code int
	Body string
}

func sceneRefusal(corpo string) string {
	achado := sceneAlert.FindStringSubmatch(corpo)
	if achado == nil {
		return ""
	}
	return html.UnescapeString(achado[1])
}

func screenSaved(tela string) string {
	inicio := strings.Index(tela, "grid-cols-3")
	if inicio < 0 {
		return ""
	}
	fim := strings.Index(tela[inicio:], "</section>")
	if fim < 0 {
		return tela[inicio:]
	}
	return tela[inicio : inicio+fim]
}
