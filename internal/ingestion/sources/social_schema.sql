CREATE TABLE IF NOT EXISTS posts_x (
    id VARCHAR PRIMARY KEY,
    politico_id VARCHAR,
    created_at TIMESTAMP,
    text TEXT,
    hashtags VARCHAR[],
    like_count INTEGER,
    retweet_count INTEGER,
    reply_count INTEGER,
    quote_count INTEGER,
    source VARCHAR DEFAULT 'x',
    ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS posts_instagram (
    shortcode VARCHAR PRIMARY KEY,
    politico_id VARCHAR,
    taken_at TIMESTAMP,
    caption TEXT,
    hashtags VARCHAR[],
    like_count INTEGER,
    comments_count INTEGER,
    media_type VARCHAR,
    media_url VARCHAR,
    source VARCHAR DEFAULT 'instagram',
    ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS posts_facebook (
    post_id VARCHAR PRIMARY KEY,
    politico_id VARCHAR,
    created_time TIMESTAMP,
    message TEXT,
    shares INTEGER,
    reactions JSON,
    source VARCHAR DEFAULT 'facebook',
    ingested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS politici (
    id VARCHAR PRIMARY KEY,
    full_name VARCHAR,
    party VARCHAR,
    screen_name_x VARCHAR,
    username_ig VARCHAR,
    page_id_fb VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE VIEW v_posts_unified AS
SELECT id AS post_id, politico_id, created_at, text, hashtags, like_count, source, ingested_at FROM posts_x
UNION ALL
SELECT shortcode, politico_id, taken_at, caption, hashtags, like_count, source, ingested_at FROM posts_instagram
UNION ALL
SELECT post_id, politico_id, created_time, message, NULL, 0, source, ingested_at FROM posts_facebook;
