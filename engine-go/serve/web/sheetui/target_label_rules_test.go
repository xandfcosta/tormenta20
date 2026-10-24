package sheetui

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode"

	"t20engine/domain/catalog"
	"t20engine/domain/engine"
)

// TODO ALVO QUE O CATÁLOGO ESCREVE TEM RÓTULO EM PORTUGUÊS (ALE-453).
//
// O `targetLabel` é uma tabela de PERMITIDOS com queda silenciosa: alvo que ela
// não conhece volta como o próprio `k`, e o crachá sai escrito
// `armorPenaltyExempt (Fur…)` na ficha de quem calhar de ter o Manto da noite.
// A queda FICA — ela é melhor que uma frase inventada —, mas ela deixa de ser
// alcançável por alvo que o catálogo de fato usa, e quem garante isso é daqui.
//
// É o irmão de tela do `TestTheCatalogOnlyUsesTargetsTheEngineApplies`, que faz
// a mesma pergunta pelo lado da CONTA: lá, "o motor aplica este alvo?"; aqui,
// "a tela sabe dizer o nome dele?". Sete alvos passaram anos sem rótulo porque
// nenhuma das duas perguntas era feita por varredura.
//
// # Duas formas do mesmo defeito, e a segunda é a que escapa
//
// A primeira é o `k` cru — "source", "nonLethalDamage". A segunda é o rótulo
// certo com um ID no complemento: `targetComplement` devolve o `name` como
// veio, então um alvo novo com `name` de id põe "Atributo da Defesa
// (intelligence)" e "O livro prevê (sangrando)" na tela. Um guarda que só
// olhasse o `k` passaria verde sobre os dois.
func TestEveryTargetTheCatalogWritesIsNamedInPortuguese(t *testing.T) {
	targets := targetsTheCatalogWrites(t)
	// O DENOMINADOR: uma varredura que não abriu nenhum catálogo e uma tabela
	// de rótulos impecável têm a mesma cor no terminal.
	if len(targets) < 25 {
		t.Fatalf("só %d formas de alvo varridas — a varredura está lendo o campo errado", len(targets))
	}

	for _, target := range targets {
		label := targetLabel(target)
		if label == "" || strings.HasPrefix(label, target.K) {
			t.Errorf("o alvo %q sai na tela como %q — a chave do motor, em inglês.\n"+
				"Escreva o rótulo dele no `targetLabel`, com a palavra que a casa já "+
				"usa para o conceito.", target.K, label)
			continue
		}
		if echoesARawName(target, label) {
			t.Errorf("o rótulo do alvo %q sai como %q, e %q é id do catálogo e não "+
				"texto de tela.\nO `targetComplement` devolve o `name` como veio: dê a "+
				"ele um tradutor, como o `attribute` já tem — ou declare o alvo em "+
				"`namesThatAreAlreadyScreenText`, com o motivo.",
				target.K, label, target.Name)
		}
	}
	t.Logf("%d formas de alvo conferidas contra a tabela de rótulos", len(targets))
}

// namesThatAreAlreadyScreenText são os alvos cujo `name` JÁ é português de
// tela, e por isso podem aparecer no rótulo como vieram.
//
// É uma lista de PERMITIDOS, e de propósito: a pergunta "este nome é um id?"
// não tem resposta por forma. A primeira versão deste guarda tentou por forma —
// "minúsculas com hífen ou ponto" — e passou verde sobre `Atributo da Defesa
// (intelligence)`, que não tem hífen nenhum. Uma lista de proibidos subconta em
// silêncio; esta reprova o que ela não conhece, e quem acrescenta um alvo
// escreve por que o nome dele pode ir cru para a tela.
var namesThatAreAlreadyScreenText = map[string]string{
	"maneuver": "o nome da manobra é o do livro em minúsculas — " +
		"\"Manobra (derrubar)\" é a frase certa (p234)",
}

// echoesARawName diz se o rótulo repetiu o `name` do alvo como ele veio.
//
// Só o `name` minúsculo: o nome de PERÍCIA é maiúsculo ("Furtividade") e é
// texto de tela legítimo, e o tipo de dano e a escola não viajam neste campo.
func echoesARawName(target engine.ModifierTarget, label string) bool {
	if target.Name == "" || namesThatAreAlreadyScreenText[target.K] != "" {
		return false
	}
	if first := []rune(target.Name)[0]; !unicode.IsLower(first) {
		return false
	}
	return strings.Contains(label, target.Name)
}

// targetsTheCatalogWrites colhe toda forma DISTINTA de alvo que os catálogos
// embutidos declaram — a chave mais os campos que o rótulo usa.
//
// Pelo `catalog.Resource` e não por caminho de arquivo: o que importa é o que
// o BINÁRIO serve, e um caminho relativo quebraria no dia em que este pacote
// mudasse de lugar — em silêncio, devolvendo zero alvos.
func targetsTheCatalogWrites(t *testing.T) []engine.ModifierTarget {
	t.Helper()
	seen := map[string]bool{}
	targets := []engine.ModifierTarget{}

	var descend func(node any)
	descend = func(node any) {
		switch v := node.(type) {
		case map[string]any:
			if raw, ok := v["target"].(map[string]any); ok {
				if target, fine := targetFrom(raw); fine {
					key := target.K + "|" + target.Name + "|" + target.Scope +
						"|" + target.Attribute + "|" + target.DamageType
					if !seen[key] {
						seen[key] = true
						targets = append(targets, target)
					}
				}
			}
			for _, child := range v {
				descend(child)
			}
		case []any:
			for _, child := range v {
				descend(child)
			}
		}
	}

	resources := catalog.Resources()
	if len(resources) < 10 {
		t.Fatalf("só %d recursos de catálogo — o guarda mediria quase nada", len(resources))
	}
	for _, name := range resources {
		raw, ok := catalog.Resource(name)
		if !ok {
			t.Fatalf("o recurso %q está na lista e não no binário", name)
		}
		var anything any
		if err := json.Unmarshal(raw, &anything); err != nil {
			t.Fatalf("%s não é JSON: %v", name, err)
		}
		descend(anything)
	}
	return targets
}

// targetFrom remonta o alvo a partir do nó cru do JSON.
func targetFrom(raw map[string]any) (engine.ModifierTarget, bool) {
	k, ok := raw["k"].(string)
	if !ok || k == "" {
		return engine.ModifierTarget{}, false
	}
	text := func(field string) string {
		s, _ := raw[field].(string)
		return s
	}
	return engine.ModifierTarget{
		K: k, Name: text("name"), Scope: text("scope"),
		Attribute: text("attribute"), DamageType: text("damageType"),
		School: text("school"),
	}, true
}
