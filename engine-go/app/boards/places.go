package boards

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"t20engine/domain/board"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/events"
)

// Archive guarda a cena atual como lugar da crônica e devolve o lugar.
//
// Sobrescreve o lugar de MESMO NOME na mesma crônica: quem reabre a taverna,
// move duas peças e encerra de novo espera uma taverna — não uma pilha de
// tavernas quase iguais. É memória do que importa, não histórico de tudo.
func (bs *Store) Archive(ctx context.Context, campaignID int64, state *board.BoardState) error {
	blob, err := json.Marshal(state)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	existente, err := bs.q.FindCampaignPlaceByName(ctx, sqlcgen.FindCampaignPlaceByNameParams{
		Campaignid: campaignID,
		Name:       state.Place,
	})
	if err == nil {
		_, err = bs.q.UpdateCampaignPlace(ctx, sqlcgen.UpdateCampaignPlaceParams{
			State: string(blob), Updatedat: now, ID: existente.ID,
		})
		return err
	}
	_, err = bs.q.SaveCampaignPlace(ctx, sqlcgen.SaveCampaignPlaceParams{
		Campaignid: campaignID,
		Name:       state.Place,
		State:      string(blob),
		Createdat:  now,
		Updatedat:  now,
	})
	return err
}

// Places lista os lugares da crônica, sem as cenas.
func (bs *Store) Places(ctx context.Context, campaignID int64) []board.Place {
	rows, err := bs.q.ListCampaignPlaces(ctx, campaignID)
	if err != nil {
		log.Printf("campaign %d: falha ao listar lugares (%v)", campaignID, err)
		return []board.Place{}
	}
	lugares := make([]board.Place, 0, len(rows))
	for _, row := range rows {
		lugares = append(lugares, board.Place{
			ID:        row.ID,
			Name:      row.Name,
			Tokens:    countTokens(row.State),
			UpdatedAt: row.Updatedat,
		})
	}
	return lugares
}

// OpenPlace põe um lugar guardado numa ABA NOVA, sem tocar no que já está na
// mesa. É o que "Reabrir" faz: as duas cenas ficam abertas, cada uma na sua aba,
// e não há arquivamento preventivo porque nada é substituído.
//
// A posse é conferida como no `RemovePlace`, e pelo mesmo motivo: o id vem do
// cliente, e sem a checagem um mestre puxaria para a própria mesa a cena de
// OUTRA campanha.
func (bs *Store) OpenPlace(ctx context.Context, campaignID, sessionID, placeID int64) (*board.BoardState, error) {
	b, err := bs.openPlaceLocked(ctx, campaignID, sessionID, placeID)
	if err != nil {
		return nil, err
	}
	bs.bus.Publish(events.BoardOpened{SessionID: sessionID})
	return b, nil
}

func (bs *Store) openPlaceLocked(ctx context.Context, campaignID, sessionID, placeID int64) (*board.BoardState, error) {
	row, err := bs.q.GetCampaignPlace(ctx, placeID)
	if err != nil {
		return nil, err
	}
	if row.Campaignid != campaignID {
		return nil, errPlaceFromAnotherCampaign
	}
	cena, err := storedScene(row.State, row.Name)
	if err != nil {
		return nil, err
	}
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	bs.hydrateLocked(ctx, sessionID)
	return bs.inNewTabLocked(sessionID, cena)
}

// inNewTabLocked acrescenta a cena como mais uma aba, com a trava na mão.
//
// UM lugar só cunha id e sequência, e é por isso que ele existe: uma segunda
// cópia disso é como uma delas esquece o TETO de abas — a diferença entre uma
// sessão com oito cenas e uma que cresce sem limite carregando tudo em toda
// hidratação.
func (bs *Store) inNewTabLocked(sessionID int64, cena *board.BoardState) (*board.BoardState, error) {
	if len(bs.boards[sessionID]) >= openBoardsCeiling {
		return nil, fmt.Errorf(
			"esta sessão já tem %d tabuleiros abertos (teto %d): feche um antes de abrir outro lugar",
			len(bs.boards[sessionID]), openBoardsCeiling)
	}
	cena.ID = bs.newID()
	cena.Seq = bs.nextSeqLocked(sessionID)
	bs.boards[sessionID] = append(bs.boards[sessionID], cena)
	return cloneBoard(cena), nil
}

// storedScene desempacota o que o acervo guardou, pronto para entrar na mesa.
// As três decisões abaixo moram aqui e não em quem chama: copiadas, uma cena
// reaberta por um caminho voltaria com o movimento proposto da semana passada e
// a do outro não.
func storedScene(blob, nome string) (*board.BoardState, error) {
	var cena board.BoardState
	if err := json.Unmarshal([]byte(blob), &cena); err != nil {
		return nil, err
	}
	// Fatia VAZIA e não nula: `null` no JSON derruba quem indexa `tokens.length`.
	if cena.Tokens == nil {
		cena.Tokens = []board.BoardToken{}
	}
	// O provisório não volta: ele é de uma cena que já acabou, e a mesa que
	// reabre a taverna não deve nada a um movimento proposto na semana passada.
	cena.Pending = nil
	// O nome vem da COLUNA e não do JSON: renomear o lugar mexeria em dois
	// lugares, e o de fora é o que a lista mostra.
	cena.Place = nome
	return &cena, nil
}

// PlaceScene devolve a cena INTEIRA de um lugar guardado — é o que o mestre
// monta sem pôr nada na mesa.
//
// A lista de lugares viaja sem as cenas de propósito (só nome e contagem), e é
// por isso que existe esta segunda pergunta: baixar o acervo inteiro para
// desenhar um menu seria pagar caro por um número, mas para EDITAR é a cena que
// se precisa.
func (bs *Store) PlaceScene(ctx context.Context, campaignID, placeID int64) (*board.BoardState, error) {
	row, err := bs.q.GetCampaignPlace(ctx, placeID)
	if err != nil {
		return nil, err
	}
	if row.Campaignid != campaignID {
		return nil, errPlaceFromAnotherCampaign
	}
	var cena board.BoardState
	if err := json.Unmarshal([]byte(row.State), &cena); err != nil {
		return nil, err
	}
	if cena.Tokens == nil {
		cena.Tokens = []board.BoardToken{}
	}
	// O nome vem da COLUNA, como no reabrir: ele é o que a lista mostra, e ter
	// duas verdades sobre como o lugar se chama é como elas divergem.
	cena.Place = row.Name
	cena.Pending = nil
	return &cena, nil
}

// SavePlaceScene grava a cena que o mestre montou, sem tocar na mesa.
//
// Quem a chama é o `EditPlace`, depois de aplicar UM gesto à cena que ele acabou
// de ler — o rascunho é a MESMA superfície do tabuleiro apontada para o acervo,
// e reusar os handlers de gesto custa menos que inventar um protocolo só dele.
//
// A CONFERÊNCIA mora aqui porque este é o único lugar por onde o acervo é
// escrito. Ela não é a fronteira contra um cliente quebrado: é o guarda contra
// uma mutação pura que produza coordenada absurda ou estoure o teto de peças —
// as puras não sabem de nenhum dos dois. Sem ela o lixo só apareceria quando a
// cena chegasse à mesa.
func (bs *Store) SavePlaceScene(ctx context.Context, campaignID, placeID int64, cena *board.BoardState) error {
	row, err := bs.q.GetCampaignPlace(ctx, placeID)
	if err != nil {
		return err
	}
	if row.Campaignid != campaignID {
		return errPlaceFromAnotherCampaign
	}
	if err := sanitizeScene(cena, bs.newID); err != nil {
		return err
	}
	cena.Place = row.Name
	blob, err := json.Marshal(cena)
	if err != nil {
		return err
	}
	_, err = bs.q.UpdateCampaignPlace(ctx, sqlcgen.UpdateCampaignPlaceParams{
		State: string(blob), Updatedat: time.Now().UTC().Format(time.RFC3339), ID: placeID,
	})
	return err
}

// sanitizeScene aplica à cena que chegou do cliente as MESMAS regras que o
// tabuleiro vivo aplica peça a peça: teto de peças, coordenada sã e tamanho
// mínimo. Recusa em vez de corrigir o que não dá para corrigir sem inventar —
// uma peça em (10^9, 0) não tem posição "quase certa".
//
// A peça nova nasce sem id (o cliente não cunha id de servidor) e ganha um
// aqui; o provisório não existe em acervo, porque ele é de uma cena que está
// acontecendo.
func sanitizeScene(cena *board.BoardState, newID func() string) error {
	if len(cena.Tokens) > board.MaxTokens {
		return fmt.Errorf("a cena tem %d peças (teto %d)", len(cena.Tokens), board.MaxTokens)
	}
	for i := range cena.Tokens {
		token := &cena.Tokens[i]
		if token.Footprint <= 0 {
			token.Footprint = 1
		}
		if err := board.AssertSaneCoords(*token); err != nil {
			return err
		}
		if token.ID == "" {
			token.ID = newID()
		}
	}
	cena.Pending = nil
	return nil
}

// countTokens conta as peças sem desserializar a cena inteira num tipo — a
// lista de lugares só quer o número, e um `board.Place` inteiro por linha seria ler o
// acervo do mestre para desenhar um menu.
func countTokens(state string) int {
	var apenasPecas struct {
		Tokens []json.RawMessage `json:"tokens"`
	}
	if err := json.Unmarshal([]byte(state), &apenasPecas); err != nil {
		return 0
	}
	return len(apenasPecas.Tokens)
}

// RemovePlace apaga um lugar do acervo da crônica.
//
// Confere a crônica antes de apagar: o id vem do cliente, e sem a checagem um
// mestre apagaria o lugar de OUTRA mesa mandando um id que não é dele. É a
// mesma regra de posse que as rotas de personagem aplicam.
func (bs *Store) RemovePlace(ctx context.Context, campaignID, placeID int64) error {
	row, err := bs.q.GetCampaignPlace(ctx, placeID)
	if err != nil {
		return err
	}
	if row.Campaignid != campaignID {
		return errPlaceFromAnotherCampaign
	}
	return bs.q.DeleteCampaignPlace(ctx, placeID)
}

var errPlaceFromAnotherCampaign = errors.New("este lugar é de outra crônica")
