package live

import "context"

// A PORTA DA FICHA (ALE-254).
//
// O regime precisa escrever PV e PM de um personagem — é a ALE-122, "o PV do
// rastreador É o PV da ficha". Mas as REGRAS dessa escrita são da ficha e não
// daqui: a ordem do dano do livro, os pools temporários, o teto de cada vital.
// Quando o `live/` foi extraído, o compilador apontou isso em três métodos de
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

	// PoolsOf devolve o poço de cada personagem pedido, para a fila refrescar os
	// máximos de quem subiu de nível no meio da sessão.
	//
	// Ela entrou na porta quando o máximo deixou de ser coluna e passou a ser
	// DERIVADO do catálogo (ALE-355): a fila lia `hpMax` direto do banco, que era
	// leitura de coluna e não de regra, e não é mais. Quem não existe mais não
	// aparece no mapa — a lista de ids vem de uma fila onde uma ficha pode ter
	// sido apagada.
	//
	// Em LOTE e não uma por uma porque ela roda a cada desenho da Mesa.
	PoolsOf(ctx context.Context, charIDs []int64) (map[int64]VitalPool, error)
}

// VitalPool é o par máximo/atual de um personagem, como o regime precisa dele.
//
// O regime tem o seu porque não pode conhecer o `sheet.Pools`: quem cumpre a
// porta traduz. São os mesmos quatro números, e a tradução é uma linha — o preço
// de o regime não importar a ficha (ver o `boundary_test.go`).
type VitalPool struct {
	HpMax     int64
	HpCurrent int64
	MpMax     int64
	MpCurrent int64
}
