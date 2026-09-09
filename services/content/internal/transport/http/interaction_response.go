package httptransport

import (
	"encoding/json"
	"net/http"

	"general-project/content/internal/application"
)

type commentResponse = CommentResponse

type commentsResponse = CommentsResponse

type likeResponse = LikeResponse

func mapCommentResponse(comment application.CommentDTO) commentResponse {
	return commentResponse{
		ID:        comment.ID.String(),
		PostID:    comment.PostID.String(),
		AuthorID:  comment.AuthorID.String(),
		Body:      comment.Body,
		CreatedAt: comment.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"),
		UpdatedAt: comment.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"),
	}
}

func writeComment(w http.ResponseWriter, status int, comment application.CommentDTO) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(mapCommentResponse(comment))
}

func writeComments(w http.ResponseWriter, status int, comments []application.CommentDTO) {
	response := commentsResponse{Comments: make([]commentResponse, 0, len(comments))}
	for _, comment := range comments {
		response.Comments = append(response.Comments, mapCommentResponse(comment))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeLike(w http.ResponseWriter, status int, like application.LikeDTO) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(likeResponse{PostID: like.PostID.String(), UserID: like.UserID.String(), Liked: like.Liked})
}
