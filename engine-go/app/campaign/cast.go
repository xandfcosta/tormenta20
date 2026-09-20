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

func NewCast(q *sqlcgen.Queries, trava session.Access) Cast {
	return Cast{queries: q, access: trava}
}

// Save grava um NPC NOVO no elenco e devolve o id dele.
//
// A VALIDAÇÃO e a NORMALIZAÇÃO ficam aqui, e não em quem monta o bloco: os dois
// caminhos que criam NPC — copiar um verbete do bestiário e escrever do zero no
// editor — faziam as duas na cena, cada um por sua conta. O que difere entre
// eles é de onde o bloco VEM; o que é um bloco válido é uma regra só.
func (c Cast) Save(
	ctx context.Context, quem app.Caller, campanhaID int64, nome string, bloco creature.Block,
) (int64, error) {
	if _, err := c.access.OwnedCampaign(ctx, quem, campanhaID); err != nil {
		return 0, err
	}
	blob, err := validBlockJSON(nome, bloco)
	if err != nil {
		return 0, err
	}
	agora := dbvalue.NowISO()
	linha, err := c.queries.CreateCampaignCreature(ctx, sqlcgen.CreateCampaignCreatureParams{
		Campaignid: campanhaID, Name: nome, Block: blob, Createdat: agora, Updatedat: agora,
	})
	if err != nil {
		return 0, fmt.Errorf("guardar %q no elenco da campanha %d: %w", nome, campanhaID, err)
	}
	return linha.ID, nil
}

// Update reescreve um NPC que já está no elenco.
func (c Cast) Update(
	ctx context.Context, quem app.Caller, campanhaID, npcID int64, nome string, bloco creature.Block,
) error {
	if _, _, err := c.Block(ctx, quem, campanhaID, npcID); err != nil {
		return err
	}
	blob, err := validBlockJSON(nome, bloco)
	if err != nil {
		return err
	}
	if _, err := c.queries.UpdateCampaignCreature(ctx, sqlcgen.UpdateCampaignCreatureParams{
		ID: npcID, Name: nome, Block: blob, Updatedat: dbvalue.NowISO(),
	}); err != nil {
		return fmt.Errorf("atualizar o npc %d da campanha %d: %w", npcID, campanhaID, err)
	}
	return nil
}

// Erase tira o NPC do elenco.
//
// NÃO mexe na FILA, e a separação é do desenho: "ele não volta mais" e "ele saiu
// desta cena" são duas perguntas, e juntá-las faria o mestre perder o combatente
// em curso ao arrumar a preparação.
func (c Cast) Erase(ctx context.Context, quem app.Caller, campanhaID, npcID int64) (string, error) {
	linha, _, err := c.Block(ctx, quem, campanhaID, npcID)
	if err != nil {
		return "", err
	}
	if err := c.queries.DeleteCampaignCreature(ctx, npcID); err != nil {
		return "", fmt.Errorf("apagar o npc %d da campanha %d: %w", npcID, campanhaID, err)
	}
	return linha.Name, nil
}

// Block lê UM NPC do elenco, com a trava, e já devolve o bloco desserializado.
//
// É o gargalo de leitura dos outros três: quem escreve passa por aqui primeiro,
// e é isso que faz a conferência de campanha existir UMA vez.
func (c Cast) Block(
	ctx context.Context, quem app.Caller, campanhaID, npcID int64,
) (sqlcgen.CampaignCreature, creature.Block, error) {
	var bloco creature.Block
	if _, err := c.access.OwnedCampaign(ctx, quem, campanhaID); err != nil {
		return sqlcgen.CampaignCreature{}, bloco, err
	}
	linha, err := c.queries.GetCampaignCreature(ctx, npcID)
	if err != nil {
		return sqlcgen.CampaignCreature{}, bloco, fmt.Errorf(
			"o npc %d não existe: %w", npcID, app.ErrNotFound)
	}
	if linha.Campaignid != campanhaID {
		return sqlcgen.CampaignCreature{}, bloco, fmt.Errorf(
			"o npc %d não é da campanha %d: %w", npcID, campanhaID, app.ErrForbidden)
	}
	if err := json.Unmarshal([]byte(linha.Block), &bloco); err != nil {
		return linha, bloco, fmt.Errorf("o bloco de %q está ilegível: %w", linha.Name, app.ErrRefused)
	}
	return linha, bloco, nil
}

// List devolve o elenco inteiro, cru. Quem monta o resumo da tela é a cena — o
// que ela desenha de cada NPC é decisão de apresentação.
func (c Cast) List(ctx context.Context, campanhaID int64) ([]sqlcgen.CampaignCreature, error) {
	linhas, err := c.queries.ListCampaignCreatures(ctx, campanhaID)
	if err != nil {
		return nil, fmt.Errorf("listar o elenco da campanha %d: %w", campanhaID, err)
	}
	return linhas, nil
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
	ctx context.Context, quem app.Caller, campanhaID, npcID int64, nome string,
) (int64, error) {
	origem, _, err := c.Block(ctx, quem, campanhaID, npcID)
	if err != nil {
		return 0, err
	}
	agora := dbvalue.NowISO()
	copia, err := c.queries.CreateCampaignCreature(ctx, sqlcgen.CreateCampaignCreatureParams{
		Campaignid: campanhaID, Name: nome, Block: origem.Block,
		Createdat: agora, Updatedat: agora,
	})
	if err != nil {
		return 0, fmt.Errorf("copiar o npc %d da campanha %d: %w", npcID, campanhaID, err)
	}
	return copia.ID, nil
}

// validBlockJSON é o que TODO bloco atravessa antes de virar linha: a validação
// do livro, a normalização e a serialização.
//
// Um erro de validação sai com `app.ErrRefused` porque é recusa de REGRA — o
// mestre digitou algo que um bloco não pode ter —, e não falha de sistema.
func validBlockJSON(nome string, bloco creature.Block) (string, error) {
	if err := creature.Validate(nome, &bloco); err != nil {
		return "", fmt.Errorf("%w: %w", app.ErrRefused, err)
	}
	creature.Normalize(&bloco)
	blob, err := json.Marshal(bloco)
	if err != nil {
		return "", fmt.Errorf("guardar o bloco de %q: %w", nome, errors.Join(err, app.ErrRefused))
	}
	return string(blob), nil
}
