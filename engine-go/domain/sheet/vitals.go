package sheet

// OS DOIS JEITOS DE OS POÇOS ACOMPANHAREM UM MÁXIMO NOVO.
//
// Quando o motor recomputa PV e PM — porque a ficha subiu de nível, ganhou um
// ponto de atributo ou acabou de nascer —, o MÁXIMO muda. O que fazer com o
// ATUAL é a decisão, e ela tem duas respostas conforme o gesto:
//
//   - **PRENDER NA FAIXA** (`ClampedToNewMax`): o atual fica onde está, só não
//     passa do teto novo. É o do nascimento — a ficha nasce cheia, e o que se
//     evita é um atual maior que o máximo.
//   - **ACOMPANHAR O DELTA** (`ShiftedByNewMax`): o atual ganha o MESMO tanto
//     que o máximo ganhou. É o de subir de nível e o do passo de atributo: um
//     herói que apanhou e sobe de nível não fica curado por isso, mas também não
//     perde o que o nível lhe deu.
//
// **A diferença entre as duas é a decisão inteira**, e é por isso que elas têm a
// mesma assinatura: quem escolhe é o gesto, e escrever "prende na faixa" no
// passo de atributo faria o ciclo `−` e `+` da cena devolver dois PV por volta.

// Vitals são os quatro números que a ficha guarda.
type Vitals struct {
	HpMax     int64 `json:"hpMax"`
	HpCurrent int64 `json:"hpCurrent"`
	MpMax     int64 `json:"mpMax"`
	MpCurrent int64 `json:"mpCurrent"`
}

// ClampedToNewMax leva o máximo novo e PRENDE o atual na faixa.
//
// Devolve também se mudou alguma coisa: quem chama grava só quando mudou, e um
// `UPDATE` que não muda nada carimbaria um `updatedAt` que a Mesa lê para
// repedir a ficha.
func ClampedToNewMax(atuais Vitals, pvMax, pmMax int) (Vitals, bool) {
	novos := Vitals{
		HpMax: int64(pvMax), HpCurrent: withinRange(atuais.HpCurrent, int64(pvMax)),
		MpMax: int64(pmMax), MpCurrent: withinRange(atuais.MpCurrent, int64(pmMax)),
	}
	return novos, novos != atuais
}

// ShiftedByNewMax leva o máximo novo e soma ao atual o MESMO delta.
func ShiftedByNewMax(atuais Vitals, pvMax, pmMax int) (Vitals, bool) {
	novos := Vitals{
		HpMax:     int64(pvMax),
		HpCurrent: withinRange(atuais.HpCurrent+(int64(pvMax)-atuais.HpMax), int64(pvMax)),
		MpMax:     int64(pmMax),
		MpCurrent: withinRange(atuais.MpCurrent+(int64(pmMax)-atuais.MpMax), int64(pmMax)),
	}
	return novos, novos != atuais
}

// withinRange prende entre zero e o teto.
func withinRange(valor, teto int64) int64 { return min(max(int64(0), valor), teto) }
