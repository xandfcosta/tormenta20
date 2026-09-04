package sheet

// MaxTibar é o teto do campo de dinheiro. Não é regra do livro — é o limite que
// mantém o número legível na ficha e a carga sã (cada mil moedas ocupam um
// espaço, p141, então isto já são mil espaços de moeda).
//
// Ele mora aqui desde a ALE-278 porque tem DOIS consumidores: a rota JSON que
// grava o tibar e a aba Mochila da ficha em Datastar. Enquanto era constante do
// `api`, a cena não a alcançava — e o caminho curto de quem precisasse dela do
// outro lado seria escrever `1_000_000` de novo.
const MaxTibar = 1_000_000
