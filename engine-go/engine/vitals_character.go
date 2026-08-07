package engine

// VitalContextFor builds the VitalContext the front assembles in enginePools:
// parsed ownership blobs + attrTotals from the base item/race effects (no
// conditionals). The server-side vitals recompute (backend buildVitalContext +
// vitals-sync) uses it to re-derive PV/PM máximos after a level/class change.
func (c *Catalogs) VitalContextFor(ch Character) VitalContext {
	effects := ComputeItemEffects(c.ActiveItemsFor(ch))
	attrTotals := map[string]int{}
	for _, a := range AttributeKeys {
		attrTotals[a] = effectiveAttribute(ch, a, effects)
	}
	level := 0
	classes := make([]ClassEntry, len(ch.Classes))
	for i, ce := range ch.Classes {
		classes[i] = ClassEntry{ClassName: ce.ClassName, Level: ce.Level}
		level += ce.Level
	}
	raceID := ""
	if len(ch.Races) > 0 {
		raceID = ch.Races[0].Race
	}
	return VitalContext{
		Level:              level,
		Classes:            classes,
		RaceID:             raceID,
		RaceAbilityChoices: parseChoiceSet(ch.RaceAbilityChoices).list,
		PowerIDs:           parseChoiceSet(ch.ClassPowers).list,
		ClassChoices:       parseClassChoices(ch.ClassChoices),
		GodPower:           ch.GodPower,
		Origin:             ch.Origin,
		OriginChoices:      parseChoiceSet(ch.OriginChoices).list,
		AttrTotals:         attrTotals,
	}
}

// VitalsForCharacter re-derives PV/PM máximos for a raw Character — the API's
// server-side vitals recompute after a level or class-level change.
func (c *Catalogs) VitalsForCharacter(ch Character) VitalPools {
	return c.ComputeVitals(c.VitalContextFor(ch))
}
