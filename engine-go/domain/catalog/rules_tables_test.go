package catalog

import (
	"encoding/json"
	"strings"
	"testing"
)

// As quatro tabelas AUTORADAS aqui. O que protege dado transcrito é validação de
// SCHEMA e de invariante, e não um `expect` por campo repetindo o mesmo número:
// o risco é typo, não regressão.
//
// O que cada teste cobre é a FORMA e os invariantes que quebram uma tela —
// faixa de rolagem com buraco, perícia fora da lista, rótulo faltando. Os
// valores em si são conferidos contra o livro na revisão, não aqui.

func decodeResource[T any](t *testing.T, name string) T {
	t.Helper()
	body, ok := Resource(name)
	if !ok {
		t.Fatalf("recurso %q não está registrado em `resources`", name)
	}
	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("%s.json não casa com o schema esperado: %v", name, err)
	}
	return out
}

type classExpertises struct {
	Fixed    []string `json:"fixed"`
	EitherOr *struct {
		Options []string `json:"options"`
	} `json:"eitherOr"`
	ChooseCount int      `json:"chooseCount"`
	ChoosePool  []string `json:"choosePool"`
}

// Concessões de perícia por classe. O erro que derruba a Forja é uma perícia
// escrita errado — a tela ofereceria uma opção que o resto do app não conhece.
func TestClassExpertisesTable(t *testing.T) {
	table := decodeResource[map[string]classExpertises](t, "class-expertises")
	known := expertiseNames(t)

	if len(table) != 14 {
		t.Errorf("a tabela tem %d classes, want 14 (as classes-base do livro)", len(table))
	}
	for class, entry := range table {
		if len(entry.Fixed) == 0 {
			t.Errorf("%s: sem perícias fixas — toda classe treina ao menos uma", class)
		}
		if entry.ChooseCount > 0 && len(entry.ChoosePool) < entry.ChooseCount {
			t.Errorf("%s: escolhe %d de um pool de %d — impossível",
				class, entry.ChooseCount, len(entry.ChoosePool))
		}
		if entry.EitherOr != nil && len(entry.EitherOr.Options) != 2 {
			t.Errorf("%s: o slot ou/ou tem %d opções, want 2", class, len(entry.EitherOr.Options))
		}
		for _, list := range [][]string{entry.Fixed, entry.ChoosePool} {
			for _, name := range list {
				if !known[name] {
					t.Errorf("%s: perícia %q não existe no catálogo de perícias", class, name)
				}
			}
		}
		if entry.EitherOr != nil {
			for _, name := range entry.EitherOr.Options {
				if !known[name] {
					t.Errorf("%s: perícia %q do slot ou/ou não existe", class, name)
				}
			}
		}
	}
}

// classeDoCatalogo é o que `classes.json` guarda de cada uma das 14.
type classeDoCatalogo struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	BookPage      int      `json:"bookPage"`
	Proficiencies []string `json:"proficiencies"`
}

// Proficiências por classe (a linha "Proficiências." de cada bloco, p36–83).
//
// O risco deste dado é typo, como o das perícias: uma categoria escrita errado
// não explode — ela some. A classe deixa de conceder o que o livro concede, o
// painel da ficha mostra a linha desmarcada, e o motor passa a aplicar a
// penalidade da p142 num personagem que devia ser proficiente. Nada disso levanta
// exceção em lugar nenhum.
//
// NINGUÉM CONCEDE `armas-simples`: ela é de todos os personagens por regra
// (p142), e uma classe listando-a duplicaria a fonte na etiqueta ("Todas as
// classes, Guerreiro") sem mudar nada — sintoma de quem transcreveu a frase do
// livro em vez da linha da classe.
func TestClassProficienciesTable(t *testing.T) {
	classes := decodeResource[[]classeDoCatalogo](t, "classes")
	if len(classes) != 14 {
		t.Errorf("a tabela tem %d classes, want 14 (as classes-base do livro)", len(classes))
	}
	known := map[string]bool{
		"armas-simples": true, "armas-marciais": true, "armas-exoticas": true,
		"armas-de-fogo": true, "armaduras-leves": true, "armaduras-pesadas": true,
		"escudos": true,
	}
	for _, c := range classes {
		seen := map[string]bool{}
		for _, cat := range c.Proficiencies {
			if !known[cat] {
				t.Errorf("%s: proficiência %q não é uma das sete categorias", c.Name, cat)
			}
			if cat == "armas-simples" {
				t.Errorf("%s: lista `armas-simples`, que é de TODO personagem (p142) e não da classe", c.Name)
			}
			if seen[cat] {
				t.Errorf("%s: proficiência %q repetida", c.Name, cat)
			}
			seen[cat] = true
		}
	}
}

// expertiseNames lê as perícias do catálogo de opções — a mesma lista que a
// criação de personagem oferece, então uma divergência aqui é a que o jogador vê.
func expertiseNames(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := Options()
	if err != nil {
		t.Fatalf("ler options.json: %v", err)
	}
	var opts struct {
		Expertises []struct {
			Name string `json:"name"`
		} `json:"expertises"`
	}
	if err := json.Unmarshal(raw, &opts); err != nil {
		t.Fatalf("options.json ilegível: %v", err)
	}
	names := map[string]bool{}
	for _, e := range opts.Expertises {
		names[e.Name] = true
	}
	if len(names) == 0 {
		t.Fatal("options.json não trouxe perícia nenhuma")
	}
	return names
}

// Termos de devoto (p96). O mapa traduz a redação do livro ("Anões", "Sereias/
// Tritões") para os nomes que o app guarda — um nome errado aqui silenciosamente
// nega devoção a quem tem direito.
func TestDevotoTermsTable(t *testing.T) {
	table := decodeResource[struct {
		OpenTerms   []string            `json:"openTerms"`
		TermToNames map[string][]string `json:"termToNames"`
	}](t, "devotee-terms")

	if len(table.OpenTerms) == 0 {
		t.Error("sem termos abertos — 'Quaisquer' deixaria de admitir todo mundo")
	}
	known := optionNames(t)
	for term, names := range table.TermToNames {
		if len(names) == 0 {
			t.Errorf("termo %q não mapeia para nada — negaria devoção em silêncio", term)
		}
		for _, n := range names {
			if !known[n] {
				t.Errorf("termo %q aponta para %q, que não é raça nem classe do app", term, n)
			}
		}
	}
}

// optionNames é o conjunto de raças + classes que o app conhece.
func optionNames(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := Options()
	if err != nil {
		t.Fatalf("ler options.json: %v", err)
	}
	var opts struct {
		Races   []string `json:"races"`
		Classes []string `json:"classes"`
	}
	if err := json.Unmarshal(raw, &opts); err != nil {
		t.Fatalf("options.json ilegível: %v", err)
	}
	names := map[string]bool{}
	for _, n := range append(opts.Races, opts.Classes...) {
		names[n] = true
	}
	return names
}

// rollRow cobre as duas formas que as tabelas usam: uma FAIXA (rollMin/rollMax,
// como "1-2: uma ameaça") ou uma rolagem única.
type rollRow struct {
	Roll    int `json:"roll"`
	RollMin int `json:"rollMin"`
	RollMax int `json:"rollMax"`
}

// span devolve a faixa que a linha cobre, normalizando a forma de rolagem única.
func (r rollRow) span() (int, int) {
	if r.RollMin != 0 || r.RollMax != 0 {
		return r.RollMin, r.RollMax
	}
	return r.Roll, r.Roll
}

// Tabelas de rolagem do Improviso. O defeito que estraga a ferramenta é um
// BURACO na faixa: o mestre rola um número que a tabela não cobre e a tela não
// mostra nada. Por isso o teste exige cobertura contígua a partir de 1.
func TestGmRollTablesCoverTheirRange(t *testing.T) {
	tables := decodeResource[struct {
		Ruin             []rollRow         `json:"ruina"`
		ChaseEvents      []rollRow         `json:"chaseEvents"`
		RewardPunishment []rollRow         `json:"rewardCastigo"`
		RewardLabels     map[string]string `json:"rewardLabels"`
		PunishmentLabels map[string]string `json:"castigoLabels"`
	}](t, "gm-tables")

	for _, tt := range []struct {
		name string
		rows []rollRow
	}{
		{"ruina", tables.Ruin},
		{"chaseEvents", tables.ChaseEvents},
		{"rewardCastigo", tables.RewardPunishment},
	} {
		assertContiguousRolls(t, tt.name, tt.rows)
	}

	if len(tables.RewardLabels) == 0 || len(tables.PunishmentLabels) == 0 {
		t.Error("rótulos de recompensa/castigo vazios — a tela mostraria a chave crua")
	}
}

// Ideias de masmorra: a mesma regra de faixa, mais os tamanhos, cujas faixas de
// salas não podem se sobrepor nem deixar vão.
func TestDungeonDesignTable(t *testing.T) {
	table := decodeResource[struct {
		Sizes     []string `json:"sizes"`
		SizeTable []struct {
			Size     string `json:"size"`
			MinRooms int    `json:"minRooms"`
			MaxRooms int    `json:"maxRooms"`
		} `json:"sizeTable"`
		RoomsPerThreat int       `json:"roomsPerThreat"`
		Ideas          []rollRow `json:"ideas"`
	}](t, "dungeon-design")

	assertContiguousRolls(t, "ideas", table.Ideas)

	if table.RoomsPerThreat <= 0 {
		t.Errorf("roomsPerThreat = %d — dividir por isso estoura a tela", table.RoomsPerThreat)
	}
	if len(table.SizeTable) != len(table.Sizes) {
		t.Errorf("%d tamanhos listados e %d linhas na tabela", len(table.Sizes), len(table.SizeTable))
	}
	// As faixas têm de encadear: o mínimo de uma é o máximo da anterior + 1.
	for i := 1; i < len(table.SizeTable); i++ {
		prev, cur := table.SizeTable[i-1], table.SizeTable[i]
		if cur.MinRooms != prev.MaxRooms+1 {
			t.Errorf("%s começa em %d salas e %s termina em %d — vão ou sobreposição",
				cur.Size, cur.MinRooms, prev.Size, prev.MaxRooms)
		}
	}
}

// assertContiguousRolls exige que as rolagens cubram 1..N sem buraco e sem
// repetição — um buraco faz a ferramenta devolver vazio para um d20 legítimo.
func assertContiguousRolls(t *testing.T, name string, rows []rollRow) {
	t.Helper()
	if len(rows) == 0 {
		t.Errorf("%s: tabela vazia", name)
		return
	}
	seen := map[int]bool{}
	max := 0
	for _, r := range rows {
		lo, hi := r.span()
		if lo == 0 || hi < lo {
			t.Errorf("%s: faixa inválida %d..%d", name, lo, hi)
			continue
		}
		for roll := lo; roll <= hi; roll++ {
			if seen[roll] {
				t.Errorf("%s: rolagem %d aparece em duas linhas", name, roll)
			}
			seen[roll] = true
		}
		if hi > max {
			max = hi
		}
	}
	for roll := 1; roll <= max; roll++ {
		if !seen[roll] {
			t.Errorf("%s: rolagem %d não está coberta (faixa vai até %d)", name, roll, max)
		}
	}
}

// spellForTruqueSweep é o subconjunto que a varredura de truque precisa: o nome,
// para a falha dizer QUAL magia, e os aprimoramentos crus.
type spellForTruqueSweep struct {
	Name string `json:"name"`
	// BaseEffect entra na varredura porque foi ONDE o truque do Hipnotismo estava:
	// colado no fim da prosa do efeito base, em vez de ser um aprimoramento como
	// nas outras treze. Assim ele não aparecia na lista, não custava zero e a
	// contagem dava treze — a falta mais silenciosa das três (ALE-339).
	BaseEffect string `json:"baseEffect"`
	Augments   []struct {
		PmCost      int    `json:"pmCost"`
		Description string `json:"description"`
		Exclusive   bool   `json:"exclusive"`
		Cantrip     bool   `json:"truque"`
	} `json:"augments"`
}

// TestEveryTruqueIsFreeAndAlone amarra as duas metades da frase da p171 ao DADO:
// o truque custa zero e não aceita companhia.
//
// A varredura existe porque o truque tem TRÊS marcas que precisam concordar — o
// prefixo `Truque:` da descrição, o `truque` e o `exclusive` — e elas moram em
// lugares diferentes do mesmo registro. Duas concordando e uma não é um truque
// que a tela anuncia e o servidor cobra, ou o contrário.
//
// O NÚMERO está preso porque é a única defesa contra a falta em silêncio: a
// primeira contagem achou ONZE, e os três que faltavam foram a fatura da
// ALE-339 — a Cura de Ferimentos (p189) e a Queda Suave (p202) cobravam 1 PM por
// um truque, e o Hipnotismo (p194) não tinha o dele. Nenhum schema reprova um
// aprimoramento que simplesmente não está lá.
func TestEveryTruqueIsFreeAndAlone(t *testing.T) {
	spells := decodeResource[map[string]spellForTruqueSweep](t, "spells")
	if len(spells) == 0 {
		t.Fatal("nenhuma magia no catálogo: não há o que varrer, e verde aqui não valeria nada")
	}

	cantrips, measured := 0, 0
	for id, m := range spells {
		inSpell := 0
		for i, a := range m.Augments {
			measured++
			written := strings.HasPrefix(a.Description, "Truque")
			if written != a.Cantrip {
				t.Errorf(
					"%s (%s) aprimoramento %d: a descrição %s de truque e o campo `truque` diz %v",
					m.Name, id, i, map[bool]string{true: "fala", false: "não fala"}[written], a.Cantrip,
				)
				continue
			}
			if !a.Cantrip {
				continue
			}
			cantrips++
			inSpell++
			if a.PmCost != 0 {
				t.Errorf("%s (%s) aprimoramento %d: o truque cobra %d PM, e ele zera o custo (p171)",
					m.Name, id, i, a.PmCost)
			}
			if !a.Exclusive {
				t.Errorf("%s (%s) aprimoramento %d: o truque não está marcado como exclusivo, e a p171 diz que ele não aceita companhia",
					m.Name, id, i)
			}
		}
		if inSpell > 1 {
			t.Errorf("%s (%s) tem %d truques, e o livro imprime no máximo um por magia", m.Name, id, inSpell)
		}
		if strings.Contains(m.BaseEffect, "Truque") {
			t.Errorf(
				"%s (%s) fala do truque no efeito base: o truque é APRIMORAMENTO, e escrito na prosa ele não entra na lista nem zera o custo",
				m.Name, id,
			)
		}
	}

	// O DENOMINADOR em duas contas: quantos aprimoramentos a varredura leu, e
	// quantos truques ela achou. Sem a primeira, um seletor que não casasse com
	// nada seria verde; sem a segunda, um truque apagado do catálogo também.
	if measured < 400 {
		t.Fatalf("a varredura leu %d aprimoramentos, e são quase 500 — o recurso é o primeiro suspeito", measured)
	}
	if cantrips != 14 {
		t.Errorf("o catálogo tem %d truques, e o livro tem 14 (contados no capítulo de Magia, p180–235)", cantrips)
	}
}
