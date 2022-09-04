CREATE TABLE `films`
(
    `id`            INTEGER PRIMARY KEY AUTOINCREMENT,
    `original_id`   VARCHAR(64),
    `name`          VARCHAR(255),
    `original_name` VARCHAR(255),
    `link`          TEXT,
    `poster_link`   TEXT,
    `created_at`    DATETIME NULL
);

CREATE UNIQUE INDEX films_original_id_index ON `films` (original_id);
CREATE INDEX films_name_index ON `films` (name);
CREATE INDEX films_created_at_index ON `films` (created_at);

CREATE TABLE `messages`
(
    `message_id`      VARCHAR(64),
    `from_id`         VARCHAR(64),
    `from_first_name` VARCHAR(64),
    `chat_id`         VARCHAR(64),
    `chat_first_name` VARCHAR(64),
    `text`            TEXT,
    `created_at`      DATETIME NULL
);

CREATE INDEX messages_message_id_index ON `messages` (message_id);
CREATE INDEX messages_from_id_index ON `messages` (from_id);
CREATE INDEX messages_chat_id_index ON `messages` (chat_id);
CREATE INDEX messages_created_at_index ON `messages` (created_at);

CREATE TABLE `chats`
(
    `chat_id`    VARCHAR(64),
    `user_id`    VARCHAR(64),
    `subscribed` INT DEFAULT 1,
    `status`      INT DEFAULT 0,
    `created_at` DATETIME NULL,
    `updated_at` DATETIME NULL
);

CREATE UNIQUE INDEX chats_chat_id_index ON `chats` (chat_id);
CREATE INDEX chats_user_id_index ON `chats` (user_id);
CREATE INDEX chats_subscribed_index ON `chats` (subscribed);
CREATE INDEX chats_status_index ON `chats` (status);
CREATE INDEX chats_created_at_index ON `chats` (created_at);
CREATE INDEX chats_updated_at_index ON `chats` (updated_at);

CREATE TABLE `watchers`
(
    `id` INTEGER PRIMARY KEY AUTOINCREMENT,
    `chat_id`    VARCHAR(64),
    `keywords`   TEXT,
    `keywords_normalised`   TEXT,
    `created_at` DATETIME NULL
);

CREATE INDEX watchers_chat_id_index ON `watchers` (chat_id);
CREATE INDEX watchers_created_at_index ON `watchers` (created_at);

CREATE VIRTUAL TABLE watchers_fts USING fts5(
    chat_id,
    keywords_normalised,
    content='watchers',
    content_rowid='id',
    tokenize="trigram"
);

CREATE TRIGGER watcher_autoinsert AFTER INSERT ON watchers
BEGIN
    INSERT INTO watchers_fts (rowid, chat_id, keywords_normalised)
    VALUES (new.id, new.chat_id, new.keywords_normalised);
END;

CREATE TRIGGER watcher_autodelete AFTER DELETE ON watchers
BEGIN
    INSERT INTO watchers_fts (watchers_fts, rowid, keywords_normalised)
    VALUES ('delete', old.id, old.keywords_normalised);
END;