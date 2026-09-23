package boards

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"t20engine/domain/board"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// O RETRATO DO TABULEIRO NO BANCO, e o banco é a FONTE DA VERDADE (ALE-375).
//
// Aqui morava uma gravação best-effort, e ela era a última do app: a mutação
// mudava a memória, a mesa via a peça andar, e a gravação saía numa goroutine
// que podia falhar depois — daí existirem uma marca de sujeira (`Dirty`), um
// `SaveFailed` e uma tarja na tela da Mesa. Agora a gravação mora DENTRO da
// mutação, como na fila (ALE-371, ALE-373): o que não pôde ser gravado não
// aconteceu, e quem clicou recebe a recusa.

// BoardSnapshots é a porta do retrato. Ela fala tipo de DOMÍNIO e não blob: o
// formato gravado é problema de quem implementa, e é por isso que o
// `json.Marshal` não aparece no store.
type BoardSnapshots interface {
	// Read devolve os tabuleiros abertos da sessão, na ordem de abertura.
	//
	// SESSÃO SEM TABULEIRO É LISTA VAZIA, e isso não é erro: é a resposta certa
	// para a mesa que ainda não abriu mapa nenhum, e o schema a diz (a linha
	// existe ou não existe). O que MUDOU é o erro de leitura, que agora SOBE em
	// vez de virar "sem tabuleiro" num log — ver o `Read` da fila, que contava a
	// mesma mentira até a ALE-373.
	Read(ctx context.Context, sessionID int64) ([]*board.BoardState, error)
	// Save grava UM tabuleiro.
	//
	// UM e não os abertos todos: com oito abas, gravar todas a cada peça que
	// anda seriam oito serializações e oito upserts por gesto. Quem chama sabe
	// qual aba mudou porque acabou de mutá-la.
	Save(ctx context.Context, sessionID int64, b *board.BoardState) error
	// Delete tira do banco o tabuleiro que foi fechado.
	Delete(ctx context.Context, sessionID int64, boardID string) error
}

// snapshotsKey marca no contexto o retrato de uma TRANSAÇÃO ABERTA. Chave de
// tipo próprio e não string: é o jeito de o valor não colidir com o de outro
// pacote — a mesma forma que o `unitKey` da unidade de trabalho usa.
type snapshotsKey struct{}

// WithSnapshots devolve um contexto que carrega o retrato de uma transação.
//
// É por aqui que o TABULEIRO entra na unidade de trabalho da fila (ALE-376). O
// store guarda o retrato dele no construtor, e um gesto que atravessa os dois
// donos de dado precisa que a gravação do tabuleiro vá pela MESMA transação —
// senão ela pega a segunda conexão do pool e bate na trava da primeira, que é
// o impasse da ALE-371.
//
// Pelo CONTEXTO e não por um parâmetro em cada método: são vinte e tantas
// mutações no store, e um parâmetro a mais em todas elas seria pago por quem
// nunca abre transação — que é quase todo chamador.
func WithSnapshots(ctx context.Context, snapshots BoardSnapshots) context.Context {
	return context.WithValue(ctx, snapshotsKey{}, snapshots)
}

// NewSnapshots é o retrato no SQLite deste app.
func NewSnapshots(q *sqlcgen.Queries) BoardSnapshots {
	return storedSnapshots{q: q}
}

type storedSnapshots struct {
	q *sqlcgen.Queries
}

func (sn storedSnapshots) Read(ctx context.Context, sessionID int64) ([]*board.BoardState, error) {
	rows, err := sn.q.ListOpenBoards(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("ler do banco os tabuleiros da sessão %d: %w", sessionID, err)
	}
	open := make([]*board.BoardState, 0, len(rows))
	for _, row := range rows {
		parsed, err := parseBoardBlob(row.State)
		if err != nil {
			return nil, fmt.Errorf("ler o tabuleiro %s da sessão %d: %w", row.Boardid, sessionID, err)
		}
		// O ID e a SEQUÊNCIA vêm da COLUNA e não do JSON: duas verdades sobre
		// quem é este tabuleiro é como elas divergem, e a de fora é a que o
		// upsert usa.
		parsed.ID = row.Boardid
		parsed.Seq = row.Openseq
		open = append(open, parsed)
	}
	return open, nil
}

func (sn storedSnapshots) Save(ctx context.Context, sessionID int64, b *board.BoardState) error {
	blob, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("serializar o tabuleiro %s da sessão %d: %w", b.ID, sessionID, err)
	}
	// `openSeq` só entra no INSERT — o upsert não o toca (ver a query). Gravar o
	// tabuleiro é dizer que ele mudou, nunca que ele nasceu de novo, e a ordem
	// das abas na tela sai daquela coluna.
	if err := sn.q.SaveOpenBoard(ctx, sqlcgen.SaveOpenBoardParams{
		Sessionid: sessionID, Boardid: b.ID, State: string(blob),
		Openseq: b.Seq, Updatedat: dbvalue.NowISO(),
	}); err != nil {
		return fmt.Errorf("gravar o tabuleiro %s da sessão %d: %w", b.ID, sessionID, err)
	}
	return nil
}

func (sn storedSnapshots) Delete(ctx context.Context, sessionID int64, boardID string) error {
	if err := sn.q.DeleteOpenBoard(ctx, sqlcgen.DeleteOpenBoardParams{
		Sessionid: sessionID, Boardid: boardID,
	}); err != nil {
		return fmt.Errorf("apagar o tabuleiro %s da sessão %d: %w", boardID, sessionID, err)
	}
	return nil
}

// errBoardBlobMalformed é um retrato gravado que não se lê.
//
// ELE SOBE, e aqui ele era engolido: o tabuleiro malformado virava `continue`
// num laço, com um `log.Printf`, e a mesa abria com uma aba a MENOS sem
// ninguém saber qual nem por quê. Servir meia sessão é pior que recusar, pela
// mesma razão que a fila deixou de responder "não há ninguém" quando não
// conseguia ler (ALE-373).
var errBoardBlobMalformed = errors.New("o tabuleiro gravado não é um JSON que este app saiba ler")

// parseBoardBlob lê um blob gravado.
func parseBoardBlob(blob string) (*board.BoardState, error) {
	var parsed board.BoardState
	if err := json.Unmarshal([]byte(blob), &parsed); err != nil {
		return nil, fmt.Errorf("%w: %v", errBoardBlobMalformed, err)
	}
	if parsed.Tokens == nil {
		parsed.Tokens = []board.BoardToken{}
	}
	return &parsed, nil
}
