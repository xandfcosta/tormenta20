package sheet

import "math"

// O DESCANSO NOTURNO do T20, como CONTA e nada mais (livro p106,
// "Recuperando PV e PM").
//
// Uma noite de pelo menos oito horas devolve PV e PM conforme o NÍVEL e a
// condição de acomodação:
//
//	ruim         metade do nível     (dormir ao relento)
//	normal       o nível             (estalagem comum)
//	confortavel  o dobro do nível
//	luxuosa      o triplo do nível
//
// O EXEMPLO TRABALHADO do livro é o que prende o arredondamento: Helior, elfo
// caçador de 7º nível, recupera 7 PV e 7 PM numa estalagem, e **3** dormindo ao
// relento — não 4. Metade de 7 é 3,5, e o livro desce. Ver o teste.
//
// E o teto é do livro também: "Você nunca pode recuperar mais pontos de vida ou
// mana do que perdeu — ou seja, não pode ultrapassar seu máximo."

// nightRestFactor é o fator por condição de acomodação.
//
// Condição desconhecida cai para `normal`, e isso é escolha: o descanso de uma
// mesa inteira não pode parar porque um cliente mandou uma palavra que o
// servidor não conhece — o mestre veria "não descansou" sem saber por quê.
var nightRestFactor = map[string]float64{
	"ruim":        0.5,
	"normal":      1,
	"confortavel": 2,
	"luxuosa":     3,
}

// NightRestGain é quanto uma noite devolve, em pontos, para um dado nível.
//
//	sheet.NightRestGain(7, "normal") // 7  — o Helior na estalagem
//	sheet.NightRestGain(7, "ruim")   // 3  — e não 4: o livro desce
func NightRestGain(level int64, condition string) int64 {
	fator, conhecida := nightRestFactor[condition]
	if !conhecida {
		fator = nightRestFactor["normal"]
	}
	return int64(math.Floor(float64(level) * fator))
}

// RestedVitals são os PV e PM ATUAIS em que uma noite de descanso deixa a ficha.
type RestedVitals struct {
	HpCurrent int64
	MpCurrent int64
}

// AfterNightRest aplica o ganho e apara no máximo.
//
// Recebe os quatro números em vez da linha do banco porque esta é a CONTA, e uma
// conta que recebesse `sqlcgen.Character` só poderia ser chamada por quem tem
// banco — que é exatamente o acoplamento que tirou esta regra do domínio até
// agora.
//
//	sheet.AfterNightRest(7, "normal", sheet.RestedVitals{HpCurrent: 10, MpCurrent: 2}, 20, 5)
//	// {17, 5} — o PM bateu no teto de 5
func AfterNightRest(
	level int64, condition string, atuais RestedVitals, hpMax, mpMax int64,
) RestedVitals {
	ganho := NightRestGain(level, condition)
	return RestedVitals{
		HpCurrent: min(hpMax, atuais.HpCurrent+ganho),
		MpCurrent: min(mpMax, atuais.MpCurrent+ganho),
	}
}
