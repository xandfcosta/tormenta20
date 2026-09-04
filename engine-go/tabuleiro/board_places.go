package tabuleiro

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"t20engine/db/sqlcgen"
	"t20engine/events"
)

// Place é uma cena guardada da crônica (ALE-124, fatia 5).
//
// O que a mesa chama de "lugar" é o tabuleiro CONGELADO: a taverna com as nove
// peças onde ficaram, para reabrir na semana seguinte sem remontar nada.
type Place struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	// Tokens é só a CONTAGEM: a lista serve para escolher onde jogar, e mandar
	// a cena inteira de cada lugar seria mandar o acervo do mestre a cada
	// abertura de menu. A cena chega ao reabrir.
	Tokens    int    `json:"tokens"`
	UpdatedAt string `json:"updatedAt"`
}

// Archive guarda a cena atual como lugar da crônica e devolve o lugar.
//
// Sobrescreve o lugar de MESMO NOME na mesma crônica: quem reabre a taverna,
// move duas peças e encerra de novo espera uma taverna — não uma pilha de
// tavernas quase iguais. É a mesma decisão do "voltar para onde estava" da
// ALE-178: memória do que importa, não histórico de tudo.
func (bs *BoardStore) Archive(ctx context.Context, campaignID int64, board *BoardState) error {
	blob, err := json.Marshal(board)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	existente, err := bs.q.FindCampaignPlaceByName(ctx, sqlcgen.FindCampaignPlaceByNameParams{
		Campaignid: campaignID,
		Name:       board.Place,
	})
	if err == nil {
		_, err = bs.q.UpdateCampaignPlace(ctx, sqlcgen.UpdateCampaignPlaceParams{
			State: string(blob), Updatedat: now, ID: existente.ID,
		})
		return err
	}
	_, err = bs.q.SaveCampaignPlace(ctx, sqlcgen.SaveCampaignPlaceParams{
		Campaignid: campaignID,
		Name:       board.Place,
		State:      string(blob),
		Createdat:  now,
		Updatedat:  now,
	})
	return err
}

// Places lista os lugares da crônica, sem as cenas.
func (bs *BoardStore) Places(ctx context.Context, campaignID int64) []Place {
	rows, err := bs.q.ListCampaignPlaces(ctx, campaignID)
	if err != nil {
		log.Printf("campaign %d: falha ao listar lugares (%v)", campaignID, err)
		return []Place{}
	}
	lugares := make([]Place, 0, len(rows))
	for _, row := range rows {
		lugares = append(lugares, Place{
			ID:        row.ID,
			Name:      row.Name,
			Tokens:    countTokens(row.State),
			UpdatedAt: row.Updatedat,
		})
	}
	return lugares
}

// Aqui moravam o `ShowPlace`, o `Reopen` e o `reopenLocked` — as três portas com
// que a ALE-191 resolvia "põe esta cena na mesa" ANTES de existirem abas.
//
// O `ShowPlace` arquivava a cena da aba e entrava no lugar dela; o `Reopen` era a
// primitiva que trocava a cena de UMA aba, ou abria a primeira se não houvesse
// nenhuma. A ALE-205 tirou o problema que os dois resolviam: com abas nada é
// substituído, então não há o que guardar antes — e desde então a rota
// `/lugares/{placeId}/reabrir` entra pelo `OpenPlace`, logo abaixo.
//
// Elas ficaram no ar com ZERO chamadores de produção e QUATRO casos de teste em
// cima (ALE-289), e é isso que as tornava caras: um daqueles casos afirmava, em
// verde, que trocar de cena ARQUIVA a que estava na mesa — o comportamento exato
// que a ALE-205 removeu. Um teste que dirige uma porta morta não fica obsoleto
// junto com ela; ele passa a afirmar o oposto do produto, e continua verde.
//
// Nenhuma garantia sumiu, as três mudaram de porta e hoje são presas contra o
// `OpenPlace`: a posse (`…ThroughOpenPlace`), as peças que voltam onde estavam e
// o movimento provisório que NÃO volta. A última mora no `storedScene`, por onde
// as duas portas sempre passaram — foi ela que impediu que a divergência entre
// os dois caminhos acontecesse enquanto ambos existiam.

// OpenPlace põe um lugar guardado numa ABA NOVA, sem tocar no que já está na
// mesa (ALE-205, fatia 3).
//
// É o que "Reabrir" passou a fazer, e a diferença com o `ShowPlace` é a issue
// inteira: lá a cena guardada ENTRAVA no lugar de outra, que ia para o acervo
// antes; aqui as duas ficam abertas, cada uma na sua aba. O arquivamento
// preventivo que a ALE-191 inventou deixou de ser necessário porque deixou de
// haver o que perder — nada é substituído.
//
// A posse é conferida como no `RemovePlace`, e pelo mesmo
// motivo: o id vem do cliente, e sem a checagem um mestre puxaria para a própria
// mesa a cena de OUTRA campanha.
func (bs *BoardStore) OpenPlace(ctx context.Context, campaignID, sessionID, placeID int64) (*BoardState, error) {
	b, err := bs.openPlaceLocked(ctx, campaignID, sessionID, placeID)
	if err != nil {
		return nil, err
	}
	bs.bus.Publish(events.BoardOpened{SessionID: sessionID})
	return b, nil
}

func (bs *BoardStore) openPlaceLocked(ctx context.Context, campaignID, sessionID, placeID int64) (*BoardState, error) {
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
// UM lugar só cunha id e sequência, e é por isso que ele existe: o `Reopen` sem
// aba e o `OpenPlace` fazem a mesma coisa, e duas cópias disso é como uma delas
// esquece o teto — que é a diferença entre uma sessão com oito cenas e uma que
// cresce sem limite carregando tudo em toda hidratação.
func (bs *BoardStore) inNewTabLocked(sessionID int64, cena *BoardState) (*BoardState, error) {
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
//
// As três decisões que ela carrega estavam soltas no `Reopen`, e a segunda porta
// de entrada (o `OpenPlace`) precisava exatamente delas — copiadas, seria a
// forma clássica de uma se esquecer: a cena reaberta por um caminho voltaria com
// o movimento proposto da semana passada e a do outro não.
func storedScene(blob, nome string) (*BoardState, error) {
	var cena BoardState
	if err := json.Unmarshal([]byte(blob), &cena); err != nil {
		return nil, err
	}
	// Fatia VAZIA e não nula: `null` no JSON derruba quem indexa `tokens.length`.
	if cena.Tokens == nil {
		cena.Tokens = []BoardToken{}
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
// monta sem pôr nada na mesa (ALE-191, fatia 2).
//
// A lista de lugares viaja sem as cenas de propósito (só nome e contagem), e é
// por isso que existe esta segunda pergunta: baixar o acervo inteiro para
// desenhar um menu seria pagar caro por um número, mas para EDITAR é a cena que
// se precisa.
func (bs *BoardStore) PlaceScene(ctx context.Context, campaignID, placeID int64) (*BoardState, error) {
	row, err := bs.q.GetCampaignPlace(ctx, placeID)
	if err != nil {
		return nil, err
	}
	if row.Campaignid != campaignID {
		return nil, errPlaceFromAnotherCampaign
	}
	var cena BoardState
	if err := json.Unmarshal([]byte(row.State), &cena); err != nil {
		return nil, err
	}
	if cena.Tokens == nil {
		cena.Tokens = []BoardToken{}
	}
	// O nome vem da COLUNA, como no reabrir: ele é o que a lista mostra, e ter
	// duas verdades sobre como o lugar se chama é como elas divergem.
	cena.Place = row.Name
	cena.Pending = nil
	return &cena, nil
}

// SavePlaceScene grava a cena que o mestre montou, sem tocar na mesa.
//
// Quem a chama é o `EditPlace` (ALE-292), depois de aplicar UM gesto à cena que
// ele acabou de ler. Ela continua sendo o único lugar por onde o acervo é
// escrito, e é por isso que a conferência mora aqui.
//
// # O que esta docstring dizia, e por que deixou de ser verdade
//
// Ela dizia: *"este é o ÚNICO lugar do tabuleiro onde um estado inteiro chega
// pelo cliente… o rascunho não tem concorrência, não tem broadcast e não tem
// vez, então um handler por gesto seria protocolo para nada."* O raciocínio
// estava escrito desde a ALE-191 e nunca foi exercido — a função passou duas
// épicas com zero chamadores de produção.
//
// Quando o gesto finalmente chegou, ele veio pelo outro caminho: o rascunho é a
// MESMA superfície do tabuleiro apontada para o acervo, então ele já tem um
// handler por gesto, e reusá-los custou menos que inventar um protocolo só
// dele. O estado inteiro NÃO chega mais pelo cliente em lugar nenhum.
//
// A CONFERÊNCIA fica, e ela não virou enfeite: ela deixou de ser a fronteira
// contra um cliente quebrado e passou a ser o guarda contra uma mutação pura
// que produza coordenada absurda ou estoure o teto de peças — as puras não
// sabem de nenhum dos dois. Sem ela, o lixo só apareceria quando a cena
// chegasse à mesa.
func (bs *BoardStore) SavePlaceScene(ctx context.Context, campaignID, placeID int64, cena *BoardState) error {
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
func sanitizeScene(cena *BoardState, newID func() string) error {
	if len(cena.Tokens) > boardMaxTokens {
		return fmt.Errorf("a cena tem %d peças (teto %d)", len(cena.Tokens), boardMaxTokens)
	}
	for i := range cena.Tokens {
		token := &cena.Tokens[i]
		if token.Footprint <= 0 {
			token.Footprint = 1
		}
		if err := assertSaneCoords(*token); err != nil {
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
// lista de lugares só quer o número, e um `Place` inteiro por linha seria ler o
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
func (bs *BoardStore) RemovePlace(ctx context.Context, campaignID, placeID int64) error {
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
