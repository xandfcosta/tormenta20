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
// A compra de pontos é do MOTOR (`engine.PointBuyWarnings`), e é ele quem sabe
// que só um atributo desce a −1 e que o orçamento é 10. Esta cena não recalcula
// nada: ela pergunta se o espalhamento resultante tem reclamação e, se tiver,
// não grava.

// attributeLabels são os nomes que a pessoa lê. Ficam aqui porque o motor
// fala `strength`, que é identificador e não texto de tela.
var attributeLabels = map[string]string{
	"strength": "Força", "dexterity": "Destreza", "constitution": "Constituição",
	"intelligence": "Inteligência", "wisdom": "Sabedoria", "charisma": "Carisma",
}

// attributesView é a cena inteira.
type attributesView struct {
	ID       int64
	HeroName string
	Rows     []attributeRow
	Spent    int
	Budget   int
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
	Base     int
	Total    int
	CanRaise bool
	CanLower bool
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
	if notices := engine.PointBuyWarnings(spread); len(notices) > 0 {
		return purchaseRefusal(notices[0]), http.StatusOK, nil
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

// purchaseRefusal traduz o aviso do motor para a frase que a cena mostra.
//
// O motor escreve para quem depura ("compra de pontos: 14 pontos gastos excedem
// o limite de 10"); a cena fala com quem está criando um herói.
func purchaseRefusal(notice string) string {
	return "Não cabe na compra de pontos (p17): " + notice
}

// heroSpread lê os seis atributos base da linha do banco.
func heroSpread(row sqlcgen.Character) map[string]int {
	return map[string]int{
		"strength": int(row.Strength), "dexterity": int(row.Dexterity),
		"constitution": int(row.Constitution), "intelligence": int(row.Intelligence),
		"wisdom": int(row.Wisdom), "charisma": int(row.Charisma),
	}
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
	spent, _ := engine.PointBuySpent(spread)
	v := attributesView{
		ID: row.ID, HeroName: row.Name, Spent: spent,
		Budget: engine.PointBuyBudget, Refusal: refusal,
	}
	for _, attribute := range book.AttributeOrder {
		v.Rows = append(v.Rows, attributeRowOf(attribute.Key, spread, sheet, spent))
	}
	return v, http.StatusOK, nil
}

// attributeRowOf monta uma das seis linhas, já dizendo se cada botão cabe.
//
// Os dois botões são desligados pela MESMA regra que recusaria o clique — a
// pergunta é feita ao motor com o espalhamento hipotético. Travar na tela é
// conveniência; quem recusa de verdade é o servidor, no `stepAttribute`.
func attributeRowOf(
	key string, spread map[string]int, sheet engine.ComputedSheet, spent int,
) attributeRow {
	return attributeRow{
		Key: key, Label: attributeLabels[key],
		Base:     spread[key],
		Total:    sheet.Attributes[key].Total,
		CanRaise: stepFits(key, spread, +1),
		CanLower: stepFits(key, spread, -1),
	}
}

// stepFits pergunta ao motor se o espalhamento continuaria legal com o passo.
func stepFits(key string, spread map[string]int, step int) bool {
	hypothesis := make(map[string]int, len(spread))
	for k, v := range spread {
		hypothesis[k] = v
	}
	hypothesis[key] += step
	return len(engine.PointBuyWarnings(hypothesis)) == 0
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
		Title:     "Atributos · Forja · Tormenta 20",
		Shape:     ui.ShellDense,
		Back:      "/personagens",
		BackLabel: "Personagens",
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
