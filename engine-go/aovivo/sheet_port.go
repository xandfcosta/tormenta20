package aovivo

import "context"

// A PORTA DA FICHA (ALE-254).
//
// O regime precisa escrever PV e PM de um personagem — é a ALE-122, "o PV do
// rastreador É o PV da ficha". Mas as REGRAS dessa escrita são da ficha e não
// daqui: a ordem do dano do livro, os pools temporários, o teto de cada vital.
// Quando o `aovivo/` foi extraído, o compilador apontou isso em três métodos de
// uma vez, e a resposta certa não era arrastar as regras da ficha para cá.
//
// Então o regime declara o que PRECISA e não sabe quem entrega. Hoje quem
// implementa é o `api/`; quando o contexto `ficha` nascer, ele assume sem que
// uma linha daqui mude — que é o ponto inteiro de a porta ser declarada do lado
// de quem chama.
//
// As duas operações são distintas de propósito e a distinção é do livro:
// DELTA é uma pancada ou uma cura e passa pelos pools temporários; ABSOLUTO é
// uma afirmação sobre o total e não drena pool nenhum. Uma porta só, com um
// sinal de "é dano?", convidaria a confundir as duas — que foi o defeito que a
// ALE-122 registrou.
type SheetVitals interface {
	// ApplyDelta move PV/PM por uma diferença. Devolve os dois valores que a
	// entrada da fila deve espelhar, INCLUSIVE o que não mudou: espelhar só o
	// que mudou faria o rastreador mostrar um número que a ficha não tem.
	ApplyDelta(ctx context.Context, charID int64, hpDelta, mpDelta *int64) (*int64, *int64, error)

	// ApplyAbsolute grava PV/PM totais. Não drena pool temporário.
	ApplyAbsolute(ctx context.Context, charID int64, hpCurrent, mpCurrent *int64) (*int64, *int64, error)
}
