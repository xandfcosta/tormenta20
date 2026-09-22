package table

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	datastar "github.com/starfederation/datastar-go/datastar"

	"t20engine/domain/board"
	"t20engine/domain/engine"
)

// ABRIR e ENCERRAR a cena na Mesa.
//
// Abrir tabuleiro NÃO inicia combate, e essa ortogonalidade é o que faz a
// taverna existir: a cena de interpretação também tem posição. Quem começa o
// turno é o `next-turn`, e é dele que sai o deslocamento — sem ele o movimento
// roda sem teto, o que é a regra e não um defeito.
//
// Encerrar ARQUIVA: a cena vira um Lugar da campanha com as peças onde estavam.
// Por isso o rótulo é "Encerrar" e não "Fechar" — fechar sugere perder.

func (s Scene) SceneRoutes(r chi.Router) {
	base := sessionPattern + "/tabuleiro"
	r.Post(base+"/abrir", s.gmBoardCommand(openBoard))
	r.Post(base+"/encerrar", s.gmBoardCommand(endBoard))
	r.Post(base+"/lugares/{placeId}/reabrir", s.gmBoardCommand(reopenPlace))
	r.Post(base+"/lugares/{placeId}/remover", s.gmBoardCommand(removeOLugar))
	// O TRAÇO e não o ponto: as duas rotas recebem de ONDE ATÉ ONDE o dedo andou
	// desde o aviso anterior do ponteiro. Um clique parado manda o mesmo par
	// duas vezes, que é um traço de uma casa. Ver `board.StrokeSquares`.
	//
	// AS PONTAS VIAJAM NO CORPO, e não no caminho: a coordenada tem de vir do
	// CLIQUE e não de um sinal da página que outro gesto pode ter mexido, e o
	// `payload` do `@post` já é isso — ele SUBSTITUI os sinais por um corpo
	// calculado no instante do clique. Sinal e corpo não são a mesma coisa.
	r.Post(base+"/terreno", s.gmContinuousCommand(paintTerrain))
	// A BORRACHA tem rota PRÓPRIA porque é a única que não nomeia espécie
	// NENHUMA, nem no caminho nem no corpo — era a espécie que a fazia apagar a
	// coisa errada em silêncio, e um corpo compartilhado com a pintura
	// devolveria o campo, e o risco, de graça.
	r.Post(base+"/terreno/limpar", s.gmContinuousCommand(clearTerrain))
	// O RETÂNGULO: os mesmos dois cantos, outra FORMA. Rota própria e não uma
	// query na de cima porque o que muda é o que o par de cantos NOMEIA — a
	// linha entre eles ou tudo o que cabe dentro —, e isso é o significado do
	// pedido, não um modo dele.
	r.Post(base+"/terreno/retangulo", s.gmContinuousCommand(fillRect))
	r.Post(base+"/terreno/limpar/retangulo", s.gmContinuousCommand(clearRect))
	r.Post(base+"/pecas", s.gmBoardCommand(poeNoMapa))
	r.Post(base+"/pecas/nova", s.gmBoardCommand(newLoosePiece))
}

// paintTerrain liga ou desliga uma espécie numa casa (T20 p238).
//
// A espécie e o quadrado vêm do CAMINHO, e o APAGAR vem da query. A divisão não
// é arbitrária: caminho é o que identifica a casa que o clique acertou — e é a
// mesma escolha do movimento —, enquanto apagar é um MODO da ferramenta, que
// vale para o arraste inteiro e não para um quadrado.
//
// Idempotente de propósito, e o `PaintTerrain` é quem garante: o pincel pinta
// ARRASTANDO e o arraste passa duas vezes pela mesma casa. Alternar faria a casa
// piscar entre brejo e chão limpo debaixo do dedo.
func paintTerrain(st Scene, c commandCtx) (*board.BoardState, error) {
	requested, trait, err := strokeFromBody(c.R)
	if err != nil {
		return nil, err
	}
	species := board.KnownTerrainKind(requested.Kind)
	on := !requested.Erase
	return st.deps.Boards().PaintStroke(c.R.Context(), c.SessionID, c.BoardID, trait, species, on)
}

// clearTerrain é a BORRACHA: o clique devolve a casa ao chão limpo, seja qual
// for o terreno nela.
//
// ROTA PRÓPRIA e não `?apagar=1` na rota de pintar: aquela precisa de uma
// ESPÉCIE, e era justamente a espécie que fazia a borracha apagar a coisa
// errada em silêncio. Sem espécie no pedido, não há como errar qual.
func clearTerrain(st Scene, c commandCtx) (*board.BoardState, error) {
	_, trait, err := strokeFromBody(c.R)
	if err != nil {
		return nil, err
	}
	return st.deps.Boards().ClearStroke(c.R.Context(), c.SessionID, c.BoardID, trait)
}

// fillRect e clearRect são os irmãos de área dos dois de cima.
//
// Eles chamam as MESMAS gravações (`PaintStroke`, `ClearStroke`): o nome fala em
// traço, mas o que elas recebem é uma lista de casas. Quem escolhe a forma é a
// rota.
func fillRect(st Scene, c commandCtx) (*board.BoardState, error) {
	requested, squares, err := rectFromBody(c.R)
	if err != nil {
		return nil, err
	}
	species := board.KnownTerrainKind(requested.Kind)
	return st.deps.Boards().PaintStroke(c.R.Context(), c.SessionID, c.BoardID, squares, species, true)
}

func clearRect(st Scene, c commandCtx) (*board.BoardState, error) {
	_, squares, err := rectFromBody(c.R)
	if err != nil {
		return nil, err
	}
	return st.deps.Boards().ClearStroke(c.R.Context(), c.SessionID, c.BoardID, squares)
}

// strokeBody é o TRAÇO como o cliente o manda: os dois cantos e, na pintura, a
// espécie.
//
// As chaves são INGLESAS porque campo JSON é FRONTEIRA, e só a ROTA saiu dessa
// lista: o endereço é o que uma pessoa vê, o corpo não.
//
// `from` e `to` são objetos e não quatro campos soltos porque o par é UM conceito
// — o segmento que o dedo andou desde o aviso anterior do ponteiro —, e separá-lo
// em `x1,y1,x2,y2` convida a mandar três dos quatro.
type strokeBody struct {
	Kind  string             `json:"kind"`
	Erase bool               `json:"erase"`
	From  struct{ X, Y int } `json:"from"`
	To    struct{ X, Y int } `json:"to"`
	// O GABARITO usa os mesmos dois pontos com outro nome na boca — a origem e
	// a mira — e acrescenta a forma e o tamanho. Um tipo só para o tabuleiro
	// inteiro é um formato só para aprender; um por gesto é como nasce a
	// terceira grafia do mesmo par de números.
	Shape string `json:"shape"`
	Size  string `json:"size"`
}

// pointsFromBody lê os DOIS CANTOS do corpo da requisição.
//
// COORDENADA NEGATIVA é lugar legítimo — o plano não tem bordas —, e é por ela
// que o valor não pode vir de um sinal da página: sinal é estado compartilhado, e
// o que se quer é o clique que ACONTECEU. O corpo resolve isso sem o caminho de
// seis parâmetros, porque o `payload` do `@post` é calculado no instante do
// gesto e não sobrevive a ele.
//
// Ela é o pedaço COMUM do traço e do retângulo. O que muda entre os dois é o que
// o par NOMEIA — a linha entre os cantos ou tudo o que cabe dentro —, e isso é do
// chamador: é o significado do pedido, não um detalhe de leitura.
func pointsFromBody(r *http.Request) (strokeBody, engine.Square, engine.Square, error) {
	var requested strokeBody
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&requested); err != nil {
		return requested, engine.Square{}, engine.Square{}, fmt.Errorf("não entendi o gesto enviado: %v", err)
	}
	return requested,
		engine.Square{X: requested.From.X, Y: requested.From.Y},
		engine.Square{X: requested.To.X, Y: requested.To.Y},
		nil
}

// squareFromBody é UM quadrado, para os gestos que apontam um lugar só — a peça
// avulsa, o marcador, a parada do movimento, a prévia.
//
// Ele reusa o `from` do mesmo corpo em vez de um campo `square` próprio, e isso é
// escolha: um formato só para todos os gestos do tabuleiro é um formato só para
// aprender, e o `to` sobra sem custo. O contrário — um campo por gesto — é como
// nasce a terceira grafia do mesmo par de números.
func squareFromBody(r *http.Request) (strokeBody, engine.Square, error) {
	requested, de, _, err := pointsFromBody(r)
	return requested, de, err
}

// squareOnly é o `squareFromBody` para quem não precisa do resto do pedido.
func squareOnly(r *http.Request) (engine.Square, error) {
	_, square, err := squareFromBody(r)
	return square, err
}

// strokeFromBody é o SEGMENTO entre os dois cantos.
func strokeFromBody(r *http.Request) (strokeBody, []engine.Square, error) {
	requested, de, ate, err := pointsFromBody(r)
	if err != nil {
		return requested, nil, err
	}
	if !board.ValidStroke(de, ate) {
		return requested, nil, fmt.Errorf("traço de %v até %v é longo demais para um gesto", de, ate)
	}
	return requested, board.StrokeSquares(de, ate), nil
}

// rectFromBody é TUDO O QUE CABE entre os dois cantos.
//
// SEM TETO DE ÁREA: o app roda LOCAL, numa mesa, e um teto de mil casas morde
// gesto de verdade — no zoom mínimo o tabuleiro visível passa de mil e novecentas
// casas, e "pinte tudo o que estou vendo" seria recusado. O irmão de cima, o
// TRAÇO, mantém o dele: ele é um quadro de 16ms, e cem casas ali continuam sendo
// impossíveis para um dedo.
func rectFromBody(r *http.Request) (strokeBody, []engine.Square, error) {
	requested, de, ate, err := pointsFromBody(r)
	if err != nil {
		return requested, nil, err
	}
	return requested, board.RectangleSquares(de, ate), nil
}

// reopenPlace traz uma cena guardada de volta para a mesa, NUMA ABA NOVA.
//
// **Sem arquivar a cena atual antes**: nada é substituído, então não há o que
// guardar — a taverna continua aberta na aba dela.
//
// Quem reabre VAI para a aba nova, como quem abre uma cena do zero: ele acabou
// de escolher aquele lugar numa lista, e deixá-lo na cena anterior faria o gesto
// parecer que não aconteceu. A MESA não é levada junto — isso é o "mostrar à
// mesa", que é gesto próprio.
func reopenPlace(st Scene, c commandCtx) (*board.BoardState, error) {
	id, err := lugarDaURL(c.R)
	if err != nil {
		return nil, err
	}
	scene, err := st.deps.Boards().OpenPlace(c.R.Context(), c.CampaignID, c.SessionID, id)
	if err != nil {
		return nil, err
	}
	st.chosenTabs.Escolhe(c.SessionID, c.User, scene.ID)
	return scene, nil
}

// removeOLugar apaga uma cena do acervo, e ela não volta.
//
// Devolve o tabuleiro ATUAL e não nil: apagar um lugar guardado não mexe na cena
// que está na mesa, e devolver nil faria o `boardCommand` publicar "não há
// tabuleiro" para a mesa inteira — o mestre limparia o acervo e a mesa perderia
// a cena em que estava jogando.
func removeOLugar(st Scene, c commandCtx) (*board.BoardState, error) {
	id, err := lugarDaURL(c.R)
	if err != nil {
		return nil, err
	}
	// APAGAR UM LUGAR QUE ESTÁ NA MESA não é apagar: a linha some do acervo e a
	// cena continua aberta, e no dia em que a aba fechar o
	// `Archive` a grava de novo com o mesmo nome. O mestre veria a taverna que
	// ele apagou ontem reaparecer sozinha — um gesto que não faz o que diz, e que
	// desfaz sozinho o trabalho de quem estava limpando o acervo.
	//
	// A recusa é do SERVIDOR e não da tela: a lista já não oferece a lixeira ao
	// que está aberto, mas quem postar na mão passaria por cima.
	name, aba, err := st.placeTab(c.R.Context(), c.CampaignID, c.SessionID, id)
	if err != nil {
		return nil, err
	}
	if aba != "" {
		return nil, fmt.Errorf(
			"%q está aberta numa aba: encerre a cena antes de apagá-la do acervo", name)
	}
	if err := st.deps.Boards().RemovePlace(c.R.Context(), c.CampaignID, id); err != nil {
		return nil, err
	}
	return st.deps.Boards().Get(c.R.Context(), c.SessionID, c.BoardID)
}

// placeTab diz em qual aba um lugar guardado está aberto, e como ele se
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
func (s Scene) placeTab(ctx context.Context, campaignID, sessionID, placeID int64) (name, boardID string, err error) {
	for _, place := range s.deps.Boards().Places(ctx, campaignID) {
		if place.ID != placeID {
			continue
		}
		open, err := s.deps.Boards().OpenBoards(ctx, sessionID)
		if err != nil {
			return "", "", err
		}
		for _, openBoard := range open {
			if openBoard.Place == place.Name {
				return place.Name, openBoard.ID, nil
			}
		}
		return place.Name, "", nil
	}
	return "", "", nil
}

// lugarDaURL lê o id do CAMINHO, como o quadrado do movimento: o valor é do
// botão que foi clicado, e não de um sinal da página que N linhas disputariam.
func lugarDaURL(r *http.Request) (int64, error) {
	raw := chi.URLParam(r, "placeId")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("lugar %q não é um id", raw)
	}
	return id, nil
}

// openBoard monta a cena com o lugar e o chão que o mestre escolheu.
//
// Ela ACRESCENTA uma aba em vez de substituir a cena que estava na mesa: o
// grupo que se separou precisa da cripta sem perder a taverna. Quem tira cena
// da mesa é o encerrar, que arquiva.
//
// E QUEM ABRE VAI PARA A ABA NOVA. O mestre digitou o nome do lugar e apertou
// abrir: deixá-lo na cena anterior faria o gesto parecer que não aconteceu —
// ele procuraria na tela uma taverna que nasceu na aba ao lado. É escolha de
// quem clicou e de mais ninguém: a mesa não é puxada, porque a aba padrão
// continua sendo a mais antiga.
func openBoard(st Scene, c commandCtx) (*board.BoardState, error) {
	place, chao, err := signalsScene(c.R)
	if err != nil {
		return nil, err
	}
	b, err := st.deps.Boards().Open(c.R.Context(), c.SessionID, place, chao)
	if err != nil {
		return nil, err
	}
	st.chosenTabs.Escolhe(c.SessionID, c.User, b.ID)
	// O formulário volta ao zero, como o do combatente: sem isto o lugar fica no
	// campo e a cena seguinte nasce com o nome da anterior.
	c.Signals["new_place"] = ""
	c.Signals["new_ground"] = board.DefaultGround()
	return b, nil
}

// endBoard arquiva e tira a cena da mesa.
//
// A falha ao ARQUIVAR não impede o encerrar: o mestre mandou tirar a cena da
// mesa, e recusar isso porque o acervo falhou deixaria a mesa presa numa cena
// que já acabou.
func endBoard(st Scene, c commandCtx) (*board.BoardState, error) {
	current, err := st.deps.Boards().Get(c.R.Context(), c.SessionID, c.BoardID)
	if err != nil {
		return nil, err
	}
	if current != nil {
		if err := st.deps.Boards().Archive(c.R.Context(), c.CampaignID, current); err != nil {
			log.Printf("session %d: falha ao arquivar o lugar (%v)", c.SessionID, err)
		}
	}
	// O FECHAR QUE FALHA RECUSA (ALE-375): o DELETE mora dentro do gesto, e uma
	// aba que sumisse da tela com a linha viva no banco voltaria no próximo boot.
	if err := st.deps.Boards().Close(c.R.Context(), c.SessionID, c.BoardID); err != nil {
		return nil, err
	}
	// AS ESCOLHAS DE ABA morrem com a ÚLTIMA cena, e não com esta.
	//
	// Fechar uma aba com outras abertas não é o fim do tabuleiro: quem estava
	// olhando a que morreu cai na padrão sozinho, porque o `chosenTabOf` confere a
	// escolha contra o que existe. Apagar tudo aqui arrastaria de volta para a
	// padrão gente que estava numa aba que continua aberta.
	left, err := st.deps.Boards().OpenBoards(c.R.Context(), c.SessionID)
	if err != nil {
		return nil, err
	}
	if len(left) == 0 {
		// A LENTE morre com a cena: "você está vendo como a mesa" sobre uma tela
		// sem tabuleiro faria o mestre concluir que o mapa sumiu PARA OS
		// JOGADORES — a resposta errada exatamente à pergunta que a lente existe
		// para responder. Apaga a de todo mundo porque a cena era de todo mundo.
		//
		// Com abas ela sobrevive ao fechamento de UMA, e isso está certo: a lente
		// é sobre "o que a mesa vê", e a mesa continua vendo as outras.
		st.lenses.Erase(c.SessionID)
		st.chosenTabs.Erase(c.SessionID)
	}
	// `nil` é a mensagem "esta sessão não tem tabuleiro", e ela só é VERDADE
	// quando não sobrou nenhum. Sobrando, quem vai é a aba padrão — ver
	// `PublishWhatIsLeft`.
	st.deps.PublishWhatIsLeft(c.R.Context(), c.SessionID)
	return nil, nil
}

// signalsScene lê o diálogo de abrir.
//
// TODOS OS NOMES SÃO MINÚSCULOS porque são chaves de `data-bind:`, e nome de
// atributo é minusculado pelo analisador de HTML: um `data-bind:` em camelCase
// liga um sinal minúsculo NOVO e deixa o declarado intocado.
//
// Os dois defaults: lugar em branco vira "Cena", porque o mestre que só quer a
// grade não deve ser barrado por um campo; e chão
// desconhecido cai no padrão em vez de recusar, porque um valor que a tela não
// oferece só chega por posse do fio, e a resposta a isso é desenhar pedra e não
// discutir.
func signalsScene(r *http.Request) (place, chao string, err error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	var signals struct {
		Place string `json:"new_place"`
		Chao  string `json:"new_ground"`
	}
	if err := datastar.ReadSignals(r, &signals); err != nil {
		return "", "", fmt.Errorf("não entendi a cena enviada: %v", err)
	}
	place = strings.TrimSpace(signals.Place)
	if place == "" {
		place = "Cena"
	}
	return place, chaoConhecido(signals.Chao), nil
}

// chaoConhecido devolve o chão pedido se ele existe, ou o padrão.
func chaoConhecido(requested string) string {
	for _, c := range board.PlaceGrounds {
		if c.ID == requested {
			return requested
		}
	}
	return board.DefaultGround()
}
