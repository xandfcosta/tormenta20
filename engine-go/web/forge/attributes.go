package forge

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"t20engine/book"
	"t20engine/sheet"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/plataforma"
	"t20engine/web/ui"
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

// asLabelsDosAtributos são os nomes que a pessoa lê. Ficam aqui porque o motor
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

// handleForjaAtributos desenha a distribuição.
func (s Scene) handleAttributes(w http.ResponseWriter, r *http.Request) {
	v, status, err := s.loadAttributes(r, "")
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	s.writeAttributes(w, r, v)
}

// handleForjaAtributoPasso soma o passo a um atributo e redesenha.
//
// O passo vai no CAMINHO e não num sinal, como o do vital na ficha: o valor é
// do botão que foi clicado, e doze botões não disputam um sinal só.
func (s Scene) handleAttributeStep(w http.ResponseWriter, r *http.Request) {
	recusa, status, err := s.stepAttribute(r)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	v, status, err := s.loadAttributes(r, recusa)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	s.writeAttributes(w, r, v)
}

// mexeNoAtributo grava o espalhamento novo, ou devolve a frase da recusa.
func (s Scene) stepAttribute(r *http.Request) (recusa string, status int, err error) {
	row, status, err := s.heroOfTheForge(r)
	if err != nil {
		return "", status, err
	}
	passo, err := oPassoDaURL(r)
	if err != nil {
		return "", http.StatusBadRequest, err
	}
	chave := chi.URLParam(r, "atributo")
	espalhamento := heroSpread(row)
	if _, conhecido := espalhamento[chave]; !conhecido {
		return "", http.StatusBadRequest, fmt.Errorf(
			"atributo %q não existe: são os seis do livro", chave)
	}
	espalhamento[chave] += passo
	if avisos := engine.PointBuyWarnings(espalhamento); len(avisos) > 0 {
		return purchaseRefusal(avisos[0]), http.StatusOK, nil
	}
	if err := s.saveAttributes(r.Context(), row.ID, espalhamento); err != nil {
		return "", http.StatusInternalServerError, err
	}
	// A Constituição mexe no PV máximo (p34). O herói ainda está sendo forjado,
	// então ele fica com os poços CHEIOS — encher aqui é o que impede alguém de
	// abrir a ficha com 18 de 20 PV sem nunca ter apanhado.
	return "", http.StatusOK, s.fillPools(r, row.ID)
}

// aRecusaDaCompra traduz o aviso do motor para a frase que a cena mostra.
//
// O motor escreve para quem depura ("compra de pontos: 14 pontos gastos excedem
// o limite de 10"); a cena fala com quem está criando um herói.
func purchaseRefusal(aviso string) string {
	return "Não cabe na compra de pontos (p17): " + aviso
}

// oEspalhamentoDoHeroi lê os seis atributos base da linha do banco.
func heroSpread(row sqlcgen.Character) map[string]int {
	return map[string]int{
		"strength": int(row.Strength), "dexterity": int(row.Dexterity),
		"constitution": int(row.Constitution), "intelligence": int(row.Intelligence),
		"wisdom": int(row.Wisdom), "charisma": int(row.Charisma),
	}
}

func (s Scene) saveAttributes(ctx context.Context, id int64, espalhamento map[string]int) error {
	return s.deps.Queries().SetCharacterAttributes(ctx, sqlcgen.SetCharacterAttributesParams{
		Strength: int64(espalhamento["strength"]), Dexterity: int64(espalhamento["dexterity"]),
		Constitution: int64(espalhamento["constitution"]), Intelligence: int64(espalhamento["intelligence"]),
		Wisdom: int64(espalhamento["wisdom"]), Charisma: int64(espalhamento["charisma"]),
		UpdatedAt: plataforma.NowISO(), ID: id,
	})
}

// oHeroiDaForja acha o herói e confere a POSSE — o mesmo gargalo da ficha:
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

// carregaAtributosDaForja monta a cena a partir do que está gravado.
func (s Scene) loadAttributes(r *http.Request, recusa string) (attributesView, int, error) {
	row, status, err := s.heroOfTheForge(r)
	if err != nil {
		return attributesView{}, status, err
	}
	sheet, err := sheet.LoadAndCompute(r.Context(), s.deps.Queries(), s.deps.Catalogs(), row)
	if err != nil {
		return attributesView{}, http.StatusInternalServerError, err
	}
	espalhamento := heroSpread(row)
	gasto, _ := engine.PointBuySpent(espalhamento)
	v := attributesView{
		ID: row.ID, HeroName: row.Name, Spent: gasto,
		Budget: engine.PointBuyBudget, Refusal: recusa,
	}
	for _, atributo := range book.AttributeOrder {
		v.Rows = append(v.Rows, attributeRowOf(atributo.Chave, espalhamento, sheet, gasto))
	}
	return v, http.StatusOK, nil
}

// aLinhaDoAtributo monta uma das seis linhas, já dizendo se cada botão cabe.
//
// Os dois botões são desligados pela MESMA regra que recusaria o clique — a
// pergunta é feita ao motor com o espalhamento hipotético. Travar na tela é
// conveniência; quem recusa de verdade é o servidor, no `mexeNoAtributo`.
func attributeRowOf(
	chave string, espalhamento map[string]int, sheet engine.ComputedSheetV2, gasto int,
) attributeRow {
	return attributeRow{
		Key: chave, Label: attributeLabels[chave],
		Base:     espalhamento[chave],
		Total:    sheet.Attributes[chave].Total,
		CanRaise: stepFits(chave, espalhamento, +1),
		CanLower: stepFits(chave, espalhamento, -1),
	}
}

// oPassoCabe pergunta ao motor se o espalhamento continuaria legal com o passo.
func stepFits(chave string, espalhamento map[string]int, passo int) bool {
	hipotese := make(map[string]int, len(espalhamento))
	for k, v := range espalhamento {
		hipotese[k] = v
	}
	hipotese[chave] += passo
	return len(engine.PointBuyWarnings(hipotese)) == 0
}

func (s Scene) writeAttributes(w http.ResponseWriter, r *http.Request, v attributesView) {
	if r.Header.Get("datastar-request") != "" {
		fragmento, err := ui.RenderFragment(r.Context(), attributesBody(v))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = datastar.NewSSE(w, r).PatchElements(fragmento)
		return
	}
	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Titulo:       "Atributos · Forja · Tormenta 20",
		Forma:        ui.ShellDense,
		Voltar:       "/personagens",
		VoltarRotulo: "Personagens",
	}, attributesScene(v))
}

// oPassoDaURL aceita o sinal de menos: o passo é para os dois lados.
//
// CÓPIA consciente do `piloto_ficha_routes.go`. A forja e a ficha leem o passo
// de rotas diferentes, e pedi-lo pela porta seria pôr sete linhas de parse numa
// interface — mais acoplamento que duplicação.
func oPassoDaURL(r *http.Request) (int, error) {
	bruto := chi.URLParam(r, "passo")
	passo, err := strconv.Atoi(bruto)
	if err != nil || passo == 0 {
		return 0, fmt.Errorf("passo %q não é um número diferente de zero", bruto)
	}
	return passo, nil
}
