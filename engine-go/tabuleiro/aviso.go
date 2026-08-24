package tabuleiro

// O AVISO de "este tabuleiro mudou" (ALE-264).
//
// Espelha o `aovivo/watch.go` linha por linha, e a razão de existir foi MEDIDA
// no navegador: o stream da Mesa assina o store da SESSÃO, e o tabuleiro é outro
// store. Mover uma peça não acordava ninguém — a mudança só aparecia no
// BATIMENTO de reserva, e o cronômetro deu 1310ms para uma peça andar um
// quadrado. Numa mesa em que o mestre arrasta o ogro com seis pessoas olhando,
// um segundo e meio é a diferença entre o mapa acompanhar a mão e o mapa
// atrasar.
//
// O batimento continua existindo e continua sendo necessário: mudanças que a
// Mesa mostra nascem FORA dos dois stores — o PV do Grupo vem da ficha, alterado
// por HTTP —, e nenhum aviso as cobriria. O aviso paga a latência, o batimento
// paga a abrangência.
//
// Um segundo store de avisos e não um compartilhado porque os dois têm travas
// próprias: chamar o aviso do outro pacote de dentro da trava daqui é como se
// escreve um abraço mortal. O stream escuta os DOIS canais, que é o lugar certo
// para juntá-los — ele já sabe reler tudo.

// Assinar registra um ouvinte deste tabuleiro e devolve o canal mais a baixa.
//
// Canal com buffer 1 e entrega NÃO BLOQUEANTE, como no rastreador: um leitor
// lento nunca pode segurar uma mutação, e um segundo aviso pendente não diz nada
// que o primeiro já não diga — "mudou" não se acumula.
func (bs *BoardStore) Assinar(sessionID int64) (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	bs.Mu.Lock()
	if bs.ouvintes == nil {
		bs.ouvintes = map[int64][]chan struct{}{}
	}
	bs.ouvintes[sessionID] = append(bs.ouvintes[sessionID], ch)
	bs.Mu.Unlock()

	return ch, func() { bs.desassinar(sessionID, ch) }
}

// desassinar tira o ouvinte da lista. Sem isto, cada aba fechada deixa um canal
// para sempre e o `avisarLocked` percorre uma lista que só cresce.
func (bs *BoardStore) desassinar(sessionID int64, alvo chan struct{}) {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	restantes := bs.ouvintes[sessionID][:0]
	for _, ch := range bs.ouvintes[sessionID] {
		if ch != alvo {
			restantes = append(restantes, ch)
		}
	}
	if len(restantes) == 0 {
		delete(bs.ouvintes, sessionID)
		return
	}
	bs.ouvintes[sessionID] = restantes
}

// avisarLocked cutuca quem escuta este tabuleiro, com a trava já na mão.
//
// Chamado dos TRÊS lugares que escrevem: o `apply`, que é o funil das mutações,
// mais o `Open` e o `Close`, que não passam por ele. Abrir e fechar são
// justamente as mudanças mais visíveis do tabuleiro — a grade aparecendo e
// sumindo —, e deixá-las de fora faria o aviso cobrir o que se move e perder o
// que nasce.
func (bs *BoardStore) avisarLocked(sessionID int64) {
	for _, ch := range bs.ouvintes[sessionID] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// Ouvintes é quantos streams estão assinados neste tabuleiro. Existe para o
// teste da BAIXA poder afirmar que o `desassinar` limpou o registro — o campo é
// interno ao pacote, e sem isto o teste diria "não sobrou ouvinte" por não
// conseguir olhar, que é o pior tipo de verde.
func (bs *BoardStore) Ouvintes(sessionID int64) int {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	return len(bs.ouvintes[sessionID])
}
