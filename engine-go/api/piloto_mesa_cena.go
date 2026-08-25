package api

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	datastar "github.com/starfederation/datastar-go/datastar"

	"t20engine/tabuleiro"
)

// ABRIR e ENCERRAR a cena na Mesa em Datastar (ALE-264, item 3).
//
// Abrir tabuleiro NÃO inicia combate, e essa ortogonalidade é o que faz a
// taverna existir: a cena de interpretação também tem posição. Quem começa o
// turno é o `next-turn`, e é dele que sai o deslocamento — sem ele o movimento
// roda sem teto, o que é a regra e não um defeito.
//
// Encerrar ARQUIVA (ALE-124): a cena vira um Lugar da campanha com as peças onde
// estavam. Por isso o rótulo é "Encerrar" e não "Fechar" — fechar sugere perder.

func (s *Server) rotasDaCena(r chi.Router) {
	base := "/mesa/{campaignId}/{sessionId}/tabuleiro"
	r.Post(base+"/abrir", s.comandoDoMestreNoTabuleiro(abreOTabuleiro))
	r.Post(base+"/encerrar", s.comandoDoMestreNoTabuleiro(encerraOTabuleiro))
}

// abreOTabuleiro monta a cena com o lugar e o chão que o mestre escolheu.
func abreOTabuleiro(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	lugar, chao, err := cenaDosSinais(c.R)
	if err != nil {
		return nil, err
	}
	b := st.boards.Open(c.R.Context(), c.SessionID, lugar, chao)
	// O formulário volta ao zero, como o do combatente: sem isto o lugar fica no
	// campo e a cena seguinte nasce com o nome da anterior.
	c.Sinais["novolugar"] = ""
	c.Sinais["novochao"] = tabuleiro.ChaoPadrao()
	return b, nil
}

// encerraOTabuleiro arquiva e tira a cena da mesa.
//
// A falha ao ARQUIVAR não impede o encerrar, e a ordem é a do `handleBoardClose`
// de propósito: o mestre mandou tirar a cena da mesa, e recusar isso porque o
// acervo falhou deixaria a mesa presa numa cena que já acabou.
func encerraOTabuleiro(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	if atual := st.boards.Get(c.R.Context(), c.SessionID); atual != nil {
		if err := st.boards.Archive(c.R.Context(), c.CampaignID, atual); err != nil {
			log.Printf("session %d: falha ao arquivar o lugar (%v)", c.SessionID, err)
		}
	}
	st.boards.Close(c.R.Context(), c.SessionID)
	// `nil` é a mensagem: "esta sessão não tem tabuleiro" é estado de verdade, e
	// não uma grade vazia. O `comandoDoTabuleiro` só publica quando não é nil, e
	// aqui o nil é justamente o que a mesa precisa saber — por isso a publicação
	// é explícita e o retorno não a repete.
	st.publishBoardState(c.SessionID, nil)
	return nil, nil
}

// cenaDosSinais lê o diálogo de abrir.
//
// TODOS OS NOMES SÃO MINÚSCULOS porque são chaves de `data-bind:`, e nome de
// atributo é minusculado pelo analisador de HTML — um `data-bind:novoChao` liga
// um sinal `novochao` e deixa o declarado intocado.
//
// Os dois defaults são a mesma decisão da SPA: lugar em branco vira "Cena",
// porque o mestre que só quer a grade não deve ser barrado por um campo; e chão
// desconhecido cai no padrão em vez de recusar, porque um valor que a tela não
// oferece só chega por posse do fio, e a resposta a isso é desenhar pedra e não
// discutir.
func cenaDosSinais(r *http.Request) (lugar, chao string, err error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	var sinais struct {
		Lugar string `json:"novolugar"`
		Chao  string `json:"novochao"`
	}
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		return "", "", fmt.Errorf("não entendi a cena enviada: %v", err)
	}
	lugar = strings.TrimSpace(sinais.Lugar)
	if lugar == "" {
		lugar = "Cena"
	}
	return lugar, chaoConhecido(sinais.Chao), nil
}

// chaoConhecido devolve o chão pedido se ele existe, ou o padrão.
func chaoConhecido(pedido string) string {
	for _, c := range tabuleiro.ChoesDoLugar {
		if c.ID == pedido {
			return pedido
		}
	}
	return tabuleiro.ChaoPadrao()
}
