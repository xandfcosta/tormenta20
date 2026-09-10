package sheetui

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"t20engine/catalog"
	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
)

// OS COMANDOS DA ABA MAGIAS (ALE-272, fatia 6).

// learnSpell põe uma magia do catálogo no grimório.
func learnSpell(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	id := chi.URLParam(r, "magia")
	if _, conhecida := catalog.LookupSpell(id); !conhecida {
		return fmt.Errorf("a magia %q não existe no livro", id)
	}
	_, err := s.deps.Queries().CreateSpell(r.Context(), sqlcgen.CreateSpellParams{
		Characterid: row.ID, Catalogspellid: id, Prepared: 0, Learnedat: plataforma.NowISO(),
	})
	return err
}

// forgetSpell tira a magia do grimório.
func forgetSpell(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	_, err := s.deps.Queries().DeleteSpell(r.Context(), sqlcgen.DeleteSpellParams{
		Characterid: row.ID, Catalogspellid: chi.URLParam(r, "magia"),
	})
	return err
}

// togglePrepared prepara ou despreparar uma magia.
//
// O comando manda a MAGIA e não o estado, pela razão de sempre: mandar
// "preparada" perde para o clique repetido e para a segunda aba aberta.
func togglePrepared(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	id := chi.URLParam(r, "magia")
	todas, err := s.deps.Queries().ListSpellsByCharacter(r.Context(), row.ID)
	if err != nil {
		return err
	}
	for _, m := range todas {
		if m.Catalogspellid != id {
			continue
		}
		depois := int64(0)
		if m.Prepared == 0 {
			depois = 1
		}
		_, err := s.deps.Queries().SetSpellPreparedByCatalog(r.Context(), sqlcgen.SetSpellPreparedByCatalogParams{
			Prepared: depois, CharacterId: row.ID, CatalogSpellId: id,
		})
		return err
	}
	return fmt.Errorf("a magia %q não está no grimório", id)
}

// castSpellFromSheet conjura, cobrando o PM.
//
// A conta e as recusas são as MESMAS da API JSON — preparação, aprimoramentos, o
// teto da p224 com a ressalva do custo mínimo, e o PM disponível. Escrevê-las de
// novo aqui daria duas regras que divergem no dia em que uma mudar, e é
// exatamente o defeito que a ALE-110 registrou: a redução de custo era exibida
// num lugar e ignorada na hora de cobrar.
func castSpellFromSheet(s Scene, r *http.Request, row sqlcgen.Character, sinais Signals) error {
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	return s.deps.CastSpell(r, dto, chi.URLParam(r, "magia"), sinais.augments())
}
