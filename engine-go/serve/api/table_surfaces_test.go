package api

import (
	"html"
	"net/http"
	"strings"
	"t20engine/serve/web/table"
	"testing"
)

func TestThePlayerHasEveryRegionExactlyOnce(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)

	html := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()

	// AS REGIÕES SÃO AS DA PRODUÇÃO, e não uma lista escrita aqui: a lista e a
	// página não podem discordar sobre quais existem.
	for _, region := range table.TableRegions(table.View{}) {
		id := region.ID
		mark := `id="` + id + `"`
		if n := strings.Count(html, mark); n != 1 {
			t.Errorf("a região %q aparece %d vezes na cena do jogador, e o remendo precisa de exatamente 1", id, n)
		}
	}
}

// Dentro da sessão as abas da ficha são COMANDO e não link. Um `<a href>` ali
// tiraria o jogador da mesa no meio do combate — e o modo de errar é silencioso,
// porque o link funciona: ele leva para uma tela legítima, só que a errada.
func TestTheSheetInTheSessionDoesNotNavigateOutOfIt(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)

	html := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()

	insideSheet := html[strings.Index(html, `id="sheet-scene"`):]
	if i := strings.Index(insideSheet, `aria-label="Seções da ficha"`); i >= 0 {
		nav, _, _ := strings.Cut(insideSheet[i:], "</nav>")
		// O LINK FICA para quem abre em outra aba, mas o toque é IMPEDIDO e vira
		// o pedido da ficha embutida: cada `href` tem o seu `__prevent`.
		links := strings.Count(nav, `href="/personagens/`)
		if links == 0 || strings.Count(nav, "data-on:click__prevent") != links {
			t.Errorf("a fileira de abas da ficha embutida tem %d links e %d toques impedidos — "+
				"o link sem toque impedido tira o jogador da mesa", links, strings.Count(nav, "data-on:click__prevent"))
		}
		if !strings.Contains(nav, "embutida=1") {
			t.Error("a aba embutida não carrega a marca que a mantém dentro da sessão")
		}
	} else {
		t.Fatal("não achei a fileira de abas dentro da ficha embutida")
	}
}

// Ele tem as SUAS superfícies — Fila, Tabuleiro e Consultas —, e não o
// seletor do jogador.
func TestTheGmDoesNotGetTheSelector(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)

	html := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	// O CONTROLE: a cena do mestre chegou inteira. Sem ele, "não achei o seletor"
	// seria verdade também num 403 ou numa página vazia.
	if !strings.Contains(html, `aria-label="Superfícies do mestre"`) {
		t.Fatal("o mestre não recebeu a mesa dele — a página não é o que este teste pensa que é")
	}
	if strings.Contains(html, "O que ver na sessão") {
		t.Error("o mestre recebeu o seletor de superfícies do jogador")
	}
}

// O padrão é a MESA (decisão do dono), e ele é escrito num lugar só: a página
// semeia o sinal a partir da mesma constante que a lista de superfícies usa.
// Digitar 'mesa' no `data-signals` seria a segunda cópia, e a que fica para trás
// no dia em que o padrão mudar — a cena nasceria com um sinal e o botão marcando
// outro.
func TestTheOpeningSurfaceIsDerivedAndNotTyped(t *testing.T) {
	f := newSceneFixture(t)
	html := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()

	if !strings.Contains(html, `surface: &#39;`+table.DefaultOpeningSurface+`&#39;`) {
		t.Errorf("a página não semeia a superfície padrão (%q)", table.DefaultOpeningSurface)
	}
	// E a superfície padrão precisa EXISTIR na lista: um padrão que não é
	// oferecido abriria a sessão com as duas abas apagadas e a tela vazia.
	offered := false
	for _, s := range table.PlayerSurfaces {
		offered = offered || s.ID == table.DefaultOpeningSurface
	}
	if !offered {
		t.Errorf("a superfície padrão %q não está entre as oferecidas", table.DefaultOpeningSurface)
	}
}

// A ficha embutida não é região do stream, então o que a mantém em dia é um par:
// o servidor escreve `sheet_version` num sinal, e um ouvinte na cena repede a
// ficha. As duas pontas estão em arquivos diferentes e nenhuma delas falha
// sozinha de forma visível — sem o ouvinte, o sinal chega e ninguém escuta;
// sem o filtro, o ouvinte dispara em qualquer remendo de sinal e a ficha é
// repedida quando o mestre puxa alguém para o mapa.
//
// O e2e prova o comportamento com dois clientes; este guarda é a rede barata que
// falha no commit em que uma das pontas some.
func TestTheSheetInTheSessionHasAWayToKnowItChanged(t *testing.T) {
	f := newSceneFixture(t)
	f.scene(t)

	// DESESCAPADO: `data-signals` e `data-on-*` são valores DINÂMICOS de
	// atributo, e o templ escapa a aspa simples deles (`&#39;`). Procurar a
	// forma crua aqui daria um guarda que reprova o código certo.
	scene := html.UnescapeString(f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String())

	if !strings.Contains(scene, "sheet_version: ''") {
		t.Error("o sinal `sheet_version` não foi declarado: o remendo do servidor não teria onde pousar")
	}
	if !strings.Contains(scene, `data-on-signal-patch=`) {
		t.Fatal("a cena não tem o ouvinte que repede a ficha")
	}
	if !strings.Contains(scene, `data-on-signal-patch-filter="{include: /^sheet_version$/}"`) {
		t.Error("o ouvinte está sem filtro: ele dispararia em QUALQUER remendo de sinal")
	}
	// A ABA viaja num sinal porque o servidor não a conhece daqui. Sem esta
	// escrita, o repedido devolveria a ficha na aba padrão.
	if !strings.Contains(scene, "$sheet_tab = ") {
		t.Error("as abas da ficha embutida não guardam a seção aberta")
	}
	if !strings.Contains(scene, "' + $sheet_tab + '") {
		t.Error("o repedido não lê a seção do sinal: ele devolveria a aba padrão")
	}
}
