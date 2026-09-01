CREATE DATABASE IF NOT EXISTS messaging_app;

USE messaging_app;

-- =========================
-- Users
-- =========================

CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE
);


-- =========================
-- Conversations
-- =========================

CREATE TABLE conversations (
    id INT AUTO_INCREMENT PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


-- =========================
-- Messages
-- =========================

CREATE TABLE messages (
    id INT AUTO_INCREMENT PRIMARY KEY,

    user_id INT NOT NULL,
    conversation_id INT NOT NULL

    content varchar(2000),
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP

    FOREIGN KEY (user_id)
        REFERENCES users(id),

    FOREIGN KEY (conversation_id)
        REFERENCES conversations(id)
);