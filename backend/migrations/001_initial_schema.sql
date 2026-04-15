-- Initial schema for Harém Brasil
-- Run with: psql -d harem -f migrations/001_initial_schema.sql

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('guest', 'user', 'creator', 'moderator', 'admin')),
    bio TEXT,
    avatar_url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    last_seen_at TIMESTAMPTZ
);

-- Sessions for refresh tokens
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

-- Posts table
CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    media_urls TEXT[],
    visibility TEXT DEFAULT 'public' CHECK (visibility IN ('public', 'subscribers', 'premium')),
    like_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Post likes
CREATE TABLE IF NOT EXISTS post_likes (
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (post_id, user_id)
);

-- Post comments
CREATE TABLE IF NOT EXISTS post_comments (
    id UUID PRIMARY KEY,
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Forum categories
CREATE TABLE IF NOT EXISTS forum_categories (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    description TEXT,
    sort_order INT DEFAULT 0,
    topic_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Forum topics
CREATE TABLE IF NOT EXISTS forum_topics (
    id UUID PRIMARY KEY,
    category_id UUID NOT NULL REFERENCES forum_categories(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    slug TEXT NOT NULL,
    reply_count INT DEFAULT 0,
    view_count INT DEFAULT 0,
    is_pinned BOOLEAN DEFAULT FALSE,
    is_locked BOOLEAN DEFAULT FALSE,
    last_reply_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Forum posts (replies)
CREATE TABLE IF NOT EXISTS forum_posts (
    id UUID PRIMARY KEY,
    topic_id UUID NOT NULL REFERENCES forum_topics(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_first_post BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Chat rooms
CREATE TABLE IF NOT EXISTS chat_rooms (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('public', 'private', 'direct')),
    description TEXT,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Chat members
CREATE TABLE IF NOT EXISTS chat_members (
    room_id UUID NOT NULL REFERENCES chat_rooms(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT DEFAULT 'member' CHECK (role IN ('admin', 'moderator', 'member')),
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (room_id, user_id)
);

-- Chat messages
CREATE TABLE IF NOT EXISTS chat_messages (
    id UUID PRIMARY KEY,
    room_id UUID NOT NULL REFERENCES chat_rooms(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Notifications
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    data JSONB,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Creator applications
CREATE TABLE IF NOT EXISTS creator_applications (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bio TEXT NOT NULL,
    social_links TEXT[],
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    submitted_at TIMESTAMPTZ DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    reviewed_by UUID REFERENCES users(id)
);

-- Subscription plans
CREATE TABLE IF NOT EXISTS plans (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    currency TEXT DEFAULT 'BRL',
    interval TEXT NOT NULL CHECK (interval IN ('month', 'year')),
    features TEXT[],
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Subscriptions
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES plans(id),
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'trialing', 'past_due', 'canceled')),
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Audit log
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    action TEXT NOT NULL,
    resource TEXT NOT NULL,
    resource_id UUID,
    details JSONB,
    ip_address INET,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_posts_author ON posts(author_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_posts_visibility ON posts(visibility) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_posts_created ON posts(created_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_comments_post ON post_comments(post_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_topics_category ON forum_topics(category_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_forum_posts_topic ON forum_posts(topic_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_chat_messages_room ON chat_messages(room_id, created_at);
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, created_at) WHERE read_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_sessions_refresh ON sessions(refresh_token) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_subscriptions_user ON subscriptions(user_id, status);
CREATE INDEX IF NOT EXISTS idx_audit_log_user ON audit_log(user_id, created_at);

-- Insert default roles/plans
INSERT INTO plans (id, name, slug, description, price, currency, interval, features)
VALUES 
    (gen_random_uuid(), 'Basic', 'basic', 'Acesso básico ao conteúdo', 9.99, 'BRL', 'month', ARRAY['Acesso a posts públicos', 'Participação no fórum']),
    (gen_random_uuid(), 'Premium', 'premium', 'Acesso premium ao conteúdo', 29.99, 'BRL', 'month', ARRAY['Acesso a posts exclusivos', 'Chat privado', 'Conteúdo de criadores', 'Sem anúncios']),
    (gen_random_uuid(), 'VIP', 'vip', 'Acesso VIP completo', 99.99, 'BRL', 'month', ARRAY['Acesso total ao conteúdo', 'Chat privado ilimitado', 'Encontros presenciais', 'Suporte prioritário'])
ON CONFLICT (slug) DO NOTHING;

INSERT INTO forum_categories (id, name, slug, description, sort_order)
VALUES 
    (gen_random_uuid(), 'Geral', 'geral', 'Discussões gerais', 1),
    (gen_random_uuid(), 'Apresentações', 'apresentacoes', 'Apresente-se para a comunidade', 2),
    (gen_random_uuid(), 'Dúvidas', 'duvidas', 'Tire suas dúvidas', 3),
    (gen_random_uuid(), 'Sugestões', 'sugestoes', 'Sugestões para a plataforma', 4)
ON CONFLICT (slug) DO NOTHING;
