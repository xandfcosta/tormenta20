package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"t20engine/plataforma"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
)

type createItemBody struct {
	CatalogID    *string   `json:"catalogId"`
	Name         *string   `json:"name"`
	Quantity     *int64    `json:"quantity"`
	Slots        *float64  `json:"slots"`
	Equipped     *string   `json:"equipped"`
	Improvements *[]string `json:"improvements"`
	Material     *string   `json:"material"`
}

// handleAddItem resolve name/slots from the
// catalog, validate slots + equip axis + the 4-vested/2-hands caps, then create.
// NOTE: overlay compatibility (improvements/material vs the item family) is not
// yet validated here — the frontend pre-validates it; ported in a later slice.
func (s *Server) handleAddItem(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	var body createItemBody
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if body.Quantity == nil {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"quantity": {"quantity must be an integer number"}})
		return
	}
	if *body.Quantity < 1 || *body.Quantity > 9999 {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"quantity": {"quantity out of range [1, 9999]"}})
		return
	}

	fields := plataforma.FieldErrorMap{}
	name := ""
	if body.Name != nil {
		name = strings.TrimSpace(*body.Name)
	}
	slots := body.Slots
	var catalog *engine.CatalogItem
	if body.CatalogID != nil && *body.CatalogID != "" {
		if s.catalogs != nil {
			catalog = s.catalogs.Item(*body.CatalogID)
		}
		if catalog == nil {
			fields["catalogId"] = []string{fmt.Sprintf("Unknown catalog item %q", *body.CatalogID)}
		} else {
			if name == "" {
				name = catalog.Name
			}
			if slots == nil {
				sv := catalog.Slots
				slots = &sv
			}
		}
	}
	if name == "" {
		fields["name"] = []string{"Name is required"}
	}
	switch {
	case slots == nil:
		fields["slots"] = []string{"Slots is required for custom items"}
	case slotsNotMultiple(*slots):
		fields["slots"] = []string{"Slots must be a multiple of 0.5"}
	}
	if len(fields) > 0 {
		plataforma.WriteValidationError(w, fields)
		return
	}

	if body.Equipped != nil && *body.Equipped != "" {
		if top, fieldMsg := equipAxisError(catalog, *body.Equipped); top != "" {
			writeAxisError(w, top, fieldMsg)
			return
		}
		if msg, err := s.equipLimitCheck(r, row.ID, 0, *body.Equipped); err != nil {
			plataforma.WriteError(w, http.StatusInternalServerError, "Could not check equip limits")
			return
		} else if msg != "" {
			plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"equipped": {msg}})
			return
		}
	}

	item, err := s.queries.CreateItem(r.Context(), sqlcgen.CreateItemParams{
		Characterid:  row.ID,
		Catalogid:    nullString(body.CatalogID),
		Name:         name,
		Quantity:     *body.Quantity,
		Slots:        *slots,
		Equipped:     nullString(body.Equipped),
		Improvements: marshalStrings(body.Improvements),
		Material:     nullString(body.Material),
		Createdat:    plataforma.NowISO(),
	})
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not create item")
		return
	}
	plataforma.WriteJSON(w, http.StatusCreated, ItemDTO{
		ID: item.ID, CatalogID: plataforma.NullToPtr(item.Catalogid), Name: item.Name,
		Quantity: item.Quantity, Slots: item.Slots, Equipped: plataforma.NullToPtr(item.Equipped),
		Improvements: item.Improvements, Material: plataforma.NullToPtr(item.Material),
	})
}

// handleUpdateItem partial patch of an
// item, validating slots + equip axis + the equip caps when `equipped` changes.
// Decodes into a raw-field map so an ABSENT field (Leave unchanged) is
// distinguished from an explicit null (unequip / clear material) — a *string
// can't tell them apart. NOTE: overlay compatibility is deferred like addItem.
func (s *Server) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	itemID, ok := intParam(w, r, "itemId")
	if !ok {
		return
	}
	var raw map[string]json.RawMessage
	if !plataforma.DecodeJSON(w, r, &raw) {
		return
	}
	item, err := s.queries.GetItem(r.Context(), itemID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && item.Characterid != row.ID) {
		plataforma.WriteError(w, http.StatusNotFound, fmt.Sprintf("Item %d not found", itemID))
		return
	}
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not Load item")
		return
	}

	var set setBuilder
	if v, has := raw["name"]; has {
		var name string
		if json.Unmarshal(v, &name) != nil {
			plataforma.WriteError(w, http.StatusBadRequest, "Invalid name")
			return
		}
		set.Add("name = ?", strings.TrimSpace(name))
	}
	if v, has := raw["quantity"]; has {
		var q int64
		if json.Unmarshal(v, &q) != nil || q < 1 || q > 9999 {
			plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"quantity": {"quantity out of range [1, 9999]"}})
			return
		}
		set.Add("quantity = ?", q)
	}
	if v, has := raw["slots"]; has {
		var sl float64
		if json.Unmarshal(v, &sl) != nil || slotsNotMultiple(sl) {
			plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"slots": {"Slots must be a multiple of 0.5"}})
			return
		}
		set.Add("slots = ?", sl)
	}
	if v, has := raw["equipped"]; has {
		var eq *string
		if json.Unmarshal(v, &eq) != nil {
			plataforma.WriteError(w, http.StatusBadRequest, "Invalid equipped")
			return
		}
		if !ptrEq(eq, plataforma.NullToPtr(item.Equipped)) && eq != nil && *eq != "" {
			var catalog *engine.CatalogItem
			if item.Catalogid.Valid && s.catalogs != nil {
				catalog = s.catalogs.Item(item.Catalogid.String)
			}
			if top, fieldMsg := equipAxisError(catalog, *eq); top != "" {
				writeAxisError(w, top, fieldMsg)
				return
			}
			if msg, err := s.equipLimitCheck(r, row.ID, itemID, *eq); err != nil {
				plataforma.WriteError(w, http.StatusInternalServerError, "Could not check equip limits")
				return
			} else if msg != "" {
				plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"equipped": {msg}})
				return
			}
		}
		set.Add("equipped = ?", nullString(eq))
	}
	if v, has := raw["improvements"]; has {
		var imp []string
		if json.Unmarshal(v, &imp) != nil {
			plataforma.WriteError(w, http.StatusBadRequest, "Invalid improvements")
			return
		}
		set.Add("improvements = ?", marshalStrings(&imp))
	}
	if v, has := raw["material"]; has {
		var mat *string
		if json.Unmarshal(v, &mat) != nil {
			plataforma.WriteError(w, http.StatusBadRequest, "Invalid material")
			return
		}
		set.Add("material = ?", nullString(mat))
	}
	if set.empty() {
		plataforma.WriteError(w, http.StatusBadRequest, "No fields to update")
		return
	}

	// No `execTouched`: character_items carries createdAt only.
	if err := set.exec(r.Context(), s.db, "UPDATE character_items", itemID); err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not update item")
		return
	}
	updated, err := s.queries.GetItem(r.Context(), itemID)
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not reload item")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, ItemDTO{
		ID: updated.ID, CatalogID: plataforma.NullToPtr(updated.Catalogid), Name: updated.Name,
		Quantity: updated.Quantity, Slots: updated.Slots, Equipped: plataforma.NullToPtr(updated.Equipped),
		Improvements: updated.Improvements, Material: plataforma.NullToPtr(updated.Material),
	})
}

// ptrEq reports whether two *string carry the same presence + value.
func ptrEq(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// handleDeleteItem 404 if the item isn't
// on this character; returns {id}.
func (s *Server) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	itemID, ok := intParam(w, r, "itemId")
	if !ok {
		return
	}
	item, err := s.queries.GetItem(r.Context(), itemID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && item.Characterid != row.ID) {
		plataforma.WriteError(w, http.StatusNotFound, fmt.Sprintf("Item %d not found", itemID))
		return
	}
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not Load item")
		return
	}
	if err := s.queries.DeleteItem(r.Context(), itemID); err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not delete item")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]int64{"id": itemID})
}

// equipLimitCheck runs the 4-vested/2-hands caps over the character's OTHER
// equipped items (excludeItemID, 0 = none). Returns a field message or "".
func (s *Server) equipLimitCheck(r *http.Request, charID, excludeItemID int64, incoming string) (string, error) {
	equipped, err := s.queries.ListEquippedItems(r.Context(), charID)
	if err != nil {
		return "", err
	}
	others := make([]string, 0, len(equipped))
	for _, e := range equipped {
		if e.ID == excludeItemID || !e.Equipped.Valid {
			continue
		}
		others = append(others, e.Equipped.String)
	}
	return equipLimitError(others, incoming), nil
}

// writeAxisError emits the equip-axis BadRequest: a custom top message + the
// equipped field error (assertEquipAxisAllowed).
func writeAxisError(w http.ResponseWriter, top, field string) {
	plataforma.WriteFieldError(w, http.StatusBadRequest, top, plataforma.FieldErrorMap{"equipped": {field}})
}

// marshalStrings JSON-encodes a string slice, normalizing nil (absent or JSON
// null) to "[]" rather than Go's "null" so the column matches JSON.stringify.
func marshalStrings(p *[]string) string {
	if p == nil || *p == nil {
		return "[]"
	}
	b, _ := json.Marshal(*p)
	return string(b)
}
