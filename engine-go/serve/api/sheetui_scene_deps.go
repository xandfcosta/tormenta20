package api

import (
	"context"
	"fmt"

	"t20engine/app"
	"t20engine/app/combat"
	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/sheetui"
)

// A CENA DA FICHA, com adaptador próprio: o núcleo mais um `sheetRules`, que é
// onde as regras moram.
type sheetHost struct {
	sceneCore
	rules sheetRules
	// strike é o caso de uso de ATACAR, e ele entra aqui porque a superfície
	// Ações rola daqui. É o MESMO que a cena da Mesa recebe — dois caminhos de
	// ataque com montagens diferentes divergiriam no dia em que um deles
	// ganhasse uma porta nova.
	strike combat.Strike
}

func (s *Server) sheetHost() sheetHost {
	return sheetHost{sceneCore: s.sceneCore(), rules: s.sheetRules(), strike: s.combatStrike()}
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

// ProposeStrikeOnTable rola o golpe — ou a manobra — da superfície Ações na
// mesa onde este personagem está.
//
// QUEM ATACA É A VEZ, resolvida CONTRA O BANCO, e é a mesma decisão do
// `proposeStrike` da cena da Mesa: o livro diz que quem age é quem está na vez
// (p231), e deixar o cliente dizer quem ataca seria o jogador rolando o ataque
// do companheiro.
//
// A POSSE JÁ FOI RESOLVIDA pelo `sheetCommand`, que recusa ficha que não é de
// quem pede antes de chegar aqui — é por isso que o `OwnsAttacker` sai `true` e
// o papel sai "player": um mestre que também joga chega por este caminho como
// dono da ficha dele, que é o que ele é.
func (h sheetHost) ProposeStrikeOnTable(
	ctx context.Context, characterID int64, strike sheetui.ActionStrike,
) error {
	row, err := h.rules.queries.GetCharacter(ctx, characterID)
	if err != nil {
		return fmt.Errorf("carregar o personagem %d para rolar o ataque: %w", characterID, err)
	}
	for _, table := range h.liveTablesOf(ctx, characterID) {
		onTurn, err := h.characterOnTurn(ctx, table.SessionID, characterID)
		if err != nil || onTurn == nil {
			continue
		}
		_, err = h.strike.Propose(ctx, app.Caller{ID: row.Ownerid}, "player", combat.Request{
			CampaignID:      table.CampaignID,
			SessionID:       table.SessionID,
			AttackerEntryID: onTurn.ID,
			TargetEntryID:   strike.TargetEntryID,
			Weapon:          strike.Weapon,
			Maneuver:        strike.Maneuver,
			OwnsAttacker:    true,
		})
		return err
	}
	// NENHUMA MESA COM A VEZ DELE é recusa e não silêncio: o gesto saiu de um
	// botão que só existe dentro de uma sessão, então chegar aqui sem vez quer
	// dizer que o turno virou entre o desenho e o clique. Calar faria o botão
	// parecer quebrado.
	return fmt.Errorf("não é a vez de %s: quem age é quem está na vez (p231)", row.Name)
}

// characterOnTurn devolve a linha da vez QUANDO ela é deste personagem, e nula
// quando não é — inclusive fora de combate, que é o caso de `TurnIndex` fora da
// faixa.
func (h sheetHost) characterOnTurn(
	ctx context.Context, sessionID, characterID int64,
) (*live.InitiativeEntry, error) {
	state, err := h.rules.sessions.State(ctx, sessionID)
	if err != nil || state == nil {
		return nil, err
	}
	if state.TurnIndex < 0 || state.TurnIndex >= len(state.Initiative) {
		return nil, nil
	}
	onTurn := state.Initiative[state.TurnIndex]
	if onTurn.CharacterID == nil || *onTurn.CharacterID != characterID {
		return nil, nil
	}
	return &onTurn, nil
}

// liveTable é uma mesa aberta, com as DUAS coordenadas.
//
// As duas e não só a sessão: o bloco de criatura do alvo é da CAMPANHA, e sem
// ela o combate leria o bloco de outra mesa (ALE-377).
type liveTable struct {
	CampaignID int64
	SessionID  int64
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
	var out []int64
	for _, table := range h.liveTablesOf(ctx, characterID) {
		out = append(out, table.SessionID)
	}
	return out
}

// liveTablesOf é a mesma varredura, devolvendo as duas coordenadas.
func (h sheetHost) liveTablesOf(ctx context.Context, characterID int64) []liveTable {
	campaigns, err := h.rules.queries.ListCampaignsForCharacter(ctx, characterID)
	if err != nil {
		return nil
	}
	var out []liveTable
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
				out = append(out, liveTable{CampaignID: c.Campaignid, SessionID: s.ID})
			}
		}
	}
	return out
}
