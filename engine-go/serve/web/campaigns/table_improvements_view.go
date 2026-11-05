package campaigns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/starfederation/datastar-go/datastar"

	"t20engine/app/campaign"
	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// A MELHORIA DA MESA, como a aba e o editor a desenham.

type tableImprovementRow struct {
	ID   string
	Name string
	Meta string
}

// tableImprovementsOf são as linhas da aba. O jogador lê as da mesa inteira e as do personagem dele,
// pela mesma razão das regras.
func (s Scene) tableImprovementsOf(ctx context.Context, campaignID int64, isGM bool, euID int64) ([]tableImprovementRow, error) {
	improvements, err := s.life.TableImprovements(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	names, mine, err := s.seatsOf(ctx, campaignID, euID)
	if err != nil {
		return nil, err
	}
	uses := map[string]int{}
	if isGM {
		if uses, err = s.life.TableImprovementUses(ctx, campaignID); err != nil {
			return nil, err
		}
	}
	out := []tableImprovementRow{}
	for _, improvement := range improvements {
		if !isGM && improvement.CharacterID != nil && !mine[*improvement.CharacterID] {
			continue
		}
		who := "Toda a mesa"
		if improvement.CharacterID != nil {
			who = names[*improvement.CharacterID]
		}
		meta := who + " · cabe em " + book.WrittenImprovementPlace(improvement.Family) + " · " + book.WrittenSum(improvement.Effect)
		if isGM {
			meta += " · " + writtenItemsUsing(uses[improvement.ID])
		}
		out = append(out, tableImprovementRow{ID: improvement.ID, Name: improvement.Name, Meta: meta})
	}
	return out, nil
}

func writtenItemsUsing(uses int) string {
	switch uses {
	case 0:
		return "em nenhum item"
	case 1:
		return "em 1 item"
	}
	return "em " + strconv.Itoa(uses) + " itens"
}

// improvementDraft é o rascunho, como os sinais o guardam.
type improvementDraft struct {
	ID        string
	Name      string
	Scope     string
	Character string
	Family    string
	Target    string
	Skill     string
	Amount    string
}

func blankImprovementDraft() improvementDraft {
	return improvementDraft{Scope: "mesa", Target: "expertise", Skill: "Luta", Amount: "1"}
}

func readImprovementDraft(r *http.Request) (improvementDraft, error) {
	var signals map[string]json.RawMessage
	if err := datastar.ReadSignals(r, &signals); err != nil {
		return improvementDraft{}, fmt.Errorf("não entendi o editor: %w", err)
	}
	text := func(key string) string { return signalText(signals, "table_improvement_"+key) }
	return improvementDraft{
		ID: text("id"), Name: text("name"), Scope: text("scope"), Character: text("character"),
		Family: text("family"), Target: text("target"), Skill: text("skill"), Amount: text("amount"),
	}, nil
}

func (d improvementDraft) normalized() improvementDraft {
	if d.Target == "expertise" && d.Skill == "" {
		d.Skill = "Luta"
	}
	return d
}

func (d improvementDraft) effect() engine.RuleEffect { return addsEffect(d.Target, d.Skill, d.Amount) }

func (d improvementDraft) toSave() campaign.TableImprovementDraft {
	_, character := seatApplies(d.Scope, d.Character)
	return campaign.TableImprovementDraft{ID: d.ID, Name: strings.TrimSpace(d.Name), CharacterID: character, Family: d.Family, Effect: d.effect()}
}

func (d improvementDraft) signals() map[string]any {
	return map[string]any{
		"table_improvement_id": d.ID, "table_improvement_name": d.Name, "table_improvement_scope": d.Scope,
		"table_improvement_character": d.Character, "table_improvement_family": d.Family,
		"table_improvement_target": d.Target, "table_improvement_skill": d.Skill, "table_improvement_amount": d.Amount,
	}
}

// tableImprovementsSignals é a declaração dos sinais do editor, na raiz da aba.
func tableImprovementsSignals() string {
	written, err := json.Marshal(blankImprovementDraft().signals())
	if err != nil {
		return "{}"
	}
	return string(written)
}

func draftOfImprovement(improvement campaign.StoredTableImprovement) improvementDraft {
	draft := improvementDraft{
		ID: improvement.ID, Name: improvement.Name, Scope: "mesa", Family: improvement.Family,
		Target: improvement.Effect.Target.K, Skill: improvement.Effect.Target.Name, Amount: strconv.Itoa(improvement.Effect.Amount),
	}
	if improvement.CharacterID != nil {
		draft.Scope, draft.Character = "um", strconv.FormatInt(*improvement.CharacterID, 10)
	}
	return draft
}

type improvementEditorView struct {
	CampaignID int64
	Draft      improvementDraft
	Families   []book.RuleOption
	Targets    []book.RuleOption
	Skills     []book.RuleOption
	Characters []book.RuleOption
	Preview    string
	Audience   string
	Uses       string
	Refusal    string
}

func (s Scene) improvementEditorOf(ctx context.Context, campaignID int64, draft improvementDraft, refusal string) (improvementEditorView, error) {
	v := improvementEditorView{
		CampaignID: campaignID, Draft: draft, Families: book.TableImprovementFamilies(), Targets: book.RuleTargets(),
		Refusal: refusal, Preview: book.WrittenImprovementPreview(draft.Family, draft.effect()),
		Audience:   "Todos da mesa a veem na lista de melhorias.",
		Characters: []book.RuleOption{{ID: "", Name: "Escolha o personagem"}},
	}
	for _, expertise := range book.Expertises() {
		v.Skills = append(v.Skills, book.RuleOption{ID: expertise.Name, Name: expertise.Name})
	}
	names, _, err := s.seatsOf(ctx, campaignID, 0)
	if err != nil {
		return v, err
	}
	members, err := s.deps.Queries().ListMembers(ctx, campaignID)
	if err != nil {
		return v, err
	}
	for _, member := range members {
		v.Characters = append(v.Characters, book.RuleOption{ID: strconv.FormatInt(member.Characterid, 10), Name: names[member.Characterid]})
	}
	if draft.Scope == "um" {
		v.Audience = "Escolha quem pode usar."
		if id, err := strconv.ParseInt(draft.Character, 10, 64); err == nil && names[id] != "" {
			v.Audience = "Só " + names[id] + " a vê na lista de melhorias."
		}
	}
	if draft.ID != "" {
		uses, err := s.life.TableImprovementUses(ctx, campaignID)
		if err != nil {
			return v, err
		}
		v.Uses = writtenItemsUsing(uses[draft.ID])
	}
	return v, nil
}
