package api

import (
	"html"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"t20engine/serve/web/sheetui"
)

// TODO SINAL NÃO TEXTUAL QUE A ABA AÇÕES LÊ É DECLARADO PELA MESA (ALE-499).
//
// A aba Ações é da ficha, mas quem a desenha é a página da Mesa, e é a Mesa que
// declara os sinais. Um sinal que a aba LÊ e ninguém declara nasce como texto
// vazio no Datastar — e o `Signals` da ficha, que espera número, falha a
// leitura INTEIRA: o servidor fica sem o `turn_target` também, e todo golpe da
// aba volta "escolha o alvo". Visto no navegador com o contador de degraus da
// Fúria (`stance_degrees`) e com a tira do impulso (`boost_*`).
//
// A varredura é sobre os campos do `Signals` que NÃO são texto: o texto vazio
// passa pela leitura, e são os outros que a derrubam. A ficha do caso tem as
// duas fontes conhecidas — Fúria com degraus e um impulso de guerreiro — e o
// controle afirma que ela as desenhou.
func TestEveryTypedSignalTheActionsSurfaceReadsIsDeclaredByTheTable(t *testing.T) {
	f, barbaro, _ := barbarianOnTurn(t)
	seedClasse(t, f.s, barbaro, "Guerreiro", 5)
	page := acoesDaMesa(t, f)

	at := strings.Index(page, `id="actions-scene"`)
	if at < 0 {
		t.Fatal("a página da Mesa não desenhou a aba Ações")
	}
	surface := html.UnescapeString(page[at:])
	for _, control := range []string{"$stance_degrees", "$boost_power"} {
		if !strings.Contains(surface, control) {
			t.Fatalf("a aba Ações não lê %s — o caso não tem o que medir", control)
		}
	}

	declared := map[string]bool{}
	for _, m := range regexp.MustCompile(`data-signals="([^"]*)"`).FindAllStringSubmatch(page, -1) {
		for _, key := range regexp.MustCompile(`([a-z_][a-z0-9_]*)\s*:`).FindAllStringSubmatch(html.UnescapeString(m[1]), -1) {
			declared[key[1]] = true
		}
	}

	typed := reflect.TypeOf(sheetui.Signals{})
	measured, missing := 0, []string{}
	for i := 0; i < typed.NumField(); i++ {
		field := typed.Field(i)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" || isText(field.Type) {
			continue
		}
		if !strings.Contains(surface, "$"+name) {
			continue
		}
		measured++
		if !declared[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if measured == 0 {
		t.Fatal("nenhum sinal tipado lido pela aba Ações — o caso mediu nada")
	}
	if len(missing) > 0 {
		t.Errorf("a aba Ações lê %v e a Mesa não os declara (de %d sinais tipados lidos).\n"+
			"Declare cada um no `tableSignalsExpr` com o valor do tipo certo: sem isso ele nasce "+
			"texto, a leitura do `Signals` falha inteira e o golpe chega sem alvo.", missing, measured)
	}
}

func isText(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	return t.Kind() == reflect.String
}
