package admin

import (
	"context"
	"fmt"
	"math"
	"t20engine/infra/platform"
	"t20engine/serve/web/ui"
	"time"
)

// A tela de ADMINISTRAÇÃO como dado: contas, convites e o estado do servidor.
//
// Ao contrário da Mesa, ela NÃO tem tempo real — quem redesenha é a resposta do
// próprio POST.

// adminView é a tela inteira.
type adminView struct {
	Players []playerRow
	Invites []inviteRow
	Machine serverInfo
}

type playerRow struct {
	ID    int64
	Name  string
	Email string
	// Posses é a frase que a linha mostra ("admin · 2 mesas · 3 fichas"), e ela
	// é montada aqui e não no template: é regra de pluralização, e regra em
	// template é regra escondida onde ninguém a testa.
	Belongings string
	// Custo é o que apagar levaria junto — a frase que o diálogo mostra ANTES
	// de o dono confirmar.
	Cost string
	// EhEu trava o apagar da própria conta. A trava de verdade é do servidor;
	// esta só evita oferecer um botão que responderia erro.
	IsMe bool
}

type inviteRow struct {
	Token   string
	Label   string
	Expires string
	URL     string
}

type serverInfo struct {
	Environment  string
	Database     string
	DatabaseSize string
	Contents     string
	LastBackup   string
}

// loadAdmin busca tudo o que a tela mostra.
//
// Quatro leituras num só handler, e é isso que a tela quer: a página só existe
// depois que as quatro responderam, então não há estado "carregando" para
// desenhar.
func (s Scene) loadAdmin(ctx context.Context, meID int64) (adminView, error) {
	linhas, err := s.deps.Queries().ListUsersWithCounts(ctx)
	if err != nil {
		return adminView{}, err
	}
	jogadores := make([]playerRow, 0, len(linhas))
	for _, u := range linhas {
		nome := u.Email
		if u.Name.Valid && u.Name.String != "" {
			nome = u.Name.String
		}
		jogadores = append(jogadores, playerRow{
			ID: u.ID, Name: nome, Email: u.Email,
			Belongings: belongings(s.deps.IsAdmin(u.Email), u.Campaigns, u.Characters),
			Cost:       deletionCost(u.Campaigns, u.Characters),
			IsMe:       u.ID == meID,
		})
	}

	convites, err := s.deps.Queries().ListOpenAccountInvites(ctx, platform.NowISO())
	if err != nil {
		return adminView{}, err
	}
	abertos := make([]inviteRow, 0, len(convites))
	for _, c := range convites {
		abertos = append(abertos, inviteRow{
			Token:   c.Token,
			Label:   "Link de convite " + firstChars(c.Token, 6),
			Expires: expiryLabel(c.Expiresat, time.Now()),
			URL:     "/register?convite=" + c.Token,
		})
	}

	contagem, err := s.deps.Queries().TableCounts(ctx)
	if err != nil {
		return adminView{}, err
	}
	servidor := serverInfo{
		Environment:  s.deps.Environment(),
		Database:     s.deps.DatabasePath(),
		DatabaseSize: inBytes(s.deps.DatabaseSize()),
		Contents: fmt.Sprintf("%d contas · %d campanhas · %d fichas",
			contagem.Users, contagem.Campaigns, contagem.Characters),
		LastBackup: "Nenhum backup ainda.",
	}
	if nome, tamanho, ok := s.deps.LastBackup(); ok {
		servidor.LastBackup = fmt.Sprintf("Último: %s · %s", nome, inBytes(tamanho))
	}

	return adminView{Players: jogadores, Invites: abertos, Machine: servidor}, nil
}

// posses é a frase de quanto a conta tem — mesas e fichas, com o plural certo.
func belongings(admin bool, campanhas, fichas int64) string {
	frase := fmt.Sprintf("%s · %s",
		ui.Plural(campanhas, "campanha", "campanhas"), ui.Plural(fichas, "ficha", "fichas"))
	if admin {
		return "admin · " + frase
	}
	return frase
}

// deletionCost é o preço que o diálogo diz ANTES de confirmar.
//
// As campanhas passam para quem apaga e as fichas vão junto — é o que o
// `DeleteAccount` do hospedeiro faz, e a frase existe para o dono ler antes e
// não descobrir depois.
func deletionCost(campanhas, fichas int64) string {
	f := ui.Plural(fichas, "ficha", "fichas")
	if campanhas == 0 {
		return fmt.Sprintf("As %s vão junto. Não há campanhas para transferir.", f)
	}
	return fmt.Sprintf("As %s vão junto, e %s para você.", f,
		ui.Plural(campanhas, "campanha passa", "campanhas passam"))
}

func firstChars(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// inBytes escreve o tamanho na escada de KB/MB/GB.
func inBytes(n int64) string {
	const k = 1024.0
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/k)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/(k*k))
	}
}

// expiryLabel traduz o prazo do convite para o que o dono precisa saber: quanto
// ainda dá para esperar.
//
// ARREDONDA em vez de truncar: um convite recém-criado, com sete dias menos
// alguns segundos, tem de dizer "7 dias" e não "6". E abaixo de um dia a escala
// vira HORAS, com piso em 1 — "0 horas" não diz se dá tempo de mandar a
// mensagem.
func expiryLabel(iso string, agora time.Time) string {
	prazo, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	restante := prazo.Sub(agora)
	if dias := restante.Hours() / 24; dias >= 1 {
		return ui.Plural(int64(math.Round(dias)), "dia", "dias")
	}
	horas := int64(math.Round(restante.Hours()))
	if horas < 1 {
		horas = 1
	}
	return ui.Plural(horas, "hora", "horas")
}
