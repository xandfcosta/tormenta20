package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"t20engine/db/sqlcgen"
	"t20engine/engine"
)

// handleListCharacters returns the caller's own characters (newest-updated first),
// each as the full aggregate —
func (s *Server) handleListCharacters(w http.ResponseWriter, r *http.Request) {
	out, err := s.characterList(r.Context(), currentUser(r).ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not list characters")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// characterList é o elenco de quem chama, agregado.
//
// Transport-agnostic, e esta é a SEXTA vez que a migração encontra a mesma
// forma — depois do `selfInitiativeEntry`, do `deleteAccount`, do trio da porta,
// do `mintAccountInvite` e do `campaignList`. Seis é padrão, não anedota: uma
// base com exatamente um transporte não tem por que separar regra de handler, e
// o segundo transporte é o que cobra a conta (ALE-239).
func (s *Server) characterList(ctx context.Context, ownerID int64) ([]CharacterDTO, error) {
	rows, err := s.queries.ListCharactersByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	out := make([]CharacterDTO, 0, len(rows))
	for _, row := range rows {
		dto, err := s.loadCharacter(ctx, row)
		if err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, nil
}

// handleGetCharacter returns one character aggregate. Access = owner OR campaign
// GM. 404 when missing, 403 when unauthorized.
func (s *Server) handleGetCharacter(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load character")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

// characterFor is the preamble every character route repeats: read {id}, load
// the row, enforce the read/mutation guard, and emit the right error. Returns
// ok=false when it already wrote the response.
//
// Twenty-three handlers spelled these nine lines out, and the copy-paste left a
// real inconsistency behind it: three of them (items, expertises, spells)
// decoded the request body BEFORE authorizing, while the rest authorized first.
// Going through one helper forces a single order — authorize, then read the
// body — so an unauthorized caller can never reach a decoder.
//
// @example row, ok := s.characterFor(w, r); if !ok { return }
func (s *Server) characterFor(w http.ResponseWriter, r *http.Request) (sqlcgen.Character, bool) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return sqlcgen.Character{}, false
	}
	row, status, err := s.authorizedCharacter(r.Context(), currentUser(r), id)
	if err != nil {
		writeError(w, status, err.Error())
		return sqlcgen.Character{}, false
	}
	return row, true
}

// authorizedCharacter loads a character and enforces the read/mutation guard
// (owner or campaign GM). Returns the row, or an HTTP status + error to emit.
func (s *Server) authorizedCharacter(ctx context.Context, user AuthUser, id int64) (sqlcgen.Character, int, error) {
	row, err := s.queries.GetCharacter(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return row, http.StatusNotFound, fmt.Errorf("Character %d not found", id)
	}
	if err != nil {
		return row, http.StatusInternalServerError, errors.New("Could not load character")
	}
	// The admin passes the same door as the owner and the campaign's GM: a table
	// they administer includes the sheets in it (ALE-120).
	if row.Ownerid == user.ID || user.IsAdmin {
		return row, http.StatusOK, nil
	}
	isGm, err := s.queries.IsCampaignGmForCharacter(ctx, sqlcgen.IsCampaignGmForCharacterParams{
		Characterid: id,
		Ownerid:     user.ID,
	})
	if err != nil {
		return row, http.StatusInternalServerError, errors.New("Could not check access")
	}
	if !isGm {
		return row, http.StatusForbidden, fmt.Errorf("Character %d belongs to another user", id)
	}
	return row, http.StatusOK, nil
}

// handleGetSheet ports GET /characters/:id/sheet — the server-computed derived sheet
// (ComputedSheetV2), the same shape the WASM engine produces for the front. No live
// consumer today (the front derives via WASM); provided for non-WASM clients + parity.
// Active conditionals (stances) aren't applied — this is the base sheet.
func (s *Server) handleGetSheet(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	if s.catalogs == nil {
		writeError(w, http.StatusServiceUnavailable, "Rules catalog not loaded")
		return
	}
	sheet, err := s.computeSheet(r.Context(), row)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not compute sheet")
		return
	}
	writeJSON(w, http.StatusOK, sheet)
}

// assertCharacterOwner is the strict owner-only check
// the WS vitals gate uses: a player may edit only a character they own. Transport-agnostic.
func (s *Server) assertCharacterOwner(ctx context.Context, userID, characterID int64) (int, error) {
	owner, err := s.queries.GetCharacterOwner(ctx, characterID)
	if errors.Is(err, sql.ErrNoRows) {
		return http.StatusNotFound, fmt.Errorf("Character %d not found", characterID)
	}
	if err != nil {
		return http.StatusInternalServerError, errors.New("Could not load character")
	}
	if owner != userID {
		return http.StatusForbidden, fmt.Errorf(
			"Caller %d can only edit their own character's vitals (character %d)", userID, characterID)
	}
	return http.StatusOK, nil
}

// loadCharacter attaches the six relations to a character row in the Prisma
// include order (races/classes/items/effects by id, expertises by name, spells by
// learnedAt).
func (s *Server) loadCharacter(ctx context.Context, c sqlcgen.Character) (CharacterDTO, error) {
	dto := characterScalarsFrom(c)

	races, err := s.queries.ListRacesByCharacter(ctx, c.ID)
	if err != nil {
		return dto, err
	}
	for _, race := range races {
		dto.Races = append(dto.Races, RaceDTO{Race: race})
	}

	classes, err := s.queries.ListClassesByCharacter(ctx, c.ID)
	if err != nil {
		return dto, err
	}
	for _, cl := range classes {
		dto.Classes = append(dto.Classes, ClassDTO{ClassName: cl.Classname, Level: cl.Level})
	}

	exps, err := s.queries.ListExpertisesByCharacter(ctx, c.ID)
	if err != nil {
		return dto, err
	}
	for _, e := range exps {
		dto.Expertises = append(dto.Expertises, ExpertiseDTO{
			Name: e.Name, Attribute: e.Attribute, Trained: e.Trained != 0, Custom: e.Custom != 0,
		})
	}

	items, err := s.queries.ListItemsByCharacter(ctx, c.ID)
	if err != nil {
		return dto, err
	}
	for _, it := range items {
		dto.Items = append(dto.Items, ItemDTO{
			ID: it.ID, CatalogID: nullToPtr(it.Catalogid), Name: it.Name,
			Quantity: it.Quantity, Slots: it.Slots, Equipped: nullToPtr(it.Equipped),
			Improvements: it.Improvements, Material: nullToPtr(it.Material),
		})
	}

	effects, err := s.queries.ListActiveEffectsByCharacter(ctx, c.ID)
	if err != nil {
		return dto, err
	}
	for _, ef := range effects {
		dto.ActiveEffects = append(dto.ActiveEffects, EffectDTO{
			ID: ef.ID, CatalogID: ef.Catalogid, Scope: ef.Scope,
			Modifiers: ef.Modifiers, CreatedAt: ef.Createdat,
		})
	}

	spells, err := s.queries.ListSpellsByCharacter(ctx, c.ID)
	if err != nil {
		return dto, err
	}
	for _, sp := range spells {
		dto.Spells = append(dto.Spells, SpellDTO{
			ID: sp.ID, CatalogSpellID: sp.Catalogspellid, Prepared: sp.Prepared != 0, LearnedAt: sp.Learnedat,
		})
	}

	// As regras opcionais da mesa entram na ficha AQUI, e num lugar só (ALE-221):
	// tudo o que calcula — o `GET /sheet`, os PV/PM do nível, o bônus de
	// iniciativa, a ficha que o navegador recalcula no WASM — passa por este
	// carregamento. Falha de leitura não derruba a ficha: o `IgnoredRules` fica
	// zerado, que significa TODAS as regras em vigor. É o lado seguro, e o único
	// em que um banco mudo não afrouxa regra sem ninguém ver.
	ignored, err := s.queries.ListIgnoredRulesForCharacter(ctx, c.ID)
	if err == nil {
		dto.IgnoredRules = engine.IgnoredRulesFrom(ignored)
	}

	// O estado de JOGO (ALE-222). Vem junto e nao por endpoint proprio: separado,
	// a ficha abriria com a Furia desligada e a ligaria um instante depois,
	// piscando os numeros que ela muda.
	//
	// Este DERRUBA a carga em caso de falha e o de cima nao, e a diferenca e
	// deliberada: sem o estado de jogo a ficha mente sobre o que esta ligado,
	// enquanto sem as regras opcionais ela cai no padrao do livro, que e o lado
	// seguro. Uma regra a mais nunca inventa numero; uma postura a menos sim.
	if err := s.loadPlayState(ctx, c.ID, &dto); err != nil {
		return dto, err
	}
	return dto, nil
}

// intParam parses a chi :id-style path param, writing a 400 (like ParseIntPipe)
// and returning false on a non-numeric value.
func intParam(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	n, err := parseInt(chi.URLParam(r, name))
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Validation failed (numeric string is expected for %q)", name))
		return 0, false
	}
	return int64(n), true
}
