package board

import (
	"errors"
	"fmt"

	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// O MOVIMENTO: quem pode mexer numa peça, quanto ela anda (T20 p106) e o
// provisório que a mesa vê antes de a peça pousar.

// PendingMove é um movimento PROPOSTO e ainda não confirmado.
//
// Ele é ESTADO, e não um evento de arraste no fio. Todo broadcast desta casa
// carrega o estado inteiro; um fantasma a 60fps seriam kilobytes por quadro
// vezes N clientes, e outras tantas gravações do blob por segundo. Como estado,
// são duas mensagens — soltar e confirmar —, o provisório sobrevive à
// reconexão, e ele é redigível por papel como qualquer outra parte do tabuleiro.
//
// No máximo UM por tabuleiro: dois movimentos provisórios simultâneos são duas
// verdades sobre a mesma cena, e a mesa não teria como saber qual confirmar.
type PendingMove struct {
	TokenID string `json:"tokenId"`
	// Path é o caminho INTEIRO, do quadrado onde a peça está até o destino. O
	// custo depende do percurso (diagonal custa o dobro, T20 p238), então
	// guardar só o destino perderia a conta que a mesa acabou de ver.
	Path []engine.Square `json:"path"`
	Cost int             `json:"cost"`
	// Diagonals e Difficult são a CONTA que produziu o custo, não um resumo
	// dela: quantos passos dobraram por serem diagonais e quantos por entrarem
	// em terreno difícil (T20 p238). Viajam para a tela poder NOMEAR a regra
	// em vez de refazer a aritmética em JavaScript.
	Diagonals int `json:"diagonals"`
	Difficult int `json:"difficult"`
	// Budget é o DESLOCAMENTO da peça em quadrados, ou -1 fora de combate.
	//
	// É DESENHO e não teto: o servidor não recusa nada por ele, porque quem põe
	// a peça no lugar é o mestre e o mestre não tem limite. A tela o usa para
	// partir a seta nas três faixas (uma ação de movimento, duas, mais que
	// duas). Por isso ele é o deslocamento da PEÇA e não a permissão de quem
	// arrasta — ver `boardDefaultSpeedSquares`.
	Budget int `json:"budget"`
	// ByUserID é quem propôs. O mestre confirma por qualquer um; o jogador só
	// confirma o que ele mesmo propôs.
	ByUserID int64 `json:"byUserId"`
	// Stops são as casas onde a pessoa CLICOU, na ordem, com a primeira sendo o
	// lugar de onde a peça saiu. O `Path` é o que elas produzem
	// (`PathThroughStops`), e não o contrário.
	//
	// Existe porque o caminho NÃO deixa descobri-las: um trecho legítimo já tem
	// uma dobra (a diagonal vem primeiro), e ela é indistinguível da dobra de uma
	// parada. Sem esta lista, "desfazer a última perna" só poderia ser adivinhado
	// — e o que se adivinha errado aqui é o movimento que a mesa está vendo.
	//
	// NULO é um valor legítimo e quer dizer "não se sabe onde ela parou": é o que
	// o `ProposeMove` deixa quando o caminho chega pronto de fora. Quem propõe por
	// paradas usa o `ProposeMoveWithStops`, e só aí o desfazer de UMA existe.
	Stops []engine.Square `json:"stops,omitempty"`
}

// boardDefaultSpeedSquares é o orçamento de quem não declarou deslocamento: 9m,
// o padrão do livro (T20 p106), que são 6 quadrados. Vale para o capanga que o
// mestre digitou à mão — inventar zero o deixaria pregado no chão, e inventar
// infinito tiraria a regra da mesa.
const boardDefaultSpeedSquares = 6

// speedOf devolve o orçamento da peça em quadrados, com o padrão do livro para
// quem nunca teve o deslocamento medido.
func speedOf(t BoardToken) int {
	if t.SpeedSquares > 0 {
		return t.SpeedSquares
	}
	return boardDefaultSpeedSquares
}

// Mover descreve quem está tentando mexer numa peça. O papel vem do socket e a
// posse do banco; a decisão de deixar ou não é a função abaixo.
type Mover struct {
	UserID int64
	Role   string
	// OwnsCharacter: a peça é de um personagem DESTE usuário. Resolvido no
	// gateway, contra o banco — o cliente não é fonte de posse.
	OwnsCharacter bool
}

// assertMovable responde "esta pessoa pode Mover esta peça agora?" e, quando
// pode, com QUANTO de orçamento (T20 p106; -1 = sem orçamento).
//
// Três regras, e as duas exceções são deliberadas:
//   - o MESTRE move qualquer peça, a qualquer hora, sem orçamento — é a saída
//     para voo, empurrão, teleporte e "pode ir";
//   - FORA DE COMBATE (`turnIndex` < 0) não existe vez nem deslocamento de
//     turno: cada um anda com a própria peça, e o contador só informa;
//   - em combate, o jogador move a própria peça só na vez dela.
func assertMovable(b *BoardState, st *live.SessionRuntimeState, tokenID string, by Mover) (*BoardToken, int, error) {
	token := FindToken(b, tokenID)
	if token == nil {
		return nil, 0, fmt.Errorf("peça %q não está no tabuleiro", tokenID)
	}
	if by.Role == "gm" {
		return token, drawingBudget(*token, st), nil
	}
	if !by.OwnsCharacter {
		return nil, 0, fmt.Errorf("a peça %q não é sua", token.Label)
	}
	if st == nil || st.TurnIndex < 0 {
		return token, -1, nil
	}
	if !isTokenOnTurn(token, st) {
		return nil, 0, fmt.Errorf("não é a vez de %s", token.Label)
	}
	return token, speedOf(*token), nil
}

// drawingBudget é o deslocamento da PEÇA, e não a permissão de quem move.
//
// O mestre NÃO recebe -1 ("sem teto") aqui, apesar de não ter limite: o número
// só serve ao DESENHO, e -1 esconderia da mesa justamente o que ela quer ver.
// Arrastando a peça de um jogador, o mestre vê as mesmas três faixas que o
// jogador vê, medidas contra o deslocamento DAQUELA peça.
//
// FORA DE COMBATE continua -1, e isso não é exceção, é a mesma frase: sem vez
// não há ação padrão para trocar por movimento (p233), então azul e vermelho
// não querem dizer nada e desenhá-los inventaria um teto que a cena não tem.
func drawingBudget(token BoardToken, st *live.SessionRuntimeState) int {
	if st == nil || st.TurnIndex < 0 {
		return -1
	}
	return speedOf(token)
}

// isTokenOnTurn amarra a peça à LINHA da iniciativa: a vez não é copiada para o
// tabuleiro, ela é perguntada ao rastreador — duas cópias da vez divergiriam no
// primeiro turno passado com o tabuleiro fechado.
func isTokenOnTurn(token *BoardToken, st *live.SessionRuntimeState) bool {
	if token.EntryID == nil || st.TurnIndex < 0 || st.TurnIndex >= len(st.Initiative) {
		return false
	}
	return st.Initiative[st.TurnIndex].ID == *token.EntryID
}

// ProposeMove mede o caminho e guarda o DESENHO dele.
//
// Não recusa por deslocamento — ver o corpo. Quem pode desenhar é quem
// `assertMovable` deixa: o mestre em qualquer peça, e o jogador na peça dele, na
// vez dele. Quem transforma desenho em pouso é o `CommitMove`, e só o mestre.
func ProposeMove(b *BoardState, st *live.SessionRuntimeState, tokenID string, path []engine.Square, by Mover) error {
	token, budget, err := assertMovable(b, st, tokenID, by)
	if err != nil {
		return err
	}
	if len(path) < 2 {
		return fmt.Errorf("o caminho tem %d quadrados: precisa de origem e destino", len(path))
	}
	if path[0].X != token.X || path[0].Y != token.Y {
		return fmt.Errorf("o caminho começa em (%d,%d) e a peça está em (%d,%d)", path[0].X, path[0].Y, token.X, token.Y)
	}
	cost := engine.PathCost(path, moveTerrainOf(b), budget)
	// O DESLOCAMENTO não recusa nada, nem aqui nem no `CommitMove`.
	//
	// Propor é DESENHAR, e o desenho é o produto: o do jogador é sempre visual —
	// ele conta à mesa o que ele quer fazer — e o do mestre vira pouso quando ele
	// confirma. Nas duas leituras, um caminho caro é uma coisa legítima de se
	// querer desenhar, e é justamente o caro que a tela precisa mostrar em azul
	// ("gasta a ação principal") e em vermelho ("não cabe nem em duas").
	//
	// Um caminho MALFORMADO continua recusado, e é por isso que o `Malformed`
	// nasceu separado do `Legal`, que é falso pelas duas razões: passo inválido
	// não é uma proposta cara, é uma proposta que não existe.
	if cost.Malformed {
		return fmt.Errorf("%s", cost.Reason)
	}
	if err := AssertSaneCoords(BoardToken{X: path[len(path)-1].X, Y: path[len(path)-1].Y}); err != nil {
		return err
	}
	b.Pending = &PendingMove{
		TokenID: tokenID, Path: path, Cost: cost.Squares, Budget: budget,
		Diagonals: cost.Diagonals, Difficult: cost.Difficult, ByUserID: by.UserID,
	}
	b.Version++
	return nil
}

// ProposeMoveWithStops propõe pelas casas em que a pessoa CLICOU, e guarda a
// lista junto.
//
// A primeira parada é onde a peça está; cada uma seguinte estende o caminho,
// contornando o que quem move quiser. É a forma que o app usa, e é ela que
// torna "desfazer a última perna" uma operação exata em vez de um palpite: o
// caminho se reconstrói pelas paradas que sobraram, e reconstruir é o que o
// `PathThroughStops` já faz de graça.
//
// A validação inteira continua sendo a do `ProposeMove` — o orçamento, a vez, a
// posse, a contiguidade. Esta função não afrouxa nada; ela só LEMBRA de onde o
// caminho veio.
func ProposeMoveWithStops(b *BoardState, st *live.SessionRuntimeState, tokenID string, stops []engine.Square, by Mover) error {
	if err := ProposeMove(b, st, tokenID, engine.PathThroughStops(stops), by); err != nil {
		return err
	}
	b.Pending.Stops = stops
	return nil
}

// CommitMove pousa a peça no fim do caminho proposto.
//
// `version` é a versão que o proponente tinha na mão: se o tabuleiro mudou
// desde a proposta, o commit é RECUSADO em vez de aplicado sobre outra cena.
// Isso mata os três casos que o last-write-wins quebra — dois clientes na mesma
// peça, o mestre arrastando enquanto o jogador confirma, e o broadcast atrasado
// que chega depois da re-hidratação. Versão 0 = "não sei em que versão eu
// estava", aceita, porque recusar um cliente honesto e desatualizado seria
// pior que aplicar o que ele acabou de ver na tela.
func CommitMove(b *BoardState, st *live.SessionRuntimeState, version int64, by Mover) error {
	pending, err := pendingFor(b, by)
	if err != nil {
		return err
	}
	if version > 0 && version != b.Version {
		return fmt.Errorf("o tabuleiro mudou (versão %d, você viu a %d): refaça o movimento", b.Version, version)
	}
	// QUEM PÕE A PEÇA NO LUGAR É O MESTRE, e só ele.
	//
	// O desenho do jogador é SEMPRE só visual: ele serve para a mesa entender o
	// que ele quer fazer, e quem decide se aconteceu é quem toca a cena. É a
	// MESMA divisa do resto da Mesa (`gmCommand`), e a razão de a trava de
	// deslocamento poder não existir logo abaixo.
	if by.Role != "gm" {
		return errors.New("só o mestre põe a peça no lugar: o seu movimento é um rascunho para a mesa ver")
	}
	// A vez é conferida DE NOVO na confirmação: entre propor e confirmar o
	// mestre pode ter passado o turno, e o que vale é a mesa no instante em que
	// a peça pousa. O mestre confirma por qualquer um, então ele passa por aqui
	// sem restrição.
	token, _, err := assertMovable(b, st, pending.TokenID, by)
	if err != nil {
		return err
	}
	// NÃO HÁ TRAVA DE DESLOCAMENTO AQUI, e a ausência é deliberada.
	//
	// Com o confirmar sendo só do mestre, o guarda não tem objeto: o mestre não
	// tem limite, ele faz o que quiser no tabuleiro. O deslocamento é DESENHO —
	// as três faixas da seta (ouro, azul, vermelho) contam à mesa quantas ações
	// aquele caminho custa, e é a mesa que decide com essa informação na tela.
	// Regra que ninguém aplica vira teatro; informação que todos veem vira
	// conversa.
	destination := pending.Path[len(pending.Path)-1]
	// DE ONDE ELA VEIO fica gravado ANTES de a peça pousar: é o que faz o
	// "voltar para onde estava" existir depois, e é aqui que a informação existe
	// pela última vez.
	//
	// No CONFIRMAR e não no propor, porque o provisório não moveu ninguém: a
	// peça só sai do lugar aqui, e gravar antes daria um "voltar" para um
	// movimento que foi cancelado.
	token.CameFrom = &engine.Square{X: token.X, Y: token.Y}
	token.X, token.Y = destination.X, destination.Y
	b.Pending = nil
	b.Version++
	return nil
}

// ReturnToken põe a peça de volta onde ela estava antes do último pouso.
//
// LIMPA o registro ao usar, e por isso o gesto só existe uma vez por movimento: um
// "voltar" que continuasse disponível andaria para trás na cena com um botão que
// não diz até onde vai. Quem quiser desfazer duas vezes desfaz, move de novo, e
// desfaz de novo — cada passo com a mesa vendo.
//
// Não confere a VEZ nem a posse, ao contrário do movimento: quem chama é o mestre
// arrumando a cena (a rota é `gmBoardCommand`), e a peça já está onde
// ele a pôs. Recusar por "não é a vez de Arwen" impediria justamente o conserto.
func ReturnToken(b *BoardState, tokenID string) error {
	token := FindToken(b, tokenID)
	if token == nil {
		return fmt.Errorf("peça %q não está no tabuleiro", tokenID)
	}
	if token.CameFrom == nil {
		return fmt.Errorf("%s não foi movida nesta cena: não há para onde voltar", token.Label)
	}
	token.X, token.Y = token.CameFrom.X, token.CameFrom.Y
	token.CameFrom = nil
	b.Version++
	return nil
}

// CancelMove desfaz o provisório sem mexer na peça.
func CancelMove(b *BoardState, by Mover) error {
	if _, err := pendingFor(b, by); err != nil {
		return err
	}
	b.Pending = nil
	b.Version++
	return nil
}

// pendingFor devolve o provisório se esta pessoa pode decidir sobre ele. O
// mestre decide por qualquer um — é ele quem toca a mesa quando o jogador
// travou ou caiu da rede —, e o jogador só sobre o que ele mesmo propôs.
func pendingFor(b *BoardState, by Mover) (*PendingMove, error) {
	if b.Pending == nil {
		return nil, fmt.Errorf("não há movimento proposto para confirmar")
	}
	if by.Role != "gm" && b.Pending.ByUserID != by.UserID {
		return nil, fmt.Errorf("o movimento proposto não é seu")
	}
	return b.Pending, nil
}

// CanMove responde "esta pessoa pode mover esta peça agora?" para a TELA.
//
// Envelope fino sobre o `assertMovable`, e a razão de existir é que a tela
// precisa da MESMA resposta que a escrita — perguntar de outro jeito é como
// nasce um botão que existe e o servidor recusa, ou uma casa clicável que leva
// a "não é a vez de Arwen" depois do clique.
//
// Não devolve o porquê: quem só desenha não tem o que fazer com a frase, e a
// frase certa é a que a RECUSA escreve, no instante em que ela acontece.
func CanMove(b *BoardState, st *live.SessionRuntimeState, tokenID string, by Mover) bool {
	can, _ := CanMoveWith(b, st, tokenID, by)
	return can
}

// CanMoveWith devolve também o ORÇAMENTO, que é o que a tela precisa para
// desenhar até onde dá para ir (-1 = sem teto).
func CanMoveWith(b *BoardState, st *live.SessionRuntimeState, tokenID string, by Mover) (bool, int) {
	if b == nil {
		return false, 0
	}
	_, budget, err := assertMovable(b, st, tokenID, by)
	return err == nil, budget
}
