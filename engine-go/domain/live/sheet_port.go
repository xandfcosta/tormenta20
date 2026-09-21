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

// SheetTurnEffects é a porta do que o GIRO DA VEZ precisa fazer com os efeitos
// da ficha, e são três coisas que acontecem juntas (p227): cobrar as
// sustentadas de quem entra, derrubar a que não foi paga, e expirar as que
// duravam a vez que acabou.
//
// SEPARADA da `SheetVitals` porque muda por outra razão: aquela é o poço de
// PV/PM, esta é o que o turno faz com efeito. O mesmo adaptador cumpre as duas
// — quem as separa é o motivo de mudar, não o número de structs.
type SheetTurnEffects interface {
	// SustainedOf lista os efeitos SUSTENTADOS da ficha, do mais antigo para o
	// mais novo. A ordem é a de pagamento quando o mana não cobre todos.
	SustainedOf(ctx context.Context, charID int64) ([]SustainedEffect, error)
	// EndSustained derruba um efeito que não foi pago.
	EndSustained(ctx context.Context, charID int64, catalogID string) error
	// ExpireTurnEffects derruba os efeitos que duravam UMA VEZ — o "1 turno"
	// do Escudo da Fé (p192).
	//
	// Uma por personagem e não em lote porque a fila mistura ficha e NPC, e só
	// a primeira tem efeito guardado: um lote exigiria montar a lista de ids
	// duas vezes, aqui e do lado de lá.
	ExpireTurnEffects(ctx context.Context, charID int64) error
	// ConditionsOf devolve as condições ligadas na ficha. O instante de quem
	// age sai delas — o inconsciente não reage, o atordoado não age —, e é
	// nelas que o sangramento da p236 mora.
	ConditionsOf(ctx context.Context, charID int64) ([]string, error)
}

// SustainedEffect é um efeito que cobra mana por turno, como o regime precisa
// dele: o id para derrubar, e o NOME para a mesa ler qual caiu.
type SustainedEffect struct {
	CatalogID string
	Label     string
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
