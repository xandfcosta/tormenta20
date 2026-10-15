package rest

import (
	"context"
	"fmt"
	"log"

	"t20engine/app"
	"t20engine/app/character"
	"t20engine/app/session"
	"t20engine/domain/engine"
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
	sessions *session.Store
	scopes   Scopes
	access   session.Access
	// plays entra por fora porque ele precisa do `*sql.DB` para as transações
	// dele, e este pacote não tem — quem monta os dois é o `serve/api`.
	plays character.Plays
}

func NewParty(
	q *sqlcgen.Queries, sessions *session.Store, catalogs *engine.Catalogs, plays character.Plays,
) Party {
	return Party{
		queries: q, sessions: sessions,
		scopes: NewScopes(q, catalogs), access: session.NewAccess(q), plays: plays,
	}
}

// StartScene é o gesto "Iniciar cena" INTEIRO: a cena abre E o grupo recebe o
// que COMEÇA com ela.
//
// # A ORDEM é o espelho do encerrar, e pela razão inversa
//
// Lá a expiração vem antes porque o estado desligado é o que a mesa vê. Aqui a
// cena abre PRIMEIRO: ela é o que o botão promete, e uma concessão que falha
// não pode deixar o mestre sem cena. É a mesma regra que a postura já tem
// escrita — a concessão que falha não derruba o que foi pago.
//
// A entrega é BEST-EFFORT e CONTADA, como o descanso do grupo: uma ficha que
// falha não pode impedir a cena das outras quatro, e descartar a contagem faria
// a falha sumir.
func (p Party) StartScene(
	ctx context.Context, who app.Caller, campaignID, sessionID int64, kind live.SceneKind,
) (*live.SessionRuntimeState, error) {
	if _, err := p.access.GM(ctx, who, campaignID, sessionID); err != nil {
		return nil, err
	}
	state, err := p.sessions.StartScene(ctx, sessionID, kind)
	if err != nil {
		return nil, fmt.Errorf("abrir a cena da sessão %d: %w", sessionID, err)
	}
	p.grantsWhatBeginsWithTheScene(ctx, campaignID, sessionID)
	return state, nil
}

// grantsWhatBeginsWithTheScene percorre o grupo entregando as reservas de
// início de cena, e registra quem falhou sem derrubar os outros.
func (p Party) grantsWhatBeginsWithTheScene(ctx context.Context, campaignID, sessionID int64) {
	sheets, err := p.memberCharacterIDs(ctx, campaignID)
	if err != nil {
		log.Printf("sessão %d: listar o grupo para o início da cena falhou (%v)", sessionID, err)
		return
	}
	for _, id := range sheets {
		row, err := p.queries.GetCharacter(ctx, id)
		if err != nil {
			log.Printf("sessão %d: carregar o personagem %d no início da cena falhou (%v)", sessionID, id, err)
			continue
		}
		if _, err := p.plays.GrantSceneStartPools(ctx, row); err != nil {
			log.Printf("sessão %d: a concessão de início de cena do personagem %d falhou (%v)",
				sessionID, id, err)
		}
	}
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
	ctx context.Context, who app.Caller, campaignID, sessionID int64,
) (*live.SessionRuntimeState, error) {
	if _, err := p.access.GM(ctx, who, campaignID, sessionID); err != nil {
		return nil, err
	}
	if _, _, err := p.expireScene(ctx, who, campaignID); err != nil {
		return nil, err
	}
	state, err := p.sessions.EndScene(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("zerar a fila da sessão %d: %w", sessionID, err)
	}
	return state, nil
}

// ExpireScene é a MESMA expiração, sem mexer na fila — o botão "Expirar efeitos
// · cena" do rodapé.
func (p Party) ExpireScene(
	ctx context.Context, who app.Caller, campaignID, sessionID int64,
) (done, total int, err error) {
	if _, err := p.access.GM(ctx, who, campaignID, sessionID); err != nil {
		return 0, 0, err
	}
	return p.expireScene(ctx, who, campaignID)
}

// RestForTheDay encerra o dia de cada ficha, cura e espelha os vitais no
// rastreador.
//
// Uma ficha só conta quando ela INTEIRA deu certo — meia ficha descansada não
// conta, senão o ack diz "5 de 5" com dois PV que não foram gravados.
func (p Party) RestForTheDay(
	ctx context.Context, who app.Caller, campaignID, sessionID int64, condition string,
) (done, total int, err error) {
	if _, err := p.access.GM(ctx, who, campaignID, sessionID); err != nil {
		return 0, 0, err
	}
	sheets, err := p.memberCharacterIDs(ctx, campaignID)
	if err != nil {
		return 0, 0, err
	}
	for _, id := range sheets {
		if p.restOne(ctx, who, sessionID, id, condition) {
			done++
		}
	}
	return done, len(sheets), nil
}

// expireScene percorre o grupo SEM conferir o papel — quem confere é o método
// exportado que chamou.
func (p Party) expireScene(
	ctx context.Context, who app.Caller, campaignID int64,
) (done, total int, err error) {
	sheets, err := p.memberCharacterIDs(ctx, campaignID)
	if err != nil {
		return 0, 0, err
	}
	for _, id := range sheets {
		if err := p.scopes.EndScene(ctx, who, id); err != nil {
			log.Printf("campanha %d: encerrar a cena do personagem %d falhou (%v)", campaignID, id, err)
			continue
		}
		done++
	}
	return done, len(sheets), nil
}

// restOne encerra o dia de UMA ficha, cura e espelha. Devolve se a ficha inteira
// deu certo.
func (p Party) restOne(
	ctx context.Context, who app.Caller, sessionID, characterID int64, condition string,
) bool {
	if err := p.scopes.EndDay(ctx, who, characterID); err != nil {
		log.Printf("sessão %d: encerrar o dia do personagem %d falhou (%v)", sessionID, characterID, err)
		return false
	}
	vitals, err := p.scopes.NightRest(ctx, who, characterID, condition)
	if err != nil {
		log.Printf("sessão %d: o descanso do personagem %d falhou (%v)", sessionID, characterID, err)
		return false
	}
	// O ESPELHO QUE FALHA REPROVA A FICHA, como os dois passos acima (ALE-372).
	//
	// Aqui o erro ia para o chão — sem nem um `_ =` para um `grep` achar —, e o
	// mestre ficava escolhendo alvo pelo PV de ANTES do descanso. A tentação é
	// tratá-lo como menos grave que os outros dois, porque a ficha já foi
	// gravada e a linha da fila é só o espelho dela. **É o contrário:** o que a
	// mesa OLHA para decidir quem cura e quem apanha é a fila, então um espelho
	// parado é a mentira chegando exatamente onde ela custa.
	//
	// E a contagem é o que torna isso honesto: "uma ficha só conta quando ela
	// INTEIRA deu certo" é o contrato desta função, e devolver `true` com o
	// espelho parado faria o ack dizer "5 de 5" sobre cinco linhas que a mesa vê
	// desatualizadas. O mestre lê "3 de 5" e clica de novo — o gesto é
	// idempotente, e a segunda passada espelha o que a primeira não conseguiu.
	if err := p.mirrorToTracker(ctx, sessionID, characterID, vitals); err != nil {
		log.Printf("sessão %d: o personagem %d descansou e a linha dele na fila não atualizou (%v)",
			sessionID, characterID, err)
		return false
	}
	return true
}

// mirrorToTracker copia os PV/PM recém-gravados para a linha viva do
// rastreador, quando o personagem está na iniciativa, para as barras mudarem
// sem recarga.
func (p Party) mirrorToTracker(ctx context.Context, sessionID, characterID int64, vitals sheet.RestedVitals) error {
	state, err := p.sessions.State(ctx, sessionID)
	if err != nil {
		return err
	}
	for _, row := range state.Initiative {
		if row.CharacterID != nil && *row.CharacterID == characterID {
			hp, mp := vitals.HpCurrent, vitals.MpCurrent
			_, err := p.sessions.PatchVitals(ctx, sessionID, row.ID, &hp, &mp)
			return err
		}
	}
	return nil
}

// memberCharacterIDs é o id do personagem de cada membro — o conjunto que um
// descanso de sessão inteira percorre.
//
// SEM filtro de papel: o mestre não tem personagem próprio na maioria das
// mesas, os NPCs dele não são membros da campanha, e a coluna `role` foi
// substituída pelo `ownerId` na ALE-287.
func (p Party) memberCharacterIDs(ctx context.Context, campaignID int64) ([]int64, error) {
	rows, err := p.queries.ListMembers(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("listar os membros da campanha %d: %w", campaignID, err)
	}
	ids := make([]int64, 0, len(rows))
	for _, m := range rows {
		ids = append(ids, m.Characterid)
	}
	return ids, nil
}
