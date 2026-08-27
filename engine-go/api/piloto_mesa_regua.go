package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/engine"
	"t20engine/tabuleiro"
)

// A RÉGUA e o GABARITO da mesa em Datastar (ALE-269, superfície 8).
//
// As duas respondem perguntas que a mesa faz em voz alta toda rodada — "dá para
// acertar daqui?" e "se eu soltar aqui, quem pega?" — e hoje se respondem
// contando quadrado com o dedo na tela, que é o que um tabuleiro digital existe
// para poupar.
//
// NÃO MUTAM NADA, e é essa a diferença que desenha estes dois caminhos. Medir
// não muda a cena: nada é publicado, nada é gravado, e a resposta é do TAMANHO
// de dois sinais — não das nove regiões que o `respondeAoMestre` reescreve. Uma
// medição que remendasse o mapa trocaria a peça debaixo do dedo de quem está
// arrastando, que é o defeito que a região `mesa-por-no-mapa` já existe para
// evitar.
//
// E é por isso que a régua NÃO é estado do tabuleiro, ao contrário do movimento
// proposto: o provisório é estado porque a mesa inteira decide sobre ele; a
// régua de um jogador não é assunto de ninguém.
//
// A CONTA é do motor Go — `engine.Measure` e `engine.AreaSquares`, os mesmos que
// a SPA chamava pelo WASM. O que vem do navegador são os cliques; o que volta é
// o desenho e a frase, e nenhuma das duas é recalculada na tela.

func (s *Server) rotasDaRegua(r chi.Router) {
	base := "/mesa/{campaignId}/{sessionId}/tabuleiro"
	r.Post(base+"/regua", s.handleReguaDaMesa)
	r.Post(base+"/gabarito/{tipo}/{tamanho}/{x}/{y}/{mx}/{my}", s.handleGabaritoDaMesa)
}

// handleReguaDaMesa devolve a leitura de CADA PERNA e o total da polilinha.
//
// A régua virou POLILINHA na ALE-203 ("a régua não permite calcular distâncias
// com mais de uma parada"), e com ela a rota deixou de ter as pontas no caminho:
// o número de paradas é variável, e um caminho com número variável de segmentos
// seria uma rota que muda de forma. As paradas chegam nos SINAIS, que é onde
// elas já moram.
//
// O que volta continua sendo só SINAL — a régua não muta nada, e uma medição que
// remendasse o mapa trocaria a peça debaixo do dedo de quem está arrastando.
func (s *Server) handleReguaDaMesa(w http.ResponseWriter, r *http.Request) {
	if _, _, _, ok := s.quemMedeAMesa(w, r); !ok {
		return
	}
	paradas, err := asParadasDaRegua(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	escreveSinais(w, r, aLeituraDaPolilinha(paradas))
}

// oMaximoDeParadas é o teto da polilinha.
//
// Doze paradas são onze pernas, e onze é mais do que qualquer movimento de um
// turno precisa contornar. O teto não é zelo com memória: é o tamanho da RESERVA
// de rótulos que o `.templ` desenha, e os dois números têm de ser o mesmo — um
// rótulo sem nó é um número que ninguém vê.
const oMaximoDeParadas = 12

// asParadasDaRegua lê as paradas dos sinais, com a MIRA no fim.
//
// A mira — o quadrado sob o ponteiro — entra como se fosse a última parada, e é
// isso que faz a perna viva ter medida antes de alguém clicar. Ela só entra
// enquanto a régua está MEDINDO: congelada, o ponteiro passeia e a medida fica.
func asParadasDaRegua(r *http.Request) ([]engine.Square, error) {
	var sinais struct {
		Pontos [][]int `json:"reguapontos"`
		MiraX  int     `json:"reguamirax"`
		MiraY  int     `json:"reguamiray"`
		Fase   int     `json:"reguafase"`
	}
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		return nil, fmt.Errorf("as paradas da régua não vieram: %w", err)
	}
	paradas := make([]engine.Square, 0, len(sinais.Pontos)+1)
	for _, p := range sinais.Pontos {
		if len(p) != 2 {
			return nil, fmt.Errorf("parada %v não é um par de números", p)
		}
		paradas = append(paradas, engine.Square{X: p[0], Y: p[1]})
	}
	if sinais.Fase == reguaMedindo {
		paradas = append(paradas, engine.Square{X: sinais.MiraX, Y: sinais.MiraY})
	}
	if len(paradas) > oMaximoDeParadas {
		return nil, fmt.Errorf("a régua tem %d paradas e o teto é %d", len(paradas), oMaximoDeParadas)
	}
	return paradas, nil
}

// aLeituraDaPolilinha escreve o rótulo de cada perna e a frase do TOTAL.
//
// Por que o rótulo da perna NÃO diz a faixa: faixa é sobre ALCANCE (p224), e
// alcance é linha reta — a perna de um caminho não é o alcance de nada. Dizer
// "alcance curto" sobre a segunda perna de um contorno seria uma resposta certa
// para uma pergunta que ninguém fez.
//
// E por que o TOTAL diz: com DUAS paradas a polilinha é uma reta, que é
// exatamente a régua de sempre, e a mesa pergunta "dá para acertar daqui?". Com
// mais paradas o total é o custo do CAMINHO, e a faixa continua sendo a leitura
// certa do número — só que da pergunta "cabe no meu deslocamento?".
func aLeituraDaPolilinha(paradas []engine.Square) map[string]any {
	rotulos := make([]string, 0, oMaximoDeParadas)
	total := 0
	for i := 1; i < len(paradas); i++ {
		perna := engine.Measure(paradas[i-1], paradas[i])
		total += perna.Squares
		rotulos = append(rotulos, aPernaEmMetros(perna))
	}
	if len(paradas) < 2 {
		return map[string]any{"reguarotulos": rotulos, "reguatexto": aDicaDaReguaVazia}
	}
	return map[string]any{
		"reguarotulos": rotulos,
		"reguatexto": leituraDaRegua(engine.Measurement{
			Squares: total,
			Metres:  float64(total) * engine.SquareMetres,
			Band:    engine.BandFor(total),
		}),
	}
}

// leituraDaRegua escreve a medida em QUADRADOS, em metros e na faixa do livro.
//
// Quadrado é a unidade da regra (p236) e metro é a unidade da conversa na mesa —
// dizer só um dos dois manda alguém converter de cabeça no meio do turno.
//
// @example leituraDaRegua(engine.Measure(a, b)) // "3 quadrados (4,5m) · alcance curto"
func leituraDaRegua(m engine.Measurement) string {
	faixa := "alcance " + string(m.Band)
	if m.Band == engine.RangeBeyond {
		faixa = "além do alcance longo"
	}
	return fmt.Sprintf("%d %s (%sm) · %s", m.Squares, quadradosEmPortugues(m.Squares), emMetros(m.Metres), faixa)
}

// aPernaEmMetros é o rótulo que pousa sobre a linha entre duas paradas (pedido
// do dono).
//
// METRO e não quadrado, ao contrário da frase do total: sobre a linha cabe UMA
// unidade, e a que a mesa fala em voz alta é o metro — "ele está a nove metros"
// é a frase do turno, e "seis quadrados" obriga a converter de cabeça. A frase
// do total continua trazendo as duas, porque lá cabe.
//
// A PERNA DE ZERO devolve VAZIO, e é o que apaga o rótulo: o instante logo depois
// de um clique tem a mira em cima da parada que acabou de nascer, e um "0,0m"
// piscando sob o dedo é ruído sobre o gesto que a pessoa está fazendo.
func aPernaEmMetros(m engine.Measurement) string {
	if m.Squares == 0 {
		return ""
	}
	return emMetros(m.Metres) + "m"
}

// aDicaDaReguaVazia é o que a barra diz com uma ponta só posta.
//
// Uma ponta não é distância nenhuma, e uma frase em branco ali faria a régua
// parecer quebrada justamente no instante em que ela acabou de começar.
const aDicaDaReguaVazia = "Clique para acrescentar paradas · duplo clique fecha · botão direito apaga"

// handleGabaritoDaMesa desenha o gabarito e diz QUEM está dentro.
//
// A lista de nomes é o que ela tem de maior, e não o desenho: o desenho mostra
// onde, mas é o nome que resolve a dúvida e é o nome que a mesa fala em voz alta.
func (s *Server) handleGabaritoDaMesa(w http.ResponseWriter, r *http.Request) {
	papel, sessionID, tabuleiroID, ok := s.quemMedeAMesa(w, r)
	if !ok {
		return
	}
	tipo, err := gabaritoDaURL(chi.URLParam(r, "tipo"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	origem, errO := quadradoDoCaminho(r, "x", "y")
	mira, errM := quadradoDoCaminho(r, "mx", "my")
	if errO != nil || errM != nil {
		http.Error(w, "a origem e a mira do gabarito precisam ser dois pares de números", http.StatusBadRequest)
		return
	}
	area := engine.Area{
		Kind:      tipo,
		Size:      tamanhoDoGabarito(chi.URLParam(r, "tamanho")),
		Direction: direcaoDoGabarito(origem, mira),
	}
	// A mira AINDA NÃO FOI DADA quando ela é a própria origem: o cone e a linha
	// precisam apontar para algum lado, e desenhá-los para um lado escolhido pelo
	// servidor seria inventar a decisão que falta.
	if apontaOGabarito(tipo) && mira == origem {
		escreveSinais(w, r, map[string]any{
			"gabaritopath": "", "gabaritotexto": "Clique de novo para apontar.",
		})
		return
	}
	casas := engine.AreaSquares(origem, area)
	// O tabuleiro passa pelo MESMO gargalo por papel do resto da Mesa: quem
	// pergunta quem o cone pega não pode descobrir por aí a peça que a cortina e
	// o `Hidden` escondem dele.
	b := tabuleiro.BoardForRole(papel, s.boards.Get(r.Context(), sessionID, tabuleiroID))
	escreveSinais(w, r, map[string]any{
		"gabaritopath":  caminhoDasCasas(casas),
		"gabaritotexto": quemOGabaritoPega(b, casas),
	})
}

// quemMedeAMesa é a mesma leitura de acesso dos comandos, sem o papel de mestre.
//
// Medir é de TODO MUNDO — "dá para acertar daqui?" é pergunta de quem ataca —, e
// a trava que sobra é a de sempre: quem não está na mesa não mede a cena dela.
//
// Devolve a ABA junto com o papel (ALE-205) porque medir é sempre sobre UMA cena:
// a régua que perguntasse "o tabuleiro da sessão" mediria a distância na taverna
// para quem está olhando a cripta, e o número sairia certo sobre o mapa errado —
// sem nada na tela dizendo que ele é de outro lugar.
func (s *Server) quemMedeAMesa(w http.ResponseWriter, r *http.Request) (papel string, sessionID int64, tabuleiroID string, ok bool) {
	campaignID, sessionID, ok := mesaParams(w, r)
	if !ok {
		return "", 0, "", false
	}
	user := currentUser(r)
	_, papel, status, err := s.sessionForCaller(r.Context(), user, campaignID, sessionID)
	if err != nil {
		http.Error(w, err.Error(), status)
		return "", 0, "", false
	}
	return papel, sessionID, s.aAbaDe(r.Context(), sessionID, user.ID), true
}

// escreveSinais responde SÓ com sinais, e é o que separa medir de comandar.
//
// O `respondeAoMestre` reescreve as nove regiões porque ele responde a uma
// MUTAÇÃO, e quem clicou precisa ver a cena nova. Aqui não há cena nova: a
// resposta são duas cordas, e mandar o mapa junto seria remendar o nó que a
// pessoa está usando para medir.
func escreveSinais(w http.ResponseWriter, r *http.Request, sinais map[string]any) {
	sse := datastar.NewSSE(w, r)
	_ = sse.MarshalAndPatchSignals(sinais)
}

// caminhoDasCasas vira UM `<path>` e não um nó por quadrado.
//
// Não é economia de bytes: é o que faz o desenho caber num SINAL. Um nó por
// casa teria de ser HTML remendado DENTRO da região do mapa, e o próximo quadro
// do stream — que não sabe do gabarito de ninguém — apagaria o desenho sem erro
// nenhum. Com um `d` só, o gabarito mora num sinal, e sinal sobrevive ao remendo.
//
// As coordenadas são as do PLANO, com sinal; quem tira a quina da moldura é o
// `transform` do grupo, que o servidor redesenha quando a moldura cresce.
//
// @example caminhoDasCasas([]engine.Square{{X: -1, Y: 2}}) // "M -1 2 h 1 v 1 h -1 Z"
func caminhoDasCasas(casas []engine.Square) string {
	if len(casas) == 0 {
		return ""
	}
	var b strings.Builder
	for i, q := range casas {
		if i > 0 {
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "M %d %d h 1 v 1 h -1 Z", q.X, q.Y)
	}
	return b.String()
}

// quemOGabaritoPega cruza a área com as peças da cena.
//
// A peça entra se QUALQUER quadrado do corpo dela cair na área — uma Colossal
// ocupa 6×6 (p107), e exigir que ela caiba inteira deixaria o dragão de fora do
// próprio incêndio.
func quemOGabaritoPega(b *tabuleiro.BoardState, casas []engine.Square) string {
	if len(casas) == 0 {
		return "Clique numa casa para pôr o gabarito."
	}
	dentro := map[engine.Square]bool{}
	for _, q := range casas {
		dentro[q] = true
	}
	var nomes []string
	if b != nil {
		for i := range b.Tokens {
			if t := &b.Tokens[i]; pegaAPeca(dentro, t) {
				nomes = append(nomes, t.Label)
			}
		}
	}
	if len(nomes) == 0 {
		return "Ninguém dentro."
	}
	return fmt.Sprintf("Pega %s: %s", pecasEmPortugues(len(nomes)), strings.Join(nomes, ", "))
}

func pegaAPeca(dentro map[engine.Square]bool, t *tabuleiro.BoardToken) bool {
	lado := t.Footprint
	if lado < 1 {
		lado = 1
	}
	for dx := 0; dx < lado; dx++ {
		for dy := 0; dy < lado; dy++ {
			if dentro[engine.Square{X: t.X + dx, Y: t.Y + dy}] {
				return true
			}
		}
	}
	return false
}

// direcaoDoGabarito é o passo unitário do segundo clique, com ZONA MORTA.
//
// Um clique quase em linha vira ortogonal e um clique perto de 45° vira
// diagonal, porque o livro desenha o cone nessas duas formas e não numa terceira
// (p225): sem a zona morta, um pixel de diferença no clique trocaria a forma
// inteira do gabarito debaixo do dedo.
//
// Isto é decisão de TELA e não regra do livro — o que o livro dá é a figura, e
// ela é transcrita em `engine.AreaSquares`. O que mora aqui é o arredondamento
// do gesto, e por isso ele fica no caminho HTTP e não no motor.
func direcaoDoGabarito(origem, mira engine.Square) engine.Square {
	dx, dy := mira.X-origem.X, mira.Y-origem.Y
	if dx == 0 && dy == 0 {
		return engine.Square{X: 1}
	}
	if moduloDe(dx) > 2*moduloDe(dy) {
		return engine.Square{X: sentidoDe(dx)}
	}
	if moduloDe(dy) > 2*moduloDe(dx) {
		return engine.Square{Y: sentidoDe(dy)}
	}
	return engine.Square{X: sentidoOuUm(dx), Y: sentidoOuUm(dy)}
}

func moduloDe(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func sentidoDe(n int) int {
	if n < 0 {
		return -1
	}
	if n > 0 {
		return 1
	}
	return 0
}

// sinalOuUm nunca devolve zero: uma diagonal com um eixo parado não é diagonal,
// e um passo (0,0) faria o cone nascer sem direção nenhuma.
func sentidoOuUm(n int) int {
	if s := sentidoDe(n); s != 0 {
		return s
	}
	return 1
}

// apontaOGabarito diz quais formas precisam de um segundo clique. A esfera vai
// para todos os lados e o quadrado não aponta para lugar nenhum (p225).
func apontaOGabarito(k engine.AreaKind) bool {
	return k == engine.AreaCone || k == engine.AreaLine
}

// aFormaNasceNaIntersecao diz onde a ORIGEM da forma pousa, e isso é REGRA do
// livro e não escolha de tela (p225, conferido no PDF p231):
//
//	"Esfera. Surge na INTERSEÇÃO DE QUATRO QUADRADOS, estendendo-se em todas as
//	 direções até o limite de seu raio."
//	"Quadrado. Surge NO QUADRADO ou quadrados escolhidos, afetando o piso."
//	"Cone/Linha. Surge ADJACENTE A VOCÊ e se afasta de você…"
//
// O dono pediu uma escolha de "montar esfera centralizada" e o livro respondeu:
// não há escolha, a esfera é SEMPRE da interseção. O `engine.sphereSquares` já
// desenhava assim — ele recebe a origem como CANTO. Quem errava era a TELA, que
// mandava o quadrado do `floor` do clique e desenhava o pingo no CENTRO dele:
// meio quadrado de deslocamento entre onde a pessoa clicou e onde a bola caiu, e
// nada na tela dizendo por quê. É a resposta ao "não tem feedback visual algum".
func aFormaNasceNaIntersecao(k engine.AreaKind) bool {
	return k == engine.AreaSphere
}

// gabaritoDaURL recusa forma que o livro não tem, e a mensagem diz o valor
// recebido e a lista do que existe.
func gabaritoDaURL(bruto string) (engine.AreaKind, error) {
	switch k := engine.AreaKind(bruto); k {
	case engine.AreaSphere, engine.AreaCone, engine.AreaLine, engine.AreaSquare:
		return k, nil
	default:
		return "", fmt.Errorf("gabarito %q não existe; são esfera, cone, linha e quadrado (p225)", bruto)
	}
}

// tamanhoDoGabarito trava a caixa de digitação em vez de recusar.
//
// Zero e negativo não desenham nada, e a recusa cairia numa frase enquanto a
// pessoa ainda está digitando o número — 60 quadrados é o alcance longo do livro
// (p224), que é o maior gabarito que cabe numa mesa.
func tamanhoDoGabarito(bruto string) int {
	n, err := intDoCaminho(bruto)
	if err != nil || n < 1 {
		return 1
	}
	if n > engine.LongRangeSquares {
		return engine.LongRangeSquares
	}
	return n
}

// quadradoDoCaminho lê um par de coordenadas com os nomes que a rota deu.
//
// Coordenada NEGATIVA é lugar legítimo — o plano não tem bordas —, e por isso
// as pontas viajam no CAMINHO e não num sinal da página: o valor é do clique que
// aconteceu, e não de um estado que outro gesto poderia ter mexido.
func quadradoDoCaminho(r *http.Request, nomeX, nomeY string) (engine.Square, error) {
	x, errX := intDoCaminho(chi.URLParam(r, nomeX))
	y, errY := intDoCaminho(chi.URLParam(r, nomeY))
	if errX != nil || errY != nil {
		return engine.Square{}, fmt.Errorf("quadrado (%q,%q) não é um par de números",
			chi.URLParam(r, nomeX), chi.URLParam(r, nomeY))
	}
	return engine.Square{X: x, Y: y}, nil
}

// emMetros escreve o metro com VÍRGULA, que é como o livro e a mesa o leem.
func emMetros(m float64) string {
	return strings.Replace(fmt.Sprintf("%.1f", m), ".", ",", 1)
}

func quadradosEmPortugues(n int) string {
	if n == 1 {
		return "quadrado"
	}
	return "quadrados"
}
