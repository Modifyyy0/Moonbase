package src

import "time"

type Message struct
{
	ID int
	UserID int
	ConversationsID int
	Content string
	SentAt time.Time
}