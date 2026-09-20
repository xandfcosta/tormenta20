package engine

import (
	"encoding/json"
	"testing"

	"t20engine/domain/catalog"
)

// OS DOIS EIXOS DO LIVRO, e o que os separa.
//
// A DURAÇÃO (p227) diz por quanto tempo o efeito vale. O LIMITE DE USO diz
// quantas vezes a habilidade pode ser acionada e quando o contador zera. Hoje
// os dois viajam na palavra "cena", em duas línguas e em dois arquivos, e é por
// isso que "dura a cena inteira e pode ser usado duas vezes por rodada" não tem
// como ser escrito.

// "Cena. A habilidade dura uma cena inteira" (p227) — o caso mais comum, e o
// único que o app já sabia ler.
func TestTheSixFormsOfTheBookAreReadFromTheCatalogSpelling(t *testing.T) {
	casos := map[string]DurationKind{
		"instantanea": DurationInstant,
		"cena":        DurationScene,
		"sustentada":  DurationSustained,
		"definida":    DurationFixed,
		"permanente":  DurationPermanent,
		"descarregar": DurationDischarge,
		// As duas grafias INGLESAS são o que a coluna guarda hoje, e continuam
		// sendo lidas até ela migrar.
		"scene": DurationScene,
	}
	for escrito, quer := range casos {
		d, err := ParseDuration(escrito)
		if err != nil {
			t.Fatalf("ler %q: %v", escrito, err)
		}
		if d.Kind != quer {
			t.Errorf("%q deu %q, queria %q", escrito, d.Kind, quer)
		}
	}
}

// "Definida. A duração pode ser medida em rodadas, horas, dias ou outra unidade
// de tempo" (p227).
//
// O `day` do catálogo NÃO é uma sétima forma: é uma DEFINIDA de um dia, e ler
// assim é o que faz "2 dias" caber sem inventar palavra nova.
func TestTheDayScopeIsAFixedDurationOfOneDay(t *testing.T) {
	d, err := ParseDuration("dia")
	if err != nil {
		t.Fatalf("ler: %v", err)
	}
	if d.Kind != DurationFixed || d.Amount != 1 || d.Unit != UnitDay {
		t.Errorf("a duração = %+v, e `day` é uma definida de 1 dia (p227)", d)
	}
}

// PALAVRA DESCONHECIDA RECUSA, e não vira "sem duração".
//
// É a diferença entre um catálogo com erro de digitação que aparece e um efeito
// que nunca expira porque ninguém soube ler a palavra dele.
func TestAnUnknownDurationRefusesInsteadOfMeaningNothing(t *testing.T) {
	if _, err := ParseDuration("ceninha"); err == nil {
		t.Error("uma duração que o livro não tem precisa RECUSAR")
	}
	if _, err := ParseDuration(""); err == nil {
		t.Error("duração vazia também recusa: ausência não é `permanente`")
	}
}

// AS TRÊS JANELAS que o catálogo usa, e o `rodada` é uma delas — 23 ativações.
func TestTheUsageWindowsAreTheThreeTheCatalogUses(t *testing.T) {
	casos := map[string]UsageWindow{"cena": WindowScene, "rodada": WindowRound, "dia": WindowDay}
	for escrito, quer := range casos {
		limite, err := ParseUsageLimit(escrito)
		if err != nil {
			t.Fatalf("ler %q: %v", escrito, err)
		}
		if limite.Window != quer {
			t.Errorf("%q deu janela %q, queria %q", escrito, limite.Window, quer)
		}
		if limite.Times != 1 {
			t.Errorf("%q deu %d vezes, e a forma do catálogo é UMA por janela", escrito, limite.Times)
		}
	}
}

// SEM LIMITE é um valor legítimo, e ele tem de ser distinguível de "não sei
// ler": 338 das 411 ativações não têm limite nenhum.
func TestNoLimitIsAValueAndNotAFailure(t *testing.T) {
	limite, err := ParseUsageLimit("")
	if err != nil {
		t.Fatalf("ler: %v", err)
	}
	if limite.Limited() {
		t.Error("ativação sem limite não é limitada")
	}
	if _, err := ParseUsageLimit("semana"); err == nil {
		t.Error("uma janela que o catálogo não usa precisa RECUSAR")
	}
}

// O QUE A FICHA COBRA é decisão registrada e não esquecimento: "1/cena" e
// "1/dia" têm contador no banco; "1/rodada" sai como crachá, porque a MESA
// conta rodadas e a ficha não.
//
// O tipo carrega a janela; o predicado carrega a decisão. Sem os dois
// separados, cobrar rodada um dia exigiria reencontrar a razão.
func TestTheRoundWindowIsNotChargedBySheet(t *testing.T) {
	porRodada, _ := ParseUsageLimit("rodada")
	if !porRodada.Limited() {
		t.Error("o limite por rodada EXISTE no livro")
	}
	if porRodada.ChargedBySheet() {
		t.Error("a ficha não conta rodadas: o limite por rodada é crachá")
	}
	for _, escrito := range []string{"cena", "dia"} {
		limite, _ := ParseUsageLimit(escrito)
		if !limite.ChargedBySheet() {
			t.Errorf("%q é cobrado: ele tem contador no banco", escrito)
		}
	}
}

// TODA PALAVRA QUE O CATÁLOGO ESCREVE NOS DOIS EIXOS É UMA QUE O LIVRO TEM.
//
// É validação de dado TRANSCRITO, e ela mora aqui porque o vocabulário é daqui:
// o `catalog` não pode importar o motor (a direção já existe ao contrário), e
// uma cópia da lista lá viraria duas listas para divergir.
//
// O DENOMINADOR vem junto: uma varredura que não abriu nenhuma entrada e um
// catálogo em conformidade se parecem no terminal.
func TestEveryTimeWordInTheCatalogIsOneTheBookHas(t *testing.T) {
	bruto, ok := catalog.Resource("activations")
	if !ok {
		t.Fatal("o catálogo de ativações não está embutido")
	}
	var ativacoes []struct {
		ID   string          `json:"id"`
		Uses json.RawMessage `json:"uses"`
	}
	if err := json.Unmarshal(bruto, &ativacoes); err != nil {
		t.Fatalf("ativações ilegíveis: %v", err)
	}

	janelas := 0
	for _, a := range ativacoes {
		var escrito string
		if json.Unmarshal(a.Uses, &escrito) != nil || escrito == "" {
			continue // `null` é sem limite, e sem limite é um valor
		}
		janelas++
		if _, err := ParseUsageLimit(escrito); err != nil {
			t.Errorf("a ativação %q usa a janela %q: %v", a.ID, escrito, err)
		}
	}
	if janelas < 50 {
		t.Fatalf("só %d janelas lidas de %d ativações — a varredura está olhando o campo errado",
			janelas, len(ativacoes))
	}

	// A DURAÇÃO vem das magias, no `buff.defaultScope`.
	bruto, ok = catalog.Resource("spells")
	if !ok {
		t.Fatal("o catálogo de magias não está embutido")
	}
	var magias map[string]struct {
		ID       string `json:"id"`
		Duration string `json:"duration"`
	}
	if err := json.Unmarshal(bruto, &magias); err != nil {
		t.Fatalf("magias ilegíveis: %v", err)
	}
	duracoes := 0
	for _, m := range magias {
		if m.Duration == "" {
			continue
		}
		duracoes++
		if _, err := ParseDuration(m.Duration); err != nil {
			t.Errorf("a magia %q dura %q: %v", m.ID, m.Duration, err)
		}
	}
	// O PISO é sobre as 198 magias: o campo `duration` está em todas, e uma
	// varredura que lesse o campo errado devolveria zero em silêncio.
	if duracoes < 150 {
		t.Fatalf("só %d durações lidas de %d magias — a varredura está olhando o campo errado",
			duracoes, len(magias))
	}
	t.Logf("%d janelas de uso e %d durações conferidas contra o vocabulário do livro", janelas, duracoes)
}
