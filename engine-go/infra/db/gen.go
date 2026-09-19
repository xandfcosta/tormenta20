package db

// Regenera a camada tipada de consultas (`sqlcgen`) depois de mexer no
// `query.sql` ou nas migracoes. O codigo gerado e COMMITADO, entao o que esta na
// arvore e a fonte de verdade.
//
//	cd engine-go && go generate ./infra/db
//
// A VERSAO E PINADA, e o numero nao e escolha: e o que o cabecalho do
// `sqlcgen/query.sql.go` declara ter gerado o que esta commitado. Com `@latest`
// uma regeracao de rotina traria o diff de uma versao nova junto com a mudanca
// de quem regerou — e o guia deste pacote ja tinha previsto o problema, pedindo
// para pinar "se a divergencia entre versoes morder" (ALE-347).
//
//go:generate go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
