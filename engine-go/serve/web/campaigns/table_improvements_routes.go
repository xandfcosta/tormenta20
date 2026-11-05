package campaigns

import (
	"errors"
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/app/campaign"
	"t20engine/serve/web/routes"
	"t20engine/serve/web/ui"
)

// O EDITOR DA MELHORIA DA MESA: o mesmo desenho do editor de regra, com os mesmos cuidados.

func (s Scene) patchImprovementEditor(w http.ResponseWriter, r *http.Request, id int64, draft improvementDraft, signals map[string]any, refusal string) {
	v, err := s.improvementEditorOf(r.Context(), id, draft, refusal)
	if err != nil {
		http.Error(w, ui.NoticeInternal, http.StatusInternalServerError)
		return
	}
	sse := datastar.NewSSE(w, r)
	if len(signals) > 0 {
		_ = sse.MarshalAndPatchSignals(signals)
	}
	fragment, err := ui.RenderFragment(r.Context(), improvementEditorForm(v))
	if err != nil {
		return
	}
	_ = sse.PatchElements(fragment)
}

// handleTableImprovementEditor abre o editor: a melhoria pedida, ou o rascunho em branco.
func (s Scene) handleTableImprovementEditor(w http.ResponseWriter, r *http.Request) {
	id, ok := s.ruleGate(w, r)
	if !ok {
		return
	}
	draft := blankImprovementDraft()
	if wanted := r.URL.Query().Get("melhoria"); wanted != "" {
		improvements, err := s.life.TableImprovements(r.Context(), id)
		if err != nil {
			http.Error(w, ui.NoticeInternal, http.StatusInternalServerError)
			return
		}
		found := slices.IndexFunc(improvements, func(i campaign.StoredTableImprovement) bool { return i.ID == wanted })
		if found < 0 {
			s.patchImprovementEditor(w, r, id, draft, draft.signals(), campaign.TableImprovementGone)
			return
		}
		draft = draftOfImprovement(improvements[found])
	}
	s.patchImprovementEditor(w, r, id, draft, draft.signals(), "")
}

// handleTableImprovementDraft redesenha o editor com o que a pessoa mudou. Só volta ao navegador o
// que o servidor mudou (`signalsThatMoved`).
func (s Scene) handleTableImprovementDraft(w http.ResponseWriter, r *http.Request) {
	id, ok := s.ruleGate(w, r)
	if !ok {
		return
	}
	draft, err := readImprovementDraft(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	asked := draft.signals()
	draft = draft.normalized()
	s.patchImprovementEditor(w, r, id, draft, signalsThatMoved(asked, draft.signals()), "")
}

func (s Scene) handleTableImprovementSave(w http.ResponseWriter, r *http.Request) {
	id, who, ok := s.requesterOf(w, r)
	if !ok {
		return
	}
	draft, err := readImprovementDraft(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = s.life.SaveTableImprovement(r.Context(), who, id, draft.toSave())
	var refusal campaign.TableRuleRefusal
	if errors.As(err, &refusal) {
		s.patchImprovementEditor(w, r, id, draft, nil, refusal.Reason)
		return
	}
	if err != nil {
		refuse(w, err)
		return
	}
	// A melhoria nova não está em item nenhum, e não muda ficha nenhuma.
	if draft.ID != "" {
		s.wakeTableSheets(r.Context(), id)
	}
	_ = datastar.NewSSE(w, r).Redirect(routes.CampaignTab(id, "regras"))
}

func (s Scene) handleTableImprovementRemove(w http.ResponseWriter, r *http.Request) {
	id, who, ok := s.requesterOf(w, r)
	if !ok {
		return
	}
	var refusal campaign.TableRuleRefusal
	if err := s.life.RemoveTableImprovement(r.Context(), who, id, chi.URLParam(r, "melhoria")); err != nil && !errors.As(err, &refusal) {
		refuse(w, err)
		return
	}
	s.wakeTableSheets(r.Context(), id)
	_ = datastar.NewSSE(w, r).Redirect(routes.CampaignTab(id, "regras"))
}
