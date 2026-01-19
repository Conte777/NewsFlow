CREATE TABLE IF NOT EXISTS delivered_messages (
    id SERIAL PRIMARY KEY,
    news_id INTEGER NOT NULL,
    user_id BIGINT NOT NULL,
    telegram_message_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_delivered_msg_news_user UNIQUE (news_id, user_id)
);

CREATE INDEX idx_delivered_messages_news_id ON delivered_messages(news_id);
CREATE INDEX idx_delivered_messages_user_id ON delivered_messages(user_id);
