// Package session é o CASO DE USO do ciclo de vida de uma sessão: iniciar,
// encerrar, renomear, reiniciar o combate e excluir.
//
// # Por que existe um grupo `app/` (ALE-344)
//
// O levantamento que abriu esta issue mediu o que já estava certo: 95% das 915
// funções de `domain/` não tocam persistência, e a regra pura já mora lá. O que
// não tinha casa era a ORQUESTRAÇÃO — carregar o estado, chamar a regra pura,
// gravar, publicar. Ela existia em dois lugares e com dois nomes: dentro dos
// stores (`BoardStore.apply`, que é um caso de uso com outro nome) e espalhada
// no `serve/api` como `*Rules`, ao lado de JWT e middleware.
//
// Este grupo é o endereço dessa camada. O que ele PODE e o que ele NÃO pode está
// no `boundary_test.go` e vale para todo pacote que nascer aqui:
//
//   - pode `domain/*` (a regra) e `infra/*` (o banco, o barramento);
//   - NÃO pode `serve/*`. Quem sabe o que é um `*http.Request`, um status 403 ou
//     um remendo de Datastar é a apresentação — e é por isso que os erros daqui
//     são TIPADOS em vez de trazerem um número da biblioteca de HTTP.
//
// # E o ciclo não conhece o transporte
//
// O `serve/api` mapeia `ErrForbidden` para 403 e o `serve/web/table` mapeia para
// a mesma coisa. Devolver o número daqui faria duas telas concordarem por
// acidente — e faria o caso de uso mentir sobre a fronteira em que vive.
package session
