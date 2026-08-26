package api

import (
	"context"

	"fmt"
	"github.com/a-h/templ"
	"io/fs"
	"net/http"
	"strconv"
	"t20engine/aovivo"
	"t20engine/tabuleiro"
	"time"

	"github.com/go-chi/chi/v5"
)

// O piloto Datastar (ALE-219): a superfície "Mesa" do jogador renderizada pelo
// SERVIDOR, ao lado da SPA e no mesmo binário.
//
// Ela mora fora do `Router()` de propósito. O `Router()` é montado sob `/api`
// em produção, e isto não é API — é uma PÁGINA, e ela precisa de uma URL que o
// jogador possa abrir e favoritar.
//
// Autenticação é a MESMA e sem uma linha nova: o `requireAuth` já lê o cookie
// `t20_session` antes do Bearer (middleware.go), e o cookie ignora porta, então
// a sessão criada pela SPA vale aqui.
//
// SAÍDA DO PILOTO: apagar `api/mesa*`, a linha do `buildMux` e a entrada do
// proxy no `vite.config.ts`.
func (s *Server) PilotoRouter() http.Handler {
	r := chi.NewRouter()
	// Os estáticos são anônimos: são o bundle do Datastar e a folha de estilo,
	// e exigir sessão para eles só quebraria o cache.
	r.Handle("/static/*", http.StripPrefix("/static/", pilotoStaticHandler()))
	// A PORTA (ALE-229) é anônima por necessidade: é ela que cria a sessão. Ela
	// tem de vir ANTES dos grupos com `requireAuth` — não por ordem de casamento
	// (o chi casa por rota, não por ordem), mas porque ficar dentro do grupo a
	// tornaria inalcançável para exatamente quem precisa dela.
	s.rotasDaPorta(r)
	// O HUB (ALE-231): o menu principal, atrás de sessão como todo o resto.
	r.Group(func(r chi.Router) {
		r.Use(s.requirePagina)
		s.rotasDoHub(r)
		s.rotasDeCampanhas(r)
		s.rotasDePersonagens(r)
		s.rotasDoGrimorio(r)
		s.rotasDoMestre(r)
		// O BUSCADOR (ALE-264) fica no grupo do Hub e não no do mestre: a caixa
		// abre em QUALQUER cena, inclusive na Mesa, e a rota tem de existir onde
		// quer que o ⌃K seja apertado.
		s.rotasDoBuscador(r)
		// O VERBETE citado por um elo (ALE-264), na casca pelo mesmo motivo do
		// buscador: a caixa abre em qualquer cena.
		s.rotasDoVerbete(r)
		// O LIVRO (ALE-264) é servido para quem ENTROU e não anonimamente como
		// os estáticos: os estáticos são o bundle do Datastar, e isto é um
		// arquivo do dono da mesa. Sem `LIVRO_PDF` a rota devolve 404 — o botão
		// que a levaria também não é desenhado.
		r.Handle(rotaDoLivro, s.LivroDoPiloto())
		// O LEITOR é uma PÁGINA e o `/livro` é o arquivo. Rotas irmãs de
		// propósito: quem quiser o PDF cru (imprimir, buscar no visualizador do
		// navegador) tem o endereço de sempre.
		r.Get(rotaDoLivro+"/ler", s.handleLeitorDoLivro)
	})
	r.Group(func(r chi.Router) {
		r.Use(s.requirePagina)
		r.Get("/mesa/{campaignId}/{sessionId}", s.handleMesaPage)
		r.Get("/mesa/{campaignId}/{sessionId}/stream", s.handleMesaStream)
		r.Post("/mesa/{campaignId}/{sessionId}/iniciativa", s.handleMesaInitiative)
		s.rotasDosComandosDaMesa(r)
		s.rotasDoBestiarioDaMesa(r)
		s.rotasDoMovimento(r)
		s.rotasDaCena(r)
		s.rotasDosMarcadores(r)
		s.rotasDaCortina(r)
		s.rotasDasCondicoes(r)
		s.rotasDaSessao(r)
		s.rotasDasNotas(r)
	})
	// A SEGUNDA superfície (ALE-219): a administração. Mesmo `requireAdmin` da
	// API — a tela não decide quem pode ver, ela só deixa de oferecer o que o
	// servidor recusaria.
	r.Group(func(r chi.Router) {
		r.Use(s.requirePagina)
		r.Use(s.requireAdmin)
		r.Get("/admin", s.handleAdminPiloto)
		r.Post("/admin/usuarios/{id}/apagar", s.handleAdminPilotoApagar)
		r.Post("/admin/backup", s.handleAdminPilotoBackup)
		r.Post("/admin/usuarios/{id}/redefinir", s.handleAdminPilotoRedefinir)
		r.Post("/admin/convites", s.handleAdminPilotoConvite)
	})
	return r
}

// mesaStaticHandler serve o bundle e a folha embutidos.
func pilotoStaticHandler() http.Handler {
	sub, err := fs.Sub(pilotoFS, "piloto/static")
	if err != nil {
		panic("piloto: static embutido ausente: " + err.Error())
	}
	return comCacheVersionado(versaoDosEstaticos, "public", http.FileServer(http.FS(sub)))
}

// mesaParams lê os dois ids da URL. Erro aqui é URL digitada errada, e a
// resposta é uma frase e não um JSON: quem está do outro lado é um navegador
// mostrando uma página.
func mesaParams(w http.ResponseWriter, r *http.Request) (campaignID, sessionID int64, ok bool) {
	campaignID, err1 := strconv.ParseInt(chi.URLParam(r, "campaignId"), 10, 64)
	sessionID, err2 := strconv.ParseInt(chi.URLParam(r, "sessionId"), 10, 64)
	if err1 != nil || err2 != nil {
		http.Error(w, "campanha e sessão precisam ser números", http.StatusBadRequest)
		return 0, 0, false
	}
	return campaignID, sessionID, true
}

// handleMesaPage é a carga fria: o documento inteiro, já com a fila desenhada.
//
// Renderizar o estado JÁ na primeira resposta (em vez de mandar uma casca que
// espera o primeiro tique do SSE) é o que faz a página não piscar vazia — e é
// a mesma lição do `settledQuery` na SPA, um andar acima (ALE-96).
func (s *Server) handleMesaPage(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, ok := mesaParams(w, r)
	if !ok {
		return
	}
	view, status, err := s.loadMesaView(r.Context(), currentUser(r), campaignID, sessionID)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	// A página é um retrato de agora, e o `escrevePagina` já a manda `no-store`:
	// guardá-la serviria uma fila velha.
	s.escrevePagina(w, r, http.StatusOK, paginaPiloto{
		Titulo: fmt.Sprintf("Mesa · Sessão %d", view.SessionNum),
		// `erro` e `erroDoComando` são DOIS sinais e não um. Um só faria a
		// recusa de "Adicionar grupo" acender a frase vermelha dentro da caixa
		// "Registrar iniciativa" do mestre que também joga: a frase certa no
		// lugar errado, que é como se lê um defeito. Uma palavra por conceito
		// vale para sinal de página como vale para identificador.
		// O chão padrão é DERIVADO e não digitado: escrever 'pedra' aqui seria a
		// terceira cópia da mesma escolha (a lista, o servidor e a página), e a
		// que fica para trás quando alguém trocar o padrão é justamente esta —
		// o formulário nasceria oferecendo um chão e o servidor abrindo outro.
		Sinais: fmt.Sprintf("{d20: 10, erro: '', erroDoComando: '', erroDoMovimento: '', novolugar: '', novochao: '%s', ferramenta: '', apagando: false, marcadorescolhido: '', escolhidosdomapa: '', qualidadedodescanso: 'normal', formdecombatente: false, linhadacondicao: '', condicoesdalinha: '', rotulodalinha: '', novonome: '', novainiciativa: 10, novopv: 0, novotipo: 'npc', edicaolinha: '', edicaonome: '', edicaoiniciativa: 0, edicaopv: 0, edicaopvmax: 0, rascunhode: '', pvdoverbete: 0, inidoverbete: 10, copiasdoverbete: 1, quadrado: %d, arrastando: '', arrastoinix: 0, arrastoiniy: 0, arrastox: 0, arrastoy: 0, notas: '', notassalvas: '', notasmodo: 'duplo', notasabertas: false, notassalvando: false, erroDasNotas: ''}", tabuleiro.ChaoPadrao(), quadradoPadrao),
		Init:   fmt.Sprintf("@get('/piloto/mesa/%d/%d/stream')", campaignID, sessionID),
	}, s.corpoDaMesa(r, view, campaignID, sessionID))
}

// corpoDaMesa escolhe QUAL DAS DUAS FORMAS a página desenha (ALE-269).
//
// O jogador recebe a coluna — uma superfície que rola, com Grupo, mapa e fila
// empilhados. O mestre recebe o PALCO: trilhos nas bordas e o tabuleiro no
// centro, que é a geometria da `session-gm-view` desde a ALE-198.
//
// SÃO DUAS FORMAS E NÃO DUAS TELAS, e a distinção é o que mantém de pé o
// argumento da ALE-265 contra uma segunda cena: as REGIÕES são as mesmas, com
// os mesmos ids e os mesmos componentes, e o que muda é onde elas são
// penduradas. Duas listas de combatente para manter em dia continuaria sendo o
// defeito da ALE-122; duas ARRUMAÇÕES da mesma lista não é.
//
// O painel do bestiário só nasce para quem pode abri-lo, pela mesma trava do
// resto da fatia: não é a tela que esconde, é a página que não o tem. Mandá-lo
// para todo mundo e escondê-lo por CSS entregaria as 80 criaturas com PV e
// defesa a quem abrisse o inspetor — e esconder PV de NPC é literalmente o que o
// olho da linha faz.
func (s *Server) corpoDaMesa(r *http.Request, view mesaView, campaignID, sessionID int64) templ.Component {
	if view.Mestre == nil {
		return mesa(view)
	}
	return palcoDoMestre(view, s.bestiarioDaMesaPara(r, campaignID, sessionID))
}

// loadMesaView busca tudo o que a tela precisa e delega a DECISÃO ao
// `mesaViewOf`. Impuro aqui, puro lá.
func (s *Server) loadMesaView(ctx context.Context, user AuthUser, campaignID, sessionID int64) (mesaView, int, error) {
	sess, role, status, err := s.sessionForCaller(ctx, user, campaignID, sessionID)
	if err != nil {
		return mesaView{}, status, err
	}
	// Hidrata do banco na primeira leitura, como o `onGetState` faz — sem isto
	// um servidor recém-subido serve fila vazia para um combate em andamento.
	if _, err := s.sessions.Load(ctx, sessionID); err != nil {
		return mesaView{}, http.StatusInternalServerError, err
	}
	// `stateForRole` e não `redactForPlayers` direto: é o mesmo gargalo que o
	// socket usa (ALE-122/ALE-210), e papel desconhecido cai em jogador. O
	// piloto não ganha uma segunda decisão sobre quem vê o quê.
	st := aovivo.StateForRole(role, s.sessions.RefreshCharacterMaxes(ctx, sessionID))
	grupo, meus, eu := s.mesaRoster(ctx, user, campaignID)
	view := mesaViewOf(st, campaignID, sessionID, sess.Sessionnumber, grupo, meus, eu)
	// O CICLO da sessão chega à tela (ALE-269): sem o estado, os verbos teriam de
	// ser oferecidos todos, e "encerrar" numa sessão que nunca começou é o gesto
	// que o servidor recusa — oferecer o que será recusado é desenhar um erro.
	view.Status = sess.Status
	if sess.Title.Valid {
		view.Titulo = sess.Title.String
	}
	// O tabuleiro passa pelo MESMO gargalo por papel que a fila: o `BoardForRole`
	// é para o mapa o que o `StateForRole` é para a lista, e é ele que tira as
	// peças escondidas antes de a cena existir. A saúde vem do estado JÁ
	// REDIGIDO, então o combatente cujo PV o mestre ocultou chega sem `HpMax` e a
	// peça dele sai sem barra — a redação alcança o mapa sem uma segunda decisão.
	// O `Mover` diz de quem é a vez e de quem é a peça, e a POSSE é resolvida
	// contra o banco (o `meus` do roster) e nunca contra o cliente — é o mesmo
	// fio de volta até a pessoa que a ALE-33 fixou.
	quemOlha := tabuleiro.Mover{UserID: user.ID, Role: role}
	view.Tabuleiro = tabuleiroViewOf(
		tabuleiro.BoardForRole(role, s.boards.Get(ctx, sessionID)), st,
		saudeDaFila(st), combatenteDaVez(st), quemOlha, meus, campaignID, sessionID,
	)
	// O ACERVO é do mestre, pela mesma razão do rastreador: a mesa não escolhe
	// onde joga. A trava é a view não ter o que desenhar, e não a tela esconder.
	if role == "gm" {
		view.Tabuleiro.Acervo = acervoDaCampanha(s.boards.Places(ctx, campaignID))
	}
	// O rastreador só é MONTADO para o mestre. A trava não é a tela esconder o
	// bloco: é a view não ter o que desenhar, pelo mesmo `role` que o
	// `stateForRole` já usou para redigir o estado.
	// AS NOTAS são do mestre e chegam JÁ EM ÁRVORE (ALE-269). Elas não entram no
	// `mesaViewOf` porque não vêm do estado ao vivo: moram na linha da sessão,
	// que é o mesmo lugar do título e do ciclo.
	if role == "gm" && sess.Notes.Valid {
		view.Notas = sess.Notes.String
		view.NotasBlocos = parseNota(sess.Notes.String)
	}
	if role == "gm" {
		membros, presentes := s.membrosEPresenca(ctx, campaignID, sessionID)
		r := mestreViewOf(st, membros, presentes, true)
		// A presença é escrita nos cartões DEPOIS de o papel ser resolvido, e é
		// isso que a mantém fora da tela do jogador sem uma segunda decisão na
		// cena: quem não é mestre não chega aqui, e lá o campo continua nil.
		marcaAPresenca(view.Grupo, r.Conectados)
		view.Mestre = &r
	}
	return view, http.StatusOK, nil
}

// mesaRoster traduz o roster da campanha nas três coisas que a tela quer: os
// cartões do Grupo, o conjunto dos MEUS personagens, e qual deles registra
// iniciativa.
//
// A ponte até "quem está olhando" é o `ownerId` do roster, e não o id do
// personagem: a ficha de um membro é o SNAPSHOT da campanha (ALE-33), então o
// dono registrado é o único fio de volta até a pessoa. Mesmo caminho do
// `myCharacterIdsOf` na SPA, pelo mesmo motivo.
func (s *Server) mesaRoster(ctx context.Context, user AuthUser, campaignID int64) ([]mesaMembro, map[int64]bool, *mesaEu) {
	rows, err := s.queries.ListMembers(ctx, campaignID)
	if err != nil {
		// Roster indisponível não derruba a fila: a iniciativa é o assunto da
		// tela, e o Grupo é o cartão ao lado.
		return nil, map[int64]bool{}, nil
	}
	grupo := make([]mesaMembro, 0, len(rows))
	meus := make(map[int64]bool, len(rows))
	var eu *mesaEu
	for _, m := range rows {
		if dono, err := s.queries.GetCharacterOwner(ctx, m.Characterid); err == nil && dono == user.ID {
			meus[m.Characterid] = true
			if eu == nil {
				eu = &mesaEu{CharacterID: m.Characterid, Nome: m.Charname}
			}
		}
		// O filtro de PAPEL é o mesmo do "Adicionar grupo" no servidor
		// (ALE-212): o mestre costuma ter um PC próprio no roster, e listá-lo
		// aqui faria duas telas discordarem sobre quem é o grupo.
		if m.Role != "player" {
			continue
		}
		grupo = append(grupo, mesaMembro{
			CharacterID: m.Characterid,
			Nome:        m.Charname,
			Iniciais:    iniciais(m.Charname),
			Nivel:       m.Charlevel,
			Classes:     s.mesaClasses(ctx, m.Characterid),
			PV:          mesaBarraDe(m.Charhpcurrent, m.Charhpmax, false),
			PM:          mesaBarraDe(m.Charmpcurrent, m.Charmpmax, true),
		})
	}
	if eu != nil {
		// O bônus é do MOTOR, nunca do template: é a mesma `ComputeSheetV2` que
		// a ficha inteira usa (ALE-213).
		if bonus, err := s.initiativeBonus(ctx, eu.CharacterID); err == nil {
			eu.Bonus = bonus
		}
	}
	return grupo, meus, eu
}

// mesaClasses monta "Guerreiro 3 / Ladino 2".
func (s *Server) mesaClasses(ctx context.Context, characterID int64) string {
	classes, err := s.queries.ListClassesByCharacter(ctx, characterID)
	if err != nil || len(classes) == 0 {
		return ""
	}
	out := ""
	for i, c := range classes {
		if i > 0 {
			out += " / "
		}
		out += c.Classname + " " + strconv.FormatInt(c.Level, 10)
	}
	return out
}

// mesaTick é a cadência do stream. 200ms é a medida que a comunidade do
// Datastar pratica (o Game of Life multiplayer do Anders Murphy re-renderiza o
// <main> inteiro nessa cadência), e ela é folgada para uma mesa de RPG: o que
// muda é um turno por vez, não um quadro por vez.
//
// Só sai byte quando o HTML MUDA — ver `handleMesaStream`.
const mesaTick = 200 * time.Millisecond

// membrosEPresenca junta o que as regras de presença precisam.
//
// Roster indisponível não derruba a cena: a presença é enfeite ao lado dos
// nomes, e a fila é o assunto da tela.
func (s *Server) membrosEPresenca(ctx context.Context, campaignID, sessionID int64) ([]aovivo.MembroDaMesa, []int64) {
	rows, err := s.queries.ListMembers(ctx, campaignID)
	if err != nil {
		return nil, nil
	}
	membros := make([]aovivo.MembroDaMesa, 0, len(rows))
	for _, m := range rows {
		dono, err := s.queries.GetCharacterOwner(ctx, m.Characterid)
		if err != nil {
			continue
		}
		membros = append(membros, aovivo.MembroDaMesa{CharacterID: m.Characterid, DonoID: dono})
	}
	var presentes []int64
	for _, u := range s.presence.Roster(sessionID) {
		presentes = append(presentes, u.UserID)
	}
	return membros, presentes
}
