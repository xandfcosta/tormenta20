package ui

import (
	"strconv"

	"t20engine/domain/engine"
)

// A ESCADA DO PV: a COR diz "quão mal", e não só a largura.
//
// Ela mora no kit porque é lida por QUATRO superfícies que não se importam entre
// si — a Mesa, o tabuleiro, a ficha e a lista de heróis — e porque é regra de
// APRESENTAÇÃO: quanto de vida vira qual tinta da casa. O `web/ui` é o único
// pacote que as quatro já importam.
//
// Privada de um pacote, uma regra não diverge — ela simplesmente NÃO ALCANÇA, e
// quem precisa dela escreve outra coisa: enquanto a escada foi privada do
// `web/table`, a ficha e a lista de heróis escreviam `--hp-full` fixo, e o mesmo
// herói a 17,5% de PV saía verde numa tela e vermelho na outra.

// hpCritical e hpHurt são os limiares, em porcento.
//
// Escritos como constantes porque são a REGRA e não números soltos: quem mudar
// um deles está mudando o que a mesa chama de "ferido", e isso se faz em um
// lugar só.
const (
	hpCritical = 25
	hpHurt     = 50
)

// VitalPercent é a largura da barra, presa em 0..100.
//
// Máximo ausente ou zero devolve ZERO em vez de dividir: quem não tem mana não
// tem barra cheia nem vazia — ele não tem barra, e quem chama decide não
// desenhar.
//
// Exemplo:
//
//	ui.VitalPercent(10, 57) // → 17
func VitalPercent(current, max int64) int {
	if max <= 0 {
		return 0
	}
	pct := int(current * 100 / max)
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

// HpFillTone é a classe que PREENCHE a barra de PV.
//
// Exemplo:
//
//	class={ "h-full", ui.HpFillTone(v.Porcento) } // → "h-full bg-hp-critical"
func HpFillTone(pct int) string {
	// AS CLASSES ESCRITAS INTEIRAS, e não montadas do degrau: o guarda das
	// tintas lê o literal, e `"bg-hp-" + tom` seria uma tinta "hp" que não
	// existe.
	return map[string]string{
		"full": "bg-hp-full", "hurt": "bg-hp-hurt", "critical": "bg-hp-critical",
	}[HpTone(pct)]
}

// HpTone é o DEGRAU da escada — `full`, `hurt` ou `critical` —, para o
// Tempestade, que pinta pelo atributo `data-tone` e não por classe.
//
//	HpTone(40) // "hurt"
func HpTone(pct int) string {
	switch {
	case pct <= hpCritical:
		return "critical"
	case pct <= hpHurt:
		return "hurt"
	}
	return "full"
}

// ── abaixo de 1 PV (T20 p236, ALE-366) ─────────────────────────────────────

// DownedWord é a palavra do estado de quem está a 0 PV ou menos — "morrendo",
// "estável" ou "morto" —, e vazio para quem está de pé.
//
// QUAL estado é decisão do motor (`engine.VitalStateOf`); aqui só se escolhe a
// palavra, como a escada escolhe a cor. Ela é só para FICHA: o livro não diz
// que o capanga do mestre sangra do mesmo jeito, e a fila não guarda condição
// dele — uma palavra ali seria inventada.
//
// Exemplo:
//
//	ui.DownedWord(-43, 87, false) // → "morto"
func DownedWord(hp, hpMax int64, bleeding bool) string {
	switch engine.VitalStateOf(hp, hpMax, bleeding) {
	case engine.VitalDying:
		return "morrendo"
	case engine.VitalStable:
		return "estável"
	case engine.VitalDead:
		return "morto"
	}
	return ""
}

// HitPoints escreve o PV com o SINAL DE MENOS tipográfico (U+2212), o mesmo
// dos botões de passo ao lado: com o hífen, "−5" e "-43" ficavam lado a lado
// com dois traços diferentes para a mesma ideia.
//
// Exemplo:
//
//	ui.HitPoints(-43) // → "−43"
func HitPoints(n int64) string {
	if n < 0 {
		return "−" + strconv.FormatInt(-n, 10)
	}
	return strconv.FormatInt(n, 10)
}
