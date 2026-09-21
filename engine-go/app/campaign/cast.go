package campaign

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"t20engine/app"
	"t20engine/app/session"
	"t20engine/domain/creature"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// O ELENCO DE NPCs de uma campanha — a tabela `campaign_creatures`.
//
// # Por que ele tem dono, e por que o dono é a CAMPANHA
//
// O NPC preparado na quinta sobrevive à sessão de sábado: a linha é do acervo da
// campanha, e a Mesa é UMA ENTRADA do gesto, não a proprietária. Antes disto a
// tabela tinha dois donos — a cena da Mesa escrevia quatro vezes e o
// `app/initiative` clonava blocos —, e a conferência de campanha existia nos
// dois, com um comentário na cena avisando contra exatamente isso: *"Duas cópias
// dariam duas travas, e a que envelhecesse seria a de menos uso"*. Ele estava a
// dois arquivos da segunda cópia (ALE-353).
//
// # A trava é uma só, e é a do dono da campanha
//
// O id do NPC vem do CAMINHO ou do rascunho, e os dois são digitáveis. Sem
// conferir a campanha, o mestre de uma mesa alcança o elenco de outra — e o
// elenco é o material mais privado que um mestre tem. Aqui ela vale para os
// quatro gestos, e devolve recusa TIPADA: quem traduz para frase é a cena, que
// é o que o `serve/web/table/session.go` já faz com os outros casos de uso.
type Cast struct {
	queries *sqlcgen.Queries
	access  session.Access
}

func NewCast(q *sqlcgen.Queries, lock session.Access) Cast {
	return Cast{queries: q, access: lock}
}

// Save grava um NPC NOVO no elenco e devolve o id dele.
//
// A VALIDAÇÃO e a NORMALIZAÇÃO ficam aqui, e não em quem monta o bloco: os dois
// caminhos que criam NPC — copiar um verbete do bestiário e escrever do zero no
// editor — faziam as duas na cena, cada um por sua conta. O que difere entre
// eles é de onde o bloco VEM; o que é um bloco válido é uma regra só.
func (c Cast) Save(
	ctx context.Context, who app.Caller, campaignID int64, name string, block creature.Block,
) (int64, error) {
	if _, err := c.access.OwnedCampaign(ctx, who, campaignID); err != nil {
		return 0, err
	}
	blob, err := validBlockJSON(name, block)
	if err != nil {
		return 0, err
	}
	now := dbvalue.NowISO()
	row, err := c.queries.CreateCampaignCreature(ctx, sqlcgen.CreateCampaignCreatureParams{
		Campaignid: campaignID, Name: name, Block: blob, Createdat: now, Updatedat: now,
	})
	if err != nil {
		return 0, fmt.Errorf("guardar %q no elenco da campanha %d: %w", name, campaignID, err)
	}
	return row.ID, nil
}

// Update reescreve um NPC que já está no elenco.
func (c Cast) Update(
	ctx context.Context, who app.Caller, campaignID, npcID int64, name string, block creature.Block,
) error {
	if _, _, err := c.Block(ctx, who, campaignID, npcID); err != nil {
		return err
	}
	blob, err := validBlockJSON(name, block)
	if err != nil {
		return err
	}
	if _, err := c.queries.UpdateCampaignCreature(ctx, sqlcgen.UpdateCampaignCreatureParams{
		ID: npcID, Name: name, Block: blob, Updatedat: dbvalue.NowISO(),
	}); err != nil {
		return fmt.Errorf("atualizar o npc %d da campanha %d: %w", npcID, campaignID, err)
	}
	return nil
}

// Erase tira o NPC do elenco.
//
// NÃO mexe na FILA, e a separação é do desenho: "ele não volta mais" e "ele saiu
// desta cena" são duas perguntas, e juntá-las faria o mestre perder o combatente
// em curso ao arrumar a preparação.
func (c Cast) Erase(ctx context.Context, who app.Caller, campaignID, npcID int64) (string, error) {
	row, _, err := c.Block(ctx, who, campaignID, npcID)
	if err != nil {
		return "", err
	}
	if err := c.queries.DeleteCampaignCreature(ctx, npcID); err != nil {
		return "", fmt.Errorf("apagar o npc %d da campanha %d: %w", npcID, campaignID, err)
	}
	return row.Name, nil
}

// Block lê UM NPC do elenco, com a trava, e já devolve o bloco desserializado.
//
// É o gargalo de leitura dos outros três: quem escreve passa por aqui primeiro,
// e é isso que faz a conferência de campanha existir UMA vez.
func (c Cast) Block(
	ctx context.Context, who app.Caller, campaignID, npcID int64,
) (sqlcgen.CampaignCreature, creature.Block, error) {
	var block creature.Block
	if _, err := c.access.OwnedCampaign(ctx, who, campaignID); err != nil {
		return sqlcgen.CampaignCreature{}, block, err
	}
	row, err := c.queries.GetCampaignCreature(ctx, npcID)
	if err != nil {
		return sqlcgen.CampaignCreature{}, block, fmt.Errorf(
			"o npc %d não existe: %w", npcID, app.ErrNotFound)
	}
	if row.Campaignid != campaignID {
		return sqlcgen.CampaignCreature{}, block, fmt.Errorf(
			"o npc %d não é da campanha %d: %w", npcID, campaignID, app.ErrForbidden)
	}
	if err := json.Unmarshal([]byte(row.Block), &block); err != nil {
		return row, block, fmt.Errorf("o bloco de %q está ilegível: %w", row.Name, app.ErrRefused)
	}
	return row, block, nil
}

// List devolve o elenco inteiro, cru. Quem monta o resumo da tela é a cena — o
// que ela desenha de cada NPC é decisão de apresentação.
func (c Cast) List(ctx context.Context, campaignID int64) ([]sqlcgen.CampaignCreature, error) {
	rows, err := c.queries.ListCampaignCreatures(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("listar o elenco da campanha %d: %w", campaignID, err)
	}
	return rows, nil
}

// CloneBlock copia um NPC do elenco sob outro nome — o "chefe que ganha nome".
//
// Ele morava no `app/initiative`, que é o pacote da FILA: clonar é gesto do
// ACERVO, e a fila só era onde o mestre clicava. Era ele a segunda cópia da
// trava de campanha.
//
// Clonar só importa quando o mestre vai EDITAR uma das linhas: sem a cópia, dar
// 30 PV ao chefe daria aos outros três zumbis também, porque o bloco é um MOLDE
// que as linhas dividem. E é o BLOCO e não a ficha: clonar personagem exigiria
// matricular a cópia na campanha, e todo membro aparece no painel do Grupo.
func (c Cast) CloneBlock(
	ctx context.Context, who app.Caller, campaignID, npcID int64, name string,
) (int64, error) {
	origin, _, err := c.Block(ctx, who, campaignID, npcID)
	if err != nil {
		return 0, err
	}
	now := dbvalue.NowISO()
	dup, err := c.queries.CreateCampaignCreature(ctx, sqlcgen.CreateCampaignCreatureParams{
		Campaignid: campaignID, Name: name, Block: origin.Block,
		Createdat: now, Updatedat: now,
	})
	if err != nil {
		return 0, fmt.Errorf("copiar o npc %d da campanha %d: %w", npcID, campaignID, err)
	}
	return dup.ID, nil
}

// validBlockJSON é o que TODO bloco atravessa antes de virar linha: a validação
// do livro, a normalização e a serialização.
//
// Um erro de validação sai com `app.ErrRefused` porque é recusa de REGRA — o
// mestre digitou algo que um bloco não pode ter —, e não falha de sistema.
func validBlockJSON(name string, block creature.Block) (string, error) {
	if err := creature.Validate(name, &block); err != nil {
		return "", fmt.Errorf("%w: %w", app.ErrRefused, err)
	}
	creature.Normalize(&block)
	blob, err := json.Marshal(block)
	if err != nil {
		return "", fmt.Errorf("guardar o bloco de %q: %w", name, errors.Join(err, app.ErrRefused))
	}
	return string(blob), nil
}
