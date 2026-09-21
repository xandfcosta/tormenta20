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

// EFEITO DE MAGIA GRAVADO COM A DURAÇÃO DA MAGIA (T20 p227).
//
// A tabela é sobre as três respostas possíveis: a magia manda, a declaração
// manda, ou não há duração nenhuma para gravar.
func TestTheSpellDurationDecidesHowLongItsEffectLasts(t *testing.T) {
	casos := []struct {
		nome      string
		daMagia   string
		nota      string
		declarada string
		quero     string
		recusa    bool
	}{
		// O QUE SE GRAVA É A GRAFIA DA FRONTEIRA, em inglês: a coluna `scope` é
		// lida por SQL que casa a palavra (`scope IN ('scene','day')`), e o
		// catálogo escreve a duração em português. Duas grafias na mesma coluna
		// seriam um efeito que nunca expira, sem erro em lugar nenhum.
		{nome: "a cena da magia manda, gravada em inglês", daMagia: "cena", quero: "scene"},
		{nome: "a declaração redundante não muda nada", daMagia: "cena", declarada: "scene", quero: "scene"},
		{nome: "a sustentada manda, e não o que o efeito dizia",
			daMagia: "sustentada", declarada: "scene", quero: "sustained"},
		{nome: "o dia da magia manda", daMagia: "dia", quero: "day"},
		// A INSTANTÂNEA não pode mandar: a consequência é outra coisa que não a
		// magia — "Curar Ferimentos age instantaneamente, mas os ferimentos
		// continuam curados" (p227).
		{nome: "a instantânea cede à consequência declarada",
			daMagia: "instantanea", declarada: "cena", quero: "scene"},
		{nome: "a instantânea sem consequência declarada é recusada",
			daMagia: "instantanea", recusa: true},
		// DEFINIDA é a ESPÉCIE e não a medida: sem as rodadas ou os dias não há
		// quando expirar, e o efeito precisa da declaração.
		{nome: "a definida sem quantia cede à declarada",
			daMagia: "definida", declarada: "cena", quero: "scene"},
		{nome: "a definida sem quantia e sem declaração é recusada",
			daMagia: "definida", recusa: true},
		// A DEFINIDA COM QUANTIA manda como qualquer outra: "1 turno" é a
		// medida que o catálogo escreve em prosa, e ela deixa de precisar de
		// declaração nenhuma (p227, e o Escudo da Fé na p192).
		{nome: "a definida de 1 turno manda, gravada em inglês",
			daMagia: "definida", nota: "1 turno", quero: "turn"},
		{nome: "a quantia da magia manda sobre a declaração",
			daMagia: "definida", nota: "1 turno", declarada: "scene", quero: "turn"},
		// NOTA QUE O APP NÃO SABE EXPIRAR não vira quantia: ela cai no caso
		// comum e legítimo da definida sem medida, que exige a declaração.
		{nome: "3 rodadas não é medida que o app saiba derrubar",
			daMagia: "definida", nota: "3 rodadas", declarada: "cena", quero: "scene"},
		{nome: "prosa condicional continua sendo definida sem quantia",
			daMagia: "definida", nota: "veja texto", recusa: true},
		{nome: "palavra que o livro não tem é recusada", daMagia: "eterna", recusa: true},
		{nome: "declaração que o livro não tem é recusada",
			daMagia: "instantanea", declarada: "eterna", recusa: true},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			teve, err := EffectScope(c.daMagia, c.nota, c.declarada)
			if c.recusa {
				if err == nil {
					t.Fatalf("%q + %q tinha de ser recusado, e devolveu %q", c.daMagia, c.declarada, teve)
				}
				return
			}
			if err != nil {
				t.Fatalf("%q + %q: %v", c.daMagia, c.declarada, err)
			}
			if teve != c.quero {
				t.Errorf("%q + %q gravou %q, quero %q", c.daMagia, c.declarada, teve, c.quero)
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
	bruto, ok := catalog.Resource("spells")
	if !ok {
		t.Fatal("o catálogo de magias não está embutido")
	}
	var magias map[string]struct {
		Duration     string `json:"duration"`
		DurationNote string `json:"durationNote"`
		Buff         *struct {
			DefaultScope string `json:"defaultScope"`
		} `json:"buff"`
	}
	if err := json.Unmarshal(bruto, &magias); err != nil {
		t.Fatalf("magias ilegíveis: %v", err)
	}
	comBuff, declarados := 0, 0
	for id, m := range magias {
		if m.Buff == nil {
			continue
		}
		comBuff++
		escopo, err := EffectScope(m.Duration, m.DurationNote, m.Buff.DefaultScope)
		if err != nil {
			t.Errorf("a magia %q: %v", id, err)
			continue
		}
		dura, _ := SpellDuration(m.Duration, m.DurationNote)
		if dura.tellsTheEffectWhenToEnd() && m.Buff.DefaultScope != "" {
			t.Errorf("a magia %q dura %q e o efeito dela declara %q por cima: apague o `defaultScope`, "+
				"que a duração da magia já responde (grava %q)", id, m.Duration, m.Buff.DefaultScope, escopo)
			continue
		}
		if m.Buff.DefaultScope != "" {
			declarados++
		}
	}
	// O DENOMINADOR: sem ele, um seletor que não casa com nada e um catálogo
	// impecável têm a mesma cor no terminal.
	if comBuff < 25 {
		t.Fatalf("só %d magias com efeito de %d — a varredura está olhando o campo errado", comBuff, len(magias))
	}
	t.Logf("%d magias com efeito conferidas; %d declaram a duração do efeito porque a magia não pode", comBuff, declarados)
}

// O RÓTULO QUE A FICHA LÊ sai da duração, e ele é o que separa um efeito que
// cai no fim da cena de um que cobra PM todo turno.
func TestTheSheetNamesHowLongAnEffectLasts(t *testing.T) {
	casos := map[string]string{
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
	for escrito, quero := range casos {
		if teve := DurationLabel(escrito); teve != quero {
			t.Errorf("%q é rotulado %q, quero %q", escrito, teve, quero)
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
	doLivro := map[string]string{
		"instantanea": "instant", "cena": "scene", "sustentada": "sustained",
		"definida": "fixed", "dia": "day", "permanente": "permanent", "descarregar": "discharge",
		// A grafia da DEFINIDA de um turno, que o giro da vez casa.
		"turn": "turn",
	}
	for escrito, quero := range doLivro {
		d, err := ParseDuration(escrito)
		if err != nil {
			t.Fatalf("%q: %v", escrito, err)
		}
		teve := d.Stored()
		if teve != quero {
			t.Errorf("%q é gravado %q, quero %q", escrito, teve, quero)
		}
		// IDA E VOLTA: o que foi gravado tem de voltar a ser lido, senão a
		// segunda leitura do próprio dado cai no erro.
		volta, err := ParseDuration(teve)
		if err != nil {
			t.Errorf("o gravado %q não é relido: %v", teve, err)
		} else if volta.Stored() != teve {
			t.Errorf("%q ida e volta virou %q", teve, volta.Stored())
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
	medida, err := SpellDuration("definida", "1 turno")
	if err != nil {
		t.Fatalf("ler a definida de 1 turno: %v", err)
	}
	if medida.Kind != DurationFixed || medida.Amount != 1 || medida.Unit != UnitTurn {
		t.Errorf("a duração = %+v, e o Escudo da Fé dura 1 turno (p192)", medida)
	}

	// O QUE FICA DE FORA, e cada um por um motivo: a unidade que ninguém conta
	// (rodadas, horas), a quantia que não é número (dados), a unidade que o
	// motor não tem (semana) e a prosa. Gravar qualquer um deles seria gravar um
	// efeito ETERNO com cara de medido — sem erro em lugar nenhum.
	semMedida := []string{"3 rodadas", "1d4 rodadas", "4d12 horas", "1 semana ou até ser descarregada",
		"veja texto", "até chegar ao solo ou cena, o que ocorrer primeiro", ""}
	for _, nota := range semMedida {
		d, err := SpellDuration("definida", nota)
		if err != nil {
			t.Fatalf("a nota %q: %v", nota, err)
		}
		if d.Amount != 0 {
			t.Errorf("a nota %q virou uma medida de %d %q, e nada no app a derruba",
				nota, d.Amount, d.Unit)
		}
	}
}

// A NOTA SÓ FALA DA DEFINIDA. A cena, a sustentada e a permanente já dizem
// quando acabam, e uma nota não as sobrescreve — senão a segunda transcrição
// voltaria a mandar na primeira, que é o defeito que o `defaultScope` causou.
func TestANoteDoesNotOverrideADurationThatAlreadySaysWhenItEnds(t *testing.T) {
	for _, escrito := range []string{"cena", "sustentada", "permanente", "dia"} {
		comNota, err := SpellDuration(escrito, "1 turno")
		if err != nil {
			t.Fatalf("%q: %v", escrito, err)
		}
		semNota, _ := ParseDuration(escrito)
		if comNota != semNota {
			t.Errorf("%q com nota deu %+v e sem nota deu %+v — a nota mandou onde não devia",
				escrito, comNota, semNota)
		}
	}
}
