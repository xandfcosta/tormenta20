package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

// Super-mestre (ALE-120): the admin reaches ANY mesa, and a player still cannot.
// Through the REAL router, because a route registered outside the guarded group
// or a handler that forgets its authorization helper only shows up here.

// mesaFixture is one player's mesa, seen by three different people.
type mesaFixture struct {
	server     *Server
	admin      int64
	dono       int64
	estranho   int64
	campaignID int64
	characterD int64
}

func seedMesa(t *testing.T) mesaFixture {
	t.Helper()
	s := newTestServer(t, adminEmail)
	f := mesaFixture{server: s, admin: seedUser(t, s, adminEmail)}
	f.dono = seedUser(t, s, "dono-da-mesa@t20.local")
	f.estranho = seedUser(t, s, "estranho@t20.local")
	f.campaignID = seedCampaign(t, s, f.dono)
	f.characterD = seedCharacter(t, s, f.dono, "Herói do dono", 10, 10, 5, 5)
	return f
}

func TestAdminReachesAnotherPlayersMesa(t *testing.T) {
	f := seedMesa(t)
	campaign := "/campaigns/" + id64(f.campaignID)

	t.Run("abre a campanha como gm", func(t *testing.T) {
		rec := authed(t, f.server, f.admin, http.MethodGet, campaign, "")

		if rec.Code != http.StatusOK {
			t.Fatalf("esperado 200, veio %d (%s)", rec.Code, rec.Body.String())
		}
		if Role, _ := jsonField(t, rec, "role").(string); Role != "gm" {
			t.Errorf("Role = %q, esperado gm (%s)", Role, rec.Body.String())
		}
		// Sem isto a tela de detalhe anuncia "Mestrando" na mesa de outro — e é
		// justamente ali que se renomeia e apaga.
		if owner, _ := jsonField(t, rec, "ownerName").(string); owner != "dono-da-mesa@t20.local" {
			t.Errorf("ownerName = %q, esperado o dono (%s)", owner, rec.Body.String())
		}
	})

	t.Run("a mesa própria não vem marcada", func(t *testing.T) {
		minha := seedCampaign(t, f.server, f.admin)

		rec := authed(t, f.server, f.admin, http.MethodGet, "/campaigns/"+id64(minha), "")

		if owner := jsonField(t, rec, "ownerName"); owner != nil {
			t.Errorf("ownerName = %v, esperado ausente na própria mesa", owner)
		}
	})

	// PATCH passa pelo `loadOwnedCampaign`, o helper que seis rotas usam —
	// renomear prova as seis de uma vez.
	t.Run("edita a campanha de outro", func(t *testing.T) {
		rec := authed(t, f.server, f.admin, http.MethodPatch, campaign, `{"name":"Renomeada pelo admin"}`)

		if rec.Code != http.StatusOK {
			t.Fatalf("esperado 200, veio %d (%s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("abre a ficha de outro", func(t *testing.T) {
		rec := authed(t, f.server, f.admin, http.MethodGet, "/characters/"+id64(f.characterD), "")

		if rec.Code != http.StatusOK {
			t.Fatalf("esperado 200, veio %d (%s)", rec.Code, rec.Body.String())
		}
	})
}

// Regressão: a marca "Mesa de Fulano" é PARA O ADMIN. Um jogador também é
// não-dono da mesa em que joga, e marcá-la troca o "Jogando" dele por "Mesa de
// Mestre" — o e2e pegou isso quando a condição olhava só a posse (ALE-120).
func TestAPlayersCampaignIsNotMarkedWithTheOwner(t *testing.T) {
	f := seedMesa(t)
	pc := seedCharacter(t, f.server, f.estranho, "PC do jogador", 10, 10, 5, 5)
	seedMember(t, f.server, f.campaignID, pc, "player")

	rec := authed(t, f.server, f.estranho, http.MethodGet, "/campaigns/"+id64(f.campaignID), "")

	if rec.Code != http.StatusOK {
		t.Fatalf("o jogador tem de ler a mesa em que joga, veio %d (%s)", rec.Code, rec.Body.String())
	}
	if Role, _ := jsonField(t, rec, "role").(string); Role != "player" {
		t.Errorf("Role = %q, esperado player", Role)
	}
	if owner := jsonField(t, rec, "ownerName"); owner != nil {
		t.Errorf("ownerName = %v, esperado ausente para o jogador", owner)
	}
}

// O contraste que dá sentido ao teste acima: sem o papel, nada disso passa.
func TestAStrangerStillReachesNothing(t *testing.T) {
	f := seedMesa(t)
	campaign := "/campaigns/" + id64(f.campaignID)

	for _, tc := range []struct{ name, method, path, body string }{
		{"abrir campanha", http.MethodGet, campaign, ""},
		{"editar campanha", http.MethodPatch, campaign, `{"name":"invadida"}`},
		{"abrir ficha", http.MethodGet, "/characters/" + id64(f.characterD), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := authed(t, f.server, f.estranho, tc.method, tc.path, tc.body)
			if rec.Code != http.StatusForbidden {
				t.Errorf("esperado 403, veio %d (%s)", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestTheAdminListSeesEveryMesaAndSaysWhoseItIs(t *testing.T) {
	f := seedMesa(t)

	rec := authed(t, f.server, f.admin, http.MethodGet, "/campaigns", "")

	var list []struct {
		ID        int64   `json:"id"`
		Role      string  `json:"role"`
		OwnerName *string `json:"ownerName"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("resposta não é uma lista (%s): %v", rec.Body.String(), err)
	}
	if len(list) != 1 || list[0].ID != f.campaignID {
		t.Fatalf("o admin tinha de ver a mesa alheia, veio %s", rec.Body.String())
	}
	if list[0].OwnerName == nil || *list[0].OwnerName != "dono-da-mesa@t20.local" {
		t.Errorf("a mesa de outro tem de vir marcada com o dono, veio %s", rec.Body.String())
	}
	if list[0].Role != "gm" {
		t.Errorf("Role = %q, esperado gm — é o papel que o admin tem ao abrir", list[0].Role)
	}
}

// Um jogador continua vendo só o que é dele: a lista do admin não pode ter
// virado a lista de todo mundo.
func TestAPlayersListStaysTheirOwn(t *testing.T) {
	f := seedMesa(t)

	rec := authed(t, f.server, f.estranho, http.MethodGet, "/campaigns", "")

	var list []campaignListDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("resposta não é uma lista (%s): %v", rec.Body.String(), err)
	}
	if len(list) != 0 {
		t.Errorf("o estranho não podia ver mesa nenhuma, veio %s", rec.Body.String())
	}
}
