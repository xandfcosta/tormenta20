package campaigns

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"t20engine/domain/sheet"
	"t20engine/infra/wire"
	"t20engine/serve/web/characters"
	"t20engine/serve/web/ui"
)

// A CARTA DE CONVITE como dado (ALE-249): entrar numa mesa.
//
// Duas portas para a mesma sala, e é isso que a struct precisa representar: um
// LINK de convite (`?token=…`, que já diz qual é a mesa) ou o NÚMERO que o
// mestre leu em voz alta. Elas são exclusivas na tela — quem chegou pelo link
// não digita número —, e por isso o `TemConvite` decide qual metade aparece.
//
// O ganho desta cena sobre a da SPA é o convite ser resolvido AQUI. Lá a tela
// monta, dispara `GET /invites/{token}` e mostra um esqueleto enquanto espera;
// aqui o nome da campanha já vem na primeira resposta. Some o estado de
// "carregando" inteiro — não porque foi escondido, porque não existe.
type joinView struct {
	Invite      string
	HasInvite   bool
	InviteValid bool
	// CampaignName só vale quando o convite vale. É o que diz à pessoa PARA
	// QUAL mesa ela foi chamada antes de comprometer um herói com ela.
	CampaignName string
	CampaignID   int64
	// TypedNumber volta preenchido numa recusa, como todo campo desta casa.
	TypedNumber string
	Heroes      []joinHero
	ChosenID    int64
	Erros       wire.FieldErrorMap
	Notice      string
}

// joinHero é uma plaqueta escolhível: o mínimo para reconhecer o herói.
type joinHero struct {
	ID       int64
	Name     string
	Subtitle string
	Initials string
	Gradient string
}

func (s Scene) LoadJoin(ctx context.Context, euID int64, token string) (joinView, error) {
	v := joinView{Invite: token, HasInvite: token != "", Erros: wire.FieldErrorMap{}}

	if v.HasInvite {
		// Convite morto NÃO é erro da página: é uma resposta, e a carta diz
		// isso em voz alta para a pessoa pedir outro link em vez de ficar
		// olhando um botão que não envia (ALE-80).
		c, err := s.deps.Queries().GetCampaignByToken(ctx, sql.NullString{String: token, Valid: true})
		switch {
		case err == nil:
			v.InviteValid, v.CampaignName, v.CampaignID = true, c.Name, c.ID
		case errors.Is(err, sql.ErrNoRows):
			// deixa `InviteValid` falso — a carta mostra a recusa
		default:
			return joinView{}, err
		}
	}

	cast, err := s.deps.CharacterList(ctx, euID)
	if err != nil {
		return joinView{}, err
	}
	for _, c := range cast {
		v.Heroes = append(v.Heroes, joinHero{
			ID: c.ID, Name: c.Name,
			Subtitle: heroSubtitle(c),
			Initials: ui.Monogram(c.Name),
			Gradient: ui.NameGradient(c.Name),
		})
	}
	return v, nil
}

// heroSubtitle é a linha de baixo da plaqueta: as classes com nível, ou o
// nível sozinho para quem ainda não tem classe. É o `classLevelLine` da SPA com
// o mesmo recuo.
func heroSubtitle(c sheet.CharacterDTO) string {
	if row := characters.ClassesOf(c); row != "" {
		return row
	}
	return "Nv " + strconv.FormatInt(c.Level, 10)
}
