package auth

// This file contains example handler implementations showing how to use the auth context.
// These are reference implementations - integrate these patterns into your actual handlers.

import (
	"encoding/json"
	"net/http"
)

// ExampleConsignmentHandler shows how to use auth context in a handler
// This is a reference implementation - adapt to your actual handler structure
func ExampleConsignmentHandler(w http.ResponseWriter, r *http.Request) {
	// Get auth context from request
	authCtx := GetAuthContext(r.Context())

	// If auth context is nil, the request came without valid Authorization header
	// Decide whether to continue or reject
	if authCtx == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Access trader information
	traderID := authCtx.TraderID

	// Parse trader context JSON if needed
	var traderMetadata map[string]interface{}
	if authCtx.TraderContext != nil && len(authCtx.TraderContext.TraderContext) > 0 {
		if err := json.Unmarshal(authCtx.TraderContext.TraderContext, &traderMetadata); err != nil {
			http.Error(w, "failed to parse trader context", http.StatusInternalServerError)
			return
		}
	}

	// Use trader information in business logic
	// Example: Associate consignment with trader
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "Consignment created",
		"trader_id": traderID,
		"metadata":  traderMetadata,
	})
}

// ExamplePublicEndpoint shows a handler that works with or without auth
func ExamplePublicEndpoint(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())

	response := map[string]interface{}{
		"data": "public data",
	}

	// If authenticated, personalize the response
	if authCtx != nil {
		response["authenticated"] = true
		response["trader_id"] = authCtx.TraderID
	} else {
		response["authenticated"] = false
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// ExampleUpdateTraderContext shows how to update trader context
func ExampleUpdateTraderContext(authService *AuthService, w http.ResponseWriter, r *http.Request) {
	// Get auth context
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body with new trader context
	var newContext map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&newContext); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert to json.RawMessage
	contextJSON, err := json.Marshal(newContext)
	if err != nil {
		http.Error(w, "failed to marshal trader context", http.StatusInternalServerError)
		return
	}

	// Update trader context in database
	// This persists the trader metadata for future requests
	err = authService.UpdateTraderContext(authCtx.TraderID, contextJSON)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Trader context updated",
	})
}

// ExampleGetTraderContext shows how to fetch a specific trader's context
func ExampleGetTraderContext(authService *AuthService, w http.ResponseWriter, r *http.Request) {
	// Get auth context
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// If needed, fetch fresh trader context from database
	// (the middleware already did this, but this shows how to do it manually)
	traderCtx, err := authService.GetTraderContext(authCtx.TraderID)
	if err != nil {
		http.Error(w, "Trader not found", http.StatusNotFound)
		return
	}

	// Parse JSON context
	var metadata map[string]interface{}
	if err := json.Unmarshal(traderCtx.TraderContext, &metadata); err != nil {
		http.Error(w, "failed to parse trader context", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"trader_id": traderCtx.TraderID,
		"context":   metadata,
	})
}

// ExampleFilterByTrader shows how to use auth context to filter results by trader
func ExampleFilterByTrader(w http.ResponseWriter, r *http.Request) {
	// Get auth context
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Filter query results by authenticated trader
	// In real implementation, pass authCtx.TraderID to database query
	traderID := authCtx.TraderID

	// Example of what your handler would do:
	// results := db.Where("trader_id = ?", traderID).Find(&items)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"filtered_by": traderID,
		"count":       0, // Replace with actual count
	})
}

// ExampleAdminOnlyEndpoint shows how to check for specific trader roles/permissions
func ExampleAdminOnlyEndpoint(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse trader context to check for admin role
	var metadata map[string]interface{}
	if authCtx.TraderContext != nil {
		if err := json.Unmarshal(authCtx.TraderContext.TraderContext, &metadata); err != nil {
			http.Error(w, "failed to parse trader context", http.StatusInternalServerError)
			return
		}
	}

	// Example: Check if trader has admin role
	role, ok := metadata["role"].(string)
	if !ok || role != "admin" {
		http.Error(w, "Forbidden - admin access required", http.StatusForbidden)
		return
	}

	// Proceed with admin operation
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Admin operation successful",
	})
}

// TODO: Integration Examples

// ExampleHandlerIntegration shows how to integrate these patterns into your handler
// File: internal/workflow/router/handler.go (example)
/*

func (w *Manager) HandleCreateConsignment(writer http.ResponseWriter, r *http.Request) {
    // Get auth context - pattern from ExampleConsignmentHandler
    authCtx := auth.GetAuthContext(r.Context())
    if authCtx == nil {
        // Decide: public endpoint or require auth
        http.Error(writer, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Parse request body
    var req CreateConsignmentRequest
    json.NewDecoder(r.Body).Decode(&req)

    // Create consignment with authenticated trader
    consignment := &model.Consignment{
        TraderID: authCtx.TraderID,  // Use trader from auth context
        Flow:     req.Flow,
        Items:    req.Items,
    }

    // Save to database
    // ws.service.CreateConsignment(authCtx.TraderID, consignment)

    writer.WriteHeader(http.StatusCreated)
    json.NewEncoder(writer).Encode(consignment)
}

func (w *Manager) HandleGetConsignmentsByTraderID(writer http.ResponseWriter, r *http.Request) {
    // Get auth context - pattern from ExampleFilterByTrader
    authCtx := auth.GetAuthContext(r.Context())
    if authCtx == nil {
        http.Error(writer, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Filter by authenticated trader - pattern from ExampleFilterByTrader
    // This ensures traders can only see their own consignments
    consignments, err := w.service.GetConsignmentsByTraderID(authCtx.TraderID)
    if err != nil {
        http.Error(writer, err.Error(), http.StatusInternalServerError)
        return
    }

    writer.Header().Set("Content-Type", "application/json")
    json.NewEncoder(writer).Encode(consignments)
}

*/

// Example curl commands to test handlers:
/*

# Test authenticated request
curl -X GET http://localhost:8080/api/v1/consignments \
  -H "Authorization: TRADER-001"

# Test public endpoint (with optional auth)
curl -X GET http://localhost:8080/api/v1/products

# Test protected endpoint without auth (should fail)
curl -X POST http://localhost:8080/api/v1/admin/action

# Test update trader context
curl -X PUT http://localhost:8080/api/v1/trader/context \
  -H "Authorization: TRADER-001" \
  -H "Content-Type: application/json" \
  -d '{"company": "Updated Co", "role": "exporter"}'

*/
