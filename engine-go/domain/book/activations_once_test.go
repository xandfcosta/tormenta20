package book

import "testing"

// O REGISTRO DE ATIVAÇÕES E O CATÁLOGO DE PODERES DE CLASSE SE DECODIFICAM UMA VEZ. Ele é embutido no binário e
// não muda, e a ficha o pergunta várias vezes por página — decodificá-lo (e o
// catálogo de poderes divinos, para a página de cada poder de deus) a cada
// pergunta custava 80% dos ~400ms de cada página da ficha, e o de classe era
// relido uma vez por magia na aba Magias (ALE-473).
func TestTheActivationsAreDecodedOnce(t *testing.T) {
	for name, read := range map[string]func(){
		"Activations":              func() { Activations() },
		"ClassPowersWithModifiers": func() { ClassPowersWithModifiers() },
		"PowersThatTeachSpells":    func() { PowersThatTeachSpells() },
	} {
		read()
		if allocs := testing.AllocsPerRun(5, read); allocs > 10 {
			t.Errorf("%s() aloca %.0f vezes por chamada: o catálogo está sendo decodificado de novo", name, allocs)
		}
	}
}
