package session

import (
	"context"
	"encoding/json"
	"fmt"

	"t20engine/domain/live"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// O RETRATO DA MESA NO BANCO, e o banco é a FONTE DA VERDADE (ALE-371).
//
// Aqui morava um `Persist` best-effort: a mutação mudava a memória, a mesa via
// a rodada avançar, e a gravação saía numa goroutine que podia falhar depois —
// daí existirem uma marca de sujeira e um aviso na tela. Agora a gravação mora
// DENTRO da mutação: o que não pôde ser gravado não aconteceu, e quem clicou
// recebe a recusa em vez de ver na tela um turno que o disco recusou.

// SessionSnapshots é a porta do retrato. Ela fala tipo de DOMÍNIO, não blob: o
// formato gravado é problema de quem implementa.
type SessionSnapshots interface {
	// Read devolve a mesa gravada. Sessão que não existe é erro — inventar uma
	// vazia foi o defeito da ALE-369.
	Read(ctx context.Context, sessionID int64) (*live.SessionRuntimeState, error)
	// Mutate lê o retrato, deixa `change` mudar e GRAVA antes de devolver.
	//
	// OS TRÊS PASSOS NÃO MORAM NUMA TRANSAÇÃO SÓ, que é o desenho óbvio contra
	// atualização perdida — e o motivo de não morarem foi medido, com o
	// mecanismo confirmado até o fim (ALE-371).
	//
	// A função de mudança NÃO É PURA: a do `NextTurn` expira efeitos, paga
	// manutenção e abre o teste de sangramento, e as três ESCREVEM NA FICHA, por
	// outra conexão. Com a transação de escrita aberta em volta dela, essas
	// escritas batem na trava do próprio chamador e esperam o `busy_timeout`.
	//
	// O que a medição mostrou, com a transação em volta:
	//
	//   - `TestWithoutManaTheSustainedAbilityEnds`: 0,037s → 9,84s, e REPROVANDO;
	//   - o pacote `serve/api` inteiro: 9,9s → 244,2s;
	//   - o erro engolido pelo `_ =` do `EndSustained` é, literalmente,
	//     `database is locked (5) (SQLITE_BUSY)`;
	//   - e o tempo da falha SEGUE o `busy_timeout` — 5000ms dá 9,84s, 1000ms dá
	//     2,02s, 200ms dá 0,42s, sempre ~2× o limite, que são dois toques
	//     bloqueados por caso. É isto que separa "a trava é a causa" de
	//     "transação é lenta".
	//
	// O que serializa as mutações é a trava do store, que é do PROCESSO — e o
	// app é um processo só, um serviço no compose. Ler aqui, e não do cache, é o
	// que garante que a mudança parta do que está gravado.
	Mutate(ctx context.Context, sessionID int64,
		change func(*live.SessionRuntimeState) error) (*live.SessionRuntimeState, error)
}

// NewSnapshots é o retrato no SQLite deste app.
func NewSnapshots(q *sqlcgen.Queries) SessionSnapshots {
	return storedSnapshots{q: q}
}

type storedSnapshots struct {
	q *sqlcgen.Queries
}

func (sn storedSnapshots) Read(ctx context.Context, sessionID int64) (*live.SessionRuntimeState, error) {
	row, err := sn.q.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("ler do banco a mesa da sessão %d: %w", sessionID, err)
	}
	return parseRuntimeBlob(row.Runtimestate), nil
}

func (sn storedSnapshots) Mutate(ctx context.Context, sessionID int64,
	change func(*live.SessionRuntimeState) error) (*live.SessionRuntimeState, error) {
	state, err := sn.Read(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := change(state); err != nil {
		return nil, err
	}
	blob, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("serializar a mesa da sessão %d: %w", sessionID, err)
	}
	if err := sn.q.ResetSessionTracker(ctx, sqlcgen.ResetSessionTrackerParams{
		RuntimeState: string(blob), UpdatedAt: dbvalue.NowISO(), ID: sessionID,
	}); err != nil {
		return nil, fmt.Errorf("gravar a mesa da sessão %d: %w", sessionID, err)
	}
	return state, nil
}

// parseRuntimeBlob lê um blob gravado e cai para um rastreador vazio quando ele
// vem malformado. Os blobs são sempre completos (o nosso Marshal e o default da
// coluna carregam rodada e turno), então não há campo parcial a preencher.
func parseRuntimeBlob(blob string) *live.SessionRuntimeState {
	if blob == "" {
		return live.EmptyRuntimeState()
	}
	var parsed live.SessionRuntimeState
	if err := json.Unmarshal([]byte(blob), &parsed); err != nil {
		return live.EmptyRuntimeState()
	}
	if parsed.Initiative == nil {
		parsed.Initiative = []live.InitiativeEntry{}
	}
	return &parsed
}
