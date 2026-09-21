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
	cases := map[string]DurationKind{
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
	for written, want := range cases {
		d, err := ParseDuration(written)
		if err != nil {
			t.Fatalf("ler %q: %v", written, err)
		}
		if d.Kind != want {
			t.Errorf("%q deu %q, queria %q", written, d.Kind, want)
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
	cases := map[string]UsageWindow{"cena": WindowScene, "rodada": WindowRound, "dia": WindowDay}
	for written, want := range cases {
		limit, err := ParseUsageLimit(written)
		if err != nil {
			t.Fatalf("ler %q: %v", written, err)
		}
		if limit.Window != want {
			t.Errorf("%q deu janela %q, queria %q", written, limit.Window, want)
		}
		if limit.Times != 1 {
			t.Errorf("%q deu %d vezes, e a forma do catálogo é UMA por janela", written, limit.Times)
		}
	}
}

// SEM LIMITE é um valor legítimo, e ele tem de ser distinguível de "não sei
// ler": 338 das 411 ativações não têm limite nenhum.
func TestNoLimitIsAValueAndNotAFailure(t *testing.T) {
	limit, err := ParseUsageLimit("")
	if err != nil {
		t.Fatalf("ler: %v", err)
	}
	if limit.Limited() {
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
	perRound, _ := ParseUsageLimit("rodada")
	if !perRound.Limited() {
		t.Error("o limite por rodada EXISTE no livro")
	}
	if perRound.ChargedBySheet() {
		t.Error("a ficha não conta rodadas: o limite por rodada é crachá")
	}
	for _, written := range []string{"cena", "dia"} {
		limit, _ := ParseUsageLimit(written)
		if !limit.ChargedBySheet() {
			t.Errorf("%q é cobrado: ele tem contador no banco", written)
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
	raw, ok := catalog.Resource("activations")
	if !ok {
		t.Fatal("o catálogo de ativações não está embutido")
	}
	var activations []struct {
		ID   string          `json:"id"`
		Uses json.RawMessage `json:"uses"`
	}
	if err := json.Unmarshal(raw, &activations); err != nil {
		t.Fatalf("ativações ilegíveis: %v", err)
	}

	windows := 0
	for _, a := range activations {
		var written string
		if json.Unmarshal(a.Uses, &written) != nil || written == "" {
			continue // `null` é sem limite, e sem limite é um valor
		}
		windows++
		if _, err := ParseUsageLimit(written); err != nil {
			t.Errorf("a ativação %q usa a janela %q: %v", a.ID, written, err)
		}
	}
	if windows < 50 {
		t.Fatalf("só %d janelas lidas de %d ativações — a varredura está olhando o campo errado",
			windows, len(activations))
	}

	// A DURAÇÃO vem das magias, no `buff.defaultScope`.
	raw, ok = catalog.Resource("spells")
	if !ok {
		t.Fatal("o catálogo de magias não está embutido")
	}
	var spells map[string]struct {
		ID       string `json:"id"`
		Duration string `json:"duration"`
	}
	if err := json.Unmarshal(raw, &spells); err != nil {
		t.Fatalf("magias ilegíveis: %v", err)
	}
	durations := 0
	for _, m := range spells {
		if m.Duration == "" {
			continue
		}
		durations++
		if _, err := ParseDuration(m.Duration); err != nil {
			t.Errorf("a magia %q dura %q: %v", m.ID, m.Duration, err)
		}
	}
	// O PISO é sobre as 198 magias: o campo `duration` está em todas, e uma
	// varredura que lesse o campo errado devolveria zero em silêncio.
	if durations < 150 {
		t.Fatalf("só %d durações lidas de %d magias — a varredura está olhando o campo errado",
			durations, len(spells))
	}
	t.Logf("%d janelas de uso e %d durações conferidas contra o vocabulário do livro", windows, durations)
}

// EFEITO DE MAGIA GRAVADO COM A DURAÇÃO DA MAGIA (T20 p227).
//
// A tabela é sobre as três respostas possíveis: a magia manda, a declaração
// manda, ou não há duração nenhuma para gravar.
func TestTheSpellDurationDecidesHowLongItsEffectLasts(t *testing.T) {
	cases := []struct {
		name          string
		spellDuration string
		note          string
		declared      string
		want          string
		refuses       bool
	}{
		// O QUE SE GRAVA É A GRAFIA DA FRONTEIRA, em inglês: a coluna `scope` é
		// lida por SQL que casa a palavra (`scope IN ('scene','day')`), e o
		// catálogo escreve a duração em português. Duas grafias na mesma coluna
		// seriam um efeito que nunca expira, sem erro em lugar nenhum.
		{name: "a cena da magia manda, gravada em inglês", spellDuration: "cena", want: "scene"},
		{name: "a declaração redundante não muda nada", spellDuration: "cena", declared: "scene", want: "scene"},
		{name: "a sustentada manda, e não o que o efeito dizia",
			spellDuration: "sustentada", declared: "scene", want: "sustained"},
		{name: "o dia da magia manda", spellDuration: "dia", want: "day"},
		// A INSTANTÂNEA não pode mandar: a consequência é outra coisa que não a
		// magia — "Curar Ferimentos age instantaneamente, mas os ferimentos
		// continuam curados" (p227).
		{name: "a instantânea cede à consequência declarada",
			spellDuration: "instantanea", declared: "cena", want: "scene"},
		{name: "a instantânea sem consequência declarada é recusada",
			spellDuration: "instantanea", refuses: true},
		// DEFINIDA é a ESPÉCIE e não a medida: sem as rodadas ou os dias não há
		// quando expirar, e o efeito precisa da declaração.
		{name: "a definida sem quantia cede à declarada",
			spellDuration: "definida", declared: "cena", want: "scene"},
		{name: "a definida sem quantia e sem declaração é recusada",
			spellDuration: "definida", refuses: true},
		// A DEFINIDA COM QUANTIA manda como qualquer outra: "1 turno" é a
		// medida que o catálogo escreve em prosa, e ela deixa de precisar de
		// declaração nenhuma (p227, e o Escudo da Fé na p192).
		{name: "a definida de 1 turno manda, gravada em inglês",
			spellDuration: "definida", note: "1 turno", want: "turn"},
		{name: "a quantia da magia manda sobre a declaração",
			spellDuration: "definida", note: "1 turno", declared: "scene", want: "turn"},
		// NOTA QUE O APP NÃO SABE EXPIRAR não vira quantia: ela cai no caso
		// comum e legítimo da definida sem medida, que exige a declaração.
		{name: "3 rodadas não é medida que o app saiba derrubar",
			spellDuration: "definida", note: "3 rodadas", declared: "cena", want: "scene"},
		{name: "prosa condicional continua sendo definida sem quantia",
			spellDuration: "definida", note: "veja texto", refuses: true},
		{name: "palavra que o livro não tem é recusada", spellDuration: "eterna", refuses: true},
		{name: "declaração que o livro não tem é recusada",
			spellDuration: "instantanea", declared: "eterna", refuses: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := EffectScope(c.spellDuration, c.note, c.declared)
			if c.refuses {
				if err == nil {
					t.Fatalf("%q + %q tinha de ser recusado, e devolveu %q", c.spellDuration, c.declared, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("%q + %q: %v", c.spellDuration, c.declared, err)
			}
			if got != c.want {
				t.Errorf("%q + %q gravou %q, quero %q", c.spellDuration, c.declared, got, c.want)
			}
		})
	}
}

// TODO EFEITO DE MAGIA DURA O QUE A MAGIA DURA, e o catálogo não guarda a
// resposta duas vezes.
//
// O `buff.defaultScope` era uma SEGUNDA transcrição da duração, em inglês, e
// oito das trinta e uma divergiam da primeira. Ele sobrevive só onde a magia
// não pode mandar — instantânea, e definida sem quantia —, e aí ele é a
// declaração que o `EffectScope` exige. Em qualquer outro lugar é uma cópia,
// e cópia envelhece.
func TestEveryBuffLastsAsLongAsItsSpell(t *testing.T) {
	raw, ok := catalog.Resource("spells")
	if !ok {
		t.Fatal("o catálogo de magias não está embutido")
	}
	var spells map[string]struct {
		Duration     string `json:"duration"`
		DurationNote string `json:"durationNote"`
		Buff         *struct {
			DefaultScope string `json:"defaultScope"`
		} `json:"buff"`
	}
	if err := json.Unmarshal(raw, &spells); err != nil {
		t.Fatalf("magias ilegíveis: %v", err)
	}
	withBuff, declaredCount := 0, 0
	for id, m := range spells {
		if m.Buff == nil {
			continue
		}
		withBuff++
		scope, err := EffectScope(m.Duration, m.DurationNote, m.Buff.DefaultScope)
		if err != nil {
			t.Errorf("a magia %q: %v", id, err)
			continue
		}
		duration, _ := SpellDuration(m.Duration, m.DurationNote)
		if duration.tellsTheEffectWhenToEnd() && m.Buff.DefaultScope != "" {
			t.Errorf("a magia %q dura %q e o efeito dela declara %q por cima: apague o `defaultScope`, "+
				"que a duração da magia já responde (grava %q)", id, m.Duration, m.Buff.DefaultScope, scope)
			continue
		}
		if m.Buff.DefaultScope != "" {
			declaredCount++
		}
	}
	// O DENOMINADOR: sem ele, um seletor que não casa com nada e um catálogo
	// impecável têm a mesma cor no terminal.
	if withBuff < 25 {
		t.Fatalf("só %d magias com efeito de %d — a varredura está olhando o campo errado", withBuff, len(spells))
	}
	t.Logf("%d magias com efeito conferidas; %d declaram a duração do efeito porque a magia não pode", withBuff, declaredCount)
}

// O RÓTULO QUE A FICHA LÊ sai da duração, e ele é o que separa um efeito que
// cai no fim da cena de um que cobra PM todo turno.
func TestTheSheetNamesHowLongAnEffectLasts(t *testing.T) {
	cases := map[string]string{
		"cena":        "cena",
		"scene":       "cena",
		"dia":         "dia",
		"day":         "dia",
		"turn":        "1 turno",
		"sustentada":  "sustentada",
		"permanente":  "permanente",
		"descarregar": "até descarregar",
		// Palavra que o livro não tem cai em "cena" no caminho do DESENHO: um
		// efeito sem rótulo SOME da lista de quem o carrega, e quem recusa a
		// palavra é a validação do catálogo, no despejo.
		"abracadabra": "cena",
	}
	for written, want := range cases {
		if got := DurationLabel(written); got != want {
			t.Errorf("%q é rotulado %q, quero %q", written, got, want)
		}
	}
}

// UMA GRAFIA POR CANAL, e a da coluna é a de fronteira.
//
// O catálogo escreve a duração em português e o SQL que expira efeito casa a
// palavra (`scope IN ('scene','day')`). Gravar a palavra do catálogo deixaria
// toda magia conjurada de hoje em diante sem nunca expirar — e sem erro em
// lugar nenhum, que é a marca desta família.
func TestEveryDurationHasOneSpellingOnTheWire(t *testing.T) {
	fromBook := map[string]string{
		"instantanea": "instant", "cena": "scene", "sustentada": "sustained",
		"definida": "fixed", "dia": "day", "permanente": "permanent", "descarregar": "discharge",
		// A grafia da DEFINIDA de um turno, que o giro da vez casa.
		"turn": "turn",
	}
	for written, want := range fromBook {
		d, err := ParseDuration(written)
		if err != nil {
			t.Fatalf("%q: %v", written, err)
		}
		got := d.Stored()
		if got != want {
			t.Errorf("%q é gravado %q, quero %q", written, got, want)
		}
		// IDA E VOLTA: o que foi gravado tem de voltar a ser lido, senão a
		// segunda leitura do próprio dado cai no erro.
		back, err := ParseDuration(got)
		if err != nil {
			t.Errorf("o gravado %q não é relido: %v", got, err)
		} else if back.Stored() != got {
			t.Errorf("%q ida e volta virou %q", got, back.Stored())
		}
	}
}

// A MEDIDA DE UMA DEFINIDA mora em PROSA, no `durationNote` (T20 p227: "pode ser
// medida em rodadas, horas, dias ou outra unidade de tempo").
//
// As 26 definidas do catálogo escrevem dezesseis notas diferentes, e a maioria é
// condicional — "veja texto", "até chegar ao solo ou cena, o que ocorrer
// primeiro". A lista lida é de PERMITIDOS: o que ela não conhece continua sendo
// definida SEM quantia, que é o caso comum e legítimo.
func TestOnlyTheMeasuresTheAppCanExpireAreReadFromTheNote(t *testing.T) {
	measure, err := SpellDuration("definida", "1 turno")
	if err != nil {
		t.Fatalf("ler a definida de 1 turno: %v", err)
	}
	if measure.Kind != DurationFixed || measure.Amount != 1 || measure.Unit != UnitTurn {
		t.Errorf("a duração = %+v, e o Escudo da Fé dura 1 turno (p192)", measure)
	}

	// O QUE FICA DE FORA, e cada um por um motivo: a unidade que ninguém conta
	// (rodadas, horas), a quantia que não é número (dados), a unidade que o
	// motor não tem (semana) e a prosa. Gravar qualquer um deles seria gravar um
	// efeito ETERNO com cara de medido — sem erro em lugar nenhum.
	unmeasured := []string{"3 rodadas", "1d4 rodadas", "4d12 horas", "1 semana ou até ser descarregada",
		"veja texto", "até chegar ao solo ou cena, o que ocorrer primeiro", ""}
	for _, note := range unmeasured {
		d, err := SpellDuration("definida", note)
		if err != nil {
			t.Fatalf("a nota %q: %v", note, err)
		}
		if d.Amount != 0 {
			t.Errorf("a nota %q virou uma medida de %d %q, e nada no app a derruba",
				note, d.Amount, d.Unit)
		}
	}
}

// A NOTA SÓ FALA DA DEFINIDA. A cena, a sustentada e a permanente já dizem
// quando acabam, e uma nota não as sobrescreve — senão a segunda transcrição
// voltaria a mandar na primeira, que é o defeito que o `defaultScope` causou.
func TestANoteDoesNotOverrideADurationThatAlreadySaysWhenItEnds(t *testing.T) {
	for _, written := range []string{"cena", "sustentada", "permanente", "dia"} {
		withNote, err := SpellDuration(written, "1 turno")
		if err != nil {
			t.Fatalf("%q: %v", written, err)
		}
		withoutNote, _ := ParseDuration(written)
		if withNote != withoutNote {
			t.Errorf("%q com nota deu %+v e sem nota deu %+v — a nota mandou onde não devia",
				written, withNote, withoutNote)
		}
	}
}
