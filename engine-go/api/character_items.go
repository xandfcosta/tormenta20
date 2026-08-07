package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

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

// handleAddItem ports CharacterItemsService.addItem: resolve name/slots from the
// catalog, validate slots + equip axis + the 4-vested/2-hands caps, then create.
// NOTE: overlay compatibility (improvements/material vs the item family) is not
// yet validated here — the frontend pre-validates it; ported in a later slice.
func (s *Server) handleAddItem(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	var body createItemBody
	if !decodeJSON(w, r, &body) {
		return
	}
	if _, status, err := s.authorizedCharacter(r.Context(), currentUser(r), id); err != nil {
		writeError(w, status, err.Error())
		return
	}
	if body.Quantity == nil {
		writeValidationError(w, FieldErrorMap{"quantity": {"quantity must be an integer number"}})
		return
	}
	if *body.Quantity < 1 || *body.Quantity > 9999 {
		writeValidationError(w, FieldErrorMap{"quantity": {"quantity out of range [1, 9999]"}})
		return
	}

	fields := FieldErrorMap{}
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
		writeValidationError(w, fields)
		return
	}

	if body.Equipped != nil && *body.Equipped != "" {
		if top, fieldMsg := equipAxisError(catalog, *body.Equipped); top != "" {
			writeAxisError(w, top, fieldMsg)
			return
		}
		if msg, err := s.equipLimitCheck(r, id, 0, *body.Equipped); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not check equip limits")
			return
		} else if msg != "" {
			writeValidationError(w, FieldErrorMap{"equipped": {msg}})
			return
		}
	}

	item, err := s.queries.CreateItem(r.Context(), sqlcgen.CreateItemParams{
		Characterid:  id,
		Catalogid:    nullString(body.CatalogID),
		Name:         name,
		Quantity:     *body.Quantity,
		Slots:        *slots,
		Equipped:     nullString(body.Equipped),
		Improvements: marshalStrings(body.Improvements),
		Material:     nullString(body.Material),
		Createdat:    nowISO(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create item")
		return
	}
	writeJSON(w, http.StatusCreated, ItemDTO{
		ID: item.ID, CatalogID: nullToPtr(item.Catalogid), Name: item.Name,
		Quantity: item.Quantity, Slots: item.Slots, Equipped: nullToPtr(item.Equipped),
		Improvements: item.Improvements, Material: nullToPtr(item.Material),
	})
}

// handleUpdateItem ports CharacterItemsService.updateItem: partial patch of an
// item, validating slots + equip axis + the equip caps when `equipped` changes.
// Decodes into a raw-field map so an ABSENT field (leave unchanged) is
// distinguished from an explicit null (unequip / clear material) — a *string
// can't tell them apart. NOTE: overlay compatibility is deferred like addItem.
func (s *Server) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	itemID, ok := intParam(w, r, "itemId")
	if !ok {
		return
	}
	var raw map[string]json.RawMessage
	if !decodeJSON(w, r, &raw) {
		return
	}
	if _, status, err := s.authorizedCharacter(r.Context(), currentUser(r), id); err != nil {
		writeError(w, status, err.Error())
		return
	}
	item, err := s.queries.GetItem(r.Context(), itemID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && item.Characterid != id) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Item %d not found", itemID))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load item")
		return
	}

	sets := []string{}
	args := []any{}
	if v, has := raw["name"]; has {
		var name string
		if json.Unmarshal(v, &name) != nil {
			writeError(w, http.StatusBadRequest, "Invalid name")
			return
		}
		sets, args = append(sets, "name = ?"), append(args, strings.TrimSpace(name))
	}
	if v, has := raw["quantity"]; has {
		var q int64
		if json.Unmarshal(v, &q) != nil || q < 1 || q > 9999 {
			writeValidationError(w, FieldErrorMap{"quantity": {"quantity out of range [1, 9999]"}})
			return
		}
		sets, args = append(sets, "quantity = ?"), append(args, q)
	}
	if v, has := raw["slots"]; has {
		var sl float64
		if json.Unmarshal(v, &sl) != nil || slotsNotMultiple(sl) {
			writeValidationError(w, FieldErrorMap{"slots": {"Slots must be a multiple of 0.5"}})
			return
		}
		sets, args = append(sets, "slots = ?"), append(args, sl)
	}
	if v, has := raw["equipped"]; has {
		var eq *string
		if json.Unmarshal(v, &eq) != nil {
			writeError(w, http.StatusBadRequest, "Invalid equipped")
			return
		}
		if !ptrEq(eq, nullToPtr(item.Equipped)) && eq != nil && *eq != "" {
			var catalog *engine.CatalogItem
			if item.Catalogid.Valid && s.catalogs != nil {
				catalog = s.catalogs.Item(item.Catalogid.String)
			}
			if top, fieldMsg := equipAxisError(catalog, *eq); top != "" {
				writeAxisError(w, top, fieldMsg)
				return
			}
			if msg, err := s.equipLimitCheck(r, id, itemID, *eq); err != nil {
				writeError(w, http.StatusInternalServerError, "Could not check equip limits")
				return
			} else if msg != "" {
				writeValidationError(w, FieldErrorMap{"equipped": {msg}})
				return
			}
		}
		sets, args = append(sets, "equipped = ?"), append(args, nullString(eq))
	}
	if v, has := raw["improvements"]; has {
		var imp []string
		if json.Unmarshal(v, &imp) != nil {
			writeError(w, http.StatusBadRequest, "Invalid improvements")
			return
		}
		sets, args = append(sets, "improvements = ?"), append(args, marshalStrings(&imp))
	}
	if v, has := raw["material"]; has {
		var mat *string
		if json.Unmarshal(v, &mat) != nil {
			writeError(w, http.StatusBadRequest, "Invalid material")
			return
		}
		sets, args = append(sets, "material = ?"), append(args, nullString(mat))
	}
	if len(sets) == 0 {
		writeError(w, http.StatusBadRequest, "No fields to update")
		return
	}

	args = append(args, itemID)
	//nolint:gosec // SET clause is built from a fixed allowlist of columns, not input.
	if _, err := s.db.ExecContext(r.Context(),
		"UPDATE character_items SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not update item")
		return
	}
	updated, err := s.queries.GetItem(r.Context(), itemID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not reload item")
		return
	}
	writeJSON(w, http.StatusOK, ItemDTO{
		ID: updated.ID, CatalogID: nullToPtr(updated.Catalogid), Name: updated.Name,
		Quantity: updated.Quantity, Slots: updated.Slots, Equipped: nullToPtr(updated.Equipped),
		Improvements: updated.Improvements, Material: nullToPtr(updated.Material),
	})
}

// ptrEq reports whether two *string carry the same presence + value.
func ptrEq(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// handleDeleteItem ports CharacterItemsService.deleteItem: 404 if the item isn't
// on this character; returns {id}.
func (s *Server) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	itemID, ok := intParam(w, r, "itemId")
	if !ok {
		return
	}
	if _, status, err := s.authorizedCharacter(r.Context(), currentUser(r), id); err != nil {
		writeError(w, status, err.Error())
		return
	}
	item, err := s.queries.GetItem(r.Context(), itemID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && item.Characterid != id) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Item %d not found", itemID))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load item")
		return
	}
	if err := s.queries.DeleteItem(r.Context(), itemID); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not delete item")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": itemID})
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
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"statusCode":  http.StatusBadRequest,
		"error":       "Bad Request",
		"message":     top,
		"fieldErrors": FieldErrorMap{"equipped": {field}},
	})
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
