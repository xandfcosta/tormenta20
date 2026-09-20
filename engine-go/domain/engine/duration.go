package engine

import "fmt"

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
func ParseDuration(escrito string) (Duration, error) {
	switch escrito {
	case "instantanea":
		return Duration{Kind: DurationInstant}, nil
	case "cena", "scene":
		return Duration{Kind: DurationScene}, nil
	case "sustentada":
		return Duration{Kind: DurationSustained}, nil
	case "definida":
		return Duration{Kind: DurationFixed}, nil
	case "dia", "day":
		return Duration{Kind: DurationFixed, Amount: 1, Unit: UnitDay}, nil
	case "permanente":
		return Duration{Kind: DurationPermanent}, nil
	case "descarregar":
		return Duration{Kind: DurationDischarge}, nil
	}
	return Duration{}, fmt.Errorf(
		"duração %q não é uma das seis do livro (p227): instantanea, cena, sustentada, definida, permanente, descarregar", escrito)
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
func ParseUsageLimit(escrito string) (UsageLimit, error) {
	switch escrito {
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
		"janela de uso %q não é uma das três que o catálogo usa: rodada, cena, dia", escrito)
}
