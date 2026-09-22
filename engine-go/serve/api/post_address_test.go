package api

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// O SEGMENTO que uma expressão preenche em tempo de execução.
//
// Ele é uma palavra qualquer de propósito: o que o guarda pergunta é se o
// ROTEADOR conhece a FORMA do caminho, e um parâmetro de chi casa com qualquer
// pedaço não vazio. Pôr um id de verdade aqui responderia sobre o dado.
const runtimeSegment = "spliced"

// datastarAddress é um endereço que a cena mandou para o navegador.
type datastarAddress struct {
	Method string
	Path   string
	Origin string
}

// firstArgument devolve o PRIMEIRO argumento de uma chamada, lendo `s` a partir
// do caractere seguinte ao parêntese de abertura.
//
// Só o primeiro: o segundo é o `{payload: …}`, e ele carrega strings que não são
// endereço nenhum. Ler a chamada inteira acharia `'M 4.5 2.5 L 9 2.5'` e
// perguntaria ao roteador por ela.
func firstArgument(s string) (string, bool) {
	depth := 0
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if c == '\\' {
				i++
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '\'', '"', '`':
			quote = c
		case '(', '{', '[':
			depth++
		case ')':
			if depth == 0 {
				return s[:i], true
			}
			depth--
		case '}', ']':
			depth--
		case ',':
			if depth == 0 {
				return s[:i], true
			}
		}
	}
	return "", false
}

// pathsInExpression resolve a expressão de endereço nos caminhos que ela pode
// produzir.
//
// A regra é o VÃO entre dois literais: vão com `+` é CONCATENAÇÃO e continua o
// mesmo endereço (`'/campanhas/1/sessoes/1/iniciativa/' + $edit_row + '/editar'` é um só);
// vão sem `+` é DESVIO e começa outro (o `evt.shiftKey ? '…/5' : '…/1'` do
// ferir são dois endereços de verdade, e os dois têm de existir).
func pathsInExpression(expr string) []string {
	var paths []string
	current, end, first := "", 0, true
	for i := 0; i < len(expr); i++ {
		c := expr[i]
		if c != '\'' && c != '"' && c != '`' {
			continue
		}
		j := i + 1
		for j < len(expr) && expr[j] != c {
			if expr[j] == '\\' {
				j++
			}
			j++
		}
		if j >= len(expr) {
			break
		}
		literal, span := expr[i+1:j], expr[end:i]
		if first || !strings.Contains(span, "+") {
			if current != "" {
				paths = append(paths, current)
			}
			current = literal
		} else {
			if strings.TrimSpace(strings.ReplaceAll(span, "+", "")) != "" {
				current += runtimeSegment
			}
			current += literal
		}
		first, end, i = false, j+1, j
	}
	if current != "" {
		paths = append(paths, current)
	}

	var addresses []string
	for _, path := range paths {
		if !strings.HasPrefix(path, "/") {
			continue
		}
		// A busca fica de fora: quem casa rota no chi é o CAMINHO, e um
		// `?abrir=1` colado nele nunca casaria.
		if cut := strings.IndexByte(path, '?'); cut >= 0 {
			path = path[:cut]
		}
		addresses = append(addresses, path)
	}
	return addresses
}

// unescapeHTML devolve o texto que o navegador vai LER.
//
// O `templ` escapa o valor do atributo, então a aspa simples de
// `@post('/campanhas/…')` chega ao HTML como `&#39;` — e um extrator que procurasse
// a aspa no texto servido acharia zero endereços, com cara de "não há nenhum".
var unescapeHTML = strings.NewReplacer(
	"&#39;", "'", "&#34;", `"`, "&quot;", `"`, "&apos;", "'",
	"&lt;", "<", "&gt;", ">", "&amp;", "&",
)

// addressesInHTML acha toda chamada do Datastar que pede um endereço.
//
// `unreadable` é a outra metade do resultado e não é descarte: uma expressão que
// o extrator não consegue resolver NÃO é uma expressão limpa, e um parser que
// descarta em silêncio o que não sabe ler produz lista de falhas com cara de
// descoberta.
func addressesInHTML(origin, html string) (findings []datastarAddress, unreadable []string) {
	text := unescapeHTML.Replace(html)
	// `window.open(` É UM ENDEREÇO QUE A CENA ESCREVE, e ele entrou aqui em vez
	// de ganhar guarda próprio porque o defeito é o mesmo: um caminho morto num
	// `@post` não faz nada e não avisa; num `window.open` ele abre uma janela com
	// um 404 dentro, igualmente mudo no console de quem clicou. O `firstArgument`
	// já para na primeira vírgula de topo, então o NOME da janela e as `features`
	// ficam de fora sozinhos.
	//
	// O método é GET porque é navegação: o que se pergunta ao chi é se existe
	// uma página naquele caminho.
	for _, verb := range []string{"post", "get", "put", "delete", "patch", "window.open"} {
		needle := "@" + verb + "("
		if verb == "window.open" {
			needle = verb + "("
		}
		for pos := 0; ; {
			cut := strings.Index(text[pos:], needle)
			if cut < 0 {
				break
			}
			start := pos + cut + len(needle)
			pos = start
			argument, closed := firstArgument(text[start:])
			if !closed {
				unreadable = append(unreadable, origin+": "+needle+" não fecha")
				continue
			}
			paths := pathsInExpression(argument)
			if len(paths) == 0 {
				unreadable = append(unreadable, origin+": "+needle+clipForMessage(argument)+")")
				continue
			}
			method := strings.ToUpper(verb)
			if verb == "window.open" {
				method = "GET"
			}
			for _, path := range paths {
				findings = append(findings, datastarAddress{
					Method: method, Path: path, Origin: origin,
				})
			}
		}
	}
	return findings, unreadable
}

func clipForMessage(s string) string {
	if len(s) > 90 {
		return s[:90] + "…"
	}
	return s
}

// routerKnows pergunta ao chi, e não ao handler.
//
// A diferença é o que separa "esta rota não existe" de "este id não existe": um
// pedido de verdade a `/personagens/spliced` devolveria 404 do HANDLER, e o
// guarda leria isso como rota faltando. `Match` responde só sobre a TABELA.
func routerKnows(mux *chi.Mux, method, path string) bool {
	return mux.Match(chi.NewRouteContext(), method, path)
}

// O EXTRATOR é a peça que pode ficar cega em silêncio, e por isso ele tem caso
// próprio com o esperado escrito à mão.
//
// As quatro formas são as que as cenas escrevem hoje, e cada uma quebra o
// extrator de um jeito diferente: a concatenação com sinal viraria dois
// endereços partidos, o ternário do ferir viraria um endereço colado que não
// existe, e a busca colada no caminho nunca casaria no chi. O dia em que uma
// delas parar de ser resolvida, o guarda grande não fica vermelho — ele fica
// com um denominador menor, que é a cor do verde.
func TestTheAddressExtractorReadsEveryShapeTheScenesWrite(t *testing.T) {
	cases := []struct {
		name, expression string
		want             []string
	}{
		{
			"o literal puro",
			`'/campanhas/1/sessoes/1/iniciativa/proxima-vez'`,
			[]string{"/campanhas/1/sessoes/1/iniciativa/proxima-vez"},
		},
		{
			"a concatenação com sinal é UM endereço",
			`'/campanhas/1/sessoes/1/iniciativa/' + $edit_row + '/editar'`,
			[]string{"/campanhas/1/sessoes/1/iniciativa/spliced/editar"},
		},
		{
			"o ternário do ferir são DOIS, e os dois têm de existir",
			`evt.shiftKey ? '/campanhas/1/sessoes/1/elenco/1/vitais/hp/ferir/5' : '/campanhas/1/sessoes/1/elenco/1/vitais/hp/ferir/1'`,
			[]string{"/campanhas/1/sessoes/1/elenco/1/vitais/hp/ferir/5", "/campanhas/1/sessoes/1/elenco/1/vitais/hp/ferir/1"},
		},
		{
			"a busca sai do caminho",
			`'/campanhas/1/sessoes/1/bestiario?abrir=1'`,
			[]string{"/campanhas/1/sessoes/1/bestiario"},
		},
	}
	// O `window.open` é a QUINTA forma, e ela não passa pelo `pathsInExpression`
	// sozinha: o que pode quebrar nela é a AGULHA (ela não tem `@`) e o MÉTODO
	// (navegação é GET, e perguntar ao chi por um POST em `/campanhas/1/sessoes/4/notas`
	// responderia "existe" pela rota de salvar — o guarda ficaria verde sobre um
	// endereço de página que não existe). Por isso o caso mede o extrator
	// inteiro, e afirma o método.
	const html = `<button data-on:click="const janela = window.open('/campanhas/1/sessoes/4/notas', 't20-notas', 'popup,width=620'); if (janela) { janela.focus() }">`
	findings, unreadable := addressesInHTML("caso", html)
	if len(unreadable) > 0 {
		t.Errorf("o extrator não leu o `window.open`: %q", unreadable)
	}
	if len(findings) != 1 || findings[0].Method != "GET" || findings[0].Path != "/campanhas/1/sessoes/4/notas" {
		t.Errorf("o `window.open` saiu como %+v, esperava um GET em /campanhas/1/sessoes/4/notas", findings)
	}

	for _, tc := range cases {
		read := pathsInExpression(tc.expression)
		if len(read) != len(tc.want) {
			t.Errorf("%s: %d endereços, esperava %d — %q", tc.name, len(read), len(tc.want), read)
			continue
		}
		for i, address := range read {
			if address != tc.want[i] {
				t.Errorf("%s: leu %q, esperava %q", tc.name, address, tc.want[i])
			}
		}
	}
}

// E o SEGUNDO argumento do `@post` fica de fora — ele é o `{payload: …}`, e as
// strings dele não são endereço de coisa nenhuma.
//
// O caso é o gabarito, que manda um traço de SVG no corpo: um extrator que
// lesse a chamada inteira acharia `'M 4.5 2.5 L 9 2.5'` e perguntaria ao
// roteador por ela.
func TestTheExtractorStopsAtThePayload(t *testing.T) {
	call := `'/campanhas/1/sessoes/1/tabuleiro/gabarito', {payload: {shape: $template, path: 'M 4.5 2.5 L 9 2.5'}}`
	argument, closed := firstArgument(call + ")")
	if !closed {
		t.Fatalf("o primeiro argumento não fechou em %q", call)
	}
	if read := pathsInExpression(argument); len(read) != 1 || read[0] != "/campanhas/1/sessoes/1/tabuleiro/gabarito" {
		t.Errorf("leu %q, e o payload não é endereço", read)
	}
}

// scenesThatWriteAddresses são as telas que este guarda VISITA.
//
// A lista é enumeração, e ela é o limite conhecido deste guarda: cena que não
// está aqui não é medida, e a ausência dela tem a mesma cor do verde. Não há
// amostragem possível — o endereço só existe RESOLVIDO, depois que o `templ`
// juntou a base, o id e o verbo, e um parser estático não sabe ler `base :=`.
//
// Duas exclusões DELIBERADAS: a porta (`/entrar`, `/criar-conta`,
// `/redefinir-senha`) não escreve endereço nenhum — ela é `<form method="post">`
// puro, com `SemEstadoDeCliente: true` —, e `/admin` mora noutro servidor,
// porque só ele leva `ADMIN_EMAILS`.
func scenesThatWriteAddresses(t *testing.T, f sceneFixture) []struct {
	Name, Path string
	User       int64
} {
	t.Helper()
	table := f.tableUrl()
	campaign := "/campanhas/" + strconv.FormatInt(f.campaignID, 10)
	sheet := "/personagens/" + strconv.FormatInt(f.charID, 10)

	scenes := []struct {
		Name, Path string
		User       int64
	}{
		{"hub", "/", f.gm},
		{"elenco", "/personagens", f.gm},
		{"forja", "/personagens/nova", f.gm},
		{"campanhas", "/campanhas", f.gm},
		{"campanha nova", "/campanhas/nova", f.gm},
		{"grimorio", "/grimorio", f.gm},
		{"buscador", "/buscador?q=fogo", f.gm},
		{"bestiario", "/mestre/bestiario", f.gm},
		{"catalogo", "/mestre/condicoes", f.gm},
		{"encontros", "/mestre/encontros", f.gm},
		{"improviso", "/mestre/improviso", f.gm},
		{"verbete", "/verbete?aba=condicoes&entrada=cego", f.gm},
		// A MESA nas DUAS formas. Não é a mesma tela duas vezes: o mestre tem
		// rodapé de comandos, trilho da fila e tabuleiro; o jogador não desenha
		// controle de mestre NENHUM (`View.Mestre` é `nil`).
		{"mesa do mestre", table, f.gm},
		{"mesa do jogador", table, f.player},
		{"tabuleiro do jogador", table + "?superficie=tabuleiro", f.player},
		// AS NOTAS NUMA JANELA: cena própria, endereço próprio, e o `@post` de
		// salvar sai dela também. Cena nova entra nesta lista no MESMO commit que
		// a cria, ou ela nasce sem medição.
		{"notas em janela", table + "/notas", f.gm},
	}
	// AS SETE ABAS da ficha, e o `?embutida=1` que a Mesa encaixa.
	for _, aba := range []string{
		"expertises", "combat", "bag", "proficiencies", "conditionals", "abilities", "spells",
	} {
		scenes = append(scenes, struct {
			Name, Path string
			User       int64
		}{"ficha " + aba, sheet + "?tab=" + aba, f.player})
	}
	scenes = append(scenes, struct {
		Name, Path string
		User       int64
	}{"ficha embutida", sheet + "?tab=combat&embutida=1", f.player})
	// AS CINCO ABAS da crônica, das quais duas só o mestre vê.
	for _, aba := range []string{"visao", "sessoes", "membros", "lugares", "config"} {
		scenes = append(scenes, struct {
			Name, Path string
			User       int64
		}{"cronica " + aba, campaign + "?tab=" + aba, f.gm})
	}
	scenes = append(scenes, struct {
		Name, Path string
		User       int64
	}{"atributos", sheet + "/atributos", f.player})
	return scenes
}

// A MESA VIVA é pré-requisito, senão o guarda passa VERDE sobre o defeito que
// veio pegar.
//
// Os comandos do rodapé nascem `disabled` quando `PodeAvancar` é falso — e ele é
// `st.InScene() && len(st.Initiative) > 0`. Na bancada recém-montada a cena
// está fria e a fila vazia, então o botão sai SEM o `data-on:click`: o endereço
// morto simplesmente não está no HTML.
func openTheLiveTable(t *testing.T, f sceneFixture) {
	t.Helper()
	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/iniciativa/adicionar",
		`{"new_name":"Ogro","new_initiative":12,"new_hp":130,"new_type":"npc"}`); rec.Code != http.StatusOK {
		t.Fatalf("pôr o Ogro na fila deu %d — sem fila o rodapé de comandos nasce todo disabled", rec.Code)
	}
	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/cena/iniciar/acao", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar a cena deu %d", rec.Code)
	}
	if rec := f.requests(t, f.gm, "POST", f.tableUrl()+"/tabuleiro/abrir", ""); rec.Code != http.StatusOK {
		t.Logf("abrir o tabuleiro deu %d", rec.Code)
	}
}

// TODO ENDEREÇO QUE UM `@post` ESCREVE EXISTE NO ROTEADOR.
//
// O defeito é o endereço montado em DUAS METADES — um helper que recebe o
// sufixo, com o sufixo chegando de outra linha —, que nenhum `grep` por um
// caminho inteiro encontra. E o Datastar DESCARTA o remendo de toda resposta
// não-2xx: o clique morre sem console, sem frase e sem nada na tela.
//
// # Por que ele RENDERIZA em vez de ler o código
//
// Porque o endereço não existe no código. Ele é montado em tempo de execução
// por `fmt.Sprintf`, por `base :=`, por helper que recebe o sufixo e por
// concatenação com sinal do Datastar — e um parser estático que tentasse
// remontar isso teria de resolver o pacote inteiro. O que o navegador recebe é
// a única forma em que o endereço EXISTE, e é ela que este guarda lê.
//
// O preço é que ele só mede o que VISITA, e a lista está em
// `scenesThatWriteAddresses` com o limite escrito.
//
// # O que ele pergunta
//
// Ao chi, com `Match`, e não ao handler: a pergunta é sobre a TABELA de rotas.
// Um pedido de verdade a `/personagens/spliced` levaria 404 do handler
// ("personagem não existe"), e o guarda leria isso como rota faltando.
func TestEveryAddressAPostWritesExistsInTheRouter(t *testing.T) {
	f := newSceneFixture(t)
	openTheLiveTable(t, f)

	mux, ok := f.s.WebRouter().(*chi.Mux)
	if !ok {
		t.Fatalf("o WebRouter deixou de ser um *chi.Mux (%T) — sem ele não há a quem perguntar", f.s.WebRouter())
	}

	var missing, unreadable []string
	measured, scenesRead := map[string]bool{}, 0
	for _, scene := range scenesThatWriteAddresses(t, f) {
		rec := f.requests(t, scene.User, "GET", scene.Path, "")
		if rec.Code != http.StatusOK {
			t.Errorf("a cena %q (%s) respondeu %d: ela saiu da lista sem ninguém tirar, e uma cena que não abre não mede nada",
				scene.Name, scene.Path, rec.Code)
			continue
		}
		scenesRead++
		findings, unread := addressesInHTML(scene.Name, rec.Body.String())
		unreadable = append(unreadable, unread...)
		for _, address := range findings {
			key := address.Method + " " + address.Path
			if measured[key] {
				continue
			}
			measured[key] = true
			if !routerKnows(mux, address.Method, address.Path) {
				missing = append(missing,
					address.Method+" "+address.Path+"  ← escrito pela cena "+address.Origin)
			}
		}
	}

	// O DENOMINADOR. Uma lista de faltantes vazia e um extrator que parou de
	// casar são a mesma cor no terminal, e por isso o guarda afirma quantos
	// endereços olhou antes de afirmar que nenhum falta.
	if scenesRead < 25 || len(measured) < 150 {
		t.Fatalf("a varredura leu %d cenas e %d endereços distintos — o extrator é o primeiro suspeito",
			scenesRead, len(measured))
	}

	sort.Strings(unreadable)
	if len(unreadable) > 0 {
		t.Errorf("%d chamadas do Datastar que o extrator NÃO soube resolver:\n  %s\n"+
			"Ramo que ignora o que não entende produz lista de falhas com cara de descoberta "+
			"(ALE-294). Ou a forma nova entra no extrator, ou ela não é forma.",
			len(unreadable), strings.Join(unreadable, "\n  "))
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("%d de %d endereços não existem no roteador:\n  %s\n"+
			"O Datastar descarta o remendo de toda resposta não-2xx, então um endereço morto "+
			"não deixa erro na tela, no console nem em lugar nenhum: o gesto simplesmente não acontece.",
			len(missing), len(measured), strings.Join(missing, "\n  "))
	}
	// A ÚLTIMA LINHA NÃO PODE DESMENTIR O VEREDITO: um `t.Logf` de sucesso escrito
	// depois do `t.Errorf` é o que se lê logo antes do FAIL, e ele diz o
	// contrário.
	if !t.Failed() {
		t.Logf("endereços: %d distintos em %d cenas, todos no roteador", len(measured), scenesRead)
	}
}
