package tabuleiro

import "t20engine/aovivo"

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"t20engine/plataforma"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/events"
)

// errNoBoard é a resposta a "mexa no tabuleiro" quando a sessão não tem um. É
// erro e não um tabuleiro vazio de cortesia: vazio desenharia uma grade de 0×0
// na tela, e o mestre acharia que abriu.
var errNoBoard = errors.New("esta sessão não tem tabuleiro aberto")

// tetoDeAbertos — quantos tabuleiros uma sessão pode ter abertos ao mesmo tempo
// (ALE-205, decisão do dono).
//
// Existe pelo mesmo motivo do `boardMaxTokens`: sem teto o estado cresce sem
// limite e TODA hidratação e TODA gravação o carregam. Oito frentes é uma cena
// que nenhuma mesa joga; o grupo que se separa costuma virar duas ou três.
const tetoDeAbertos = 8

// BoardStore guarda os tabuleiros vivos de cada sessão em memória, com lastro na
// tabela open_boards.
//
// Mutex PRÓPRIO, separado do `sessionStore`: lá o mutex é global a todas as
// sessões, e um tabuleiro movimentado numa mesa serializaria a edição de PV de
// outra mesa. Aqui a mesma trava vale para todos os tabuleiros — quando o custo
// aparecer, ela vira uma por sessão sem mudar quem chama (ALE-124).
type BoardStore struct {
	Mu sync.Mutex
	// boards é uma LISTA por sessão, e a ordem dela é a de abertura (ALE-205):
	// ela é a ordem das abas na tela, e a primeira é a aba PADRÃO de quem ainda
	// não escolheu. Um mapa por id perderia a ordem e faria a barra de abas
	// mudar de forma a cada carga.
	boards map[int64][]*BoardState
	// loaded marca a sessão já consultada no banco, para "sem tabuleiro" não
	// virar uma ida ao disco por mensagem.
	loaded map[int64]bool
	// Dirty: a última gravação falhou. Espelha o do rastreador, e é o que
	// transforma "gravação falhando em silêncio" em aviso na tela da mesa.
	//
	// Continua por SESSÃO e não por tabuleiro, porque o aviso é da mesa: "o
	// tabuleiro parou de gravar" é a frase que o mestre precisa ler, e uma
	// tarja por aba faria ele conferir oito lugares para saber se o disco está
	// vivo.
	Dirty map[int64]bool
	// bus é por onde as mudanças deste tabuleiro viram notícia (ALE-279).
	//
	// Aqui morava `ouvintes map[int64][]chan struct{}`, num arquivo `aviso.go`
	// que espelhava o do `aovivo` linha por linha. O comentário dele explicava
	// por que eram DOIS registros e não um compartilhado: os dois stores têm
	// travas próprias, e chamar o aviso do outro pacote de dentro da trava daqui
	// é como se escreve um abraço mortal.
	//
	// O barramento não tem esse risco porque é FOLHA — pega só a própria trava e
	// nunca chama de volta um store (ver `events.Bus.Publish`) —, e mesmo assim
	// a publicação sai de fora da trava.
	bus   *events.Bus
	newID func() string
	q     *sqlcgen.Queries
}

func NewBoardStore(q *sqlcgen.Queries, newID func() string, bus *events.Bus) *BoardStore {
	return &BoardStore{
		bus:    bus,
		boards: map[int64][]*BoardState{},
		loaded: map[int64]bool{},
		Dirty:  map[int64]bool{},
		newID:  newID,
		q:      q,
	}
}

// cloneBoard copia o tabuleiro para o broadcast. As peças são valores; a cópia
// da fatia é o que impede uma mensagem concorrente de mexer no que já está
// sendo serializado.
func cloneBoard(b *BoardState) *BoardState {
	if b == nil {
		return nil
	}
	out := *b
	out.Tokens = make([]BoardToken, len(b.Tokens))
	copy(out.Tokens, b.Tokens)
	// Os marcadores são valores, mas a FATIA é compartilhada: sem a cópia, uma
	// mensagem concorrente mexeria no que já está sendo serializado (ALE-132).
	out.Markers = make([]BoardMarker, len(b.Markers))
	copy(out.Markers, b.Markers)
	// O provisório é PONTEIRO, e uma cópia rasa deixaria o instantâneo do
	// broadcast apontando para o mesmo movimento que a mensagem seguinte
	// substitui. É a família de defeito da `cloneState` (ALE-132): o que sai
	// no fio tem de ser cópia de verdade.
	if b.Pending != nil {
		pending := *b.Pending
		pending.Path = append([]engine.Square(nil), b.Pending.Path...)
		out.Pending = &pending
	}
	return &out
}

// Get devolve UM tabuleiro da sessão (nil quando não há), hidratando do banco na
// primeira leitura.
//
// `tabuleiroID` vazio significa "a aba padrão", e não "erro": quem entra na
// sessão pela primeira vez ainda não escolheu aba nenhuma, e o que ele vê é o
// primeiro tabuleiro aberto. Um id que não existe devolve NIL em vez de cair no
// padrão — a aba que o mestre fechou tem de sumir da tela de quem estava nela,
// e não virar outra cena em silêncio.
func (bs *BoardStore) Get(ctx context.Context, sessionID int64, tabuleiroID string) *BoardState {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	bs.hydrateLocked(ctx, sessionID)
	return cloneBoard(bs.achaLocked(sessionID, tabuleiroID))
}

// Abertos devolve os tabuleiros da sessão na ordem de abertura — é o que a
// barra de abas desenha.
//
// Cópias, como o `Get`: quem recebe a lista a redige por papel e a serializa,
// e devolver os ponteiros vivos deixaria o `BoardForRole` do chamador
// escrevendo no estado da mesa.
func (bs *BoardStore) Abertos(ctx context.Context, sessionID int64) []*BoardState {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	bs.hydrateLocked(ctx, sessionID)
	abertos := make([]*BoardState, 0, len(bs.boards[sessionID]))
	for _, b := range bs.boards[sessionID] {
		abertos = append(abertos, cloneBoard(b))
	}
	return abertos
}

// APadrao é o id da aba de quem ainda não escolheu: a mais antiga.
//
// A MAIS ANTIGA e não a última aberta, e a diferença aparece na cena que esta
// issue existe para servir: o mestre abre a taverna sob cortina enquanto a mesa
// olha a cripta, e por "a última" a mesa inteira seria puxada para uma cortina
// sem ninguém pedir. Quem move a mesa de propósito é o FORÇAR, que é gesto.
func (bs *BoardStore) APadrao(ctx context.Context, sessionID int64) string {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	bs.hydrateLocked(ctx, sessionID)
	if abertos := bs.boards[sessionID]; len(abertos) > 0 {
		return abertos[0].ID
	}
	return ""
}

// proximaSeqLocked é o número da PRÓXIMA aba desta sessão.
//
// `max + 1` e não `len + 1`: fechar a aba do meio deixaria dois tabuleiros com
// o mesmo número, e dois números iguais é o empate que esta coluna existe para
// não ter. Buraco na sequência não custa nada — ela só serve para ordenar.
func (bs *BoardStore) proximaSeqLocked(sessionID int64) int64 {
	var maior int64
	for _, b := range bs.boards[sessionID] {
		if b.Seq > maior {
			maior = b.Seq
		}
	}
	return maior + 1
}

// achaLocked resolve o id na lista da sessão, com a trava já na mão.
func (bs *BoardStore) achaLocked(sessionID int64, tabuleiroID string) *BoardState {
	abertos := bs.boards[sessionID]
	if len(abertos) == 0 {
		return nil
	}
	if tabuleiroID == "" {
		return abertos[0]
	}
	for _, b := range abertos {
		if b.ID == tabuleiroID {
			return b
		}
	}
	return nil
}

// hydrateLocked traz os tabuleiros do banco na primeira leitura da sessão.
//
// O `loaded` só é marcado no SUCESSO, e essa ordem é a correção da ALE-155.
// Marcando antes da query, um erro transiente de banco na primeira leitura
// ficava cacheado como "esta sessão não tem tabuleiro" **até o processo
// reiniciar** — a falha de LEITURA com o mesmo modo silencioso que a de
// gravação tinha antes da ALE-124.
//
// A lista VAZIA é a exceção deliberada: "sessão sem tabuleiro" é uma resposta
// legítima e definitiva, então ela MARCA e evita uma ida ao disco por mensagem.
// Qualquer outro erro deixa a sessão sem marca, e a mensagem seguinte tenta de
// novo. (Era `sql.ErrNoRows` enquanto a consulta devolvia uma linha só; com uma
// consulta de várias, ausência chega como fatia vazia e não como erro.)
func (bs *BoardStore) hydrateLocked(ctx context.Context, sessionID int64) {
	if bs.loaded[sessionID] {
		return
	}
	rows, err := bs.q.ListOpenBoards(ctx, sessionID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		// Sem marcar: a próxima mensagem tenta de novo em vez de servir um
		// "sem tabuleiro" que só existe porque o banco piscou.
		log.Printf("session %d: board Load failed (%v); tentará de novo", sessionID, err)
		return
	}
	bs.loaded[sessionID] = true
	abertos := make([]*BoardState, 0, len(rows))
	for _, row := range rows {
		var parsed BoardState
		if err := json.Unmarshal([]byte(row.State), &parsed); err != nil {
			log.Printf("session %d: board %s blob malformed (%v); tratando como sem tabuleiro",
				sessionID, row.Boardid, err)
			continue
		}
		if parsed.Tokens == nil {
			parsed.Tokens = []BoardToken{}
		}
		// O ID e a SEQUÊNCIA vêm da COLUNA e não do JSON, pela mesma razão do
		// nome do lugar no `Reopen`: duas verdades sobre quem é este tabuleiro é
		// como elas divergem, e a de fora é a que o upsert usa.
		parsed.ID = row.Boardid
		parsed.Seq = row.Openseq
		abertos = append(abertos, &parsed)
	}
	if len(abertos) > 0 {
		bs.boards[sessionID] = abertos
	}
}

// Open abre MAIS UM tabuleiro na sessão e devolve o que nasceu (ALE-205).
//
// Ele ACRESCENTA em vez de substituir, que é a issue inteira: o grupo se separou
// e a cripta não pode custar a taverna. Quem tira uma cena da mesa é o `Close`,
// que arquiva; abrir nunca destrói.
//
// A versão nasce em 1 e não continua a de ninguém. Ela era continuada quando
// abrir SUBSTITUÍA — um cliente com o tabuleiro velho na mão precisava
// reconhecer o novo como mais recente. Agora são dois tabuleiros diferentes, com
// dois contadores, e continuar um no outro faria o número mentir sobre quantas
// vezes ESTA cena mudou.
func (bs *BoardStore) Open(ctx context.Context, sessionID int64, place, terrain string) (*BoardState, error) {
	b, err := bs.openLocked(ctx, sessionID, place, terrain)
	if err != nil {
		return nil, err
	}
	bs.bus.Publish(events.BoardOpened{SessionID: sessionID})
	return b, nil
}

func (bs *BoardStore) openLocked(ctx context.Context, sessionID int64, place, terrain string) (*BoardState, error) {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	bs.hydrateLocked(ctx, sessionID)
	if len(bs.boards[sessionID]) >= tetoDeAbertos {
		return nil, fmt.Errorf(
			"esta sessão já tem %d tabuleiros abertos (teto %d): feche um antes de abrir outro",
			len(bs.boards[sessionID]), tetoDeAbertos)
	}
	b := newBoard(bs.newID(), place, terrain)
	b.Seq = bs.proximaSeqLocked(sessionID)
	bs.boards[sessionID] = append(bs.boards[sessionID], b)
	return cloneBoard(b), nil
}

// Close encerra UM tabuleiro: some da memória e do banco. As posições não são
// arquivadas AQUI — quem arquiva é o gateway, antes de chamar (ALE-124).
//
// Devolve as transições de saúde como o `Persist`, e pelo mesmo motivo
// (ALE-155): se o DELETE falha, a memória diz "fechado" e o banco mantém a
// linha — no próximo boot o tabuleiro FANTASMA volta, com as peças de uma cena
// que a mesa já encerrou. Antes isso morria numa linha de log.
func (bs *BoardStore) Close(ctx context.Context, sessionID int64, tabuleiroID string) (Dirty, changed bool) {
	bs.Mu.Lock()
	alvo := bs.achaLocked(sessionID, tabuleiroID)
	if alvo == nil {
		bs.Mu.Unlock()
		return bs.Dirty[sessionID], false
	}
	fechado := alvo.ID
	restantes := make([]*BoardState, 0, len(bs.boards[sessionID]))
	for _, b := range bs.boards[sessionID] {
		if b.ID != fechado {
			restantes = append(restantes, b)
		}
	}
	if len(restantes) == 0 {
		delete(bs.boards, sessionID)
	} else {
		bs.boards[sessionID] = restantes
	}
	bs.loaded[sessionID] = true
	bs.Mu.Unlock()
	bs.bus.Publish(events.BoardClosed{SessionID: sessionID})

	err := bs.q.DeleteOpenBoard(ctx, sqlcgen.DeleteOpenBoardParams{
		Sessionid: sessionID, Boardid: fechado,
	})

	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	prev := bs.Dirty[sessionID]
	Dirty = err != nil
	changed = prev != Dirty
	if Dirty {
		bs.Dirty[sessionID] = true
		log.Printf("session %d: board delete failed (%v)", sessionID, err)
		return Dirty, changed
	}
	delete(bs.Dirty, sessionID)
	return Dirty, changed
}

// apply roda uma mutação pura sobre UM tabuleiro, sob a trava, e devolve o
// instantâneo para o broadcast. Recusa quando aquele tabuleiro não existe: mexer
// no que não existe é erro de quem chamou, não um tabuleiro criado por acidente.
func (bs *BoardStore) apply(
	ctx context.Context, sessionID int64, tabuleiroID string, fn func(*BoardState) error,
) (*BoardState, error) {
	b, err := bs.applyLocked(ctx, sessionID, tabuleiroID, fn)
	if err != nil {
		return nil, err
	}
	bs.bus.Publish(events.BoardChanged{SessionID: sessionID})
	return b, nil
}

func (bs *BoardStore) applyLocked(
	ctx context.Context, sessionID int64, tabuleiroID string, fn func(*BoardState) error,
) (*BoardState, error) {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	bs.hydrateLocked(ctx, sessionID)
	b := bs.achaLocked(sessionID, tabuleiroID)
	if b == nil {
		return nil, errNoBoard
	}
	if err := fn(b); err != nil {
		return nil, err
	}
	return cloneBoard(b), nil
}

// AddToken põe a peça no tabuleiro. Sem posição declarada, ela nasce no
// primeiro quadrado livre da fileira de entrada — senão duas peças criadas
// seguidas ficariam uma em cima da outra (ALE-166).
func (bs *BoardStore) AddToken(ctx context.Context, sessionID int64, tabuleiroID string, t BoardToken, temPosicao bool) (*BoardState, error) {
	return bs.apply(ctx, sessionID, tabuleiroID, func(b *BoardState) error {
		if !temPosicao {
			spot := nextFreeSpot(b)
			t.X, t.Y = spot.x, spot.y
		}
		return AddToken(b, t, bs.newID)
	})
}

func (bs *BoardStore) RemoveToken(ctx context.Context, sessionID int64, tabuleiroID, tokenID string) (*BoardState, error) {
	return bs.apply(ctx, sessionID, tabuleiroID, func(b *BoardState) error { RemoveToken(b, tokenID); return nil })
}

// DuplicateToken põe outra igual ao lado, numerada pelo SERVIDOR: dois clientes
// duplicando ao mesmo tempo não podem inventar o mesmo "Zumbi 3" (ALE-192).
func (bs *BoardStore) DuplicateToken(ctx context.Context, sessionID int64, tabuleiroID, tokenID string) (*BoardState, error) {
	return bs.apply(ctx, sessionID, tabuleiroID, func(b *BoardState) error {
		return DuplicateToken(b, tokenID, bs.newID)
	})
}

func (bs *BoardStore) UpdateToken(ctx context.Context, sessionID int64, tabuleiroID, tokenID string, patch tokenPatch) (*BoardState, error) {
	return bs.apply(ctx, sessionID, tabuleiroID, func(b *BoardState) error { return UpdateToken(b, tokenID, patch) })
}

// LimpaACasa é a BORRACHA (ALE-203): tira todo terreno de um quadrado.
func (bs *BoardStore) LimpaACasa(ctx context.Context, sessionID int64, tabuleiroID string, square engine.Square) (*BoardState, error) {
	return bs.LimpaOTraco(ctx, sessionID, tabuleiroID, []engine.Square{square})
}

// LimpaOTraco apaga o segmento inteiro numa gravação só.
//
// UMA transação para o traço e não uma por casa, e a diferença não é desempenho:
// `apply` sobe a versão do tabuleiro e publica para a mesa. Uma gravação por casa
// faria a mesa receber dez quadros para um gesto só, e cada um deles com metade
// do traço desenhada.
func (bs *BoardStore) LimpaOTraco(ctx context.Context, sessionID int64, tabuleiroID string, traco []engine.Square) (*BoardState, error) {
	return bs.apply(ctx, sessionID, tabuleiroID, func(b *BoardState) error {
		for _, casa := range traco {
			LimpaACasa(b, casa)
		}
		return nil
	})
}

// VoltaAPeca desfaz o último pouso (ALE-206).
func (bs *BoardStore) VoltaAPeca(ctx context.Context, sessionID int64, tabuleiroID, tokenID string) (*BoardState, error) {
	return bs.apply(ctx, sessionID, tabuleiroID, func(b *BoardState) error { return VoltaAPeca(b, tokenID) })
}

// Marcadores (ALE-195): o lugar apontado no mapa que não é peça.
func (bs *BoardStore) AddMarker(ctx context.Context, sessionID int64, tabuleiroID string, m BoardMarker) (*BoardState, error) {
	return bs.apply(ctx, sessionID, tabuleiroID, func(b *BoardState) error { return AddMarker(b, m, bs.newID) })
}

func (bs *BoardStore) UpdateMarker(ctx context.Context, sessionID int64, tabuleiroID, markerID string, patch markerPatch) (*BoardState, error) {
	return bs.apply(ctx, sessionID, tabuleiroID, func(b *BoardState) error { return UpdateMarker(b, markerID, patch) })
}

func (bs *BoardStore) RemoveMarker(ctx context.Context, sessionID int64, tabuleiroID, markerID string) (*BoardState, error) {
	return bs.apply(ctx, sessionID, tabuleiroID, func(b *BoardState) error { RemoveMarker(b, markerID); return nil })
}

func (bs *BoardStore) PaintTerrain(
	ctx context.Context, sessionID int64, tabuleiroID string, square engine.Square, especie EspecieDeTerreno, ligado bool,
) (*BoardState, error) {
	return bs.PintaOTraco(ctx, sessionID, tabuleiroID, []engine.Square{square}, especie, ligado)
}

// PintaOTraco pinta o segmento inteiro numa gravação só — ver `LimpaOTraco`.
func (bs *BoardStore) PintaOTraco(
	ctx context.Context, sessionID int64, tabuleiroID string, traco []engine.Square, especie EspecieDeTerreno, ligado bool,
) (*BoardState, error) {
	return bs.apply(ctx, sessionID, tabuleiroID, func(b *BoardState) error {
		for _, casa := range traco {
			PaintTerrain(b, casa, especie, ligado)
		}
		return nil
	})
}

func (bs *BoardStore) Populate(
	ctx context.Context, sessionID int64, tabuleiroID string, st *aovivo.SessionRuntimeState, chosen EntrySelection,
) (*BoardState, error) {
	return bs.apply(ctx, sessionID, tabuleiroID, func(b *BoardState) error {
		populateBoard(b, st, bs.newID, chosen)
		return nil
	})
}

// SetSpeeds grava o orçamento de várias peças de uma vez. Uma mutação só, e um
// broadcast só: um `UpdateToken` por peça faria a mesa receber seis tabuleiros
// seguidos ao trazer o grupo.
func (bs *BoardStore) SetSpeeds(ctx context.Context, sessionID int64, tabuleiroID string, speeds map[string]int) (*BoardState, error) {
	return bs.apply(ctx, sessionID, tabuleiroID, func(b *BoardState) error {
		for i := range b.Tokens {
			if squares, ok := speeds[b.Tokens[i].ID]; ok && squares > 0 {
				b.Tokens[i].SpeedSquares = squares
			}
		}
		b.Version++
		return nil
	})
}

// Persist grava UM tabuleiro e devolve as transições de saúde, como o do
// rastreador: a mesa não para porque o disco piscou, mas ela precisa SABER
// quando parou de gravar.
//
// UM e não todos os abertos da sessão, e a razão é o tamanho: com oito abas,
// gravar todas a cada peça que anda seria oito serializações e oito upserts por
// gesto — a mesma multiplicação que fez a resposta do pincel sair de 353 KB
// (ALE-203). Quem chama sabe qual aba mudou porque acabou de mutá-la.
//
// A saúde continua sendo da SESSÃO: ver o campo `Dirty`.
//
// Esta devolução nasceu de um defeito real: a tabela do tabuleiro sumiu do banco
// de desenvolvimento (a migração constava aplicada e a tabela não existia), e o
// tabuleiro passou um dia vivendo só em memória — cada gravação falhava numa
// linha de log que ninguém lê, e na tela estava tudo perfeito até o processo
// reiniciar. Falha permanente de gravação não é "o disco piscou".
func (bs *BoardStore) Persist(ctx context.Context, sessionID int64, tabuleiroID string) (Dirty, changed bool) {
	bs.Mu.Lock()
	b := cloneBoard(bs.achaLocked(sessionID, tabuleiroID))
	bs.Mu.Unlock()
	if b == nil {
		return false, false
	}
	blob, err := json.Marshal(b)
	if err != nil {
		log.Printf("session %d: board marshal failed (%v)", sessionID, err)
		return false, false
	}
	// `openSeq` só entra no INSERT — o upsert não o toca (ver a query). Gravar o
	// tabuleiro é dizer que ele mudou, nunca que ele nasceu de novo, e a ordem
	// das abas na tela sai daquela coluna.
	err = bs.q.SaveOpenBoard(ctx, sqlcgen.SaveOpenBoardParams{
		Sessionid: sessionID, Boardid: b.ID, State: string(blob),
		Openseq: b.Seq, Updatedat: plataforma.NowISO(),
	})

	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	prev := bs.Dirty[sessionID] // ausente ⇒ false (saudável)
	Dirty = err != nil
	changed = prev != Dirty
	if Dirty {
		bs.Dirty[sessionID] = true
		log.Printf("session %d: board %s Persist failed (%v)", sessionID, b.ID, err)
		return Dirty, changed
	}
	delete(bs.Dirty, sessionID)
	return Dirty, changed
}

// ProposeMove, CommitMove e CancelMove são as três portas do movimento (ALE-124).
// A posse e o orçamento chegam RESOLVIDOS do gateway: quem consulta o banco é
// ele, e a trava daqui não pode esperar por I/O.

func (bs *BoardStore) ProposeMove(ctx context.Context, sessionID int64, tabuleiroID string, st *aovivo.SessionRuntimeState, tokenID string, path []engine.Square, by Mover, speedSquares int) (*BoardState, error) {
	return bs.apply(ctx, sessionID, tabuleiroID, func(b *BoardState) error {
		// O orçamento fresco do motor entra na peça ANTES da medição: sem isso,
		// a armadura vestida no meio da sessão só valeria no movimento seguinte.
		if token := FindToken(b, tokenID); token != nil && speedSquares > 0 {
			token.SpeedSquares = speedSquares
		}
		return ProposeMove(b, st, tokenID, path, by)
	})
}

// ProposeMoveComParadas é a porta de quem monta o movimento CLICANDO, e ela
// existe para a lista de paradas ser guardada junto (ALE-269, item 10).
//
// Mesma trava, mesmo orçamento fresco, mesma medição: o que muda é a memória de
// ONDE a pessoa parou, que o caminho sozinho não deixa reconstruir. Sem ela,
// "desfazer a última perna" seria um palpite sobre o movimento que a mesa está
// vendo.
func (bs *BoardStore) ProposeMoveComParadas(ctx context.Context, sessionID int64, tabuleiroID string, st *aovivo.SessionRuntimeState, tokenID string, paradas []engine.Square, by Mover, speedSquares int) (*BoardState, error) {
	return bs.apply(ctx, sessionID, tabuleiroID, func(b *BoardState) error {
		if token := FindToken(b, tokenID); token != nil && speedSquares > 0 {
			token.SpeedSquares = speedSquares
		}
		return ProposeMoveComParadas(b, st, tokenID, paradas, by)
	})
}

func (bs *BoardStore) CommitMove(ctx context.Context, sessionID int64, tabuleiroID string, st *aovivo.SessionRuntimeState, version int64, by Mover) (*BoardState, error) {
	return bs.apply(ctx, sessionID, tabuleiroID, func(b *BoardState) error { return CommitMove(b, st, version, by) })
}

func (bs *BoardStore) CancelMove(ctx context.Context, sessionID int64, tabuleiroID string, by Mover) (*BoardState, error) {
	return bs.apply(ctx, sessionID, tabuleiroID, func(b *BoardState) error { return CancelMove(b, by) })
}

// SetCurtain fecha ou abre a CORTINA (ALE-202). Devolve `changed` falso quando
// o estado já era o pedido: fechar cortina fechada não é erro — dois cliques
// no telefone do mestre, ou dois abas abertas — mas também não é mutação, e
// publicar quadro por não-mudança acorda a mesa inteira à toa.
//
// A cortina é POR TABULEIRO desde que ela existe, porque `Curtained` é campo do
// `BoardState`. O que a ALE-205 mudou não foi ela: foi haver mais de um estado
// para ela morar. O mestre monta a taverna com a cortina fechada enquanto a mesa
// olha a cripta, e isso saiu de graça.
func (bs *BoardStore) SetCurtain(ctx context.Context, sessionID int64, tabuleiroID string, fechada bool) (*BoardState, bool, error) {
	var mudou bool
	b, err := bs.apply(ctx, sessionID, tabuleiroID, func(b *BoardState) error {
		mudou = b.Curtained != fechada
		if !mudou {
			return nil
		}
		b.Curtained = fechada
		// `Version` significa "o tabuleiro mudou", e fechar a cortina é uma
		// mudança: toda mutação aceita neste arquivo sobe o contador, e uma que
		// não subisse faria o número mentir para quem o lê — o descarte de
		// quadro atrasado do hub (ALE-238 #1) e o `commitMove`, que recusa
		// confirmar sobre um tabuleiro que mudou desde a proposta.
		//
		// MEDIDO, porque eu primeiro escrevi aqui que sem o bump a mesa nunca
		// veria a cortina: é FALSO. O `EmitOrdered` descarta com `Seq <
		// ultimaSeq`, estritamente menor, então versão repetida PASSA — tirei o
		// bump, subi o servidor e o e2e de dois clientes continuou verde. O
		// contador não é o que faz a cortina chegar; é o que a mantém honesta.
		b.Version++
		return nil
	})
	return b, mudou, err
}

// MoveOGrupo desloca as peças marcadas pelo mesmo delta (ALE-203, item 10).
//
// Uma transação para o grupo inteiro, pelo mesmo motivo do `PintaOTraco`: o
// gesto é UM, e uma gravação por peça faria a mesa ver a horda chegar pela
// metade.
func (bs *BoardStore) MoveOGrupo(
	ctx context.Context, sessionID int64, tabuleiroID string, ids []string, dx, dy int,
) (*BoardState, error) {
	return bs.apply(ctx, sessionID, tabuleiroID, func(b *BoardState) error {
		return MoveOGrupo(b, ids, dx, dy)
	})
}
