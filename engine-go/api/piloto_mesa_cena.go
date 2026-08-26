package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	datastar "github.com/starfederation/datastar-go/datastar"

	"t20engine/engine"
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
	r.Post(base+"/lugares/{placeId}/reabrir", s.comandoDoMestreNoTabuleiro(reabreOLugar))
	r.Post(base+"/lugares/{placeId}/remover", s.comandoDoMestreNoTabuleiro(removeOLugar))
	// O TRAÇO e não o ponto (ALE-203): as duas rotas recebem de ONDE ATÉ ONDE o
	// dedo andou desde o aviso anterior do ponteiro. Um clique parado manda o
	// mesmo par duas vezes, que é um traço de uma casa. Ver `tabuleiro.CasasDoTraco`.
	r.Post(base+"/terreno/{especie}/{x}/{y}/ate/{x2}/{y2}", s.comandoContinuoDoMestre(pintaOTerreno))
	r.Post(base+"/terreno/limpar/{x}/{y}/ate/{x2}/{y2}", s.comandoContinuoDoMestre(limpaOTerreno))
	r.Post(base+"/pecas", s.comandoDoMestreNoTabuleiro(poeNoMapa))
}

// pintaOTerreno liga ou desliga uma espécie numa casa (T20 p238).
//
// A espécie e o quadrado vêm do CAMINHO, e o APAGAR vem da query. A divisão não
// é arbitrária: caminho é o que identifica a casa que o clique acertou — e é a
// mesma escolha do movimento —, enquanto apagar é um MODO da ferramenta, que
// vale para o arraste inteiro e não para um quadrado.
//
// Idempotente de propósito, e o `PaintTerrain` é quem garante: o pincel pinta
// ARRASTANDO e o arraste passa duas vezes pela mesma casa. Alternar faria a casa
// piscar entre brejo e chão limpo debaixo do dedo.
func pintaOTerreno(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	traco, err := tracoDaURL(c.R)
	if err != nil {
		return nil, err
	}
	if st.boards.Get(c.R.Context(), c.SessionID) == nil {
		return nil, fmt.Errorf("não há tabuleiro aberto para pintar")
	}
	especie := tabuleiro.EspecieConhecida(chi.URLParam(c.R, "especie"))
	ligado := c.R.URL.Query().Get("apagar") == ""
	return st.boards.PintaOTraco(c.R.Context(), c.SessionID, traco, especie, ligado)
}

// limpaOTerreno é a BORRACHA (ALE-203): o clique devolve a casa ao chão limpo,
// seja qual for o terreno nela.
//
// ROTA PRÓPRIA e não `?apagar=1` na rota de pintar, e a diferença é o que
// conserta o defeito: aquela precisa de uma ESPÉCIE no caminho, e era justamente
// a espécie que fazia a borracha apagar a coisa errada em silêncio. Sem espécie
// no caminho, não há como errar qual.
func limpaOTerreno(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	traco, err := tracoDaURL(c.R)
	if err != nil {
		return nil, err
	}
	if st.boards.Get(c.R.Context(), c.SessionID) == nil {
		return nil, fmt.Errorf("não há tabuleiro aberto para apagar")
	}
	return st.boards.LimpaOTraco(c.R.Context(), c.SessionID, traco)
}

// tracoDaURL lê o segmento que o dedo percorreu e devolve as casas dele.
//
// A RECUSA de um traço grande demais vem do `tabuleiro.TracoValido` e é do
// DOMÍNIO, não do transporte: o que ela protege é o tabuleiro gravado, e a razão
// está escrita lá.
func tracoDaURL(r *http.Request) ([]engine.Square, error) {
	de, err := quadradoDaURL(r)
	if err != nil {
		return nil, err
	}
	ate, err := segundoQuadradoDaURL(r)
	if err != nil {
		return nil, err
	}
	if !tabuleiro.TracoValido(de, ate) {
		return nil, fmt.Errorf("traço de %v até %v é longo demais para um gesto", de, ate)
	}
	return tabuleiro.CasasDoTraco(de, ate), nil
}

// reabreOLugar traz uma cena guardada de volta para a mesa.
//
// `ShowPlace` e não `Reopen`: ele ARQUIVA a cena atual antes de trocar, que é o
// que deixa o mestre pular da taverna para a cripta com a mesa jogando sem
// perder a taverna. E a política dele é o OPOSTO da do encerrar — falhar ao
// guardar RECUSA a troca, porque trocar em cima de um acervo que não gravou é
// justamente perder a cena que se queria guardar.
func reabreOLugar(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	id, err := lugarDaURL(c.R)
	if err != nil {
		return nil, err
	}
	return st.boards.ShowPlace(c.R.Context(), c.CampaignID, c.SessionID, id)
}

// removeOLugar apaga uma cena do acervo, e ela não volta.
//
// Devolve o tabuleiro ATUAL e não nil: apagar um lugar guardado não mexe na cena
// que está na mesa, e devolver nil faria o `comandoDoTabuleiro` publicar "não há
// tabuleiro" para a mesa inteira — o mestre limparia o acervo e a mesa perderia
// a cena em que estava jogando.
func removeOLugar(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	id, err := lugarDaURL(c.R)
	if err != nil {
		return nil, err
	}
	if err := st.boards.RemovePlace(c.R.Context(), c.CampaignID, id); err != nil {
		return nil, err
	}
	return st.boards.Get(c.R.Context(), c.SessionID), nil
}

// lugarDaURL lê o id do CAMINHO, como o quadrado do movimento: o valor é do
// botão que foi clicado, e não de um sinal da página que N linhas disputariam.
func lugarDaURL(r *http.Request) (int64, error) {
	bruto := chi.URLParam(r, "placeId")
	id, err := strconv.ParseInt(bruto, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("lugar %q não é um id", bruto)
	}
	return id, nil
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
	// A LENTE morre com a cena (ALE-193): "você está vendo como a mesa" sobre uma
	// tela sem tabuleiro faria o mestre concluir que o mapa sumiu PARA OS
	// JOGADORES — a resposta errada exatamente à pergunta que a lente existe para
	// responder. Apaga a de todo mundo porque a cena era de todo mundo.
	st.lentes.Apaga(c.SessionID)
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
