package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

// Quem pode COMANDAR o combate.
//
// Este arquivo é o `realtime_gating_test.go` traduzido para HTTP (ALE-253). A
// invariante não mudou: o risco real não é a linha da porta estar errada — é um
// comando NOVO esquecer dela, e um jogador reiniciar o combate da mesa inteira
// sem nada acusar.
//
// O que mudou foi de ONDE se lê o registro. Antes era `sock.On("nome", …)`,
// agora é o roteador chi. A identidade do comando passou a ser `MÉTODO caminho`,
// que é a MESMA chave do `realtime-wire.test.ts` do front — os dois guardas
// falam da mesma coisa pelo mesmo nome, de propósito.
//
// E um detalhe que o teste antigo não precisava ter: a porta agora mora em
// PREÂMBULO comum (`boardGmCommand` e irmãos), então procurar `requireGmRole`
// só no corpo do handler leria "sem porta" em quase todo comando de tabuleiro.
// A resolução é TRANSITIVA — um guarda só mede o que ele visita, e aqui ele
// precisa visitar quem o handler chama.

// gmGate declara, por rota, se ela é do MESTRE. As exceções são deliberadas e
// estão anotadas uma a uma. A rota é escrita como o chi a registra, com os
// prefixos do `r.Route` já juntados e sem o `/{id}` da sessão.
var gmGate = map[string]bool{
	"POST /initiative":                        true,
	"POST /initiative/self":                   false, // o JOGADOR rola a PRÓPRIA iniciativa: exceção deliberada
	"PATCH /initiative/{entryId}":             true,
	"DELETE /initiative/{entryId}":            true,
	"POST /initiative/next-turn":              true,
	"POST /initiative/previous-turn":          true,
	"DELETE /initiative":                      true, // reiniciar o combate
	"POST /initiative/populate":               true,
	"POST /scene/start":                       true, // quem decide que a cena começou é o mestre (ALE-210)
	"POST /scene/end":                         true,
	"PATCH /initiative/{entryId}/vitals":      false, // regra mais fina: assertVitalsEditableFor
	"POST /initiative/{entryId}/vitals/delta": false, // idem
	"POST /rest":                              true,
	"POST /initiative/{entryId}/effects":      true,
	// Tabuleiro (ALE-124): abrir, montar e esconder peça é do mestre. tabuleiro.Mover tem
	// regra mais fina (assertMovable) e por isso NÃO leva a porta larga — igual
	// aos vitais: o mestre move qualquer peça, o jogador move a própria na vez
	// dela, e fora de combate cada um anda com a sua.
	"POST /board":                            true,
	"DELETE /board":                          true,
	"POST /board/curtain":                    true,  // a cortina é o gesto do mestre; não há como o jogador abrir a sua (ALE-202)
	"GET /board":                             false, // ler o tabuleiro é de todo mundo na mesa — REDIGIDO por papel
	"POST /board/tokens":                     true,
	"DELETE /board/tokens/{tokenId}":         true,
	"PATCH /board/tokens/{tokenId}":          true,
	"POST /board/tokens/{tokenId}/duplicate": true, // "mais um zumbi" é montar encontro, e montar é do mestre
	"POST /board/populate":                   true,
	// O marcador é o LUGAR apontado no mapa (ALE-195): ele nasce escondido e o
	// mestre revela, então marcar, revelar e apagar são todos dele.
	"POST /board/markers":                 true,
	"PATCH /board/markers/{markerId}":     true,
	"DELETE /board/markers/{markerId}":    true,
	"POST /board/terrain":                 true, // o chão é da cena, e a cena é do mestre
	"GET /board/as-player":                true, // "ver como jogador" é a lente do mestre sobre a própria cena
	"GET /board/places":                   true, // o acervo de cenas guardadas é preparação do mestre
	"POST /board/places/{placeId}/reopen": true,
	"GET /board/places/{placeId}/scene":   true, // a cena guardada é preparação: o jogador não a pede
	"PUT /board/places/{placeId}/scene":   true, // montar o lugar é do mestre, e não toca na mesa
	"DELETE /board/places/{placeId}":      true,
	"POST /board/tokens/{tokenId}/move":   false, // assertMovable: papel, posse e a VEZ
	"POST /board/move/cancel":             false, // desfaz o próprio provisório; o mestre desfaz o de qualquer um
	"POST /board/move/commit":             false, // idem
}

var (
	rotaChi     = regexp.MustCompile(`^\s*r\.(Get|Post|Put|Patch|Delete)\("([^"]+)",\s*s\.(\w+)\)`)
	prefixoChi  = regexp.MustCompile(`^\s*r\.Route\("([^"]+)"`)
	inicioFunc  = `func (s *Server) %s(`
	chamadaFunc = regexp.MustCompile(`\bs\.(\w+)\(`)
)

// comandoRota é uma rota do `mountLiveRoutes` com o handler dela.
type comandoRota struct {
	chave   string // "POST /initiative/next-turn"
	handler string
}

func fonteDosComandos(t *testing.T) string {
	t.Helper()
	var tudo strings.Builder
	for _, arquivo := range []string{"session_commands.go", "board_commands.go"} {
		bruto, err := os.ReadFile(arquivo)
		if err != nil {
			t.Fatalf("ler %s: %v", arquivo, err)
		}
		tudo.Write(bruto)
	}
	return tudo.String()
}

// rotasAoVivo lê o `mountLiveRoutes` juntando os prefixos dos `r.Route`.
func rotasAoVivo(t *testing.T) []comandoRota {
	t.Helper()
	fonte := fonteDosComandos(t)
	corpo := fonte[strings.Index(fonte, "func (s *Server) mountLiveRoutes"):]
	var rotas []comandoRota
	var prefixos []string
	for _, linha := range strings.Split(corpo, "\n") {
		if m := prefixoChi.FindStringSubmatch(linha); m != nil {
			prefixos = append(prefixos, m[1])
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(linha), "})") && len(prefixos) > 0 {
			prefixos = prefixos[:len(prefixos)-1]
		}
		m := rotaChi.FindStringSubmatch(linha)
		if m == nil {
			continue
		}
		caminho := strings.Join(prefixos, "")
		if m[2] != "/" {
			caminho += m[2]
		}
		caminho = strings.Replace(caminho, "/{id}", "", 1)
		if caminho == "" {
			caminho = "/"
		}
		rotas = append(rotas, comandoRota{chave: strings.ToUpper(m[1]) + " " + caminho, handler: m[3]})
	}
	return rotas
}

// corpoDaFuncao devolve o corpo de um método de `Server` — do cabeçalho até o
// próximo `\n}` de topo, que em Go é o fim da função.
func corpoDaFuncao(t *testing.T, fonte, nome string) string {
	t.Helper()
	inicio := strings.Index(fonte, strings.ReplaceAll(inicioFunc, "%s", nome))
	if inicio < 0 {
		t.Fatalf("não achei a função %q nos comandos", nome)
	}
	fim := strings.Index(fonte[inicio:], "\n}\n")
	if fim < 0 {
		t.Fatalf("função %q não termina", nome)
	}
	return fonte[inicio : inicio+fim]
}

// temPortaDoMestre resolve a porta ATRAVESSANDO as chamadas. A porta mora em
// preâmbulo comum (`boardGmCommand`, `boardTokenCommand`, `boardPlaceCommand`),
// e um guarda que olhasse só o corpo do handler leria "sem porta" em quase todo
// comando de tabuleiro — verde por cegueira, que é o pior resultado possível.
func temPortaDoMestre(t *testing.T, fonte, handler string) bool {
	t.Helper()
	visitados := map[string]bool{}
	var visita func(string) bool
	visita = func(nome string) bool {
		if visitados[nome] {
			return false
		}
		visitados[nome] = true
		if !strings.Contains(fonte, strings.ReplaceAll(inicioFunc, "%s", nome)) {
			return false // não é método de Server: nada a atravessar
		}
		corpo := corpoDaFuncao(t, fonte, nome)
		if strings.Contains(corpo, "requireGmRole") {
			return true
		}
		for _, m := range chamadaFunc.FindAllStringSubmatch(corpo, -1) {
			if visita(m[1]) {
				return true
			}
		}
		return false
	}
	return visita(handler)
}

func TestPortaDoMestreEmTodaRotaDaMesaAoVivo(t *testing.T) {
	fonte := fonteDosComandos(t)
	rotas := rotasAoVivo(t)
	if len(rotas) == 0 {
		t.Fatal("nenhuma rota reconhecida — o registro mudou de forma e este teste ficou cego")
	}

	vistas := map[string]bool{}
	for _, rota := range rotas {
		vistas[rota.chave] = true
		querGm, declarada := gmGate[rota.chave]
		if !declarada {
			t.Errorf("rota %q é nova: decida se é do mestre e declare em gmGate", rota.chave)
			continue
		}
		if temPortaDoMestre(t, fonte, rota.handler) != querGm {
			t.Errorf("%s (%s): porta do mestre=%v, declarado=%v",
				rota.chave, rota.handler, !querGm, querGm)
		}
	}

	for chave := range gmGate {
		if !vistas[chave] {
			t.Errorf("gmGate declara %q, mas o roteador não registra mais essa rota", chave)
		}
	}
}

// saidaDeEstado é toda linha que ENTREGA estado da mesa: o `Emit` do broadcast e
// o `plataforma.WriteJSON` que responde a quem comandou.
//
// O `board-state` entrou aqui junto com o primeiro Emit do tabuleiro (ALE-124):
// enquanto o regex só conhecia `session-state`, uma peça escondida podia sair
// sem redação nenhuma e este teste passava verde — é assim que uma rede fica
// cega, e o próprio arquivo avisa que é assim.
var saidaDeEstado = regexp.MustCompile(
	`(?m)^.*(sse\.Emit(Ordered)?\([^,]+, [^,]+, "(session|board)-state"|plataforma.WriteJSON\(w, http\.StatusOK, (state|board)\b).*$`)

// O PV oculto é do mestre, e o broadcast não é o único caminho do estado até a
// tela: a RESPOSTA do próprio comando hidrata quem o mandou, inclusive jogador.
// Foi exatamente por aí que ele vazou (ALE-122) — a redação existia e o ack
// passava por fora dela. Com HTTP o buraco é o mesmo, só que se chama `plataforma.WriteJSON`.
func TestEstadoSaiDoServidorFiltradoPorPapel(t *testing.T) {
	linhas := saidaDeEstado.FindAllString(fonteDosComandos(t), -1)
	if len(linhas) == 0 {
		t.Fatal("nenhuma saída de estado reconhecida — a forma mudou e este teste ficou cego")
	}

	for _, linha := range linhas {
		filtrada := strings.Contains(linha, "aovivo.StateForRole") ||
			strings.Contains(linha, "aovivo.RedactForPlayers") ||
			strings.Contains(linha, "tabuleiro.BoardForRole") ||
			strings.Contains(linha, `, "gm", `)
		if !filtrada {
			t.Errorf("estado sai sem filtro de papel:\n%s\ndiga para quem:"+
				" aovivo.StateForRole(ctx.Role, …) na resposta, ou o papel no Emit", strings.TrimSpace(linha))
		}
	}
}

// "Ver como jogador" redige para o JOGADOR, e não para quem pediu (ALE-193).
//
// O teste acima já obriga toda saída de estado a passar por um filtro de papel,
// e o `tabuleiro.BoardForRole(ctx.Role, …)` satisfaz aquela regra — só que aqui `ctx.Role`
// é "gm", então o mestre receberia a PRÓPRIA cena de volta: o botão acenderia,
// a peça escondida continuaria na tela, e ele concluiria que a mesa está vendo
// a emboscada. Um modo que mente sobre o segredo é pior que não ter modo.
func TestVerComoJogadorRedigeParaAMesaENaoParaQuemPediu(t *testing.T) {
	corpo := corpoDaFuncao(t, fonteDosComandos(t), "handleBoardAsPlayer")

	if !strings.Contains(corpo, `tabuleiro.BoardForRole("player"`) {
		t.Errorf("a resposta de as-player não redige para o jogador:\n%s", corpo)
	}
	if strings.Contains(corpo, "tabuleiro.BoardForRole(ctx.Role") {
		t.Error("as-player redige para o papel de QUEM PEDIU: o mestre receberia a própria cena de volta")
	}
}

// Quem MOVE peça resolve posse e orçamento contra o BANCO.
//
// O `gmGate` acima diz que as rotas de movimento não levam a porta larga do
// mestre — e é isso mesmo, porque a regra é mais fina. O risco que sobra é o
// inverso: uma rota nova de movimento que acredita no papel que o CLIENTE
// mandou, ou no orçamento que ele mandou junto. As duas coisas se resolvem em
// `moverFor`, que lê o dono do personagem e o deslocamento da ficha computada.
func TestMoveHandlersResolveOwnershipOnTheServer(t *testing.T) {
	fonte := fonteDosComandos(t)

	achados := 0
	for _, rota := range rotasAoVivo(t) {
		if !strings.Contains(rota.chave, "/move") {
			continue
		}
		achados++
		// `moverFor(` sem o receptor: ele saiu do gateway para o `Server` na
		// ALE-253, e prender o receptor faria este guarda acusar uma MUDANÇA DE
		// DONO como se fosse perda da regra. O que importa é que o autor seja
		// resolvido no servidor, não em qual struct o método mora.
		if !strings.Contains(corpoDaFuncao(t, fonte, rota.handler), "moverFor(") {
			t.Errorf("%s (%s) move peça sem resolver quem é o autor no servidor:"+
				" chame moverFor(ctx, tokenId) em vez de confiar no corpo", rota.chave, rota.handler)
		}
	}
	if achados == 0 {
		t.Fatal("nenhuma rota de movimento reconhecida — os caminhos mudaram e este teste ficou cego")
	}
}

// O tempo real entrou debaixo da política de origem do resto da API (ALE-253).
//
// Isto é o que sobrou do `TestSocketOriginFollowsTheHttpPolicy` (ALE-158), e
// sobrou MENOR de propósito: o `guardSocketOrigin` existia porque o socket.io
// tinha caminho próprio no mux, fora do `Router()`, e por isso precisava repetir
// a política de origem por conta — duas cópias da mesma regra. O `/events` é uma
// rota chi como as outras, então a política é a do `cors.Handler` do `Router()`,
// que já é testado onde mora.
//
// O que este teste ainda prende é o que a mudança poderia ter perdido sem
// ninguém ver: que o fluxo está DENTRO do roteador, e não pendurado à parte.
func TestOFluxoDeEventosMoraDentroDoRoteador(t *testing.T) {
	s := newTestServer(t)
	s.cfg.CORSOrigins = []string{"http://localhost:5173"}

	req := httptest.NewRequest(http.MethodOptions, "/campaigns/1/sessions/7/events", nil)
	req.Header.Set("Origin", "https://site-do-mal.example")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	s.Router().ServeHTTP(rec, req)

	if origem := rec.Header().Get("Access-Control-Allow-Origin"); origem != "" {
		t.Errorf("o preflight de terceiro foi refletido como %q — o /events saiu de baixo do CORS", origem)
	}
}
