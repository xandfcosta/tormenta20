package campaigns

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/app/campaign"
	"t20engine/domain/engine"
	"t20engine/serve/web/routes"
	"t20engine/serve/web/ui"
)

// ruleGate confere o mestre ANTES do fluxo abrir, pela razão do `handleToggleRule`: depois do
// `NewSSE` o status não muda mais.
func (s Scene) ruleGate(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, who, ok := s.requesterOf(w, r)
	if !ok {
		return 0, false
	}
	if _, err := s.access.OwnedCampaign(r.Context(), who, id); err != nil {
		refuse(w, err)
		return 0, false
	}
	return id, true
}

func (s Scene) patchRuleEditor(w http.ResponseWriter, r *http.Request, id int64, draft ruleDraft, signals map[string]any, refusal string) {
	v, err := s.ruleEditorOf(r.Context(), id, draft, refusal)
	if err != nil {
		http.Error(w, ui.NoticeInternal, http.StatusInternalServerError)
		return
	}
	sse := datastar.NewSSE(w, r)
	if len(signals) > 0 {
		_ = sse.MarshalAndPatchSignals(signals)
	}
	fragment, err := ui.RenderFragment(r.Context(), ruleEditorForm(v))
	if err != nil {
		return
	}
	_ = sse.PatchElements(fragment)
}

// handleTableRuleEditor abre o editor: a regra pedida, ou o rascunho em branco.
func (s Scene) handleTableRuleEditor(w http.ResponseWriter, r *http.Request) {
	id, ok := s.ruleGate(w, r)
	if !ok {
		return
	}
	draft := blankRuleDraft()
	if wanted := r.URL.Query().Get("regra"); wanted != "" {
		rules, err := s.life.TableRules(r.Context(), id)
		if err != nil {
			http.Error(w, ui.NoticeInternal, http.StatusInternalServerError)
			return
		}
		found := slices.IndexFunc(rules, func(rule campaign.StoredTableRule) bool { return rule.ID == wanted })
		if found < 0 {
			s.patchRuleEditor(w, r, id, draft, draft.signals(), campaign.TableRuleGone)
			return
		}
		draft = draftOfRule(rules[found])
	}
	// Sem `normalized`: abrir só para ler não troca o valor que saiu do catálogo. Gravar é que o recusa.
	s.patchRuleEditor(w, r, id, draft, draft.signals(), "")
}

// handleTableRuleDraft redesenha o editor com o que a pessoa mudou. `mais` acrescenta uma condição, e
// `tira` tira a linha pedida.
func (s Scene) handleTableRuleDraft(w http.ResponseWriter, r *http.Request) {
	id, ok := s.ruleGate(w, r)
	if !ok {
		return
	}
	draft, err := readRuleDraft(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	asked := draft.signals()
	if r.URL.Query().Get("mais") != "" {
		draft.When = append(draft.When, engine.RuleCondition{})
	}
	if slot, err := strconv.Atoi(r.URL.Query().Get("tira")); err == nil && slot >= 1 && slot <= len(draft.When) {
		draft.When = append(draft.When[:slot-1], draft.When[slot:]...)
	}
	draft = draft.normalized()
	s.patchRuleEditor(w, r, id, draft, signalsThatMoved(asked, draft.signals()), "")
}

// handleTableRuleSave grava. A recusa volta dentro do editor, com o que foi escrito.
func (s Scene) handleTableRuleSave(w http.ResponseWriter, r *http.Request) {
	id, who, ok := s.requesterOf(w, r)
	if !ok {
		return
	}
	draft, err := readRuleDraft(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = s.life.SaveTableRule(r.Context(), who, id, draft.toSave())
	var refusal campaign.TableRuleRefusal
	if errors.As(err, &refusal) {
		s.patchRuleEditor(w, r, id, draft, nil, refusal.Reason)
		return
	}
	if err != nil {
		refuse(w, err)
		return
	}
	_ = datastar.NewSSE(w, r).Redirect(routes.CampaignTab(id, "regras"))
}

// handleTableRuleFlip liga ou desliga, e redesenha a lista.
func (s Scene) handleTableRuleFlip(w http.ResponseWriter, r *http.Request) {
	id, who, ok := s.requesterOf(w, r)
	if !ok {
		return
	}
	// A regra que sumiu não é erro aqui: a lista redesenhada já diz, e devolve o interruptor ao lugar.
	var refusal campaign.TableRuleRefusal
	if err := s.life.FlipTableRule(r.Context(), who, id, chi.URLParam(r, "regra")); err != nil && !errors.As(err, &refusal) {
		refuse(w, err)
		return
	}
	rows, err := s.tableRulesOf(r.Context(), id, true, 0)
	if err != nil {
		http.Error(w, ui.NoticeInternal, http.StatusInternalServerError)
		return
	}
	fragment, err := ui.RenderFragment(r.Context(), tableRulesList(oneView{ID: id, IsGM: true, TableRules: rows}))
	if err != nil {
		return
	}
	_ = datastar.NewSSE(w, r).PatchElements(fragment)
}

func (s Scene) handleTableRuleRemove(w http.ResponseWriter, r *http.Request) {
	id, who, ok := s.requesterOf(w, r)
	if !ok {
		return
	}
	var refusal campaign.TableRuleRefusal
	if err := s.life.RemoveTableRule(r.Context(), who, id, chi.URLParam(r, "regra")); err != nil && !errors.As(err, &refusal) {
		refuse(w, err)
		return
	}
	_ = datastar.NewSSE(w, r).Redirect(routes.CampaignTab(id, "regras"))
}

func ruleURL(id int64, path string) string {
	return fmt.Sprintf("/campanhas/%d/regras-da-mesa%s", id, path)
}
