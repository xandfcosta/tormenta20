package api

import (
	"html"
	"regexp"
	"strings"
	"testing"
)

// Os ajudantes de leitura da cena da ficha, COPIADOS de `web/sheetui`.
//
// A cópia é deliberada: importar o ajudante do pacote que está sendo testado faz
// o teste andar junto com o defeito. Eles leem HTML servido e não dependem de
// uma linha da cena.

// sceneAlert acha a frase da recusa: a cena responde 200 com a página inteira
// redesenhada, e a única marca do "não deu" é o `role="alert"`. Ver a armadilha
// do Datastar que diz por que a recusa é CONTEÚDO e não status.
var sceneAlert = regexp.MustCompile(`role="alert"[^>]*>([^<]*)</p>`)

func actionsSlice(screen string) string {
	start := strings.Index(screen, ">Ações</h3>")
	if start < 0 {
		return ""
	}
	end := strings.Index(screen[start:], "Passivas ·")
	if end < 0 {
		return screen[start:]
	}
	return screen[start : start+end]
}
func existe(m map[string]bool, key string) bool {
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

var panelTitle = map[string]string{
	"proficiencies": "Proficiências",
	"combat":        "Combate",
	"expertises":    "Perícias",
	"conditionals":  "Efeitos",
	"spells":        "Grimório",
	"bag":           "Mochila",
	"abilities":     "Poderes",
}

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

func screenSaved(screen string) string {
	start := strings.Index(screen, "grid-cols-3")
	if start < 0 {
		return ""
	}
	end := strings.Index(screen[start:], "</section>")
	if end < 0 {
		return screen[start:]
	}
	return screen[start : start+end]
}

// hpTintOf devolve a tinta com que a FAIXA de PV foi pintada na tela, e falha
// alto quando não acha — sem isso, um seletor que não casa com nada e uma
// tinta ausente se parecem no terminal.
//
// Ele lê o trecho ANTES do `data-vital="PV"`, que é onde a barra mora, e aceita
// as duas grafias que o repositório usa para a mesma tinta: a classe da paleta
// (`bg-hp-full`) e o valor arbitrário (`bg-[color:var(--hp-full)]`). Aceitar as
// duas é o que impede o guarda de passar verde sobre um renome de grafia.
func hpTintOf(t *testing.T, screen string) string {
	t.Helper()
	cut := strings.Index(screen, `data-vital="PV"`)
	if cut < 0 {
		t.Fatal("a tela não desenha o PV: o guarda mediria o vazio")
	}
	findings := regexp.MustCompile(`hp-(full|hurt|critical)`).FindAllStringSubmatch(screen[:cut], -1)
	if len(findings) == 0 {
		t.Fatal("a faixa de PV não foi pintada com nenhuma tinta da escada")
	}
	// A ÚLTIMA é a da barra: as anteriores podem ser de outra fileira da cena.
	return findings[len(findings)-1][1]
}
