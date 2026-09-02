package api

import (
	"errors"
	"fmt"
	"net/http"
	"t20engine/plataforma"

	"t20engine/db/sqlcgen"
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

// syncLevelVitals recomputes the pools for the (already mutated) aggregate and
// persists the level-shifted currents — the server-side syncVitalsForProjection.
func (s *Server) syncLevelVitals(r *http.Request, id int64, dto sheet.CharacterDTO) (storedVitals, error) {
	stored := storedVitals{HpMax: dto.HpMax, HpCurrent: dto.HpCurrent, MpMax: dto.MpMax, MpCurrent: dto.MpCurrent}
	if s.catalogs == nil || len(dto.Classes) == 0 {
		return stored, nil // no engine pools (0/0) → keep what is stored
	}
	ec, err := sheet.EngineCharacterFrom(dto)
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
	dto, classes, total, vitals, err := s.applyClassLevel(r, row, body.ClassName, *body.Level)
	if err != nil {
		writeLevelFailure(w, err)
		return
	}
	_ = dto
	plataforma.WriteJSON(w, http.StatusOK, classLevelResult{Level: total, Classes: classes, Vitals: vitals})
}

// classLevelError separa a recusa de REGRA da falha de infraestrutura, para
// os dois chamadores traduzirem cada uma no idioma da própria tela — a API JSON
// num erro de campo, o piloto numa frase no rodapé.
type classLevelError struct {
	Campo string
	Frase string
}

func (e classLevelError) Error() string { return e.Frase }

// writeLevelFailure traduz a recusa para o formato da API JSON.
func writeLevelFailure(w http.ResponseWriter, err error) {
	var recusa classLevelError
	if errors.As(err, &recusa) {
		plataforma.WriteFieldError(w, http.StatusBadRequest, recusa.Frase,
			plataforma.FieldErrorMap{recusa.Campo: {recusa.Frase}})
		return
	}
	plataforma.WriteError(w, http.StatusInternalServerError, "Could not update class level")
}

// applyClassLevel é A REGRA do degrau de nível, e ela é UMA para as duas
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
func (s *Server) applyClassLevel(
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
		return dto, nil, 0, storedVitals{}, classLevelError{
			Campo: "className",
			Frase: fmt.Sprintf("Character does not have class %q", classe),
		}
	}
	if total > 20 {
		return dto, nil, 0, storedVitals{}, classLevelError{
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
