package engine

// Compra de pontos (book p17, Tabela 1-1).
// All six attributes start at 0 with a 10-point budget; costs 1→1, 2→2, 3→4,
// 4→7; −1 refunds 1 point. Racial mods apply after, outside the budget. The forge
// only INFORMS with these numbers: the choice is free, since the book's other
// method (rolls, p17) does not fit a budget.
const PointBuyBudget = 10

var pointBuyCosts = map[int]int{-1: -1, 0: 0, 1: 1, 2: 2, 3: 4, 4: 7}

// PointBuySpent totals the points a BASE attribute spread costs. ok=false when
// any value is outside −1..4, which the table has no cost for.
func PointBuySpent(attrs map[string]int) (int, bool) {
	spent := 0
	for _, k := range AttributeKeys {
		cost, has := pointBuyCosts[attrs[k]]
		if !has {
			return 0, false
		}
		spent += cost
	}
	return spent, true
}
