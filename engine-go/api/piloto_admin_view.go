package api

import (
	"context"
	"fmt"
	"math"
	"time"
)

// A tela de ADMINISTRAÇÃO como dado — a segunda superfície do piloto (ALE-219).
//
// Ela existe para responder o que a Mesa não conseguia responder sozinha. A
// Mesa foi escolhida por ser o caso mais favorável: leitura pura, estado que já
// era do servidor, zero diálogo. Se o piloto parasse ali, o resultado
// comportaria duas leituras — "o Datastar funciona" e "eu escolhi a tela mais
// fácil" —, e elas levam a decisões opostas.
//
// O admin é o oposto em três eixos, e é por isso que é ele:
//
//  1. NÃO tem tempo real. O melhor resultado da Mesa foi reusar o
//     `redactForPlayers` e o SSE, e nada disso transfere para CRUD comum — que é
//     a maior parte do app.
//  2. Tem um DESTRUTIVO com confirmação (apagar conta), que é o risco nº 3 da
//     análise: acessibilidade de diálogo sem Kobalte. Aqui ele é testado na tela
//     de menor risco do produto, que só o dono vê.
//  3. Tem dez testes de integração hoje, então o custo de migrar deixa de ser
//     estimativa.

// adminView é a tela inteira.
type adminView struct {
	Jogadores []adminJogador
	Convites  []adminConvite
	Servidor  adminServidor
}

type adminJogador struct {
	ID    int64
	Nome  string
	Email string
	// Posses é a frase que a linha mostra ("admin · 2 mesas · 3 fichas"), e ela
	// é montada aqui e não no template: é regra de pluralização, e regra em
	// template é regra escondida onde ninguém a testa.
	Posses string
	// Custo é o que apagar levaria junto — a frase que o diálogo mostra ANTES
	// de o dono confirmar.
	Custo string
	// EhEu trava o apagar da própria conta. A trava de verdade é do servidor;
	// esta só evita oferecer um botão que responderia erro.
	EhEu bool
}

type adminConvite struct {
	Token  string
	Rotulo string
	Expira string
	URL    string
}

type adminServidor struct {
	Ambiente     string
	Banco        string
	TamanhoBanco string
	Conteudo     string
	UltimoBackup string
}

// carregaAdmin busca tudo o que a tela mostra.
//
// Quatro leituras num só handler, e é isso que a tela quer: na SPA são quatro
// queries que chegam em quatro instantes, com um esqueleto por cima enquanto
// elas voam. Aqui a página só existe depois que as quatro responderam — o
// esqueleto deixa de ser necessário porque o estado "carregando" deixa de
// existir. É a diferença mais visível entre os dois modelos, e ela vale
// registrar como GANHO: três `Show` e um `SkeletonCardGrid` somem.
func (s *Server) carregaAdmin(ctx context.Context, eu AuthUser) (adminView, error) {
	linhas, err := s.queries.ListUsersWithCounts(ctx)
	if err != nil {
		return adminView{}, err
	}
	jogadores := make([]adminJogador, 0, len(linhas))
	for _, u := range linhas {
		nome := u.Email
		if u.Name.Valid && u.Name.String != "" {
			nome = u.Name.String
		}
		jogadores = append(jogadores, adminJogador{
			ID: u.ID, Nome: nome, Email: u.Email,
			Posses: posses(s.cfg.IsAdmin(u.Email), u.Campaigns, u.Characters),
			Custo:  custoDeApagar(u.Campaigns, u.Characters),
			EhEu:   u.ID == eu.ID,
		})
	}

	convites, err := s.queries.ListOpenAccountInvites(ctx, nowISO())
	if err != nil {
		return adminView{}, err
	}
	abertos := make([]adminConvite, 0, len(convites))
	for _, c := range convites {
		abertos = append(abertos, adminConvite{
			Token:  c.Token,
			Rotulo: "Link de convite " + primeiros(c.Token, 6),
			Expira: expiraEm(c.Expiresat, time.Now()),
			URL:    "/register?convite=" + c.Token,
		})
	}

	contagem, err := s.queries.TableCounts(ctx)
	if err != nil {
		return adminView{}, err
	}
	servidor := adminServidor{
		Ambiente:     string(s.cfg.AppEnv),
		Banco:        s.cfg.DatabasePath,
		TamanhoBanco: emBytes(fileSize(s.cfg.DatabasePath)),
		Conteudo: fmt.Sprintf("%d contas · %d campanhas · %d fichas",
			contagem.Users, contagem.Campaigns, contagem.Characters),
		UltimoBackup: "Nenhum backup ainda.",
	}
	if lista := s.listBackups(); len(lista) > 0 {
		servidor.UltimoBackup = fmt.Sprintf("Último: %s · %s", lista[0].Name, emBytes(lista[0].Size))
	}

	return adminView{Jogadores: jogadores, Convites: abertos, Servidor: servidor}, nil
}

// posses espelha o `belongings` do `players-panel.tsx`.
func posses(admin bool, campanhas, fichas int64) string {
	frase := fmt.Sprintf("%s · %s",
		plural(campanhas, "campanha", "campanhas"), plural(fichas, "ficha", "fichas"))
	if admin {
		return "admin · " + frase
	}
	return frase
}

// custoDeApagar espelha o `deletionCost`: o diálogo diz o preço ANTES.
//
// As campanhas passam para quem apaga, as fichas vão junto — é a decisão que o
// `handleAdminDeleteUser` implementa, e a frase existe para que o dono a leia
// antes de confirmar, e não descubra depois.
func custoDeApagar(campanhas, fichas int64) string {
	f := plural(fichas, "ficha", "fichas")
	if campanhas == 0 {
		return fmt.Sprintf("As %s vão junto. Não há campanhas para transferir.", f)
	}
	return fmt.Sprintf("As %s vão junto, e %s para você.", f,
		plural(campanhas, "campanha passa", "campanhas passam"))
}

func plural(n int64, um, muitos string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, um)
	}
	return fmt.Sprintf("%d %s", n, muitos)
}

func primeiros(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// emBytes é a mesma escada do `bytes()` do `server-panel.tsx`.
func emBytes(n int64) string {
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

// expiraEm traduz o prazo do convite para o que o dono precisa saber: quanto
// ainda dá para esperar. Porte do `expiryLabel` do `open-invites-panel.tsx`.
//
// Ele existe porque a MIGRAÇÃO O PERDEU. A primeira versão desta tela
// renderizava o ISO cru — e as quatro asserções que o protegem na SPA teriam
// pegado isso na hora, se eu as tivesse portado ANTES de escrever o template.
// Fica como a medida mais honesta do custo de trocar de camada de teste: o que
// se perde não é o teste, é a regra que ele guardava.
//
// ARREDONDA em vez de truncar: um convite recém-criado, com sete dias menos
// alguns segundos, tem de dizer "7 dias" e não "6". E abaixo de um dia a escala
// vira HORAS, com piso em 1 — "0 horas" não diz se dá tempo de mandar a
// mensagem.
func expiraEm(iso string, agora time.Time) string {
	prazo, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	restante := prazo.Sub(agora)
	if dias := restante.Hours() / 24; dias >= 1 {
		return plural(int64(math.Round(dias)), "dia", "dias")
	}
	horas := int64(math.Round(restante.Hours()))
	if horas < 1 {
		horas = 1
	}
	return plural(horas, "hora", "horas")
}
