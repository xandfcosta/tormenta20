package api

import (
	"bufio"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ABRIR A MESA REGISTRA A PRESENÇA (ALE-287).
//
// O anel de cada carta do elenco dizia quem está com a aba aberta, e ficava
// CINZA para sempre: quem preenchia o registro era o handshake da rota
// `/events` da SPA, apagada na ALE-277 por não ter consumidor. Ninguém em
// produção chamava `Join` desde que a SPA saiu (ALE-272), então o mestre lia a
// mesa deserta com três jogadores conectados — e "todos fora" tem cara de
// medição, não de ausência de medição.
//
// # Por que este caso não é o mesmo que o do anel
//
// Já havia um guarda do DESENHO (`TestTheGmSeesWhoIsAtTheTableAndThePlayerDoesNot`)
// e ele passava: só que ele chamava `Presence().Join` ele mesmo, arranjando um
// estado que a produção não sabia produzir. É a terceira vez nesta issue que um
// verde vinha da bancada e não do app.
//
// Por isso este caso ANDA pelo fluxo de verdade: servidor HTTP real, o
// `/stream` aberto, e a pergunta feita ao registro — sem tocar no `Join`.
func TestOpeningTheTableStreamRegistersPresence(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	srv := httptest.NewServer(f.s.WebRouter())
	defer srv.Close()

	// O CONTROLE, e ele é obrigatório: sem a leitura de antes, "vi o jogador no
	// elenco" não distingue o fluxo tendo registrado de um registro que já
	// estava lá.
	if antes := f.s.tableHost().Presence().Roster(f.sessionID); len(antes) != 0 {
		t.Fatalf("o elenco já tinha %d gente antes de alguém abrir a mesa", len(antes))
	}

	// O CANCELAMENTO É DEFERIDO, e isso não é higiene: sem ele qualquer
	// `t.Fatalf` abaixo pula o `fechar()`, o stream fica aberto, e o
	// `srv.Close()` do defer de cima espera por ele PARA SEMPRE. O caso deixa de
	// reprovar e passa a TRAVAR — foi o que aconteceu ao sabotar o `Join`: o
	// veredito virou "test timed out after 1m0s", que não diz nada sobre
	// presença. Limpeza não pode falar mais alto que o defeito (ALE-245).
	ctx, fechar := context.WithCancel(context.Background())
	defer fechar()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+f.tableUrl()+"/stream", nil)
	if err != nil {
		t.Fatalf("montar pedido: %v", err)
	}
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Authorization", "Bearer "+f.token(t, f.jogador))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("abrir stream: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	// LÊ o primeiro quadro antes de perguntar: sem isso a asserção corre contra
	// o handshake, e um teste que às vezes ganha a corrida é pior que um que
	// nunca ganha — ele reprova em máquina lenta e some da vista na rápida.
	esperaOPrimeiroQuadro(t, resp)

	presentes := f.s.tableHost().Presence().Roster(f.sessionID)
	if len(presentes) != 1 || presentes[0].UserID != f.jogador {
		t.Fatalf("o elenco presente = %+v, queria só o jogador %d", presentes, f.jogador)
	}
	if presentes[0].Role != "player" {
		t.Errorf("o jogador entrou como %q — o papel sai do `view.Mestre`", presentes[0].Role)
	}

	// FECHAR A ABA TIRA A PESSOA, e é a metade que uma rota nova erraria: quem
	// avisa da saída é o `r.Context()` cancelado, não um gesto do cliente.
	fechar()
	_ = resp.Body.Close()
	ateSumir(t, f)
}

func esperaOPrimeiroQuadro(t *testing.T, resp *http.Response) {
	t.Helper()
	zr, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("o corpo não é gzip: %v", err)
	}
	leitor := bufio.NewScanner(zr)
	leitor.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for leitor.Scan() {
		if strings.HasPrefix(leitor.Text(), "event:") {
			return
		}
	}
	t.Fatal("o stream fechou sem mandar quadro nenhum")
}

// ateSumir espera a goroutine do stream perceber o cancelamento.
//
// Sondagem e não `sleep` fixo: o `defer` do lado do servidor roda quando o
// runtime escalona a goroutine, e um tempo fixo escolhido na minha máquina é
// exatamente o teste que pisca na de outra pessoa.
func ateSumir(t *testing.T, f pilotoFixture) {
	t.Helper()
	limite := time.Now().Add(2 * time.Second)
	for time.Now().Before(limite) {
		if len(f.s.tableHost().Presence().Roster(f.sessionID)) == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("o jogador continuou no elenco depois de fechar a aba: %+v",
		f.s.tableHost().Presence().Roster(f.sessionID))
}
