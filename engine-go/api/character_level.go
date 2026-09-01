package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"t20engine/plataforma"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/sheet"
)

type storedVitals struct {
	HpMax     int64 `json:"hpMax"`
	HpCurrent int64 `json:"hpCurrent"`
	MpMax     int64 `json:"mpMax"`
	MpCurrent int64 `json:"mpCurrent"`
}

type levelResult struct {
	Level  int64        `json:"level"`
	Vitals storedVitals `json:"vitals"`
}

type classLevelResult struct {
	Level   int64            `json:"level"`
	Classes []sheet.ClassDTO `json:"classes"`
	Vitals  storedVitals     `json:"vitals"`
}

// levelVitalsNext ports vitals-sync.helpers levelVitalsPatch: new maxes from the
// engine, with currents FOLLOWING the max delta (level up heals the gain, down
// walks it back), clamped to [0, newMax]. Returns the next pools + changed flag.
func levelVitalsNext(stored storedVitals, pvMax, pmMax int) (storedVitals, bool) {
	next := storedVitals{
		HpMax:     int64(pvMax),
		HpCurrent: clampCurrent(stored.HpCurrent+(int64(pvMax)-stored.HpMax), int64(pvMax)),
		MpMax:     int64(pmMax),
		MpCurrent: clampCurrent(stored.MpCurrent+(int64(pmMax)-stored.MpMax), int64(pmMax)),
	}
	return next, next != stored
}

func clampCurrent(c, hi int64) int64 { return min(max(int64(0), c), hi) }

// engineCharacterFrom bridges the API aggregate to engine.Character via JSON —
// both mirror the frontend Character contract, so the round-trip is lossless.
func engineCharacterFrom(dto sheet.CharacterDTO) (engine.Character, error) {
	var ec engine.Character
	b, err := json.Marshal(dto)
	if err != nil {
		return ec, err
	}
	return ec, json.Unmarshal(b, &ec)
}

// computeSheet builds the engine input from an already-loaded character row and returns the
// server-computed ComputedSheetV2 (base sheet, no active conditionals). Shared by GET /sheet
// and the power-grant temp-HP amount so the Load→engine→compute wiring lives in one place.
// Caller must ensure s.catalogs is primed.
func (s *Server) ComputeSheet(ctx context.Context, row sqlcgen.Character) (engine.ComputedSheetV2, error) {
	dto, err := s.LoadCharacter(ctx, row)
	if err != nil {
		return engine.ComputedSheetV2{}, err
	}
	return s.sheetFromDTO(dto)
}

// sheetFromDTO computa a ficha de um agregado JÁ CARREGADO.
//
// Separado do `computeSheet` para a cena de personagens (ALE-239), que precisa
// da ficha de TODOS de uma vez: ela já tem os agregados na mão, e passar por
// `computeSheet` faria cada personagem ser lido do banco DUAS vezes — uma na
// lista e outra dentro dele. Com uma dúzia de heróis isso é o dobro das
// consultas para o mesmo resultado.
func (s *Server) sheetFromDTO(dto sheet.CharacterDTO) (engine.ComputedSheetV2, error) {
	ec, err := engineCharacterFrom(dto)
	if err != nil {
		return engine.ComputedSheetV2{}, err
	}
	return s.catalogs.ComputeSheetV2(ec, map[string]bool{}), nil
}

// syncLevelVitals recomputes the pools for the (already mutated) aggregate and
// persists the level-shifted currents — the server-side syncVitalsForProjection.
func (s *Server) syncLevelVitals(r *http.Request, id int64, dto sheet.CharacterDTO) (storedVitals, error) {
	stored := storedVitals{HpMax: dto.HpMax, HpCurrent: dto.HpCurrent, MpMax: dto.MpMax, MpCurrent: dto.MpCurrent}
	if s.catalogs == nil || len(dto.Classes) == 0 {
		return stored, nil // no engine pools (0/0) → keep what is stored
	}
	ec, err := engineCharacterFrom(dto)
	if err != nil {
		return stored, err
	}
	pools := s.catalogs.VitalsForCharacter(ec)
	next, changed := levelVitalsNext(stored, pools.PvMax, pools.PmMax)
	if changed {
		if err := s.queries.SetCharacterVitals(r.Context(), sqlcgen.SetCharacterVitalsParams{
			HpMax: next.HpMax, HpCurrent: next.HpCurrent, MpMax: next.MpMax, MpCurrent: next.MpCurrent,
			UpdatedAt: plataforma.NowISO(), ID: id,
		}); err != nil {
			return stored, err
		}
	}
	return next, nil
}

// handleUpdateLevel ports updateLevel: set the total level, resync the derived
// pools. (Pools track class levels, so setting the total alone usually leaves
// them unchanged.)
func (s *Server) handleUpdateLevel(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	var body struct {
		Level *int64 `json:"level"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if msg := levelRangeError(body.Level); msg != "" {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"level": {msg}})
		return
	}
	if err := s.queries.SetCharacterLevel(r.Context(), sqlcgen.SetCharacterLevelParams{
		Level: *body.Level, UpdatedAt: plataforma.NowISO(), ID: row.ID,
	}); err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not update level")
		return
	}
	dto, err := s.LoadCharacter(r.Context(), row)
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not Load character")
		return
	}
	dto.Level = *body.Level
	vitals, err := s.syncLevelVitals(r, row.ID, dto)
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not sync vitals")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, levelResult{Level: *body.Level, Vitals: vitals})
}

// handleUpdateClassLevel ports updateClassLevel: bump one class, recompute the
// total (≤ 20 cap) and the derived pools with the level-shift.
func (s *Server) handleUpdateClassLevel(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	var body struct {
		ClassName string `json:"className"`
		Level     *int64 `json:"level"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if msg := levelRangeError(body.Level); msg != "" {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"level": {msg}})
		return
	}
	dto, classes, total, vitals, err := s.aplicaONivelDaClasse(r, row, body.ClassName, *body.Level)
	if err != nil {
		escreveAFalhaDoNivel(w, err)
		return
	}
	_ = dto
	plataforma.WriteJSON(w, http.StatusOK, classLevelResult{Level: total, Classes: classes, Vitals: vitals})
}

// erroDeClasseDoNivel separa a recusa de REGRA da falha de infraestrutura, para
// os dois chamadores traduzirem cada uma no idioma da própria tela — a API JSON
// num erro de campo, o piloto numa frase no rodapé.
type erroDeClasseDoNivel struct {
	Campo string
	Frase string
}

func (e erroDeClasseDoNivel) Error() string { return e.Frase }

// escreveAFalhaDoNivel traduz a recusa para o formato da API JSON.
func escreveAFalhaDoNivel(w http.ResponseWriter, err error) {
	var recusa erroDeClasseDoNivel
	if errors.As(err, &recusa) {
		plataforma.WriteFieldError(w, http.StatusBadRequest, recusa.Frase,
			plataforma.FieldErrorMap{recusa.Campo: {recusa.Frase}})
		return
	}
	plataforma.WriteError(w, http.StatusInternalServerError, "Could not update class level")
}

// aplicaONivelDaClasse é A REGRA do degrau de nível, e ela é UMA para as duas
// telas (ALE-272).
//
// Ela estava dentro do handler JSON, e extraí-la é o que impede a ficha em
// Datastar de divergir da antiga: o nível de um personagem é a SOMA dos níveis
// de classe, e uma segunda cópia dessa conta é como as duas telas passam a
// discordar de quanto vale o Guerreiro 3 / Ladino 2.
//
// As três garantias que ela carrega, e que a tela não pode reescrever:
//
//   - a classe TEM de ser do personagem — subir "Bardo" em quem não é bardo
//     criaria um nível que não existe em lugar nenhum;
//   - o TOTAL é limitado a 20 (p32), e a soma é a conta que decide, não o campo;
//   - os POOLS acompanham, porque PV e PM máximos derivam dos níveis de classe.
//     Gravar o nível sem sincronizar deixa a ficha com o número novo e a vida
//     velha, que é o defeito que ninguém liga ao botão que o causou.
func (s *Server) aplicaONivelDaClasse(
	r *http.Request, row sqlcgen.Character, classe string, nivel int64,
) (sheet.CharacterDTO, []sheet.ClassDTO, int64, storedVitals, error) {
	dto, err := s.LoadCharacter(r.Context(), row)
	if err != nil {
		return dto, nil, 0, storedVitals{}, err
	}
	achou := false
	var total int64
	for i := range dto.Classes {
		if dto.Classes[i].ClassName == classe {
			dto.Classes[i].Level = nivel
			achou = true
		}
		total += dto.Classes[i].Level
	}
	if !achou {
		return dto, nil, 0, storedVitals{}, erroDeClasseDoNivel{
			Campo: "className",
			Frase: fmt.Sprintf("Character does not have class %q", classe),
		}
	}
	if total > 20 {
		return dto, nil, 0, storedVitals{}, erroDeClasseDoNivel{
			Campo: "level",
			Frase: fmt.Sprintf("Total level %d exceeds 20", total),
		}
	}
	if _, err := s.queries.SetCharacterClassLevel(r.Context(), sqlcgen.SetCharacterClassLevelParams{
		Level: nivel, CharacterId: row.ID, ClassName: classe,
	}); err != nil {
		return dto, nil, 0, storedVitals{}, err
	}
	if err := s.queries.SetCharacterLevel(r.Context(), sqlcgen.SetCharacterLevelParams{
		Level: total, UpdatedAt: plataforma.NowISO(), ID: row.ID,
	}); err != nil {
		return dto, nil, 0, storedVitals{}, err
	}
	dto.Level = total
	vitals, err := s.syncLevelVitals(r, row.ID, dto)
	return dto, dto.Classes, total, vitals, err
}

// levelRangeError applies the UpdateLevelDto range (@IsInt @Min(1) @Max(20)).
func levelRangeError(level *int64) string {
	switch {
	case level == nil:
		return "level must be an integer number"
	case *level < 1:
		return "level must not be less than 1"
	case *level > 20:
		return "level must not be greater than 20"
	}
	return ""
}
