package initiative

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"t20engine/app"
	"t20engine/app/session"
	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// Combatant resolve os números de UM personagem para uma linha de fila,
// cobrando as regras da campanha.
//
// A ORDEM da checagem é personagem → campanha → filiação → autorização, e ela
// importa: perguntar a filiação antes de a campanha existir não distingue
// "campanha que não existe" de "personagem que não é dela".
//
// Quem pede tem de ser o DONO do personagem ou o mestre da campanha. É mais
// estreito que a trava da ficha (que aceita o admin): pôr alguém na fila é um
// gesto de mesa, e a mesa tem dono.
func (r Roster) Combatant(
	ctx context.Context, quem app.Caller, campaignID, characterID int64,
) (Combatant, error) {
	ficha, err := r.queries.GetCharacter(ctx, characterID)
	if errors.Is(err, sql.ErrNoRows) {
		return Combatant{}, fmt.Errorf("o personagem %d não existe: %w", characterID, app.ErrNotFound)
	}
	if err != nil {
		return Combatant{}, fmt.Errorf("carregar o personagem %d: %w", characterID, err)
	}
	campanha, err := r.queries.GetCampaign(ctx, campaignID)
	if errors.Is(err, sql.ErrNoRows) {
		return Combatant{}, fmt.Errorf("a campanha %d não existe: %w", campaignID, app.ErrNotFound)
	}
	if err != nil {
		return Combatant{}, fmt.Errorf("carregar a campanha %d: %w", campaignID, err)
	}
	membro, err := r.queries.IsCharacterMember(ctx, sqlcgen.IsCharacterMemberParams{
		Campaignid: campaignID, Characterid: characterID,
	})
	if err != nil {
		return Combatant{}, fmt.Errorf("conferir a filiação do personagem %d: %w", characterID, err)
	}
	if !membro {
		return Combatant{}, fmt.Errorf(
			"o personagem %d não é membro da campanha %d: %w", characterID, campaignID, app.ErrRefused)
	}
	if quem.ID != ficha.Ownerid && quem.ID != campanha.Ownerid {
		return Combatant{}, fmt.Errorf(
			"%d não mestra a campanha %d nem é dono do personagem %d: %w",
			quem.ID, campaignID, characterID, app.ErrForbidden)
	}
	poco, err := r.onePool(ctx, characterID)
	if err != nil {
		return Combatant{}, err
	}
	return Combatant{
		CharacterID: characterID, Name: ficha.Name,
		HpCurrent: poco.HpCurrent, HpMax: poco.HpMax,
		MpCurrent: poco.MpCurrent, MpMax: poco.MpMax,
	}, nil
}

// onePool é o poço derivado de um personagem só.
//
// As quatro colunas de `characters` ainda existem e ainda batem, porque o funil
// as espelha — mas elas saem, e ler a regra pelo espelho é o hábito que faria a
// saída delas quebrar a fila em silêncio (ALE-355).
func (r Roster) onePool(ctx context.Context, characterID int64) (sheet.Pools, error) {
	pocos, err := sheet.PoolsForCharacters(ctx, r.queries, r.catalogs, []int64{characterID})
	if err != nil {
		return sheet.Pools{}, fmt.Errorf("derivar o poço do personagem %d: %w", characterID, err)
	}
	return pocos[characterID], nil
}

// Aqui morava o clone do bloco de criatura, e ele não existe mais neste pacote:
// escreve `campaign_creatures`, que é acervo da CAMPANHA, e este é o pacote da
// FILA. Ele carregava a segunda cópia da trava de campanha — a primeira estava
// na cena da Mesa, com um comentário avisando contra exatamente isso. Hoje é o
// `campaign.Cast.CloneBlock` (ALE-353).

// PartyCombatants é todo personagem da campanha com os vitais vivos — o "pôr o
// grupo na fila" de um clique do mestre.
//
// SEM filtro de papel, e isso é a regra e não um esquecimento: o mestre não tem
// personagem próprio na maioria das mesas, os NPCs dele não são membros da
// campanha, e a coluna `role` foi substituída pelo `ownerId` na ALE-287.
// Filtrar aqui esconderia da fila o bardo que o mestre também joga.
func (r Roster) PartyCombatants(ctx context.Context, campaignID int64) ([]Combatant, error) {
	linhas, err := r.queries.ListMembers(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("listar os membros da campanha %d: %w", campaignID, err)
	}
	ids := make([]int64, len(linhas))
	for i, m := range linhas {
		ids[i] = m.Characterid
	}
	// Os poços do grupo INTEIRO de uma vez: o `ListMembers` traz as quatro
	// colunas espelhadas, e usá-las seria ler pelo espelho a regra que já tem
	// dono.
	pocos, err := sheet.PoolsForCharacters(ctx, r.queries, r.catalogs, ids)
	if err != nil {
		return nil, fmt.Errorf("derivar os poços da campanha %d: %w", campaignID, err)
	}
	grupo := make([]Combatant, 0, len(linhas))
	for _, m := range linhas {
		poco := pocos[m.Characterid]
		grupo = append(grupo, Combatant{
			CharacterID: m.Characterid, Name: m.Charname,
			HpCurrent: poco.HpCurrent, HpMax: poco.HpMax,
			MpCurrent: poco.MpCurrent, MpMax: poco.MpMax,
		})
	}
	return grupo, nil
}

// Queue é a fila de uma sessão, com o store por trás.
type Queue struct {
	roster   Roster
	sessions *session.Store
}

func NewQueue(q *sqlcgen.Queries, catalogs *engine.Catalogs, sessions *session.Store) Queue {
	return Queue{roster: NewRoster(q, catalogs), sessions: sessions}
}

// Roster é o montador de linhas, exposto para quem precisa só dele.
func (q Queue) Roster() Roster { return q.roster }

// PopulateParty põe na fila, com iniciativa 0 e vitais vivos, cada combatente
// que ainda não está lá.
//
// IDEMPOTENTE de propósito: o botão continua clicável, e o mestre que aceitou um
// jogador atrasado clica de novo e leva só o que faltava.
//
// Devolve o estado mais recente JUNTO com o primeiro erro de `Add`: pôr quatro
// dos cinco e tropeçar no quinto deixa a mesa com quatro combatentes novos, e é
// esse o estado que as outras telas precisam receber.
func (q Queue) PopulateParty(
	sessionID int64, quem []Combatant,
) (*live.SessionRuntimeState, error) {
	jaEstao := map[int64]bool{}
	for _, linha := range q.sessions.GetState(sessionID).Initiative {
		if linha.CharacterID != nil {
			jaEstao[*linha.CharacterID] = true
		}
	}
	var estado *live.SessionRuntimeState
	for _, c := range quem {
		if jaEstao[c.CharacterID] {
			continue
		}
		id, pvAtual, pvMax, pmAtual, pmMax := c.CharacterID, c.HpCurrent, c.HpMax, c.MpCurrent, c.MpMax
		novo, err := q.sessions.AddInitiativeEntry(sessionID, live.InitiativeEntry{
			Label: c.Name, Initiative: 0, Type: "character", CharacterID: &id,
			HpCurrent: &pvAtual, HpMax: &pvMax, MpCurrent: &pmAtual, MpMax: &pmMax,
		})
		if err != nil {
			return estado, fmt.Errorf("pôr %q na fila da sessão %d: %w", c.Name, sessionID, err)
		}
		estado = novo
	}
	return estado, nil
}
