package catalog

import (
	"encoding/json"
	"fmt"
	"sort"
	"testing"
)

// A INVARIANTE que faz o empilhamento por `bonusType` corresponder à p226.
//
// O livro empilha por ORIGEM ("efeitos de habilidades acumulam entre si, exceto
// quando vierem da mesma habilidade") e o motor empilha por TIPO. Medindo os 18
// personagens-oráculo, os dois modelos concordam em TODOS os casos: onze alvos
// recebem contribuições de fontes diferentes e os onze estão corretos — inclusive
// o exemplo que o próprio livro dá ("o bônus na Defesa da Pele de Ferro acumula
// com o da Esquiva Sagaz") e a ressalva explícita de armadura + escudo. E não há
// um único alvo em que o motor compita onde o livro mandaria somar.
//
// Eles concordam porque os `bonusType` do catálogo já correspondem às fontes que
// a p226 lista (`armor`, `shield`, `item`, `training`, `morale`, `enhancement`,
// `condition`), e o `untyped` ficou para o que genuinamente acumula.
//
// O que sustenta isso é esta invariante, que até agora ninguém tinha escrito:
// UMA ENTIDADE NÃO DECLARA DOIS MODIFICADORES NÃO-CONDICIONAIS NO MESMO ALVO E
// TIPO. Quebrá-la faz o motor somar dois efeitos da MESMA habilidade, que é
// exatamente o que a p226 proíbe — e entraria em silêncio, porque nenhum teste
// de regra olha o catálogo (ALE-110).
//
// Duas exceções, e as duas são modelagem legítima que este teste ACHOU ao ser
// escrito, corrigindo a regra que eu tinha em mente:
//
//   - CONDICIONAIS descrevem situações distintas ("+2 se em terreno natural"), e
//     é assim que o catálogo modela o "os bônus dobram" da Força da Natureza.
//   - ESCALAS distintas são componentes distintos da mesma habilidade. O "Duro
//     como Pedra" do Anão — "+3 PV no 1º nível e +1 por nível seguinte" (p20) —
//     é um `maxPv` plano de +2 mais um `maxPv` de +1 por nível; no 1º nível dá
//     os 3 do livro. Somar é o certo ali.
//
// O que sobra depois dessas duas é o caso real: o MESMO efeito declarado duas
// vezes.
func TestNoEntityStacksWithItself(t *testing.T) {
	resources := []string{
		"items", "class-powers", "general-powers", "granted-powers",
		"race-defs", "origins", "divine-powers", "races", "origens",
	}
	total := 0
	for _, name := range resources {
		body, ok := Resource(name)
		if !ok {
			t.Fatalf("recurso %q não está registrado", name)
		}
		var tree any
		if err := json.Unmarshal(body, &tree); err != nil {
			t.Fatalf("%s.json ilegível: %v", name, err)
		}
		for _, owner := range collectModifierOwners(tree, name) {
			total++
			seen := map[string][]float64{}
			for _, m := range owner.modifiers {
				if m["condition"] != nil {
					continue
				}
				key := modifierBucket(m)
				amount, _ := m["amount"].(float64)
				seen[key] = append(seen[key], amount)
			}
			for _, key := range sortedKeys(seen) {
				if len(seen[key]) > 1 {
					t.Errorf("%s: %q declara %d modificadores não-condicionais em %s (%v) — "+
						"o motor os SOMA, e a p226 diz que efeitos da mesma habilidade não acumulam",
						name, owner.id, len(seen[key]), key, seen[key])
				}
			}
		}
	}
	if total == 0 {
		t.Fatal("nenhuma entidade com modificadores encontrada — o teste não está olhando nada")
	}
	t.Logf("%d entidades com modificadores conferidas", total)
}

type modifierOwner struct {
	id        string
	modifiers []map[string]any
}

// collectModifierOwners caminha o JSON procurando toda lista `modifiers`, seja
// qual for a forma do catálogo — os itens a trazem na raiz, as origens dentro de
// `benefits` e de `poderUnico`, as raças dentro das habilidades. Caminhar em vez
// de modelar cada forma faz o teste cobrir um catálogo novo sem mudança.
func collectModifierOwners(node any, path string) []modifierOwner {
	var out []modifierOwner
	switch v := node.(type) {
	case map[string]any:
		label := path
		for _, field := range []string{"id", "name"} {
			if s, ok := v[field].(string); ok && s != "" {
				label = s
				break
			}
		}
		if raw, ok := v["modifiers"].([]any); ok {
			mods := make([]map[string]any, 0, len(raw))
			for _, m := range raw {
				if mm, ok := m.(map[string]any); ok {
					mods = append(mods, mm)
				}
			}
			if len(mods) > 0 {
				out = append(out, modifierOwner{id: label, modifiers: mods})
			}
		}
		for _, key := range sortedAnyKeys(v) {
			if key == "modifiers" {
				continue
			}
			out = append(out, collectModifierOwners(v[key], label)...)
		}
	case []any:
		for _, item := range v {
			out = append(out, collectModifierOwners(item, path)...)
		}
	}
	return out
}

// modifierBucket é a identidade que o motor usa para empilhar: alvo + tipo.
// Espelha o `targetKey` do engine — dois modificadores que caem no mesmo balde
// somam ou competem, dependendo do tipo.
func modifierBucket(m map[string]any) string {
	target, _ := m["target"].(map[string]any)
	kind, _ := target["k"].(string)
	bonus, _ := m["bonusType"].(string)

	// A escala faz parte da identidade: um bônus plano e um por nível são
	// componentes DIFERENTES da mesma habilidade, não o mesmo efeito repetido.
	scale := "flat"
	if sc, ok := m["scale"].(map[string]any); ok {
		per, _ := sc["per"].(string)
		attr, _ := sc["attribute"].(string)
		step, _ := sc["step"].(float64)
		scale = fmt.Sprintf("%s/%s/%g", per, attr, step)
	}

	detail := ""
	switch kind {
	case "expertise", "expertiseRemovePenalty", "attribute":
		detail, _ = target["name"].(string)
	case "expertiseByAttribute":
		detail, _ = target["attribute"].(string)
	case "attack", "damage", "defense":
		detail, _ = target["scope"].(string)
	case "catalyst":
		detail, _ = target["school"].(string)
	case "flag":
		detail, _ = target["flag"].(string)
	}
	if detail != "" {
		kind += ":" + detail
	}
	return fmt.Sprintf("%s [%s] escala=%s", kind, bonus, scale)
}

func sortedKeys(m map[string][]float64) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedAnyKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
