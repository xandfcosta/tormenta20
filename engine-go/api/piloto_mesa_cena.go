package api

import (
	"context"
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

func (s *Server) SceneRoutes(r chi.Router) {
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
	// O RETÂNGULO (ALE-203, item 10): os mesmos dois cantos, outra FORMA. Rota
	// própria e não uma query na de cima porque o que muda é o que o par de
	// cantos NOMEIA — a linha entre eles ou tudo o que cabe dentro —, e isso é o
	// significado do pedido, não um modo dele.
	r.Post(base+"/terreno/{especie}/retangulo/{x}/{y}/{x2}/{y2}", s.comandoContinuoDoMestre(enchaORetangulo))
	r.Post(base+"/terreno/limpar/retangulo/{x}/{y}/{x2}/{y2}", s.comandoContinuoDoMestre(limpaORetangulo))
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
	if st.boards.Get(c.R.Context(), c.SessionID, c.TabuleiroID) == nil {
		return nil, fmt.Errorf("não há tabuleiro aberto para pintar")
	}
	especie := tabuleiro.KnownTerrainKind(chi.URLParam(c.R, "especie"))
	ligado := c.R.URL.Query().Get("apagar") == ""
	return st.boards.PaintStroke(c.R.Context(), c.SessionID, c.TabuleiroID, traco, especie, ligado)
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
	if st.boards.Get(c.R.Context(), c.SessionID, c.TabuleiroID) == nil {
		return nil, fmt.Errorf("não há tabuleiro aberto para apagar")
	}
	return st.boards.ClearStroke(c.R.Context(), c.SessionID, c.TabuleiroID, traco)
}

// enchaORetangulo e limpaORetangulo são os irmãos de área dos dois de cima.
//
// Eles chamam as MESMAS gravações (`PintaOTraco`, `LimpaOTraco`) — o nome fala em
// traço porque foi ele que as pediu primeiro, e o que elas recebem sempre foi uma
// lista de casas. Quem escolhe a forma é a rota.
func enchaORetangulo(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	casas, err := oRetanguloDaURL(c.R)
	if err != nil {
		return nil, err
	}
	if st.boards.Get(c.R.Context(), c.SessionID, c.TabuleiroID) == nil {
		return nil, fmt.Errorf("não há tabuleiro aberto para pintar")
	}
	especie := tabuleiro.KnownTerrainKind(chi.URLParam(c.R, "especie"))
	return st.boards.PaintStroke(c.R.Context(), c.SessionID, c.TabuleiroID, casas, especie, true)
}

func limpaORetangulo(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	casas, err := oRetanguloDaURL(c.R)
	if err != nil {
		return nil, err
	}
	if st.boards.Get(c.R.Context(), c.SessionID, c.TabuleiroID) == nil {
		return nil, fmt.Errorf("não há tabuleiro aberto para apagar")
	}
	return st.boards.ClearStroke(c.R.Context(), c.SessionID, c.TabuleiroID, casas)
}

// oRetanguloDaURL lê os dois cantos e devolve as casas de dentro.
//
// A RECUSA vem do domínio (`tabuleiro.RetanguloValido`) e o teto dele é maior que
// o do traço, pela razão escrita lá: o retângulo é um gesto DELIBERADO de dois
// cantos, e o traço é um quadro de 16ms.
func oRetanguloDaURL(r *http.Request) ([]engine.Square, error) {
	de, err := quadradoDaURL(r)
	if err != nil {
		return nil, err
	}
	ate, err := segundoQuadradoDaURL(r)
	if err != nil {
		return nil, err
	}
	if !tabuleiro.ValidRectangle(de, ate) {
		return nil, fmt.Errorf("o retângulo de %v até %v é grande demais para um gesto", de, ate)
	}
	return tabuleiro.RectangleSquares(de, ate), nil
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
	if !tabuleiro.ValidStroke(de, ate) {
		return nil, fmt.Errorf("traço de %v até %v é longo demais para um gesto", de, ate)
	}
	return tabuleiro.StrokeSquares(de, ate), nil
}

// reabreOLugar traz uma cena guardada de volta para a mesa, NUMA ABA NOVA
// (ALE-205, fatia 3).
//
// Era `ShowPlace`, que arquivava a cena atual e entrava no lugar dela — a saída
// que a ALE-191 inventou para o mestre pular da taverna para a cripta sem perder
// a taverna. **Com abas, o problema que ela resolvia deixou de existir**: nada é
// substituído, então não há o que guardar antes, e a taverna continua aberta na
// aba dela. O arquivamento preventivo saiu com a razão dele.
//
// Quem reabre VAI para a aba nova, como quem abre uma cena do zero: ele acabou
// de escolher aquele lugar numa lista, e deixá-lo na cena anterior faria o gesto
// parecer que não aconteceu. A MESA não é levada junto — isso é o "mostrar à
// mesa", que é gesto próprio desde a fatia 2.
func reabreOLugar(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	id, err := lugarDaURL(c.R)
	if err != nil {
		return nil, err
	}
	cena, err := st.boards.OpenPlace(c.R.Context(), c.CampaignID, c.SessionID, id)
	if err != nil {
		return nil, err
	}
	st.abas.Escolhe(c.SessionID, c.User.ID, cena.ID)
	return cena, nil
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
	// APAGAR UM LUGAR QUE ESTÁ NA MESA não é apagar (ALE-205, fatia 3): a linha
	// some do acervo e a cena continua aberta, e no dia em que a aba fechar o
	// `Archive` a grava de novo com o mesmo nome. O mestre veria a taverna que
	// ele apagou ontem reaparecer sozinha — um gesto que não faz o que diz, e que
	// desfaz sozinho o trabalho de quem estava limpando o acervo.
	//
	// A recusa é do SERVIDOR e não da tela: a lista já não oferece a lixeira ao
	// que está aberto, mas quem postar na mão passaria por cima.
	if nome, aba := st.aAbaComOLugar(c.R.Context(), c.CampaignID, c.SessionID, id); aba != "" {
		return nil, fmt.Errorf(
			"%q está aberta numa aba: encerre a cena antes de apagá-la do acervo", nome)
	}
	if err := st.boards.RemovePlace(c.R.Context(), c.CampaignID, id); err != nil {
		return nil, err
	}
	return st.boards.Get(c.R.Context(), c.SessionID, c.TabuleiroID), nil
}

// aAbaComOLugar diz em qual aba um lugar guardado está aberto, e como ele se
// chama. Aba vazia é "não está na mesa".
//
// A JUNÇÃO É PELO NOME, e não por um id do lugar guardado dentro do tabuleiro.
// Não é atalho: **nome é a identidade que este app já dá ao lugar**, porque é
// assim que o `Archive` decide se sobrescreve ou cria — encerrar a taverna duas
// vezes produz UMA taverna. Um `placeId` dentro do `BoardState` seria uma segunda
// identidade, e ela discordaria da primeira exatamente no caso que a mesa faz
// toda semana: abrir "Taverna do Javali" do zero e encerrar por cima da guardada.
//
// A consequência, dita para ninguém a redescobrir: uma cena ABERTA do zero com o
// nome de um lugar guardado é tratada como aquele lugar. É a mesma conta que o
// arquivamento fará quando ela fechar.
func (s *Server) aAbaComOLugar(ctx context.Context, campaignID, sessionID, placeID int64) (nome, tabuleiroID string) {
	for _, lugar := range s.boards.Places(ctx, campaignID) {
		if lugar.ID != placeID {
			continue
		}
		for _, aberto := range s.boards.OpenBoards(ctx, sessionID) {
			if aberto.Place == lugar.Name {
				return lugar.Name, aberto.ID
			}
		}
		return lugar.Name, ""
	}
	return "", ""
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
//
// Desde a ALE-205 ela ACRESCENTA uma aba em vez de substituir a cena que estava
// na mesa: é a issue inteira, e o caso de uso é o grupo que se separou — mostrar
// a cripta não pode custar a taverna. Quem tira cena da mesa continua sendo o
// encerrar, que arquiva.
//
// E QUEM ABRE VAI PARA A ABA NOVA. O mestre digitou o nome do lugar e apertou
// abrir: deixá-lo na cena anterior faria o gesto parecer que não aconteceu —
// ele procuraria na tela uma taverna que nasceu na aba ao lado. É escolha de
// quem clicou e de mais ninguém: a mesa não é puxada, porque a aba padrão
// continua sendo a mais antiga.
func abreOTabuleiro(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	lugar, chao, err := cenaDosSinais(c.R)
	if err != nil {
		return nil, err
	}
	b, err := st.boards.Open(c.R.Context(), c.SessionID, lugar, chao)
	if err != nil {
		return nil, err
	}
	st.abas.Escolhe(c.SessionID, c.User.ID, b.ID)
	// O formulário volta ao zero, como o do combatente: sem isto o lugar fica no
	// campo e a cena seguinte nasce com o nome da anterior.
	c.Sinais["novolugar"] = ""
	c.Sinais["novochao"] = tabuleiro.DefaultGround()
	return b, nil
}

// encerraOTabuleiro arquiva e tira a cena da mesa.
//
// A falha ao ARQUIVAR não impede o encerrar, e a ordem é a do `handleBoardClose`
// de propósito: o mestre mandou tirar a cena da mesa, e recusar isso porque o
// acervo falhou deixaria a mesa presa numa cena que já acabou.
func encerraOTabuleiro(st *Server, c mesaComando) (*tabuleiro.BoardState, error) {
	if atual := st.boards.Get(c.R.Context(), c.SessionID, c.TabuleiroID); atual != nil {
		if err := st.boards.Archive(c.R.Context(), c.CampaignID, atual); err != nil {
			log.Printf("session %d: falha ao arquivar o lugar (%v)", c.SessionID, err)
		}
	}
	st.boards.Close(c.R.Context(), c.SessionID, c.TabuleiroID)
	// AS ESCOLHAS DE ABA morrem com a ÚLTIMA cena, e não com esta (ALE-205).
	//
	// Fechar uma aba com outras abertas não é o fim do tabuleiro: quem estava
	// olhando a que morreu cai na padrão sozinho, porque o `aAbaDe` confere a
	// escolha contra o que existe. Apagar tudo aqui arrastaria de volta para a
	// padrão gente que estava numa aba que continua aberta.
	if len(st.boards.OpenBoards(c.R.Context(), c.SessionID)) == 0 {
		// A LENTE morre com a cena (ALE-193): "você está vendo como a mesa" sobre
		// uma tela sem tabuleiro faria o mestre concluir que o mapa sumiu PARA OS
		// JOGADORES — a resposta errada exatamente à pergunta que a lente existe
		// para responder. Apaga a de todo mundo porque a cena era de todo mundo.
		//
		// Com abas ela sobrevive ao fechamento de UMA, e isso está certo: a lente
		// é sobre "o que a mesa vê", e a mesa continua vendo as outras.
		st.lentes.Apaga(c.SessionID)
		st.abas.Apaga(c.SessionID)
	}
	// `nil` é a mensagem "esta sessão não tem tabuleiro", e ela só é VERDADE
	// quando não sobrou nenhum. Sobrando, quem vai é a aba padrão — ver
	// `publicaOQueSobrou`.
	st.publishWhatIsLeft(c.R.Context(), c.SessionID)
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
	for _, c := range tabuleiro.PlaceGrounds {
		if c.ID == pedido {
			return pedido
		}
	}
	return tabuleiro.DefaultGround()
}
