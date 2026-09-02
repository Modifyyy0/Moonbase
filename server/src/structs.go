package src

import "time"

// Capitalized first letter = exported (public)
// lowercase first letter = unexported (private)

type User struct
{
	ID int
	Username string
}

type Conversations struct
{
	ID int
	Name string
	CreatedAt time.Time
}

type User_in_conversation struct
{
	UserID int
	ConversationID int
}

type Message struct
{
	ID int
	UserID int
	ConversationsID int
	Content string
	SentAt time.Time
}