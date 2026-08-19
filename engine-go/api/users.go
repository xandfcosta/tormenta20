package api

import "net/http"

type userDTO struct {
	ID        int64   `json:"id"`
	Email     string  `json:"email"`
	Name      *string `json:"name"`
	CreatedAt string  `json:"createdAt"`
}

// handleListUsers devolve o chamador mais quem ele enxerga por MESA, e a
// visibilidade é sempre pelo eixo mestre↔jogador: o mestre vê os donos dos
// personagens das mesas dele, o jogador vê os mestres das mesas em que joga.
// Dois jogadores da mesma crônica NÃO se veem — o comentário antigo dizia
// "todo usuário com quem compartilha uma campanha", que é mais do que as duas
// queries fazem (ALE-186).
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	ids := map[int64]bool{user.ID: true}
	if players, err := s.queries.VisiblePlayerOwners(r.Context(), user.ID); err == nil {
		for _, p := range players {
			ids[p] = true
		}
	}
	if gms, err := s.queries.VisibleGmOwners(r.Context(), user.ID); err == nil {
		for _, g := range gms {
			ids[g] = true
		}
	}
	idList := make([]int64, 0, len(ids))
	for id := range ids {
		idList = append(idList, id)
	}
	rows, err := s.queries.ListUsersByIDs(r.Context(), idList)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not list users")
		return
	}
	out := make([]userDTO, 0, len(rows))
	for _, u := range rows {
		out = append(out, userDTO{ID: u.ID, Email: u.Email, Name: nullToPtr(u.Name), CreatedAt: u.Createdat})
	}
	writeJSON(w, http.StatusOK, out)
}
