package engine

import (
	"fmt"
	"strings"
)

// OS DOIS EIXOS DO TEMPO, que o livro nomeia separadamente e o app colapsou
// numa palavra (ALE-365).
//
// Eles moram no MOTOR e não no catálogo tipado, apesar de descreverem dado de
// catálogo: `book` importa `engine`, então um tipo no `book` seria inalcançável
// para o `collect.go` — que é justamente um dos dois leitores. E a direção está
// certa: isto é vocabulário de REGRA do livro (p227), e o motor é onde regra
// pura mora.
//
// A DURAÇÃO (p227) responde "por quanto tempo este efeito vale". O LIMITE DE
// USO responde "quantas vezes esta habilidade pode ser acionada, e quando o
// contador zera". São perguntas diferentes sobre coisas diferentes, e as duas
// vinham escritas como `"cena"` — a primeira em inglês, no `defaultScope` das
// magias; a segunda em português, no `uses` das ativações. Com uma palavra só
// não há como dizer "dura a cena inteira e pode ser usado duas vezes por
// rodada", que o livro tem.

// ── DURAÇÃO (p227) ──────────────────────────────────────────────────────────

// DurationKind é uma das SEIS formas que o livro imprime.
//
// O valor sai em inglês porque ele é FRONTEIRA — ele viaja na coluna
// `active_effects.scope` e no JSON do catálogo.
type DurationKind string

const (
	// DurationInstant — "o efeito da habilidade termina assim que ela é usada,
	// mas suas consequências podem durar mais tempo" (p227). Curar Ferimentos
	// age e some; os ferimentos continuam curados.
	DurationInstant DurationKind = "instant"
	// DurationScene — dura a cena inteira, encerrando quando ela acaba.
	DurationScene DurationKind = "scene"
	// DurationSustained — "o personagem deve gastar 1 PM como uma ação livre no
	// início de cada turno seu para manter o efeito ativo" (p227). Várias
	// habilidades sustentadas convivem; **uma só magia**.
	DurationSustained DurationKind = "sustained"
	// DurationFixed — medida em rodadas, horas, dias ou outra unidade.
	DurationFixed DurationKind = "fixed"
	// DurationPermanent — fica até ser encerrada de outra forma.
	DurationPermanent DurationKind = "permanent"
	// DurationDischarge — dorme até um gatilho; descarrega quando ele acontece,
	// ou se encerra sem efeito quando a duração transcorre.
	DurationDischarge DurationKind = "discharge"
)

// TimeUnit é a unidade de uma duração DEFINIDA.
type TimeUnit string

const (
	UnitRound TimeUnit = "round"
	UnitHour  TimeUnit = "hour"
	UnitDay   TimeUnit = "day"
	// UnitTurn é a vez de um personagem — "cada jogador tem o seu turno, a sua
	// vez de realizar ações" (p233). O livro não a lista entre as unidades da
	// definida; ela é a "outra unidade de tempo" que a p227 deixa aberta, e o
	// Escudo da Fé a escreve (p192).
	UnitTurn TimeUnit = "turn"
)

// Duration é quanto tempo um efeito vale.
type Duration struct {
	Kind DurationKind
	// Amount e Unit só existem na DEFINIDA, e só quando a unidade é CONHECIDA —
	// o `dia` do catálogo. A definida sem unidade é o caso comum: a prosa dela
	// ("até chegar ao solo ou cena, o que ocorrer primeiro") mora no
	// `durationNote` do catálogo, porque a duração definida do livro muitas
	// vezes é condicional e não um número.
	Amount int
	Unit   TimeUnit
}

// ParseDuration lê a palavra que o catálogo escreve.
//
// AS SEIS FORMAS JÁ ESTÃO TRANSCRITAS, em português, no campo `duration` das
// 198 magias — `cena` 69, `instantanea` 50, `sustentada` 32, `definida` 26,
// `dia` 19, `permanente` 2 — e nada as lia. O app usava, em vez disso, um
// `buff.defaultScope` com duas palavras em inglês, presente em 31 delas
// (ALE-365).
//
// As duas grafias inglesas continuam sendo aceitas porque são o que a COLUNA
// `active_effects.scope` guarda hoje; quando ela migrar, elas saem daqui.
//
// O `dia` não é uma sétima forma: é uma DEFINIDA de um dia. O livro não lista
// "dia" entre as durações — ele diz que a definida "pode ser medida em rodadas,
// horas, dias ou outra unidade". A `definida` sem unidade é o caso comum e
// legítimo: a prosa dela mora no `durationNote` do catálogo ("até chegar ao
// solo ou cena, o que ocorrer primeiro"), porque a duração definida do livro
// muitas vezes é CONDICIONAL e não um número.
//
// PALAVRA DESCONHECIDA RECUSA, inclusive a vazia. É a diferença entre um erro
// de digitação que aparece e um efeito que nunca expira porque ninguém soube
// ler a palavra dele — e o segundo não deixa rastro nenhum.
func ParseDuration(written string) (Duration, error) {
	switch written {
	case "instantanea":
		return Duration{Kind: DurationInstant}, nil
	case "instant":
		return Duration{Kind: DurationInstant}, nil
	case "sustained":
		return Duration{Kind: DurationSustained}, nil
	case "fixed":
		return Duration{Kind: DurationFixed}, nil
	case "permanent":
		return Duration{Kind: DurationPermanent}, nil
	case "discharge":
		return Duration{Kind: DurationDischarge}, nil
	case "cena", "scene":
		return Duration{Kind: DurationScene}, nil
	case "sustentada":
		return Duration{Kind: DurationSustained}, nil
	case "definida":
		return Duration{Kind: DurationFixed}, nil
	case "dia", "day":
		return Duration{Kind: DurationFixed, Amount: 1, Unit: UnitDay}, nil
	case "turn":
		return Duration{Kind: DurationFixed, Amount: 1, Unit: UnitTurn}, nil
	case "permanente":
		return Duration{Kind: DurationPermanent}, nil
	case "descarregar":
		return Duration{Kind: DurationDischarge}, nil
	}
	return Duration{}, fmt.Errorf(
		"duração %q não é uma das seis do livro (p227): instantanea, cena, sustentada, definida, permanente, descarregar", written)
}

// storableNotes são as MEDIDAS que o catálogo escreve em prosa e que o app sabe
// EXPIRAR, e a lista é de PERMITIDOS: nota que ela não conhece continua sendo
// definida SEM quantia, que é o caso comum e legítimo.
//
// As 26 definidas escrevem dezesseis notas diferentes, e quase todas são
// condicionais ("veja texto", "até chegar ao solo ou cena, o que ocorrer
// primeiro"). Das que têm número, só duas unidades têm QUEM AS DERRUBE: o dia
// cai no descanso e o turno cai no giro da vez. Medir "3 rodadas" sem ninguém
// contar rodadas gravaria um efeito eterno com cara de medido — que é
// exatamente o defeito que o `defaultScope` produziu uma vez.
//
// Ela não é um analisador de prosa de propósito. A forma "N unidade" convidaria
// a aceitar "1 semana" e "1d4 rodadas", e as duas cairiam do lado errado: a
// primeira por unidade que o motor não tem, a segunda por quantia que não é
// número. Uma tabela de duas entradas tem denominador à vista.
var storableNotes = map[string]Duration{
	"1 turno": {Kind: DurationFixed, Amount: 1, Unit: UnitTurn},
	"1 dia":   {Kind: DurationFixed, Amount: 1, Unit: UnitDay},
}

// TurnScope é a grafia com que uma duração de UMA VEZ vai para a coluna
// `scope`, e ela tem nome porque DOIS lugares a expiram: o giro da vez, enquanto
// o combate corre, e o fim da cena, para a reação que nunca chegou a girar.
// Duas grafias à mão divergiriam, que é o defeito que esta fatia veio consertar.
//
// @example TurnScope() // "turn"
func TurnScope() string {
	return Duration{Kind: DurationFixed, Amount: 1, Unit: UnitTurn}.Stored()
}

// SpellDuration é a duração da magia COM a medida, quando há medida.
//
// A espécie vem do campo `duration` e a medida vem do `durationNote`, que é
// prosa — "Definida. A duração pode ser medida em rodadas, horas, dias ou outra
// unidade de tempo" (p227) nomeia a espécie e deixa a medida para o verbete.
//
// A NOTA SÓ FALA DA DEFINIDA SEM QUANTIA. Deixá-la mandar sobre uma cena ou uma
// sustentada seria pôr a segunda transcrição por cima da primeira, que é o
// defeito do `defaultScope` com outro nome.
//
// @example SpellDuration("definida", "1 turno") // {Fixed, 1, turn}
func SpellDuration(written, note string) (Duration, error) {
	duration, err := ParseDuration(written)
	if err != nil {
		return Duration{}, err
	}
	if duration.Kind != DurationFixed || duration.Amount != 0 {
		return duration, nil
	}
	if measure, found := storableNotes[strings.ToLower(strings.TrimSpace(note))]; found {
		return measure, nil
	}
	return duration, nil
}

// EffectScope diz com que duração o efeito de uma magia é GRAVADO na ficha.
//
// A DURAÇÃO DA MAGIA MANDA. O catálogo escrevia a mesma coisa duas vezes — a
// duração na magia e um `defaultScope` no efeito dela, em duas línguas — e oito
// divergiam: Velocidade dizia "sustentada" e deixava na ficha um efeito de
// cena, que nunca cobra PM e nunca cai (ALE-365).
//
// `declarada` só é consultada quando a magia NÃO PODE mandar, e os dois casos
// são do livro:
//
//   - INSTANTÂNEA — a consequência não é a magia. "Curar Ferimentos age
//     instantaneamente, mas os ferimentos continuam curados" (p227), e quanto o
//     ferimento fica curado a duração da magia não diz.
//   - DEFINIDA SEM QUANTIA — "definida" é a ESPÉCIE e não a medida ("pode ser
//     medida em rodadas, horas, dias", p227). Sem a medida não há quando
//     expirar, e gravar "definida" seria gravar um efeito eterno.
//
// Uma declaração REDUNDANTE não é erro aqui e sim dado sujo: quem a recusa é o
// `TestEveryBuffLastsAsLongAsItsSpell`, na varredura. Fazer a conjuração falhar
// por isso trocaria um efeito com duração errada por um efeito nenhum.
//
// @example EffectScope("sustentada", "", "scene") // "sustained", nil
func EffectScope(spellDuration, note, declared string) (string, error) {
	duration, err := SpellDuration(spellDuration, note)
	if err != nil {
		return "", fmt.Errorf("a magia dura %q: %w", spellDuration, err)
	}
	if !duration.tellsTheEffectWhenToEnd() {
		if declared == "" {
			return "", fmt.Errorf(
				"a magia dura %q, que não diz quando o EFEITO acaba: o efeito tem de declarar a duração dele (p227)", spellDuration)
		}
		effectDuration, err := ParseDuration(declared)
		if err != nil {
			return "", fmt.Errorf("o efeito declara durar %q: %w", declared, err)
		}
		return effectDuration.Stored(), nil
	}
	return duration.Stored(), nil
}

// tellsTheEffectWhenToEnd diz se a duração da magia serve de duração do efeito.
func (d Duration) tellsTheEffectWhenToEnd() bool {
	if d.Kind == DurationInstant {
		return false
	}
	return !(d.Kind == DurationFixed && d.Amount == 0)
}

// Stored é a ÚNICA grafia com que uma duração vai para a coluna `scope`.
//
// Em inglês porque a coluna é FRONTEIRA (CLAUDE.md, "Idioma"), e uma só porque
// o SQL que expira efeito casa a palavra: `scope IN ('scene','day')`. O
// catálogo escreve em português e a tela lê em português — quem converte é
// esta função e o `DurationLabel`, um em cada ponta.
//
// @example ParseDuration("sustentada").Stored() // "sustained"
func (d Duration) Stored() string {
	if d.Kind == DurationFixed && d.Amount == 1 {
		switch d.Unit {
		case UnitDay:
			return "day"
		case UnitTurn:
			return "turn"
		}
	}
	return string(d.Kind)
}

// DurationLabel é a palavra que a MESA lê ao lado do efeito na ficha.
//
// Palavra que o livro não tem cai em "cena" de propósito: este é o caminho do
// DESENHO, e um efeito sem rótulo some da lista de quem o carrega. Quem recusa
// a palavra é a validação do catálogo, no despejo.
//
// @example DurationLabel("sustentada") // "sustentada"
func DurationLabel(written string) string {
	duration, err := ParseDuration(written)
	if err != nil {
		return "cena"
	}
	switch duration.Kind {
	case DurationSustained:
		return "sustentada"
	case DurationPermanent:
		return "permanente"
	case DurationDischarge:
		return "até descarregar"
	case DurationFixed:
		switch duration.Unit {
		case UnitDay:
			return "dia"
		case UnitTurn:
			return "1 turno"
		}
	}
	return "cena"
}

// ── LIMITE DE USO ───────────────────────────────────────────────────────────

// UsageWindow é quando o contador de usos zera.
type UsageWindow string

const (
	WindowNone  UsageWindow = ""
	WindowRound UsageWindow = "round"
	WindowScene UsageWindow = "scene"
	WindowDay   UsageWindow = "day"
)

// UsageLimit é "N vezes por janela".
//
// São DOIS campos e não uma string porque o livro escreve "duas vezes por
// rodada", e um `"rodada"` sozinho só sabe dizer uma.
type UsageLimit struct {
	Times  int
	Window UsageWindow
}

// Limited diz se há limite. Sem limite é um VALOR — 338 das 411 ativações — e
// ele precisa ser distinguível de "não soube ler".
func (l UsageLimit) Limited() bool { return l.Window != WindowNone }

// ChargedBySheet diz se a FICHA mantém o contador.
//
// "1/cena" e "1/dia" têm contador no banco; "1/rodada" sai como crachá e nada
// mais. A decisão é antiga e continua valendo — **a mesa conta rodadas, a ficha
// não** —, e ela mora aqui, separada da janela, porque são duas coisas: a
// janela é do LIVRO e o cobrar é deste app. Juntá-las faria cobrar rodada um
// dia exigir reencontrar a razão de não cobrar.
func (l UsageLimit) ChargedBySheet() bool {
	return l.Window == WindowScene || l.Window == WindowDay
}

// ParseUsageLimit lê o `uses` da ativação — em PORTUGUÊS, que é como o catálogo
// o escreve.
func ParseUsageLimit(written string) (UsageLimit, error) {
	switch written {
	case "":
		return UsageLimit{}, nil
	case "rodada":
		return UsageLimit{Times: 1, Window: WindowRound}, nil
	case "cena":
		return UsageLimit{Times: 1, Window: WindowScene}, nil
	case "dia":
		return UsageLimit{Times: 1, Window: WindowDay}, nil
	}
	return UsageLimit{}, fmt.Errorf(
		"janela de uso %q não é uma das três que o catálogo usa: rodada, cena, dia", written)
}
