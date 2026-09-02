package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"t20engine/plataforma"

	"github.com/go-chi/chi/v5"
	"t20engine/db/sqlcgen"
	"t20engine/sheet"
)

// expertiseNames mirrors t20-data EXPERTISE_NAMES — the builtin perícias. A
// custom expertise may not reuse one, and only these can be edited via PATCH.
var expertiseNames = sheet.ToStringSet([]string{
	"Acrobacia", "Adestramento", "Atletismo", "Atuação", "Cavalgar", "Conhecimento",
	"Cura", "Diplomacia", "Enganação", "Fortitude", "Furtividade", "Guerra",
	"Iniciativa", "Intimidação", "Intuição", "Investigação", "Jogatina", "Ladinagem",
	"Luta", "Misticismo", "Nobreza", "Ofício", "Percepção", "Pilotagem", "Pontaria",
	"Reflexos", "Religião", "Sobrevivência", "Vontade",
})

var attributeKeys = sheet.ToStringSet([]string{
	"strength", "dexterity", "constitution", "intelligence", "wisdom", "charisma",
})

// saveNewCraft é a regra de quem pode virar ofício, e ela é a MESMA para a
// API JSON e para a ficha em Datastar (ALE-272).
//
// Três recusas, e a do meio é a que importa: um ofício não pode ROUBAR o nome de
// uma das 29 do livro, porque a ficha passaria a ter duas linhas com o mesmo
// nome e a decomposição de uma cairia sobre a outra.
func (s *Server) saveNewCraft(ctx context.Context, characterID int64, nome string) error {
	if nome == "" {
		return fmt.Errorf("dê um nome ao ofício")
	}
	if expertiseNames[nome] {
		return fmt.Errorf("%q é uma perícia do livro — escolha outro nome", nome)
	}
	_, err := s.queries.GetExpertiseMeta(ctx, sqlcgen.GetExpertiseMetaParams{
		Characterid: characterID, Name: nome,
	})
	if err == nil {
		return fmt.Errorf("esta ficha já tem %q", nome)
	}
	return nil
}

func expertiseDTO(name, attribute string, trained, custom int64) sheet.ExpertiseDTO {
	return sheet.ExpertiseDTO{Name: name, Attribute: attribute, Trained: trained != 0, Custom: custom != 0}
}

// handleAddExpertise ports addCustomExpertise: a user-named perícia (not a
// builtin, not a duplicate), created trained + custom.
func (s *Server) handleAddExpertise(w http.ResponseWriter, r *http.Request) {
	character, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	var body struct {
		Name      string `json:"name"`
		Attribute string `json:"attribute"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	fields := plataforma.FieldErrorMap{}
	name := strings.TrimSpace(body.Name)
	// A REGRA DO NOME é UMA SÓ, e a extração é da ALE-272: a ficha em Datastar
	// cria ofício pelo mesmo caminho, e duas validações divergiriam no dia em que
	// uma regra nova chegasse — a esquecida aceitaria o que a outra recusa.
	if err := s.saveNewCraft(r.Context(), character.ID, name); err != nil {
		fields["name"] = []string{err.Error()}
	}
	if !attributeKeys[body.Attribute] {
		fields["attribute"] = []string{"attribute must be a valid AttributeKey"}
	}
	if len(fields) > 0 {
		plataforma.WriteValidationError(w, fields)
		return
	}
	row, err := s.queries.CreateExpertise(r.Context(), sqlcgen.CreateExpertiseParams{
		Characterid: character.ID, Name: name, Attribute: body.Attribute, Trained: 1, Custom: 1,
	})
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not create expertise")
		return
	}
	plataforma.WriteJSON(w, http.StatusCreated, expertiseDTO(row.Name, row.Attribute, row.Trained, row.Custom))
}

// handleUpdateExpertise ports updateExpertise: patch a BUILTIN perícia's attribute
// and/or trained flag.
func (s *Server) handleUpdateExpertise(w http.ResponseWriter, r *http.Request) {
	character, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	var body struct {
		Name      string  `json:"name"`
		Attribute *string `json:"attribute"`
		Trained   *bool   `json:"trained"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	// O NOME NÃO PRECISA SER UMA DAS 29 DO LIVRO, e a exigência saiu aqui na
	// ALE-272: ela recusava o OFÍCIO que o próprio jogador criou.
	//
	// A ficha desenha o botão de treino e o seletor de atributo em toda linha,
	// inclusive nos ofícios — e nos ofícios os dois davam 400. Era promessa de
	// tela que o servidor não cumpria, e ninguém tinha notado porque nenhum teste
	// mexia num ofício depois de criá-lo.
	//
	// Quem confere a existência é o `UpdateExpertise` logo abaixo, e ele confere
	// MELHOR: a checagem é por PERSONAGEM, então um nome que não é de ninguém
	// vira 404 em vez de passar por uma lista global. A `expertiseNames` continua
	// valendo onde ela é a regra certa — no `handleAddExpertise`, para um ofício
	// novo não roubar o nome de uma perícia do livro.
	if body.Attribute == nil && body.Trained == nil {
		plataforma.WriteError(w, http.StatusBadRequest, "No fields to update")
		return
	}
	if body.Attribute != nil && !attributeKeys[*body.Attribute] {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"attribute": {"attribute must be a valid AttributeKey"}})
		return
	}
	row, err := s.queries.UpdateExpertise(r.Context(), sqlcgen.UpdateExpertiseParams{
		Attribute:   nullString(body.Attribute),
		Trained:     nullBool(body.Trained),
		CharacterId: character.ID,
		Name:        body.Name,
	})
	if errors.Is(err, sql.ErrNoRows) {
		plataforma.WriteError(w, http.StatusNotFound, fmt.Sprintf("Expertise %q not found", body.Name))
		return
	}
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not update expertise")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, expertiseDTO(row.Name, row.Attribute, row.Trained, row.Custom))
}

// handleDeleteExpertise ports deleteExpertise: only custom perícias can be
// removed; builtins 400, missing 404.
func (s *Server) handleDeleteExpertise(w http.ResponseWriter, r *http.Request) {
	character, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	name := chi.URLParam(r, "name")
	if decoded, err := url.PathUnescape(name); err == nil {
		name = decoded
	}
	meta, err := s.queries.GetExpertiseMeta(r.Context(), sqlcgen.GetExpertiseMetaParams{Characterid: character.ID, Name: name})
	if errors.Is(err, sql.ErrNoRows) {
		plataforma.WriteError(w, http.StatusNotFound, fmt.Sprintf("Expertise %q not found", name))
		return
	}
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not Load expertise")
		return
	}
	if meta.Custom == 0 {
		plataforma.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Expertise %q is builtin and cannot be removed", name))
		return
	}
	if err := s.queries.DeleteExpertiseByID(r.Context(), meta.ID); err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not delete expertise")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]string{"name": name})
}

func nullBool(p *bool) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: boolToInt(*p), Valid: true}
}
