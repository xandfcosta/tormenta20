package api

import (
	"context"
	"database/sql"
	"net/http"

	"t20engine/db/sqlcgen"
	"t20engine/sheet"
	"t20engine/web/sheetui"
)

// A CENA DA FICHA, com adaptador próprio (ALE-278, fatia 6).
//
// Das dezoito assinaturas que a porta pede, quatro são do núcleo e catorze são
// escrita de ficha. É a maior porta do projeto e a que mais mudou de dono nesta
// fatia: o adaptador é o núcleo mais um `sheetRules`, que é onde as regras
// foram morar.
type sheetHost struct {
	sceneCore
	rules sheetRules
}

func (s *Server) sheetHost() sheetHost {
	return sheetHost{sceneCore: s.sceneCore(), rules: s.sheetRules()}
}

// O adaptador cumprindo a porta da FICHA (`sheetui.Deps`, ALE-278).
//
// Ela é a maior cena do repositório e a porta tem dezoito métodos — contra onze
// das campanhas e dois do trilho do mestre. O tamanho não é vício: cinco deles
// são ESCRITA, e três nasceram nesta fatia trocando SQL que a cena montava à mão
// pela pergunta correspondente.
//
// Como nas outras, o sinal de que a fronteira está no lugar é nenhum destes
// métodos desenhar nada — e nenhum handler da cena tocar banco fora do
// `Queries`.

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

// CastSpell gasta o PM e resolve os aprimoramentos.
func (h sheetHost) CastSpell(
	r *http.Request, dto sheet.CharacterDTO, magia string, aprimoramentos []sheet.AugmentPick,
) error {
	return h.rules.castSpellForCharacter(r, dto, magia, aprimoramentos)
}

// ConsumeItem gasta uma dose do consumível.
//
// O RESULTADO não atravessa: a cena descarta a dose inteira, e a única recusa
// que ela precisa — a porção diária — já chega como erro. É a regra da menor
// pergunta chegando no mesmo lugar que o leitor: às vezes ela é nenhuma. O
// `doseUsed` carrega o corpo da resposta JSON, com tag `json:` em cada campo, e
// uma tela que o lesse dependeria do formato de um endpoint que ela não serve.
func (h sheetHost) ConsumeItem(
	r *http.Request, row sqlcgen.Character, itemID int64, pvRolado, pmRolado *int64,
) error {
	_, err := h.rules.consumeItemForCharacter(r.Context(), row, itemID, pvRolado, pmRolado)
	return err
}

// ApplyClassLevel sobe ou desce uma classe.
//
// O hospedeiro devolve quatro valores e um deles é o `storedVitals`, que é tipo
// DELE — uma porta que devolvesse isso não seria porta. A cena redesenha a ficha
// inteira depois de gravar, então ela não precisa de nenhum dos quatro.
func (h sheetHost) ApplyClassLevel(r *http.Request, id int64, classe string, nivel int64) error {
	row, err := h.rules.queries.GetCharacter(r.Context(), id)
	if err != nil {
		return err
	}
	_, _, _, _, err = h.rules.applyClassLevel(r, row, classe, nivel)
	return err
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
	return h.rules.powerTempHpAmount(r, row, atributo)
}

// ── As CINCO escritas ────────────────────────────────────────────────────────
//
// As três últimas substituem `setBuilder` + `"UPDATE …"` montados dentro da
// cena. Cena que compõe SQL é cena com o banco dentro, e o remédio é sempre o
// mesmo: quem sabe o nome da coluna, o que é NULL e se a tabela tem carimbo é o
// hospedeiro — a decisão que o `SaveText` das campanhas deixou escrita.

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
// A cena manda a LISTA e o nome do material; a serialização em JSON e a
// tradução de material vazio para NULL são daqui. Nenhuma das duas tabelas de
// item tem `updatedAt`, e é por isso que o `exec` e não o `execTouched`.
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
//
// A cena tinha a transação inteira escrita dentro dela, e o `applyPool` da rota
// JSON tinha a MESMA sequência — duas cópias de um `BeginTx` sobre a mesma
// regra. As duas passaram pelo `applyPoolTx`, e então a rota JSON foi apagada
// na ALE-277: sobrou uma chamadora e a extração continua valendo, porque a
// conta é do `sheet`, "considere apenas o maior valor" (p256).
func (h sheetHost) ApplyPowerTempHp(
	ctx context.Context, id int64, powerID, escopo string, quanto int,
) error {
	_, _, err := h.rules.applyPoolTx(ctx, id, "power", powerID, escopo, quanto, "PV temporários")
	return err
}
