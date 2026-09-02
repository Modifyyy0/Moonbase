package src

import "time"

//Adding a private / public, comes from the first letter being capitalized

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

type Message struct
{
	ID int
	UserID int
	ConversationsID int
	Content string
	SentAt time.Time
}