package api

import (
	"html"
	"regexp"
	"strings"
	"testing"
)

// Os ajudantes de leitura da cena da ficha.
//
// Eles moram aqui e não em `web/sheetui` de propósito: importar o ajudante do
// pacote que está sendo testado faz o teste andar junto com o defeito. Eles leem
// HTML servido e não dependem de uma linha da cena.

// sceneAlert acha a frase da recusa: a cena responde 200 com a página inteira
// redesenhada, e a única marca do "não deu" é o `role="alert"`. Ver a armadilha
// do Datastar que diz por que a recusa é CONTEÚDO e não status.
var sceneAlert = regexp.MustCompile(`role="alert"[^>]*>([^<]*)</p>`)

func hasKey(m map[string]bool, key string) bool {
	_, found := m[key]
	return found
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

var panelTitle = map[string]string{
	"proficiencies": "Proficiências",
	"combat":        "Combate",
	"expertises":    "Perícias",
	"conditionals":  "Efeitos",
	"spells":        "Grimório",
	"bag":           "Mochila",
	"abilities":     "Poderes",
}

// powerPanel corta a LISTA, deixando de fora os diálogos que vêm depois
// dela — o de escolher poderes mostra o catálogo inteiro de opções.
func powerPanel(screen string) string {
	// O CORTE é no ABRIR do primeiro diálogo, e não no primeiro `</section>`: as
	// duas seções da lista são `<section>` ANINHADAS, e cortar no primeiro
	// fechamento deixaria de fora justamente as passivas. Os diálogos começam
	// depois do painel, e todos são sobreposições de tela cheia.
	end := strings.Index(screen, `class="fixed inset-0`)
	if end < 0 {
		return screen
	}
	return screen[:end]
}

type responseRecorderLike struct {
	Code int
	Body string
}

func sceneRefusal(body string) string {
	found := sceneAlert.FindStringSubmatch(body)
	if found == nil {
		return ""
	}
	return html.UnescapeString(found[1])
}

// hpTintOf devolve o degrau da escada com que a barra de PV foi pintada na
// tela, e falha alto quando não acha — sem isso, um seletor que não casa com
// nada e uma tinta ausente se parecem no terminal.
func hpTintOf(t *testing.T, screen string) string {
	t.Helper()
	if !strings.Contains(screen, `aria-label="PV `) {
		t.Fatal("a tela não desenha o PV: o guarda mediria o vazio")
	}
	// A BARRA DE PV da faixa e a da coluna levam o mesmo tom, e a primeira basta.
	found := regexp.MustCompile(`data-kind="pv" data-tone="(full|hurt|critical)"`).FindStringSubmatch(screen)
	if found == nil {
		t.Fatal("a barra de PV não foi pintada com nenhum degrau da escada")
	}
	return found[1]
}
