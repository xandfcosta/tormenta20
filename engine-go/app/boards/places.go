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
	// O VÍNCULO COM A FILA FICA NA SESSÃO (ALE-377). Sobre uma CÓPIA, porque o
	// que entra aqui é o tabuleiro VIVO da mesa: desamarrar o original tiraria
	// as barras de PV da tela no instante em que o mestre arquiva.
	guardado := *state
	guardado.Tokens = append([]board.BoardToken(nil), state.Tokens...)
	board.UnbindOrphanTokens(&guardado, nil)
	blob, err := json.Marshal(&guardado)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	existing, err := bs.q.FindCampaignPlaceByName(ctx, sqlcgen.FindCampaignPlaceByNameParams{
		Campaignid: campaignID,
		Name:       state.Place,
	})
	if err == nil {
		_, err = bs.q.UpdateCampaignPlace(ctx, sqlcgen.UpdateCampaignPlaceParams{
			State: string(blob), Updatedat: now, ID: existing.ID,
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
	places := make([]board.Place, 0, len(rows))
	for _, row := range rows {
		places = append(places, board.Place{
			ID:        row.ID,
			Name:      row.Name,
			Tokens:    countTokens(row.State),
			UpdatedAt: row.Updatedat,
		})
	}
	return places
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
	scene, err := storedScene(row.State, row.Name)
	if err != nil {
		return nil, err
	}
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	// A HIDRATAÇÃO QUE FALHA RECUSA (ALE-375): sem saber quais abas já estão
	// abertas, abrir mais uma passa por cima do teto e pode duplicar a cena que
	// já estava na mesa.
	if err := bs.hydrateLocked(ctx, sessionID); err != nil {
		return nil, err
	}
	return bs.inNewTabLocked(sessionID, scene)
}

// inNewTabLocked acrescenta a cena como mais uma aba, com a trava na mão.
//
// UM lugar só cunha id e sequência, e é por isso que ele existe: uma segunda
// cópia disso é como uma delas esquece o TETO de abas — a diferença entre uma
// sessão com oito cenas e uma que cresce sem limite carregando tudo em toda
// hidratação.
func (bs *Store) inNewTabLocked(sessionID int64, scene *board.BoardState) (*board.BoardState, error) {
	if len(bs.boards[sessionID]) >= openBoardsCeiling {
		return nil, fmt.Errorf(
			"esta sessão já tem %d tabuleiros abertos (teto %d): feche um antes de abrir outro lugar",
			len(bs.boards[sessionID]), openBoardsCeiling)
	}
	scene.ID = bs.newID()
	scene.Seq = bs.nextSeqLocked(sessionID)
	bs.boards[sessionID] = append(bs.boards[sessionID], scene)
	return cloneBoard(scene), nil
}

// storedScene desempacota o que o acervo guardou, pronto para entrar na mesa.
// As três decisões abaixo moram aqui e não em quem chama: copiadas, uma cena
// reaberta por um caminho voltaria com o movimento proposto da semana passada e
// a do outro não.
func storedScene(blob, name string) (*board.BoardState, error) {
	var scene board.BoardState
	if err := json.Unmarshal([]byte(blob), &scene); err != nil {
		return nil, err
	}
	// Fatia VAZIA e não nula: `null` no JSON derruba quem indexa `tokens.length`.
	if scene.Tokens == nil {
		scene.Tokens = []board.BoardToken{}
	}
	// O provisório não volta: ele é de uma cena que já acabou, e a mesa que
	// reabre a taverna não deve nada a um movimento proposto na semana passada.
	scene.Pending = nil
	// NEM O VÍNCULO COM A FILA, e aqui ele é a rede para o que JÁ ESTÁ GRAVADO:
	// o corte na ida (o `Archive`) só vale para o que for arquivado de agora em
	// diante, e este é o gargalo por onde todo leitor do acervo passa (ALE-377).
	board.UnbindOrphanTokens(&scene, nil)
	// O nome vem da COLUNA e não do JSON: renomear o lugar mexeria em dois
	// lugares, e o de fora é o que a lista mostra.
	scene.Place = name
	return &scene, nil
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
	var scene board.BoardState
	if err := json.Unmarshal([]byte(row.State), &scene); err != nil {
		return nil, err
	}
	if scene.Tokens == nil {
		scene.Tokens = []board.BoardToken{}
	}
	// O nome vem da COLUNA, como no reabrir: ele é o que a lista mostra, e ter
	// duas verdades sobre como o lugar se chama é como elas divergem.
	scene.Place = row.Name
	scene.Pending = nil
	return &scene, nil
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
func (bs *Store) SavePlaceScene(ctx context.Context, campaignID, placeID int64, scene *board.BoardState) error {
	row, err := bs.q.GetCampaignPlace(ctx, placeID)
	if err != nil {
		return err
	}
	if row.Campaignid != campaignID {
		return errPlaceFromAnotherCampaign
	}
	if err := sanitizeScene(scene, bs.newID); err != nil {
		return err
	}
	scene.Place = row.Name
	blob, err := json.Marshal(scene)
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
func sanitizeScene(scene *board.BoardState, newID func() string) error {
	if len(scene.Tokens) > board.MaxTokens {
		return fmt.Errorf("a cena tem %d peças (teto %d)", len(scene.Tokens), board.MaxTokens)
	}
	for i := range scene.Tokens {
		token := &scene.Tokens[i]
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
	scene.Pending = nil
	// O VÍNCULO COM A FILA NÃO ATRAVESSA PARA O ACERVO (ALE-377).
	//
	// O `EntryID` é um id de LINHA DA FILA, e fila é da SESSÃO; o acervo é da
	// CAMPANHA e não tem fila nenhuma. Sem esta linha ele era gravado no blob e
	// voltava intacto ao reabrir o lugar — MEDIDO: numa sessão B, a peça
	// ressuscitava apontando para uma linha da sessão A, numa fila com zero
	// linhas.
	//
	// `nil` como fila porque é a verdade: aqui não existe nenhuma linha viva, e
	// a mesma primitiva que reconcilia a sessão responde ao acervo.
	board.UnbindOrphanTokens(scene, nil)
	return nil
}

// countTokens conta as peças sem desserializar a cena inteira num tipo — a
// lista de lugares só quer o número, e um `board.Place` inteiro por linha seria ler o
// acervo do mestre para desenhar um menu.
func countTokens(state string) int {
	var tokensOnly struct {
		Tokens []json.RawMessage `json:"tokens"`
	}
	if err := json.Unmarshal([]byte(state), &tokensOnly); err != nil {
		return 0
	}
	return len(tokensOnly.Tokens)
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
