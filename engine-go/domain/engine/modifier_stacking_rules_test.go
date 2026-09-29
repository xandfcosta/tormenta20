package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
// O que sustenta isso é esta invariante: UMA ENTIDADE NÃO DECLARA DOIS
// MODIFICADORES NÃO-CONDICIONAIS NO MESMO ALVO E TIPO. Quebrá-la faz o motor
// somar dois efeitos da MESMA habilidade, que é exatamente o que a p226 proíbe
// — e entraria em silêncio, porque nenhum teste de regra olha o catálogo.
//
// Duas exceções, e as duas são modelagem legítima:
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
//
// # Por que ele mora AQUI, e não no `domain/catalog`
//
// Porque a identidade do balde é o `targetKey`, e o `targetKey` é daqui. Lá ele
// era ESPELHADO num `switch` próprio, e o espelho divergiu em três pontos
// (ALE-419): a `flag` era lida de `target["flag"]` e o catálogo a escreve em
// `name`; o `maneuver` e o `displacement` não separavam por escopo.
//
// A divergência era INERTE — medida, ela muda 12 baldes de 108 e nenhum
// veredito, porque os casos que colidiriam são condicionais e este guarda pula
// condicional. O que ela não era é segura: um segundo modificador de flag
// não-condicional na mesma entidade passaria despercebido, e nada acusaria.
//
// Aqui não há espelho: o balde sai do `targetKey` de verdade, e do `Modifier`
// de verdade — o que faz o guarda atravessar o `UnmarshalJSON` junto, que é a
// fronteira que a ALE-415 pegou quebrada.
func TestNoEntityStacksWithItself(t *testing.T) {
	arquivos, err := filepath.Glob(filepath.Join("..", "catalog", "data", "*.json"))
	if err != nil {
		t.Fatalf("listar o catálogo: %v", err)
	}
	if len(arquivos) < 10 {
		t.Fatalf("só %d arquivos de catálogo — o guarda mediria quase nada", len(arquivos))
	}

	total := 0
	for _, caminho := range arquivos {
		bruto, err := os.ReadFile(caminho)
		if err != nil {
			t.Fatalf("ler %s: %v", caminho, err)
		}
		var tree any
		if err := json.Unmarshal(bruto, &tree); err != nil {
			t.Fatalf("%s ilegível: %v", caminho, err)
		}
		nome := filepath.Base(caminho)
		for _, owner := range collectModifierOwners(tree, nome) {
			total++
			seen := map[string][]int{}
			for _, m := range modifiersOf(t, owner) {
				if m.Condition != nil {
					continue
				}
				key := stackingBucket(m)
				seen[key] = append(seen[key], m.Amount)
			}
			for _, key := range sortedKeys(seen) {
				if len(seen[key]) > 1 {
					t.Errorf("%s: %q declara %d modificadores não-condicionais em %s (%v) — "+
						"o motor os SOMA, e a p226 diz que efeitos da mesma habilidade não acumulam",
						nome, owner.id, len(seen[key]), key, seen[key])
				}
			}
		}
	}
	if total == 0 {
		t.Fatal("nenhuma entidade com modificadores encontrada — o teste não está olhando nada")
	}
	t.Logf("%d entidades com modificadores conferidas", total)
}

// stackingBucket é a identidade que o motor usa para empilhar: o ALVO, o tipo e
// a escala.
//
// O alvo sai do `targetKey` e não de uma cópia dele. A escala faz parte da
// identidade porque um bônus plano e um por nível são componentes DIFERENTES da
// mesma habilidade, não o mesmo efeito repetido.
func stackingBucket(m Modifier) string {
	escala := "flat"
	if s := m.Scale; s != nil {
		escala = fmt.Sprintf("%s/%s/%d", s.Per, s.Attribute, s.Step)
	}
	return fmt.Sprintf("%s [%s] escala=%s", targetKey(m.Target), m.BonusType, escala)
}

// modifiersOf lê a lista do dono no tipo do MOTOR.
//
// Passa pelo JSON de novo porque o caminhador entrega `map[string]any` — ele
// precisa ser genérico para achar toda lista `modifiers` seja qual for a forma
// do catálogo. Reconstruir o `Modifier` campo a campo aqui seria uma segunda
// definição do formato, que é o que este guarda acabou de deixar de ter.
func modifiersOf(t *testing.T, owner modifierOwner) []Modifier {
	t.Helper()
	bruto, err := json.Marshal(owner.modifiers)
	if err != nil {
		t.Fatalf("%s: reescrever os modificadores: %v", owner.id, err)
	}
	var lidos []Modifier
	if err := json.Unmarshal(bruto, &lidos); err != nil {
		t.Fatalf("%s: os modificadores não cabem no `Modifier`: %v", owner.id, err)
	}
	return lidos
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

func sortedKeys(m map[string][]int) []string {
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
