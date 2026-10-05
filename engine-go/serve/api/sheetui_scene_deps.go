package api

import (
	"context"
	"fmt"

	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// A CENA DA FICHA, com adaptador próprio: o núcleo mais um `sheetRules`, que é
// onde as regras moram.
type sheetHost struct {
	sceneCore
	rules sheetRules
}

func (s *Server) sheetHost() sheetHost {
	return sheetHost{sceneCore: s.sceneCore(), rules: s.sheetRules()}
}

// O adaptador cumprindo a porta da FICHA (`sheetui.Deps`).
//
// O sinal de que a fronteira está no lugar é nenhum destes métodos desenhar
// nada — e nenhum handler da cena tocar banco fora do `Queries`.

// LoadCharacter e ComputeSheet atravessam pelo adaptador, e não pelo núcleo:
// só a ficha e a Mesa as pedem, e o núcleo é o que quase toda cena pede.
func (h sheetHost) LoadCharacter(ctx context.Context, c sqlcgen.Character) (sheet.CharacterDTO, error) {
	return h.rules.LoadCharacter(ctx, c)
}

// CharacterChanged avisa a MESA que esta ficha mexeu.
func (h sheetHost) CharacterChanged(characterID int64) { h.rules.characterChanged(characterID) }

// ActionFitsOnTurn e SpendActionOnTurn levam o gesto da ficha ao turno da mesa
// em que este personagem está.
//
// Achar a mesa é do hospedeiro porque só ele tem o `session.Store`: a ficha
// pergunta por personagem, e a resposta atravessa a porta já decidida.
func (h sheetHost) ActionFitsOnTurn(ctx context.Context, characterID int64, cost engine.ActionCost) error {
	return h.rules.sessions.CharacterActionFits(ctx, characterID, cost)
}

func (h sheetHost) SpendActionOnTurn(ctx context.Context, characterID int64, cost engine.ActionCost) error {
	return h.rules.sessions.SpendCharacterAction(ctx, characterID, cost)
}

// PublishSkillTest leva o teste rolado na ficha para a mesa (p220-221).
//
// O NOME DE QUEM ROLOU sai daqui e não da cena da ficha: a faixa da mesa diz
// "Arwen · Atletismo", e a ficha não tem por que montar a frase que outra tela
// desenha. Quem sabe o nome é o agregado que o hospedeiro já carrega.
func (h sheetHost) PublishSkillTest(
	ctx context.Context, characterID int64, skill string, test engine.SkillTest, byHand bool,
) error {
	row, err := h.rules.queries.GetCharacter(ctx, characterID)
	if err != nil {
		return fmt.Errorf("carregar o personagem %d para publicar o teste: %w", characterID, err)
	}
	roll := live.SkillTestRoll{
		Who: row.Name, Skill: skill,
		Roll: test.Roll, Modifier: test.Modifier, Total: test.Total,
		Natural20: test.Natural20, Natural1: test.Natural1,
		ByHand: byHand,
	}
	for _, sessionID := range h.liveSessionsOf(ctx, characterID) {
		if _, err := h.rules.sessions.RecordSkillTest(ctx, sessionID, roll); err != nil {
			return err
		}
	}
	return nil
}

// liveSessionsOf são as sessões EM CURSO das mesas onde este personagem está.
//
// PELA CAMPANHA e não pela FILA, e isso é a correção de um buraco que o caso de
// integração pegou: o `LiveSessionsWithCharacter` do regime só acha sessão onde
// o personagem tem LINHA NA INICIATIVA, e um teste é rolado fora de combate o
// tempo todo — a Percepção que abre a cena vem antes de haver cena.
//
// SEM SESSÃO NÃO É ERRO e nem lista vazia é defeito: a ficha aberta sozinha rola
// e mostra o número a quem está olhando; o que não existe é mesa para onde
// publicar. Pelo mesmo motivo os erros de leitura são ENGOLIDOS — uma campanha
// ilegível não pode transformar um teste de perícia numa recusa.
func (h sheetHost) liveSessionsOf(ctx context.Context, characterID int64) []int64 {
	campaigns, err := h.rules.queries.ListCampaignsForCharacter(ctx, characterID)
	if err != nil {
		return nil
	}
	var out []int64
	for _, c := range campaigns {
		// `Campaignid` E NÃO `ID`: a consulta devolve a linha de
		// `campaign_members`, então o `ID` dela é o da LINHA DE ELENCO. Ler o
		// campo errado pede as sessões de outra campanha — ou de nenhuma —, e o
		// teste rolado some sem erro, sem recusa e sem faixa.
		//
		// Passou despercebido porque na bancada a campanha e a linha de elenco
		// eram as duas a de número 1; quem pegou foi o banco de desenvolvimento,
		// onde um jogador está em duas campanhas.
		sessions, err := h.rules.queries.ListSessions(ctx, c.Campaignid)
		if err != nil {
			continue
		}
		for _, s := range sessions {
			if s.Status == "active" {
				out = append(out, s.ID)
			}
		}
	}
	return out
}
