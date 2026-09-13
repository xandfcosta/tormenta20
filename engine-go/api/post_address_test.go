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
	Metodo  string
	Caminho string
	Origem  string
}

// firstArgument devolve o PRIMEIRO argumento de uma chamada, lendo `s` a partir
// do caractere seguinte ao parêntese de abertura.
//
// Só o primeiro: o segundo é o `{payload: …}`, e ele carrega strings que não são
// endereço nenhum. Ler a chamada inteira acharia `'M 4.5 2.5 L 9 2.5'` e
// perguntaria ao roteador por ela.
func firstArgument(s string) (string, bool) {
	profundidade := 0
	var aspa byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if aspa != 0 {
			if c == '\\' {
				i++
				continue
			}
			if c == aspa {
				aspa = 0
			}
			continue
		}
		switch c {
		case '\'', '"', '`':
			aspa = c
		case '(', '{', '[':
			profundidade++
		case ')':
			if profundidade == 0 {
				return s[:i], true
			}
			profundidade--
		case '}', ']':
			profundidade--
		case ',':
			if profundidade == 0 {
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
// mesmo endereço (`'/mesa/1/1/iniciativa/' + $edit_row + '/editar'` é um só);
// vão sem `+` é DESVIO e começa outro (o `evt.shiftKey ? '…/5' : '…/1'` do
// ferir são dois endereços de verdade, e os dois têm de existir).
func pathsInExpression(expr string) []string {
	var caminhos []string
	atual, fim, primeiro := "", 0, true
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
		literal, vao := expr[i+1:j], expr[fim:i]
		if primeiro || !strings.Contains(vao, "+") {
			if atual != "" {
				caminhos = append(caminhos, atual)
			}
			atual = literal
		} else {
			if strings.TrimSpace(strings.ReplaceAll(vao, "+", "")) != "" {
				atual += runtimeSegment
			}
			atual += literal
		}
		primeiro, fim, i = false, j+1, j
	}
	if atual != "" {
		caminhos = append(caminhos, atual)
	}

	var enderecos []string
	for _, caminho := range caminhos {
		if !strings.HasPrefix(caminho, "/") {
			continue
		}
		// A busca fica de fora: quem casa rota no chi é o CAMINHO, e um
		// `?abrir=1` colado nele nunca casaria.
		if corte := strings.IndexByte(caminho, '?'); corte >= 0 {
			caminho = caminho[:corte]
		}
		enderecos = append(enderecos, caminho)
	}
	return enderecos
}

// unescapeHTML devolve o texto que o navegador vai LER.
//
// O `templ` escapa o valor do atributo, então a aspa simples de
// `@post('/mesa/…')` chega ao HTML como `&#39;` — e um extrator que procurasse
// a aspa no texto servido acharia zero endereços, com cara de "não há nenhum".
var unescapeHTML = strings.NewReplacer(
	"&#39;", "'", "&#34;", `"`, "&quot;", `"`, "&apos;", "'",
	"&lt;", "<", "&gt;", ">", "&amp;", "&",
)

// addressesInHTML acha toda chamada do Datastar que pede um endereço.
//
// `ilegiveis` é a outra metade do resultado e não é descarte: uma expressão que
// o extrator não consegue resolver NÃO é uma expressão limpa, e a casa já
// pagou por um parser que descartava em silêncio o que não sabia ler (ALE-294).
func addressesInHTML(origem, html string) (achados []datastarAddress, ilegiveis []string) {
	texto := unescapeHTML.Replace(html)
	for _, verbo := range []string{"post", "get", "put", "delete", "patch"} {
		agulha := "@" + verbo + "("
		for pos := 0; ; {
			corte := strings.Index(texto[pos:], agulha)
			if corte < 0 {
				break
			}
			inicio := pos + corte + len(agulha)
			pos = inicio
			argumento, fechou := firstArgument(texto[inicio:])
			if !fechou {
				ilegiveis = append(ilegiveis, origem+": "+agulha+" não fecha")
				continue
			}
			caminhos := pathsInExpression(argumento)
			if len(caminhos) == 0 {
				ilegiveis = append(ilegiveis, origem+": "+agulha+clipForMessage(argumento)+")")
				continue
			}
			for _, caminho := range caminhos {
				achados = append(achados, datastarAddress{
					Metodo: strings.ToUpper(verbo), Caminho: caminho, Origem: origem,
				})
			}
		}
	}
	return achados, ilegiveis
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
func routerKnows(mux *chi.Mux, metodo, caminho string) bool {
	return mux.Match(chi.NewRouteContext(), metodo, caminho)
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
	casos := []struct {
		nome, expressao string
		esperado        []string
	}{
		{
			"o literal puro",
			`'/mesa/1/1/iniciativa/proxima-vez'`,
			[]string{"/mesa/1/1/iniciativa/proxima-vez"},
		},
		{
			"a concatenação com sinal é UM endereço",
			`'/mesa/1/1/iniciativa/' + $edit_row + '/editar'`,
			[]string{"/mesa/1/1/iniciativa/spliced/editar"},
		},
		{
			"o ternário do ferir são DOIS, e os dois têm de existir",
			`evt.shiftKey ? '/mesa/1/1/elenco/1/vitais/hp/ferir/5' : '/mesa/1/1/elenco/1/vitais/hp/ferir/1'`,
			[]string{"/mesa/1/1/elenco/1/vitais/hp/ferir/5", "/mesa/1/1/elenco/1/vitais/hp/ferir/1"},
		},
		{
			"a busca sai do caminho",
			`'/mesa/1/1/bestiario?abrir=1'`,
			[]string{"/mesa/1/1/bestiario"},
		},
	}
	for _, caso := range casos {
		lido := pathsInExpression(caso.expressao)
		if len(lido) != len(caso.esperado) {
			t.Errorf("%s: %d endereços, esperava %d — %q", caso.nome, len(lido), len(caso.esperado), lido)
			continue
		}
		for i, endereco := range lido {
			if endereco != caso.esperado[i] {
				t.Errorf("%s: leu %q, esperava %q", caso.nome, endereco, caso.esperado[i])
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
	chamada := `'/mesa/1/1/tabuleiro/gabarito', {payload: {shape: $template, path: 'M 4.5 2.5 L 9 2.5'}}`
	argumento, fechou := firstArgument(chamada + ")")
	if !fechou {
		t.Fatalf("o primeiro argumento não fechou em %q", chamada)
	}
	if lido := pathsInExpression(argumento); len(lido) != 1 || lido[0] != "/mesa/1/1/tabuleiro/gabarito" {
		t.Errorf("leu %q, e o payload não é endereço", lido)
	}
}

// scenesThatWriteAddresses são as telas que este guarda VISITA.
//
// A lista é enumeração, e ela é o limite conhecido deste guarda: cena que não
// está aqui não é medida, e a ausência dela tem a mesma cor do verde. Não há
// amostragem possível — o endereço só existe RESOLVIDO, depois que o `templ`
// juntou a base, o id e o verbo, e ler isso do código-fonte é o parser que a
// ALE-307 já mostrou não saber ler `base :=`.
//
// Duas exclusões DELIBERADAS: a porta (`/entrar`, `/criar-conta`,
// `/redefinir-senha`) não escreve endereço nenhum — ela é `<form method="post">`
// puro, com `SemEstadoDeCliente: true` —, e `/admin` mora noutro servidor,
// porque só ele leva `ADMIN_EMAILS`.
func scenesThatWriteAddresses(t *testing.T, f pilotoFixture) []struct {
	Nome, Caminho string
	Usuario       int64
} {
	t.Helper()
	mesa := f.tableUrl()
	campanha := "/campanhas/" + strconv.FormatInt(f.campaignID, 10)
	ficha := "/personagens/" + strconv.FormatInt(f.charID, 10)

	cenas := []struct {
		Nome, Caminho string
		Usuario       int64
	}{
		{"hub", "/", f.mestre},
		{"elenco", "/personagens", f.mestre},
		{"forja", "/personagens/nova", f.mestre},
		{"campanhas", "/campanhas", f.mestre},
		{"campanha nova", "/campanhas/nova", f.mestre},
		{"grimorio", "/grimorio", f.mestre},
		{"buscador", "/buscador?q=fogo", f.mestre},
		{"bestiario", "/mestre/bestiario", f.mestre},
		{"catalogo", "/mestre/condicoes", f.mestre},
		{"encontros", "/mestre/encontros", f.mestre},
		{"improviso", "/mestre/improviso", f.mestre},
		{"verbete", "/verbete?aba=condicoes&entrada=cego", f.mestre},
		// A MESA nas DUAS formas. Não é a mesma tela duas vezes: o mestre tem
		// rodapé de comandos, trilho da fila e tabuleiro; o jogador não desenha
		// controle de mestre NENHUM (`View.Mestre` é `nil`).
		{"mesa do mestre", mesa, f.mestre},
		{"mesa do jogador", mesa, f.jogador},
		{"tabuleiro do jogador", mesa + "?superficie=tabuleiro", f.jogador},
	}
	// AS SETE ABAS da ficha, e o `?embutida=1` que a Mesa encaixa.
	for _, aba := range []string{
		"expertises", "combat", "bag", "proficiencies", "conditionals", "abilities", "spells",
	} {
		cenas = append(cenas, struct {
			Nome, Caminho string
			Usuario       int64
		}{"ficha " + aba, ficha + "?tab=" + aba, f.jogador})
	}
	cenas = append(cenas, struct {
		Nome, Caminho string
		Usuario       int64
	}{"ficha embutida", ficha + "?tab=combat&embutida=1", f.jogador})
	// AS CINCO ABAS da crônica, das quais duas só o mestre vê.
	for _, aba := range []string{"visao", "sessoes", "membros", "lugares", "config"} {
		cenas = append(cenas, struct {
			Nome, Caminho string
			Usuario       int64
		}{"cronica " + aba, campanha + "?tab=" + aba, f.mestre})
	}
	cenas = append(cenas, struct {
		Nome, Caminho string
		Usuario       int64
	}{"atributos", ficha + "/atributos", f.jogador})
	return cenas
}

// A MESA VIVA é pré-requisito, e descobri isso pelo caminho mais barato que
// existe: a primeira versão deste guarda passou VERDE sobre o defeito que ela
// veio pegar.
//
// O `‹` do rodapé nasce `disabled` quando `PodeAvancar` é falso — e ele é
// `st.SceneActive && len(st.Initiative) > 0`. Na bancada recém-montada a cena
// está fria e a fila vazia, então o botão sai SEM o `data-on:click`: o endereço
// morto simplesmente não está no HTML. É a armadilha do ramo, medida no guarda
// que existe para medir ramos.
func openTheLiveTable(t *testing.T, f pilotoFixture) {
	t.Helper()
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/iniciativa/adicionar",
		`{"new_name":"Ogro","new_initiative":12,"new_hp":130,"new_type":"npc"}`); rec.Code != http.StatusOK {
		t.Fatalf("pôr o Ogro na fila deu %d — sem fila o rodapé de comandos nasce todo disabled", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/cena/iniciar", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar a cena deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/tabuleiro/abrir", ""); rec.Code != http.StatusOK {
		t.Logf("abrir o tabuleiro deu %d", rec.Code)
	}
}

// TODO ENDEREÇO QUE UM `@post` ESCREVE EXISTE NO ROTEADOR (ALE-308).
//
// # O defeito que o fez nascer
//
// A ALE-304 traduziu as rotas para português e deixou o `‹` do rodapé da mesa
// postando em `/mesa/{c}/{s}/initiative/previous-turn`, que não existe — a rota
// é `iniciativa/vez-anterior`. O endereço é montado em DUAS METADES
// (`tableCommand(v, "POST", "initiative/"+rota)`, com `rota` chegando de outra
// linha), e por isso nenhum `grep` por `/iniciativa/` nem por um caminho
// inteiro o encontrava.
//
// E o Datastar DESCARTA o remendo de qualquer resposta não-2xx: o clique morria
// sem console, sem frase e sem nada na tela.
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
	f := novoPiloto(t)
	openTheLiveTable(t, f)

	mux, ok := f.s.WebRouter().(*chi.Mux)
	if !ok {
		t.Fatalf("o WebRouter deixou de ser um *chi.Mux (%T) — sem ele não há a quem perguntar", f.s.WebRouter())
	}

	var faltando, ilegiveis []string
	medidos, cenasLidas := map[string]bool{}, 0
	for _, cena := range scenesThatWriteAddresses(t, f) {
		rec := f.pede(t, cena.Usuario, "GET", cena.Caminho, "")
		if rec.Code != http.StatusOK {
			t.Errorf("a cena %q (%s) respondeu %d: ela saiu da lista sem ninguém tirar, e uma cena que não abre não mede nada",
				cena.Nome, cena.Caminho, rec.Code)
			continue
		}
		cenasLidas++
		achados, naoLidos := addressesInHTML(cena.Nome, rec.Body.String())
		ilegiveis = append(ilegiveis, naoLidos...)
		for _, endereco := range achados {
			chave := endereco.Metodo + " " + endereco.Caminho
			if medidos[chave] {
				continue
			}
			medidos[chave] = true
			if !routerKnows(mux, endereco.Metodo, endereco.Caminho) {
				faltando = append(faltando,
					endereco.Metodo+" "+endereco.Caminho+"  ← escrito pela cena "+endereco.Origem)
			}
		}
	}

	// O DENOMINADOR. Uma lista de faltantes vazia e um extrator que parou de
	// casar são a mesma cor no terminal, e por isso o guarda afirma quantos
	// endereços olhou antes de afirmar que nenhum falta.
	if cenasLidas < 25 || len(medidos) < 150 {
		t.Fatalf("a varredura leu %d cenas e %d endereços distintos — o extrator é o primeiro suspeito",
			cenasLidas, len(medidos))
	}

	sort.Strings(ilegiveis)
	if len(ilegiveis) > 0 {
		t.Errorf("%d chamadas do Datastar que o extrator NÃO soube resolver:\n  %s\n"+
			"Ramo que ignora o que não entende produz lista de falhas com cara de descoberta "+
			"(ALE-294). Ou a forma nova entra no extrator, ou ela não é forma.",
			len(ilegiveis), strings.Join(ilegiveis, "\n  "))
	}
	sort.Strings(faltando)
	if len(faltando) > 0 {
		t.Errorf("%d de %d endereços não existem no roteador:\n  %s\n"+
			"O Datastar descarta o remendo de toda resposta não-2xx, então um endereço morto "+
			"não deixa erro na tela, no console nem em lugar nenhum: o gesto simplesmente não acontece.",
			len(faltando), len(medidos), strings.Join(faltando, "\n  "))
	}
	// A ÚLTIMA LINHA NÃO PODE DESMENTIR O VEREDITO. Os dois guardas da ALE-307
	// escrevem o `t.Logf` de sucesso depois do `t.Errorf`, e o que se lê antes
	// do FAIL é "nenhuma com coordenada no caminho" — a família do `finally` da
	// ALE-245, dentro do guarda que existe para combatê-la.
	if !t.Failed() {
		t.Logf("endereços: %d distintos em %d cenas, todos no roteador", len(medidos), cenasLidas)
	}
}
