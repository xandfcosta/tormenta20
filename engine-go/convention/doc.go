// Package convention guarda as convenções do REPOSITÓRIO — as que não são de
// nenhum pacote e que por isso não teriam onde morar.
//
// O CLAUDE.md da raiz diz que uma convenção mecanizável vira guarda, e que um
// guarda mora no pacote que a possui. As duas frases se contradizem para uma
// regra sobre TODOS os pacotes: pô-la em qualquer um deles é escolher um dono
// arbitrário, e o `api` — que seria a escolha natural pelo tamanho — está sendo
// dividido em um pacote por cena (ALE-278), então o guarda mudaria de casa junto
// com a próxima fatia.
//
// Aqui não há código de produção, e é de propósito: se um dia houver, a regra
// deixou de ser sobre o repositório e passou a ser sobre alguma coisa.
package convention
