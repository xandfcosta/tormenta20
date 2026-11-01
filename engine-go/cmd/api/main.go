// O comando api é o processo INTEIRO: ele abre e migra o SQLite e serve as
// cenas em templ, a API JSON sob `/api/` e o fluxo ao vivo por SSE — tudo na
// mesma porta, em todo ambiente.
//
// O ambiente vem do `.env.<APP_ENV>` ao lado do pacote: o `air` sobe como
// desenvolvimento, o `pnpm start` como produção.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/infra/config"
	"t20engine/infra/db"
	"t20engine/serve/api"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	// Fatal e não aviso: subir em produção com chave de assinatura forjável é
	// pior que não subir, e aviso rola para fora da tela.
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config: %v", err)
	}

	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer func() { _ = database.Close() }()

	srv := api.NewServer(cfg, database, primeCatalogs())
	// UM roteador: as cenas, os estáticos, as fontes, a saúde e a API em `/api`
	// saem todos do `WebRouter`, já comprimidos. O `cmd` não monta rota nenhuma —
	// ele abre o banco, prima os catálogos e escuta.
	mux := srv.WebRouter()

	// Um sinal encerra a mesa com ordem, em vez de no meio de uma gravação: sem
	// isto, um Ctrl-C durante um `VACUUM INTO` ou um persist do rastreador morre
	// no meio, e o `defer database.Close()` acima NUNCA roda — o processo morre
	// por sinal antes de qualquer defer.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go srv.ScheduleBackups(ctx)

	announce(cfg) // por último, para o endereço ser a última linha da tela
	if err := serve(ctx, cfg, mux, srv.CloseLiveStreams); err != nil {
		log.Fatalf("listen: %v", err)
	}
	// Aqui morava um `srv.WaitForBackground()`, que esperava a gravação do
	// tabuleiro em goroutine antes do `defer database.Close()`. Ela deixou de
	// rodar em goroutine na ALE-375 — hoje o `Shutdown` do `net/http`, que espera
	// as REQUISIÇÕES, já espera tudo que escreve.
}

// httpServerFor monta o servidor com os timeouts da casa. Separado da `serve`
// para os testes poderem afirmar as escolhas — inclusive a AUSÊNCIA do
// `WriteTimeout`, que é a mais fácil de alguém "consertar" sem saber.
func httpServerFor(cfg config.Config, mux http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

// serve sobe o HTTP e espera o sinal para desligar com ordem.
//
// Os timeouts são escolhidos, não copiados de um exemplo:
//
//   - `ReadHeaderTimeout` existe porque sem ele uma conexão que abre e nunca
//     manda o cabeçalho segura uma goroutine para sempre (slowloris);
//   - `IdleTimeout` recolhe conexões ociosas do keep-alive;
//   - `WriteTimeout` fica de FORA de propósito. Ele mataria o fluxo SSE, que é
//     conexão longa por natureza, e o download do PDF do livro numa rede ruim.
//     É o timeout que parece obrigatório e é justamente o errado aqui.
func serve(ctx context.Context, cfg config.Config, mux http.Handler, closeLiveStreams func()) error {
	server := httpServerFor(cfg, mux)
	server.RegisterOnShutdown(closeLiveStreams)
	failed := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			failed <- err
		}
	}()

	select {
	case err := <-failed:
		return err
	case <-ctx.Done():
	}

	log.Print("encerrando: esperando as requisições em curso")
	// A janela existe para a gravação em curso terminar; passado o prazo, o
	// desligamento continua — travar o encerramento seria trocar um problema
	// por outro.
	deadline, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(deadline); err != nil {
		log.Printf("encerramento forçado: %v", err)
	}
	return nil
}

// primeCatalogs carrega os catálogos de regra, e DERRUBA o processo sem eles.
//
// # Ele já foi melhor esforço, e a promessa que o sustentava morreu
//
// O comentário aqui dizia: "entrar, ler e mexer em vitais funciona sem eles;
// criar e equipar, não". Era verdade enquanto o PV máximo fosse uma COLUNA —
// sem catálogo a ficha abria com os números gravados, e só as validações de
// mutação desligavam.
//
// O PV máximo passou a ser DERIVADO (ALE-355). Sem catálogo primado não há poço
// para derivar, e as três saídas eram: servir zero PV para a mesa inteira,
// cair na coluna velha (que é a segunda verdade que a mudança existe para
// apagar), ou não subir. Decisão do dono: não subir.
//
// É a mesma classe de falha do `cfg.Validate()` acima, e pela mesma razão: um
// processo que sobe servindo ficha errada em silêncio é pior que um processo
// que não sobe.
//
// O catálogo vem embutido no binário desde a ALE-494, então o arquivo ausente
// deixou de ser um caso; o que ainda derruba é um catálogo que não se indexa.
func primeCatalogs() *engine.Catalogs {
	catalogs, err := book.EngineCatalogs()
	if err != nil {
		log.Fatalf("catalogs: prime failed: %v — o PV máximo é derivado do "+
			"catálogo, e sem ele toda ficha sairia errada em silêncio", err)
	}
	log.Printf("catalogs primed from the embedded book")
	return catalogs
}

func announce(cfg config.Config) {
	log.Printf("t20 %s server listening on :%s (http, db=%s)", cfg.AppEnv, cfg.Port, cfg.DatabasePath)
	for _, url := range lanURLs(cfg) {
		log.Printf("  players can open %s", url)
	}
}

// lanURLs lista os IPv4 não-loopback desta máquina. O servidor já escuta em
// todas as interfaces; o que falta é serem descobríveis. São os endereços que a
// mesa abre, em HTTP: não há nada na frente do processo (ALE-531).
func lanURLs(cfg config.Config) []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Printf("interfaces: %v — LAN address unknown", err)
		return nil
	}
	var urls []string
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
			continue
		}
		urls = append(urls, fmt.Sprintf("%s://%s:%s", "http", ipNet.IP, cfg.Port))
	}
	return urls
}
