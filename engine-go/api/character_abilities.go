package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"t20engine/book"
	"t20engine/plataforma"
	"t20engine/sheet"

	"t20engine/db/sqlcgen"
)

// handleUpdateAbilities ports updateAbilityChoices: patch any subset of the
// character's ability-choice JSON blobs, echoing back only the fields written.
// NOTE: classChoices sanitization (devoto/caminho validation vs the DEUS/CAMINHOS
// catalogs) is deferred — the frontend pre-validates; stored as sent.
func (s *Server) handleUpdateAbilities(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	var body struct {
		RaceAbilityChoices *[]string        `json:"raceAbilityChoices"`
		OriginChoices      *[]string        `json:"originChoices"`
		ClassPowers        *[]string        `json:"classPowers"`
		ClassChoices       *json.RawMessage `json:"classChoices"`
		PowerChoices       *json.RawMessage `json:"powerChoices"`
		// raceAttributeChoices entra aqui na ALE-169: a forja oferece criar o
		// personagem com o bônus de atributo da raça por colocar ("dá para
		// criar assim e terminar na ficha"), e sem esta coluna a ficha não
		// tinha como terminar. Guardado como veio, igual aos vizinhos, e é
		// seguro: o motor RECUSA uma escolha inválida — contagem errada,
		// repetida ou no atributo proibido — e não aplica bônus nenhum, em vez
		// de aplicar demais (engine/collect_rules.go, resolveFloating).
		RaceAttributeChoices *json.RawMessage `json:"raceAttributeChoices"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}

	var set setBuilder
	resp := map[string]string{}
	Add := func(column string, value string) {
		set.Add(column+" = ?", value)
		resp[column] = value
	}
	if body.RaceAbilityChoices != nil {
		Add("raceAbilityChoices", sheet.MarshalStrings(body.RaceAbilityChoices))
	}
	if body.OriginChoices != nil {
		Add("originChoices", sheet.MarshalStrings(body.OriginChoices))
	}
	if body.ClassPowers != nil {
		Add("classPowers", sheet.MarshalStrings(body.ClassPowers))
	}
	if body.ClassChoices != nil {
		Add("classChoices", compactJSON(*body.ClassChoices))
	}
	if body.PowerChoices != nil {
		Add("powerChoices", compactJSON(*body.PowerChoices))
	}
	if body.RaceAttributeChoices != nil {
		Add("raceAttributeChoices", compactJSON(*body.RaceAttributeChoices))
	}
	if set.empty() {
		plataforma.WriteError(w, http.StatusBadRequest, "No fields to update")
		return
	}
	// A FRONTEIRA DAS ESCOLHAS (ALE-272, fatia 8). Este endpoint gravava os
	// cinco blobs sem conferir nada: quantos poderes cabem no nível, quantos
	// benefícios a origem dá e quais caminhos a classe aceita eram regra só da
	// tela, em `shared/rules/abilities-*.ts`. Um pedido montado à mão punha
	// vinte poderes num personagem de nível 1, e o motor somava todos.
	//
	// A conferência é sobre o RESULTADO — a ficha depois da escrita —, e por
	// isso ela roda com o patch já aplicado sobre o DTO carregado.
	if msg := s.invalidChoiceAfterPatch(r, row, resp); msg != "" {
		plataforma.WriteFieldError(w, http.StatusBadRequest, msg,
			plataforma.FieldErrorMap{"classPowers": {msg}})
		return
	}

	if err := set.execTouched(r.Context(), s.db, "UPDATE characters", row.ID); err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not update abilities")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, resp)
}

// handleUpdateProficiencies ports updateProficiencies: validate every category
// against the catalog, dedup, store, return {proficiencies}.
func (s *Server) handleUpdateProficiencies(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	var body struct {
		Proficiencies []string `json:"proficiencies"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if body.Proficiencies == nil {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"proficiencies": {"proficiencies must be an array"}})
		return
	}
	proficiencies, unknown, err := s.saveProficiencies(r.Context(), row.ID, body.Proficiencies)
	if len(unknown) > 0 {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"proficiencies": unknown})
		return
	}
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not update proficiencies")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]string{"proficiencies": proficiencies})
}

// saveProficiencies valida contra o catálogo, tira as repetidas e grava.
//
// Extraída na ALE-272 (fatia 2) porque a ficha em Datastar passou a gravar
// proficiencia pelo mesmo caminho, e a regra e uma so: o painel do piloto e o
// `PATCH /characters/{id}/proficiencies` guardam com a MESMA validacao. Duas
// copias divergiriam no dia em que uma categoria nova chegasse — e a copia
// esquecida aceitaria o que a outra recusa.
//
// Devolve a lista de DESCONHECIDAS separada do erro porque as duas respostas sao
// diferentes: categoria invalida e 422 com o campo, falha de banco e 500.
func (s *Server) saveProficiencies(
	ctx context.Context, id int64, categorias []string,
) (string, []string, error) {
	var unknown []string
	seen := map[string]bool{}
	dedup := []string{}
	for _, cat := range categorias {
		if !book.IsProficiencyCategory(cat) {
			unknown = append(unknown, fmt.Sprintf("Unknown category %q", cat))
		}
		if !seen[cat] {
			seen[cat] = true
			dedup = append(dedup, cat)
		}
	}
	if len(unknown) > 0 {
		return "", unknown, nil
	}
	proficiencies := sheet.MarshalStrings(&dedup)
	if err := s.queries.SetProficiencies(ctx, sqlcgen.SetProficienciesParams{
		Proficiencies: proficiencies, UpdatedAt: plataforma.NowISO(), ID: id,
	}); err != nil {
		return "", nil, err
	}
	return proficiencies, nil, nil
}

// compactJSON normalizes an object blob to compact JSON (matching JSON.stringify
// of the sanitized value, minus the deferred sanitization).
func compactJSON(raw json.RawMessage) string {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return "{}"
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// invalidChoiceAfterPatch aplica o patch sobre a ficha carregada e devolve
// a frase da recusa, ou "" quando ela fica válida.
//
// Ele lê as colunas do `resp`, que é o que o handler já montou para devolver ao
// cliente: um segundo mapeamento de nome de coluna daria duas listas para
// manter iguais.
func (s *Server) invalidChoiceAfterPatch(
	r *http.Request, row sqlcgen.Character, patch map[string]string,
) string {
	dto, err := s.LoadCharacter(r.Context(), row)
	if err != nil {
		return ""
	}
	if v, tem := patch["classPowers"]; tem {
		dto.ClassPowers = v
	}
	if v, tem := patch["originChoices"]; tem {
		dto.OriginChoices = v
	}
	if v, tem := patch["classChoices"]; tem {
		dto.ClassChoices = v
	}
	if err := sheet.WithChoicesValid(dto); err != nil {
		return err.Error()
	}
	return ""
}
