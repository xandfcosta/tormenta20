package api

// O aviso de "esta sessão mudou" — o passo (b) do piloto Datastar (ALE-219).
//
// Isto NÃO é CQRS, e a diferença importa. As treze mutações da fila já
// passam todas por `sessionStore.apply()` sob uma trava só: o barramento de
// comandos já existe. E `stateForRole`/`redactForPlayers` já é a projeção de
// leitura por papel. O que faltava era só o terceiro pedaço — alguém avisar
// quem lê —, e ele cabe neste arquivo.
//
// Store de escrita e de leitura separados, objetos de comando e log de eventos
// comprariam auditoria, replay e escala de leitura. Numa mesa de seis pessoas
// nenhum dos três se paga, e cada um seria um segundo modelo para manter
// honesto.
//
// A versão fica FORA do `SessionRuntimeState` de propósito. Pôr um campo lá
// mudaria o que o socket transmite e o que o banco guarda, para um número que
// só o stream usa — e o `redactForPlayers` devolve `emptyRuntimeState()` fora
// de cena, então a cópia do jogador zeraria a versão e passaria a mentir. O
// `BoardState.Version` continua sendo o precedente para versão NO FIO, onde ela
// serve para recusar movimento concorrente; aqui a pergunta é outra.

// assinar registra um ouvinte desta sessão e devolve o canal mais a baixa.
//
// O canal tem buffer 1 e a entrega é NÃO BLOQUEANTE: um leitor lento nunca pode
// segurar uma mutação, e um segundo aviso pendente não diz nada que o primeiro
// já não diga — "mudou" não se acumula.
func (st *sessionStore) assinar(sessionID int64) (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	st.mu.Lock()
	if st.ouvintes == nil {
		st.ouvintes = map[int64][]chan struct{}{}
	}
	st.ouvintes[sessionID] = append(st.ouvintes[sessionID], ch)
	st.mu.Unlock()

	return ch, func() { st.desassinar(sessionID, ch) }
}

// desassinar tira o ouvinte da lista. Sem isto, cada jogador que fecha a aba
// deixa um canal para sempre — e o `avisarLocked` passa a percorrer uma lista
// que só cresce, escrevendo em canais que ninguém lê.
func (st *sessionStore) desassinar(sessionID int64, alvo chan struct{}) {
	st.mu.Lock()
	defer st.mu.Unlock()
	restantes := st.ouvintes[sessionID][:0]
	for _, ch := range st.ouvintes[sessionID] {
		if ch != alvo {
			restantes = append(restantes, ch)
		}
	}
	if len(restantes) == 0 {
		delete(st.ouvintes, sessionID)
		return
	}
	st.ouvintes[sessionID] = restantes
}

// avisarLocked cutuca todo mundo que escuta esta sessão. Chamado de DENTRO do
// `apply`, com a trava já na mão — é o que garante que nenhuma mutação escapa
// sem aviso, porque `apply` é o funil único das treze.
func (st *sessionStore) avisarLocked(sessionID int64) {
	for _, ch := range st.ouvintes[sessionID] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
