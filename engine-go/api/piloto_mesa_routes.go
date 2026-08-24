package api

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
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
	})
	r.Group(func(r chi.Router) {
		r.Use(s.requirePagina)
		r.Get("/mesa/{campaignId}/{sessionId}", s.handleMesaPage)
		r.Get("/mesa/{campaignId}/{sessionId}/stream", s.handleMesaStream)
		r.Post("/mesa/{campaignId}/{sessionId}/iniciativa", s.handleMesaInitiative)
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
	return http.FileServer(http.FS(sub))
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
		Sinais: "{d20: 10, erro: ''}",
		Init:   fmt.Sprintf("@get('/piloto/mesa/%d/%d/stream')", campaignID, sessionID),
	}, mesa(view))
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
	if _, err := s.sessions.load(ctx, sessionID); err != nil {
		return mesaView{}, http.StatusInternalServerError, err
	}
	// `stateForRole` e não `redactForPlayers` direto: é o mesmo gargalo que o
	// socket usa (ALE-122/ALE-210), e papel desconhecido cai em jogador. O
	// piloto não ganha uma segunda decisão sobre quem vê o quê.
	st := stateForRole(role, s.sessions.refreshCharacterMaxes(ctx, sessionID))
	grupo, meus, eu := s.mesaRoster(ctx, user, campaignID)
	return mesaViewOf(st, campaignID, sessionID, sess.Sessionnumber, grupo, meus, eu), http.StatusOK, nil
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
			Nome:    m.Charname,
			Nivel:   m.Charlevel,
			Classes: s.mesaClasses(ctx, m.Characterid),
			PV:      mesaBarraDe(m.Charhpcurrent, m.Charhpmax, false),
			PM:      mesaBarraDe(m.Charmpcurrent, m.Charmpmax, true),
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
