package rest

import (
	"context"
	"fmt"
	"log"

	"t20engine/app"
	"t20engine/app/session"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// Party são os mesmos gestos, aplicados ao GRUPO INTEIRO pelo mestre.
//
// # Best-effort por ficha, mas CONTADO
//
// Uma ficha que falha não pode impedir o descanso das outras quatro — só que o
// resultado volta em número: descartar a contagem faz o mestre ler "descansou"
// com duas de cinco fichas de fora. Best-effort é sobre continuar apesar da
// falha, não sobre escondê-la.
type Party struct {
	queries  *sqlcgen.Queries
	sessions *live.SessionStore
	scopes   Scopes
	access   session.Access
}

func NewParty(q *sqlcgen.Queries, sessions *live.SessionStore) Party {
	return Party{queries: q, sessions: sessions, scopes: NewScopes(q), access: session.NewAccess(q)}
}

// EndScene é o gesto "Encerrar cena" INTEIRO: a duração "cena" acaba para o
// grupo E a fila volta ao começo.
//
// # A ORDEM importa, e a recusa também
//
// A expiração vem ANTES porque o estado desligado é o que a mesa vê: desligar a
// cena e só então falhar deixaria o mestre com a fila zerada e as bênçãos vivas
// — o defeito da ALE-220 outra vez, agora com o botão parecendo ter funcionado.
// Falha aqui é falha do gesto inteiro, e o mestre clica de novo.
//
// É por isso que os dois passos moram na MESMA função em vez de a cena chamá-los
// em sequência: a ordem é a regra, e uma sequência escrita no chamador é uma
// sequência que o segundo chamador escreve ao contrário.
func (p Party) EndScene(
	ctx context.Context, quem app.Caller, campaignID, sessionID int64,
) (*live.SessionRuntimeState, error) {
	if _, err := p.access.GM(ctx, quem, campaignID, sessionID); err != nil {
		return nil, err
	}
	if _, _, err := p.expireScene(ctx, quem, campaignID); err != nil {
		return nil, err
	}
	estado, err := p.sessions.EndScene(sessionID)
	if err != nil {
		return nil, fmt.Errorf("zerar a fila da sessão %d: %w", sessionID, err)
	}
	return estado, nil
}

// ExpireScene é a MESMA expiração, sem mexer na fila — o botão "Expirar efeitos
// · cena" do rodapé.
func (p Party) ExpireScene(
	ctx context.Context, quem app.Caller, campaignID, sessionID int64,
) (feitos, total int, err error) {
	if _, err := p.access.GM(ctx, quem, campaignID, sessionID); err != nil {
		return 0, 0, err
	}
	return p.expireScene(ctx, quem, campaignID)
}

// RestForTheDay encerra o dia de cada ficha, cura e espelha os vitais no
// rastreador.
//
// Uma ficha só conta quando ela INTEIRA deu certo — meia ficha descansada não
// conta, senão o ack diz "5 de 5" com dois PV que não foram gravados.
func (p Party) RestForTheDay(
	ctx context.Context, quem app.Caller, campaignID, sessionID int64, condicao string,
) (feitos, total int, err error) {
	if _, err := p.access.GM(ctx, quem, campaignID, sessionID); err != nil {
		return 0, 0, err
	}
	fichas, err := p.memberCharacterIDs(ctx, campaignID)
	if err != nil {
		return 0, 0, err
	}
	for _, id := range fichas {
		if p.restOne(ctx, quem, sessionID, id, condicao) {
			feitos++
		}
	}
	return feitos, len(fichas), nil
}

// expireScene percorre o grupo SEM conferir o papel — quem confere é o método
// exportado que chamou.
func (p Party) expireScene(
	ctx context.Context, quem app.Caller, campaignID int64,
) (feitos, total int, err error) {
	fichas, err := p.memberCharacterIDs(ctx, campaignID)
	if err != nil {
		return 0, 0, err
	}
	for _, id := range fichas {
		if err := p.scopes.EndScene(ctx, quem, id); err != nil {
			log.Printf("campanha %d: encerrar a cena do personagem %d falhou (%v)", campaignID, id, err)
			continue
		}
		feitos++
	}
	return feitos, len(fichas), nil
}

// restOne encerra o dia de UMA ficha, cura e espelha. Devolve se a ficha inteira
// deu certo.
func (p Party) restOne(
	ctx context.Context, quem app.Caller, sessionID, characterID int64, condicao string,
) bool {
	if err := p.scopes.EndDay(ctx, quem, characterID); err != nil {
		log.Printf("sessão %d: encerrar o dia do personagem %d falhou (%v)", sessionID, characterID, err)
		return false
	}
	vitais, err := p.scopes.NightRest(ctx, quem, characterID, condicao)
	if err != nil {
		log.Printf("sessão %d: o descanso do personagem %d falhou (%v)", sessionID, characterID, err)
		return false
	}
	p.mirrorToTracker(sessionID, characterID, vitais)
	return true
}

// mirrorToTracker copia os PV/PM recém-gravados para a linha viva do
// rastreador, quando o personagem está na iniciativa, para as barras mudarem
// sem recarga.
func (p Party) mirrorToTracker(sessionID, characterID int64, vitais sheet.RestedVitals) {
	for _, linha := range p.sessions.GetState(sessionID).Initiative {
		if linha.CharacterID != nil && *linha.CharacterID == characterID {
			hp, mp := vitais.HpCurrent, vitais.MpCurrent
			_, _ = p.sessions.PatchVitals(sessionID, linha.ID, &hp, &mp)
			return
		}
	}
}

// memberCharacterIDs é o id do personagem de cada membro — o conjunto que um
// descanso de sessão inteira percorre.
//
// SEM filtro de papel: o mestre não tem personagem próprio na maioria das
// mesas, os NPCs dele não são membros da campanha, e a coluna `role` foi
// substituída pelo `ownerId` na ALE-287.
func (p Party) memberCharacterIDs(ctx context.Context, campaignID int64) ([]int64, error) {
	linhas, err := p.queries.ListMembers(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("listar os membros da campanha %d: %w", campaignID, err)
	}
	ids := make([]int64, 0, len(linhas))
	for _, m := range linhas {
		ids = append(ids, m.Characterid)
	}
	return ids, nil
}
