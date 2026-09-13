CREATE DATABASE IF NOT EXISTS messaging_app;

USE messaging_app;

CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE
);


CREATE TABLE conversations (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name varchar(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE user_in_conversation (
	user_id  int,
	conversation_id int,
	PRIMARY KEY (user_id, conversation_id),
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);


CREATE TABLE messages (
    id INT AUTO_INCREMENT PRIMARY KEY,

    user_id INT NOT NULL,
    conversation_id INT NOT NULL,

    content varchar(2000),
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE CASCADE,

    FOREIGN KEY (conversation_id)
        REFERENCES conversations(id) ON DELETE CASCADE
);

create table sessions(
	username VARCHAR(50) not null,
	session_token VARCHAR(200),
	createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (username) REFERENCES users(username) ON DELETE CASCADE
);