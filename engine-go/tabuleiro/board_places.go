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

// ShowPlace põe um lugar guardado na mesa — e GUARDA ANTES a cena que estava
// lá (ALE-191).
//
// Sem isso, mostrar a cripta à mesa DESTRUÍA a taverna: o `Reopen` troca o
// tabuleiro vivo, e o que estava nele não ia para lugar nenhum. Até agora o
// caminho era inalcançável, porque a lista de Lugares só aparecia na cena
// vazia; é esta issue que o abre, ao deixar o mestre trocar de cena com a mesa
// jogando.
//
// A falha ao guardar RECUSA a troca, e aqui a política é o oposto da do
// encerrar: lá o mestre mandou tirar a cena da mesa e prendê-lo numa cena que
// já acabou seria pior; aqui ele mandou trocar, e trocar em cima de um acervo
// que não gravou é justamente perder a taverna.
func (bs *BoardStore) ShowPlace(ctx context.Context, campaignID, sessionID int64, tabuleiroID string, placeID int64) (*BoardState, error) {
	row, err := bs.q.GetCampaignPlace(ctx, placeID)
	if err != nil {
		return nil, err
	}
	// O id vem do cliente: sem conferir a crônica, um mestre puxaria para a
	// própria mesa a cena de OUTRA campanha. É a mesma posse que o `RemovePlace`
	// confere, e pelo mesmo motivo.
	if row.Campaignid != campaignID {
		return nil, errPlaceFromAnotherCampaign
	}
	// A troca acontece DENTRO DE UMA ABA (ALE-205): é a cena daquela aba que vai
	// para o acervo e é ali que a nova entra. Com vários tabuleiros abertos,
	// arquivar "o tabuleiro da sessão" seria arquivar o de alguém que não pediu
	// nada — o mestre troca a cripta pela ponte e a taverna da outra aba some.
	if atual := bs.Get(ctx, sessionID, tabuleiroID); atual != nil {
		if err := bs.Archive(ctx, campaignID, atual); err != nil {
			return nil, fmt.Errorf("não consegui guardar %q antes de trocar de cena: %w", atual.Place, err)
		}
	}
	return bs.Reopen(ctx, sessionID, tabuleiroID, placeID)
}

// Reopen põe o lugar guardado de volta na mesa. É a PRIMITIVA: não confere de
// que crônica o lugar é, nem guarda a cena que estava na mesa — quem faz as
// duas coisas é o `ShowPlace`, que é por onde o gateway entra.
//
// A VERSÃO continua a do tabuleiro que estava aberto, e não a que foi
// arquivada: um cliente com a cena velha na mão precisa reconhecer esta como
// mais recente. Reabrir é uma mutação de agora, não uma volta ao passado.
//
// COM ABA ele TROCA a cena daquela aba; SEM aba nenhuma ele ABRE a primeira
// (ALE-205). Os dois casos são o mesmo gesto do mestre — "põe esta na mesa" —, e
// o que muda é se já havia mesa: recusar o segundo deixaria o acervo inalcançável
// justamente na sessão que ainda não abriu tabuleiro, que é quando ele é mais
// usado. O teto de abertos vale nos dois.
func (bs *BoardStore) Reopen(ctx context.Context, sessionID int64, tabuleiroID string, placeID int64) (*BoardState, error) {
	b, ev, err := bs.reopenLocked(ctx, sessionID, tabuleiroID, placeID)
	if err != nil {
		return nil, err
	}
	bs.bus.Publish(ev)
	return b, nil
}

// reopenLocked faz o trabalho e DEVOLVE o evento que ele produziu.
//
// Devolver em vez de publicar aqui dentro é o que permite dizer a verdade: com
// aba já aberta a cena TROCA dentro dela, e sem aba ela NASCE. Um wrapper que
// adivinhasse o evento pelo nome do método chamaria as duas de "abriu".
func (bs *BoardStore) reopenLocked(ctx context.Context, sessionID int64, tabuleiroID string, placeID int64) (*BoardState, events.Event, error) {
	row, err := bs.q.GetCampaignPlace(ctx, placeID)
	if err != nil {
		return nil, nil, err
	}
	cena, err := storedScene(row.State, row.Name)
	if err != nil {
		return nil, nil, err
	}
	guardado := *cena

	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	bs.hydrateLocked(ctx, sessionID)
	aberto := bs.findLocked(sessionID, tabuleiroID)
	if aberto == nil {
		b, err := bs.inNewTabLocked(sessionID, &guardado)
		return b, events.BoardOpened{SessionID: sessionID}, err
	}
	if aberto.Version >= guardado.Version {
		guardado.Version = aberto.Version + 1
	}
	// A ABA continua a mesma: o que muda é a cena que está nela. O id é a
	// identidade da aba na barra e no que cada pessoa escolheu olhar — trocá-lo
	// aqui tiraria da tela de todo mundo a aba que ainda está lá.
	guardado.ID = aberto.ID
	// A POSIÇÃO na barra também é da aba, e não da cena que entrou nela: trocar
	// a taverna pela cripta não pode fazer a aba pular de lugar debaixo do dedo
	// de quem estava clicando nela.
	guardado.Seq = aberto.Seq
	*aberto = guardado
	return cloneBoard(aberto), events.BoardChanged{SessionID: sessionID}, nil
}

// OpenPlace põe um lugar guardado numa ABA NOVA, sem tocar no que já está na
// mesa (ALE-205, fatia 3).
//
// É o que "Reabrir" passou a fazer, e a diferença com o `ShowPlace` é a issue
// inteira: lá a cena guardada ENTRAVA no lugar de outra, que ia para o acervo
// antes; aqui as duas ficam abertas, cada uma na sua aba. O arquivamento
// preventivo que a ALE-191 inventou deixou de ser necessário porque deixou de
// haver o que perder — nada é substituído.
//
// A posse é conferida como no `ShowPlace` e no `RemovePlace`, e pelo mesmo
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
// Este é o ÚNICO lugar do tabuleiro onde um estado inteiro chega pelo cliente —
// nos outros o cliente manda a intenção ("mova esta peça") e o servidor produz o
// estado. É deliberado: o rascunho não tem concorrência (só o mestre o vê),
// não tem broadcast e não tem vez, então um handler por gesto seria protocolo
// para nada. O preço é este: o que chega tem de ser CONFERIDO antes de virar
// acervo, senão um cliente quebrado guarda lixo que só aparece quando a cena
// chega à mesa.
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
