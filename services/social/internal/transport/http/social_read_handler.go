package httptransport

import (
	"encoding/json"
	"net/http"
	"strconv"
	"uuid"

	"github.com/go-chi/chi/v5"

	"general-project/social/internal/application"
)

// GetRelationships возвращает друзей, подписчиков и подписки текущего пользователя.






func (h *Handler) GetRelationships(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUser(w, r)
	if !ok {
		return
	}
	result, err := h.social.GetRelationships(r.Context(), userID)
	if err != nil {
		writeSocialError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mapRelationshipsResponse(result))
}

// GetPublicFriends возвращает друзей указанного профиля с учётом приватности.








func (h *Handler) GetPublicFriends(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := authenticatedUser(w, r)
	if !ok {
		return
	}
	targetID, err := uuid.Parse(chi.URLParam(r, "targetID"))
	if err != nil {
		writeSocialError(w, application.ErrValidation)
		return
	}
	friends, err := h.social.GetPublicFriends(r.Context(), viewerID, targetID)
	if err != nil {
		writeSocialError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, publicFriendsResponse{Friends: mapUUIDs(friends.Friends), Visible: friends.Visible})
}

// GetRecommendations возвращает кандидатов в друзья, отсортированных по числу общих друзей.








func (h *Handler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUser(w, r)
	if !ok {
		return
	}
	limit := 10
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeSocialError(w, application.ErrValidation)
			return
		}
		limit = parsed
	}
	result, err := h.social.GetRecommendations(r.Context(), userID, limit)
	if err != nil {
		writeSocialError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mapRecommendationsResponse(result))
}

type relationshipsResponse = RelationshipsResponse

type publicFriendsResponse = PublicFriendsResponse

type recommendationsResponse = RecommendationsResponse

type friendRequestsResponse = FriendRequestsResponse

type friendRequestResponse = FriendRequestResponse

type recommendationResponse = RecommendationResponse

func mapRelationshipsResponse(result application.RelationshipsDTO) relationshipsResponse {
	return relationshipsResponse{
		Friends:       mapUUIDs(result.Friends),
		Subscribers:   mapUUIDs(result.Subscribers),
		Subscriptions: mapUUIDs(result.Subscriptions),
	}
}

func mapRecommendationsResponse(result []application.RecommendationDTO) recommendationsResponse {
	response := recommendationsResponse{Recommendations: make([]recommendationResponse, 0, len(result))}
	for _, recommendation := range result {
		response.Recommendations = append(response.Recommendations, recommendationResponse{UserID: recommendation.UserID.String(), CommonFriends: recommendation.CommonFriends})
	}
	return response
}

func mapUUIDs(values []uuid.UUID) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.String())
	}
	return result
}

func mapFriendRequests(requests []application.FriendRequestDTO) []friendRequestResponse {
	result := make([]friendRequestResponse, 0, len(requests))
	for _, request := range requests {
		result = append(result, friendRequestResponse{
			ID: request.ID.String(), SenderID: request.SenderID.String(), RecipientID: request.RecipientID.String(),
			Status: request.Status, CreatedAt: request.CreatedAt, RespondedAt: request.RespondedAt,
		})
	}
	return result
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
