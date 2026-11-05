package campaigns

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/starfederation/datastar-go/datastar"

	"t20engine/app/campaign"
	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// A ABA REGRAS DA MESA: a lista e o editor guiado. O rascunho mora nos sinais, e o servidor
// redesenha o editor a cada troca: as opções de uma condição dependem da espécie dela, e mandar a
// lista de toda espécie em toda linha seria milhares de opções escondidas.

// tableRuleRow é uma linha da lista.
type tableRuleRow struct {
	ID      string
	Name    string
	Meta    string
	Enabled bool
}

// tableRulesOf são as regras que quem olha lê: todas para o mestre, e as ligadas para o jogador.
func (s Scene) tableRulesOf(ctx context.Context, campaignID int64, isGM bool, euID int64) ([]tableRuleRow, error) {
	rules, err := s.life.TableRules(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	members, err := s.deps.Queries().ListMembers(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	names, mine := map[int64]string{}, map[int64]bool{}
	for _, member := range members {
		names[member.Characterid] = memberName(member.Charname, member.Characterid)
		mine[member.Characterid] = member.Charownerid == euID
	}
	out := []tableRuleRow{}
	for _, rule := range rules {
		// O jogador lê o que vale para ele: a regra individual de outra pessoa é da outra pessoa.
		if !isGM && (!rule.Enabled || (rule.CharacterID != nil && !mine[*rule.CharacterID])) {
			continue
		}
		scope := "Toda a mesa"
		if rule.CharacterID != nil {
			scope = names[*rule.CharacterID]
		}
		out = append(out, tableRuleRow{
			ID: rule.ID, Name: rule.Name, Enabled: rule.Enabled,
			Meta: scope + " · " + book.WrittenTableRule(rule.When, rule.Effect),
		})
	}
	return out, nil
}

// O RASCUNHO, como os sinais o guardam. As condições são sinais numerados, de 1 até a conta.

func ruleKindSignal(slot int) string  { return "table_rule_kind_" + strconv.Itoa(slot) }
func ruleValueSignal(slot int) string { return "table_rule_value_" + strconv.Itoa(slot) }

type ruleDraft struct {
	ID        string
	Name      string
	Scope     string
	Character string
	When      []engine.RuleCondition
	Target    string
	Skill     string
	Amount    string
}

func blankRuleDraft() ruleDraft {
	return ruleDraft{Scope: "mesa", Target: "attack", Amount: "1", When: []engine.RuleCondition{}}
}

// readRuleDraft lê os sinais do editor.
func readRuleDraft(r *http.Request) (ruleDraft, error) {
	var signals map[string]json.RawMessage
	if err := datastar.ReadSignals(r, &signals); err != nil {
		return ruleDraft{}, fmt.Errorf("não entendi o editor: %w", err)
	}
	text := func(key string) string {
		var written string
		if json.Unmarshal(signals[key], &written) == nil {
			return written
		}
		return strings.TrimSpace(string(signals[key])) // é número, ou não veio
	}
	draft := ruleDraft{
		ID: text("table_rule_id"), Name: text("table_rule_name"), Scope: text("table_rule_scope"),
		Character: text("table_rule_character"), Target: text("table_rule_target"),
		Skill: text("table_rule_skill"), Amount: text("table_rule_amount"), When: []engine.RuleCondition{},
	}
	count, _ := strconv.Atoi(text("table_rule_count"))
	for slot := 1; slot <= min(count, book.MaxRuleConditions); slot++ {
		draft.When = append(draft.When, engine.RuleCondition{Kind: text(ruleKindSignal(slot)), Value: text(ruleValueSignal(slot))})
	}
	return draft, nil
}

// normalized acerta o que a troca deixou para trás: a espécie mudou e o valor era da outra.
func (d ruleDraft) normalized() ruleDraft {
	for i, condition := range d.When {
		if condition.Kind == "" {
			condition.Kind = engine.RuleOnItem
		}
		options := book.RuleOptions(condition.Kind)
		if condition.Kind == engine.RuleOnVested {
			condition.Value = ""
		} else if len(options) > 0 && !optionListed(options, condition.Value) {
			condition.Value = options[0].ID
		}
		d.When[i] = condition
	}
	if d.Target == "expertise" && d.Skill == "" {
		d.Skill = "Luta"
	}
	return d
}

func optionListed(options []book.RuleOption, id string) bool {
	for _, option := range options {
		if option.ID == id {
			return true
		}
	}
	return false
}

func (d ruleDraft) effect() engine.RuleEffect {
	amount, _ := strconv.Atoi(strings.TrimPrefix(d.Amount, "+"))
	target := engine.ModifierTarget{K: d.Target}
	if d.Target == "expertise" {
		target.Name = d.Skill
	}
	return engine.RuleEffect{Kind: engine.RuleAdds, Target: target, Amount: amount}
}

func (d ruleDraft) applies() (engine.Selector, *int64) {
	if d.Scope != "um" {
		return engine.EveryoneIn(), nil
	}
	id, err := strconv.ParseInt(d.Character, 10, 64)
	if err != nil {
		id = 0
	}
	return engine.OnlyCharacter(int(id)), &id
}

func (d ruleDraft) toSave() campaign.TableRuleDraft {
	_, character := d.applies()
	return campaign.TableRuleDraft{ID: d.ID, Name: strings.TrimSpace(d.Name), CharacterID: character, When: d.When, Effect: d.effect()}
}

// signals são os sinais do rascunho, para o editor abrir e para o que `normalized` acertou.
func (d ruleDraft) signals() map[string]any {
	out := map[string]any{
		"table_rule_id": d.ID, "table_rule_name": d.Name, "table_rule_scope": d.Scope,
		"table_rule_character": d.Character, "table_rule_target": d.Target, "table_rule_skill": d.Skill,
		"table_rule_amount": d.Amount, "table_rule_count": len(d.When),
	}
	for slot := 1; slot <= book.MaxRuleConditions; slot++ {
		kind, value := "", ""
		if slot <= len(d.When) {
			kind, value = d.When[slot-1].Kind, d.When[slot-1].Value
		}
		out[ruleKindSignal(slot)], out[ruleValueSignal(slot)] = kind, value
	}
	return out
}

// signalsThatMoved é o que o servidor mudou no rascunho, e só isso volta ao navegador. Devolver o
// pedido inteiro desfaria o que a pessoa escreveu enquanto a resposta vinha.
func signalsThatMoved(asked, answered map[string]any) map[string]any {
	moved := map[string]any{}
	for name, value := range answered {
		if asked[name] != value {
			moved[name] = value
		}
	}
	return moved
}

// tableRulesSignals é a declaração dos sinais do editor, na raiz da aba.
func tableRulesSignals() string {
	written, err := json.Marshal(blankRuleDraft().signals())
	if err != nil {
		return "{}"
	}
	return string(written)
}

// draftOfRule abre uma regra guardada no editor.
func draftOfRule(rule campaign.StoredTableRule) ruleDraft {
	draft := ruleDraft{
		ID: rule.ID, Name: rule.Name, Scope: "mesa", When: rule.When,
		Target: rule.Effect.Target.K, Skill: rule.Effect.Target.Name, Amount: strconv.Itoa(rule.Effect.Amount),
	}
	if rule.CharacterID != nil {
		draft.Scope, draft.Character = "um", strconv.FormatInt(*rule.CharacterID, 10)
	}
	return draft
}

// O EDITOR, como a tela o desenha.

type ruleConditionLine struct {
	Slot    int
	Kind    string
	Value   string
	Options []book.RuleOption
}

type ruleEditorView struct {
	CampaignID int64
	Draft      ruleDraft
	Lines      []ruleConditionLine
	Kinds      []book.RuleOption
	Targets    []book.RuleOption
	Skills     []book.RuleOption
	Characters []book.RuleOption
	Preview    string
	Reaches    string
	Refusal    string
	CanAdd     bool
}

func (s Scene) ruleEditorOf(ctx context.Context, campaignID int64, draft ruleDraft, refusal string) (ruleEditorView, error) {
	v := ruleEditorView{
		CampaignID: campaignID, Draft: draft, Kinds: book.RuleConditionKinds(), Targets: book.RuleTargets(),
		Refusal: refusal, CanAdd: len(draft.When) < book.MaxRuleConditions,
	}
	for i, condition := range draft.When {
		line := ruleConditionLine{Slot: i + 1, Kind: condition.Kind, Value: condition.Value}
		if condition.Kind != engine.RuleOnVested {
			line.Options = book.RuleOptions(condition.Kind)
		}
		v.Lines = append(v.Lines, line)
	}
	for _, expertise := range book.Expertises() {
		v.Skills = append(v.Skills, book.RuleOption{ID: expertise.Name, Name: expertise.Name})
	}
	members, err := s.deps.Queries().ListMembers(ctx, campaignID)
	if err != nil {
		return v, err
	}
	applies, _ := draft.applies()
	rule := engine.TableRule{Applies: applies, When: book.RuleConditionsForReach(draft.When), Effect: draft.effect()}
	reached := []string{}
	v.Characters = append(v.Characters, book.RuleOption{ID: "", Name: "Escolha o personagem"})
	for _, member := range members {
		name := memberName(member.Charname, member.Characterid)
		v.Characters = append(v.Characters, book.RuleOption{ID: strconv.FormatInt(member.Characterid, 10), Name: name})
		row, err := s.deps.Queries().GetCharacter(ctx, member.Characterid)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return v, err
		}
		dto, err := sheet.Load(ctx, s.deps.Queries(), s.deps.Catalogs(), row)
		if err != nil {
			return v, err
		}
		ec, err := sheet.EngineCharacterFrom(dto)
		if err != nil {
			return v, err
		}
		if dto.Ruleset.RuleReaches(ec, rule) {
			reached = append(reached, name)
		}
	}
	v.Preview = book.WrittenTableRule(draft.When, draft.effect())
	v.Reaches = "Hoje não alcança ninguém."
	if len(reached) > 0 {
		v.Reaches = "Alcança hoje: " + strings.Join(reached, ", ") + "."
	}
	return v, nil
}
