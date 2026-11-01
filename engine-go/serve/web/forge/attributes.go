package forge

import (
	"fmt"
	"net/http"
	"strconv"
	"t20engine/domain/book"
	"t20engine/domain/sheet"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/domain/engine"

	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/ui"
)

// A SEGUNDA CENA DA FORJA: distribuir os atributos (p17, Tabela 1-1).
//
// Ela vem DEPOIS do nascimento e não antes, e é o que faz a forja curta caber
// em duas telas: o herói já existe, então cada `+` e cada `−` é um comando
// sobre uma linha do banco — a mesma forma dos comandos da ficha, com a mesma
// recusa em 200 e a cena inteira de volta.
//
// A escolha é LIVRE, e a compra de pontos só INFORMA. O livro dá dois métodos
// (p17), e travar nos pontos deixaria sem saída quem rolou na mesa: as rolagens
// dão mais de um negativo e dão −2. A conta dos pontos continua sendo do motor
// (`engine.PointBuySpent`); esta cena só a mostra.

// attributesView é a cena inteira.
type attributesView struct {
	ID       int64
	HeroName string
	Rows     []attributeRow
	// PointsLine é o contador, e ele não recusa nada.
	PointsLine string
	// Refusal é a frase da recusa. Ela é CONTEÚDO e não status: o Datastar
	// descarta o remendo de uma resposta que não é 2xx, então uma recusa
	// devolvida em 422 deixaria a tela parada e muda.
	Refusal string
}

// attributeRow é uma das seis linhas.
type attributeRow struct {
	Key   string
	Label string
	// Base é o que a compra de pontos gasta; Total é o que a ficha usa, já com
	// o modificador da raça somado pelo motor.
	Base  int
	Total int
	// Sum é de onde vem o total, em palavras; vazio quando nada soma ao escolhido.
	Sum string
}

// handleAttributes desenha a distribuição.
func (s Scene) handleAttributes(w http.ResponseWriter, r *http.Request) {
	v, status, err := s.loadAttributes(r, "")
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	s.writeAttributes(w, r, v)
}

// handleAttributeStep soma o passo a um atributo e redesenha.
//
// O passo vai no CAMINHO e não num sinal, como o do vital na ficha: o valor é
// do botão que foi clicado, e doze botões não disputam um sinal só.
func (s Scene) handleAttributeStep(w http.ResponseWriter, r *http.Request) {
	refusal, status, err := s.stepAttribute(r)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	v, status, err := s.loadAttributes(r, refusal)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	s.writeAttributes(w, r, v)
}

// stepAttribute grava o espalhamento novo, ou devolve a frase da recusa.
func (s Scene) stepAttribute(r *http.Request) (refusal string, status int, err error) {
	row, status, err := s.heroOfTheForge(r)
	if err != nil {
		return "", status, err
	}
	step, err := stepFromURL(r)
	if err != nil {
		return "", http.StatusBadRequest, err
	}
	key := chi.URLParam(r, "atributo")
	spread := heroSpread(row)
	if _, known := spread[key]; !known {
		return "", http.StatusBadRequest, fmt.Errorf(
			"atributo %q não existe: são os seis do livro", key)
	}
	spread[key] += step
	if refusal, err := s.leavesChosenExpertises(r, row, spread); err != nil || refusal != "" {
		return refusal, http.StatusOK, err
	}
	if err := s.births.SpreadAttributes(r.Context(), row.ID, spread); err != nil {
		return "", http.StatusInternalServerError, err
	}
	// A Constituição mexe no PV máximo (p34), e NÃO HÁ NADA A FAZER sobre isso.
	//
	// Aqui morava um passo que regravava as quatro colunas de espelho, e ele
	// deixou de existir junto com elas na 00015: o máximo é derivado e o atual
	// é `máximo − dano`, então o teto se move sozinho e a dívida fica. É o que
	// impede o ciclo `−`/`+` de virar uma bomba de cura de dois cliques, e hoje
	// sai de graça em vez de sair de uma conta de delta (ALE-355).
	return "", http.StatusOK, nil
}

// heroSpread lê os seis atributos base da linha do banco.
func heroSpread(row sqlcgen.Character) map[string]int {
	return map[string]int{
		"strength": int(row.Strength), "dexterity": int(row.Dexterity),
		"constitution": int(row.Constitution), "intelligence": int(row.Intelligence),
		"wisdom": int(row.Wisdom), "charisma": int(row.Charisma),
	}
}

// leavesChosenExpertises recusa o espalhamento que baixaria a Inteligência
// abaixo das perícias escolhidas no Especialista (p73, ALE-501). A recusa é
// frase da cena, como a da compra de pontos.
func (s Scene) leavesChosenExpertises(r *http.Request, row sqlcgen.Character, spread map[string]int) (string, error) {
	dto, err := sheet.Load(r.Context(), s.deps.Queries(), s.deps.Catalogs(), row)
	if err != nil {
		return "", err
	}
	if !sheet.HasExpertiseChoice(dto.PowerChoices) &&
		!sheet.HasRaceAbilityPicks(row.Raceattributechoices, row.Secondaryracechoices) {
		return "", nil
	}
	refused := sheet.ChangeKeepsChosenExpertises(dto, func(d *sheet.CharacterDTO) {
		d.Strength, d.Dexterity = int64(spread["strength"]), int64(spread["dexterity"])
		d.Constitution, d.Intelligence = int64(spread["constitution"]), int64(spread["intelligence"])
		d.Wisdom, d.Charisma = int64(spread["wisdom"]), int64(spread["charisma"])
	})
	if refused != nil {
		return refused.Error(), nil
	}
	return "", nil
}

// heroOfTheForge acha o herói e confere a POSSE — o mesmo gargalo da ficha:
// quem não é dono não distribui atributo nenhum.
func (s Scene) heroOfTheForge(r *http.Request) (sqlcgen.Character, int, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return sqlcgen.Character{}, http.StatusBadRequest, fmt.Errorf("id inválido: %q", chi.URLParam(r, "id"))
	}
	row, err := s.deps.Queries().GetCharacter(r.Context(), id)
	if err != nil {
		return sqlcgen.Character{}, http.StatusNotFound, fmt.Errorf("personagem %d não existe", id)
	}
	if row.Ownerid != s.deps.CurrentUserID(r) {
		return sqlcgen.Character{}, http.StatusForbidden, fmt.Errorf("este herói não é seu")
	}
	return row, http.StatusOK, nil
}

// loadAttributes monta a cena a partir do que está gravado.
func (s Scene) loadAttributes(r *http.Request, refusal string) (attributesView, int, error) {
	row, status, err := s.heroOfTheForge(r)
	if err != nil {
		return attributesView{}, status, err
	}
	sheet, err := sheet.LoadAndCompute(r.Context(), s.deps.Queries(), s.deps.Catalogs(), row)
	if err != nil {
		return attributesView{}, http.StatusInternalServerError, err
	}
	spread := heroSpread(row)
	v := attributesView{
		ID: row.ID, HeroName: row.Name,
		PointsLine: pointsLine(engine.PointBuySpent(spread)),
		Refusal:    refusal,
	}
	for _, attribute := range book.AttributeOrder {
		v.Rows = append(v.Rows, attributeRowOf(attribute.Key, spread, sheet))
	}
	return v, http.StatusOK, nil
}

// attributeRowOf monta uma das seis linhas.
func attributeRowOf(key string, spread map[string]int, sheet engine.ComputedSheet) attributeRow {
	computed := sheet.Attributes[key]
	return attributeRow{
		Key: key, Label: book.AttributeLabel(key),
		Base:  spread[key],
		Total: computed.Total,
		Sum:   sumWhenThereIsOne(spread[key], computed.Contributions),
	}
}

// sumWhenThereIsOne é vazia quando nada soma ao valor livre: ele já é o total, e a linha só
// repetiria o número.
func sumWhenThereIsOne(base int, contributions []engine.SourceAmount) string {
	if len(contributions) == 0 {
		return ""
	}
	return sheet.AttributeSum(base, contributions)
}

// pointsLine diz onde o espalhamento está em relação aos pontos do livro (p17).
//
// @example pointsLine(14, true) // "4 pontos além dos 10"
func pointsLine(spent int, inTable bool) string {
	switch {
	case !inTable:
		return "Fora da tabela de pontos"
	case spent > engine.PointBuyBudget:
		return fmt.Sprintf("%d pontos além dos %d", spent-engine.PointBuyBudget, engine.PointBuyBudget)
	default:
		return fmt.Sprintf("%d de %d pontos", engine.PointBuyBudget-spent, engine.PointBuyBudget)
	}
}

func (s Scene) writeAttributes(w http.ResponseWriter, r *http.Request, v attributesView) {
	if r.Header.Get("datastar-request") != "" {
		fragment, err := ui.RenderFragment(r.Context(), attributesBody(v))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = datastar.NewSSE(w, r).PatchElements(fragment)
		return
	}
	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Title:        "Atributos · Forja · Tormenta 20",
		Shape:        ui.ShellDense,
		System:       ui.SystemTempestade,
		Back:         "/personagens",
		BackLabel:    "Personagens",
		VisibleTitle: v.HeroName,
	}, attributesScene(v))
}

// stepFromURL aceita o sinal de menos: o passo é para os dois lados.
//
// CÓPIA consciente do `routes.go` da ficha. A forja e a ficha leem o passo
// de rotas diferentes, e pedi-lo pela porta seria pôr sete linhas de parse numa
// interface — mais acoplamento que duplicação.
func stepFromURL(r *http.Request) (int, error) {
	raw := chi.URLParam(r, "passo")
	step, err := strconv.Atoi(raw)
	if err != nil || step == 0 {
		return 0, fmt.Errorf("passo %q não é um número diferente de zero", raw)
	}
	return step, nil
}
