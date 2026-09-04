package domain

// Summary содержит агрегированные показатели принятых интеграционных событий.
type Summary struct {
	EventsCount     int64
	UniqueUsers     int64
	UsersRegistered int64
	PostsCreated    int64
	CommentsCreated int64
	LikesAdded      int64
	LikesRemoved    int64
	PhotosDeleted   int64
}
