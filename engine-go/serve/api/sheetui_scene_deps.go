package api

import (
	"context"
	"database/sql"
	"net/http"

	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/sheetui"
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

// SaveProficiencies grava as categorias, devolvendo o blob e a lista limpa.
func (h sheetHost) SaveProficiencies(
	ctx context.Context, id int64, categorias []string,
) (string, []string, error) {
	return h.rules.saveProficiencies(ctx, id, categorias)
}

// SaveNewCraft acrescenta a perícia que o livro não tem.
func (h sheetHost) SaveNewCraft(ctx context.Context, id int64, nome string) error {
	return h.rules.saveNewCraft(ctx, id, nome)
}

// ApplySpellBuffEffect liga o efeito de uma magia de melhoria.
func (h sheetHost) ApplySpellBuffEffect(
	ctx context.Context, id int64, magia string, escopo *string,
) (sheet.EffectDTO, int, error) {
	return h.rules.applySpellBuffEffect(ctx, id, magia, escopo)
}

// PowerTempHpAmount é quanto de PV temporário um poder concede.
func (h sheetHost) PowerTempHpAmount(
	r *http.Request, row sqlcgen.Character, atributo string,
) (int, bool) {
	return h.rules.powerTempHpAmount(r.Context(), row, atributo)
}

// ── As ESCRITAS ──────────────────────────────────────────────────────────────
//
// Cena que compõe SQL é cena com o banco dentro. Quem sabe o nome da coluna, o
// que é NULL e se a tabela tem carimbo é o HOSPEDEIRO.

// SaveCustomItem grava nome, quantidade e espaços de um item da mochila.
//
// `espacos` é `float64` porque a coluna `slots` é REAL: a carga do livro conta
// de meio em meio (uma adaga ocupa 1, um bálsamo 0,5, p141).
func (h sheetHost) SaveCustomItem(
	ctx context.Context, itemID int64, nome string, quantidade int64, espacos float64,
) error {
	var set setBuilder
	set.Add("name = ?", nome)
	set.Add("quantity = ?", quantidade)
	set.Add("slots = ?", espacos)
	return set.exec(ctx, h.rules.db, "UPDATE character_items", itemID)
}

// SaveEquipped grava o slot em que o item está vestido, ou NULL.
func (h sheetHost) SaveEquipped(ctx context.Context, itemID int64, valor sql.NullString) error {
	var set setBuilder
	set.Add("equipped = ?", valor)
	return set.exec(ctx, h.rules.db, "UPDATE character_items", itemID)
}

// SaveItemOverlays grava a melhoria e o material escolhidos.
//
// A cena manda a LISTA e o nome do material; a serialização em JSON e a tradução
// de material vazio para NULL são daqui. É `exec` e não `execTouched` porque
// nenhuma das tabelas de item tem `updatedAt`.
func (h sheetHost) SaveItemOverlays(
	ctx context.Context, itemID int64, melhorias []string, material string,
) error {
	var set setBuilder
	set.Add("improvements = ?", sheet.MarshalStrings(&melhorias))
	set.Add("material = ?", sql.NullString{String: material, Valid: material != ""})
	return set.exec(ctx, h.rules.db, "UPDATE character_items", itemID)
}

// SaveChoices grava as colunas de escolha que a cena diz terem mudado.
//
// A cena declara o `ChoiceWrite` e este método o traduz em colunas — mesma
// direção do `ListRow` das campanhas. Nulo é "não toque nesta", e um pedido sem
// nenhuma coluna não vira `UPDATE`: gravar só o carimbo diria que a ficha mudou
// quando ela não mudou, e o carimbo é o que a Mesa lê para repedir a ficha.
func (h sheetHost) SaveChoices(ctx context.Context, id int64, escolhas sheetui.ChoiceWrite) error {
	var set setBuilder
	for _, campo := range []struct {
		coluna string
		valor  *string
	}{
		{"classPowers", escolhas.ClassPowers},
		{"originChoices", escolhas.OriginChoices},
		{"classChoices", escolhas.ClassChoices},
		{"raceAbilityChoices", escolhas.RaceAbilityChoices},
		{"raceAttributeChoices", escolhas.RaceAttributeChoices},
	} {
		if campo.valor != nil {
			set.Add(campo.coluna+" = ?", *campo.valor)
		}
	}
	if set.empty() {
		return nil
	}
	return set.execTouched(ctx, h.rules.db, "UPDATE characters", id)
}

// ApplyPowerTempHp aplica a reserva de PV temporários de um poder.
func (h sheetHost) ApplyPowerTempHp(
	ctx context.Context, id int64, powerID, escopo string, quanto int,
) error {
	_, err := h.rules.applyTempHpPool(ctx, id, "power", powerID, escopo, quanto, "PV temporários")
	return err
}
