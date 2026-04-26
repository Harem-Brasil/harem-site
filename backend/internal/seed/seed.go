package seed

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12
const defaultPassword = "Seed123!"

type Seeder struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Seeder {
	return &Seeder{db: db}
}

func (s *Seeder) Run(ctx context.Context) error {
	slog.Info("starting database seeding...")

	userIDs, err := s.seedUsers(ctx)
	if err != nil {
		return fmt.Errorf("seed users: %w", err)
	}

	postIDs, err := s.seedPosts(ctx, userIDs)
	if err != nil {
		return fmt.Errorf("seed posts: %w", err)
	}

	if err := s.seedComments(ctx, userIDs, postIDs); err != nil {
		return fmt.Errorf("seed comments: %w", err)
	}

	if err := s.seedLikes(ctx, userIDs, postIDs); err != nil {
		return fmt.Errorf("seed likes: %w", err)
	}

	categoryIDs, err := s.seedForumCategories(ctx)
	if err != nil {
		return fmt.Errorf("seed forum categories: %w", err)
	}

	topicIDs, err := s.seedForumTopics(ctx, userIDs, categoryIDs)
	if err != nil {
		return fmt.Errorf("seed forum topics: %w", err)
	}

	if err := s.seedForumPosts(ctx, userIDs, topicIDs); err != nil {
		return fmt.Errorf("seed forum posts: %w", err)
	}

	roomIDs, err := s.seedChatRooms(ctx, userIDs)
	if err != nil {
		return fmt.Errorf("seed chat rooms: %w", err)
	}

	if err := s.seedChatMessages(ctx, userIDs, roomIDs); err != nil {
		return fmt.Errorf("seed chat messages: %w", err)
	}

	slog.Info("database seeding completed successfully")
	return nil
}

// --- Users ---

type seedUser struct {
	id         string
	email      string
	screenName string
	role       string
	bio        string
}

func (s *Seeder) seedUsers(ctx context.Context) ([]string, error) {
	users := []seedUser{
		{"", "admin@harembrasil.com.br", "Admin", "admin", "Administrador da plataforma"},
		{"", "mod@harembrasil.com.br", "Moderadora", "moderator", "Moderadora da comunidade"},
		{"", "carla@harembrasil.com.br", "CarlaSensual", "creator", "Criadora de conteúdo exclusivo 🔥"},
		{"", "bianca@harembrasil.com.br", "BiancaLux", "creator", "Modelo e influenciadora digital"},
		{"", "diana@harembrasil.com.br", "DianaSecret", "creator", "Conteúdo premium e chat exclusivo"},
		{"", "user1@example.com", "JoãoSilva", "user", "Fã da plataforma"},
		{"", "user2@example.com", "PedroSantos", "user", "Apreciador de conteúdo"},
		{"", "user3@example.com", "LucasOliveira", "user", "Membro ativo"},
		{"", "user4@example.com", "RafaelCosta", "user", "Entusiasta"},
		{"", "user5@example.com", "MarcosSouza", "user", "Novo membro"},
		{"", "user6@example.com", "AnaPaula", "user", "Curiosa pela plataforma"},
		{"", "user7@example.com", "JulianaR", "user", "Explorando o site"},
		{"", "user8@example.com", "FernandaM", "user", "Primeira vez aqui"},
		{"", "user9@example.com", "CamilaDias", "user", "Adorando o conteúdo"},
		{"", "user10@example.com", "LarissaP", "user", "Membro recente"},
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	ids := make([]string, 0, len(users))
	now := time.Now().UTC()

	for i, u := range users {
		id := uuid.New().String()
		createdAt := now.Add(-time.Duration(len(users)-i) * 24 * time.Hour)

		_, err := s.db.Exec(ctx,
			`INSERT INTO users (id, email, screen_name, password_hash, role, bio, accept_terms_version, email_verified_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, '1.0', $7, $8, $8)
			 ON CONFLICT (email) DO NOTHING`,
			id, u.email, u.screenName, string(hashedPassword), u.role, u.bio, createdAt, createdAt,
		)
		if err != nil {
			slog.Warn("failed to seed user", "email", u.email, "error", err)
			continue
		}

		// If ON CONFLICT DO NOTHING, the user already exists — fetch its ID
		if err := s.db.QueryRow(ctx,
			`SELECT id FROM users WHERE email = $1`, u.email,
		).Scan(&id); err != nil {
			slog.Warn("failed to get user id", "email", u.email, "error", err)
			continue
		}

		ids = append(ids, id)
		slog.Info("seeded user", "screen_name", u.screenName, "role", u.role)
	}

	slog.Info("users seeded", "count", len(ids))
	return ids, nil
}

// --- Posts ---

func (s *Seeder) seedPosts(ctx context.Context, userIDs []string) ([]string, error) {
	postContents := []struct {
		authorIdx  int
		content    string
		visibility string
	}{
		{2, "Novo ensaio fotográfico disponível! Confiram o conteúdo exclusivo 📸🔥", "public"},
		{3, "Dicas de como cuidar da pele no verão ☀️✨", "public"},
		{4, "Conteúdo premium disponível para assinantes! Acessem o chat exclusivo 💬", "subscribers"},
		{2, "Bastidores da produção de hoje — só para membros VIP 💎", "premium"},
		{3, "Promoção de verão! 30% de desconto no plano Premium 🎉", "public"},
		{4, "Live especial na sexta-feira às 20h! Não percam 🎬", "public"},
		{5, "Primeiro post aqui! Estou adorando a plataforma 😊", "public"},
		{6, "Alguém mais acompanhou a live ontem? Foi incrível!", "public"},
		{7, "Dúvida: como funciona o conteúdo exclusivo dos criadores?", "public"},
		{8, "Acabei de assinar o plano Premium — vale muito a pena!", "public"},
		{9, "Boa noite pessoal! Alguém no chat?", "public"},
		{10, "O fórum tá parado, vamos interagir mais! 💪", "public"},
		{2, "Preview do novo ensaio — completo só para subscribers 🔓", "subscribers"},
		{3, "Tutorial de maquiagem para a balada 💄", "public"},
		{4, "Mensagem especial para meus inscritos ❤️", "subscribers"},
	}

	ids := make([]string, 0, len(postContents))
	now := time.Now().UTC()

	for i, p := range postContents {
		if p.authorIdx >= len(userIDs) {
			continue
		}
		id := uuid.New().String()
		createdAt := now.Add(-time.Duration(len(postContents)-i) * 12 * time.Hour)

		_, err := s.db.Exec(ctx,
			`INSERT INTO posts (id, author_id, content, visibility, like_count, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, 0, $5, $5)`,
			id, userIDs[p.authorIdx], p.content, p.visibility, createdAt,
		)
		if err != nil {
			slog.Warn("failed to seed post", "index", i, "error", err)
			continue
		}

		ids = append(ids, id)
	}

	slog.Info("posts seeded", "count", len(ids))
	return ids, nil
}

// --- Comments ---

func (s *Seeder) seedComments(ctx context.Context, userIDs, postIDs []string) error {
	commentTexts := []string{
		"Incrível! 🔥",
		"Adorei o conteúdo!",
		"Muito bom, continue assim!",
		"Onde posso acessar o conteúdo completo?",
		"Show de bola! 👏",
		"Que lindo! Parabéns!",
		"Vou assinar agora mesmo!",
		"Sempre surpreendendo 😍",
		"Alguém sabe se tem desconto para plano anual?",
		"Primeiro! 😂",
	}

	count := 0
	now := time.Now().UTC()

	for i, postID := range postIDs {
		// 2-4 comments per post
		nComments := 2 + rand.Intn(3)
		for j := 0; j < nComments && j < len(commentTexts); j++ {
			id := uuid.New().String()
			authorIdx := 5 + rand.Intn(len(userIDs)-5) // regular users comment
			if authorIdx >= len(userIDs) {
				authorIdx = len(userIDs) - 1
			}
			createdAt := now.Add(-time.Duration(len(postIDs)-i) * 12 * time.Hour).Add(time.Duration(j) * 5 * time.Minute)

			_, err := s.db.Exec(ctx,
				`INSERT INTO post_comments (id, post_id, author_id, content, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, $5, $5)`,
				id, postID, userIDs[authorIdx], commentTexts[j%len(commentTexts)], createdAt,
			)
			if err != nil {
				slog.Warn("failed to seed comment", "error", err)
				continue
			}
			count++
		}
	}

	slog.Info("comments seeded", "count", count)
	return nil
}

// --- Likes ---

func (s *Seeder) seedLikes(ctx context.Context, userIDs, postIDs []string) error {
	count := 0

	for _, postID := range postIDs {
		// 3-8 likes per post from random users
		nLikes := 3 + rand.Intn(6)
		likedBy := make(map[int]bool)
		for j := 0; j < nLikes; j++ {
			idx := 5 + rand.Intn(len(userIDs)-5)
			if likedBy[idx] || idx >= len(userIDs) {
				continue
			}
			likedBy[idx] = true

			_, err := s.db.Exec(ctx,
				`INSERT INTO post_likes (post_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				postID, userIDs[idx],
			)
			if err != nil {
				continue
			}
			count++
		}

		// Update like_count on the post
		s.db.Exec(ctx, `UPDATE posts SET like_count = (SELECT COUNT(*) FROM post_likes WHERE post_id = $1) WHERE id = $1`, postID)
	}

	slog.Info("likes seeded", "count", count)
	return nil
}

// --- Forum Categories ---

func (s *Seeder) seedForumCategories(ctx context.Context) ([]string, error) {
	// Default categories are already created by migration 001, fetch them
	categories := []struct {
		name string
		slug string
	}{
		{"Geral", "geral"},
		{"Apresentações", "apresentacoes"},
		{"Dúvidas", "duvidas"},
		{"Sugestões", "sugestoes"},
		{"Criadores", "criadores"},
		{"Eventos", "eventos"},
	}

	// Create extra categories beyond the defaults
	for i := 4; i < len(categories); i++ {
		id := uuid.New().String()
		_, err := s.db.Exec(ctx,
			`INSERT INTO forum_categories (id, name, slug, description, sort_order) VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (slug) DO NOTHING`,
			id, categories[i].name, categories[i].slug, fmt.Sprintf("Categoria de %s", strings.ToLower(categories[i].name)), i+1,
		)
		if err != nil {
			slog.Warn("failed to seed forum category", "slug", categories[i].slug, "error", err)
		}
	}

	// Fetch all category IDs
	ids := make([]string, 0)
	rows, err := s.db.Query(ctx, `SELECT id FROM forum_categories ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}

	slog.Info("forum categories seeded", "count", len(ids))
	return ids, nil
}

// --- Forum Topics ---

func (s *Seeder) seedForumTopics(ctx context.Context, userIDs, categoryIDs []string) ([]string, error) {
	topics := []struct {
		catIdx    int
		authorIdx int
		title     string
	}{
		{0, 5, "Bem-vindos ao fórum do Harem Brasil!"},
		{0, 6, "Regras da comunidade — leiam antes de postar"},
		{0, 7, "Dicas para novos membros"},
		{1, 8, "Olá pessoal! Me chamo Rafael, sou de SP"},
		{1, 9, "Cheguei agora, ainda conhecendo a plataforma"},
		{2, 10, "Como cancelar a assinatura?"},
		{2, 11, "Qual a diferença entre os planos Basic e Premium?"},
		{3, 12, "Sugestão: chat por voz ao vivo"},
		{3, 13, "Seria legal ter um sistema de badges/perfis"},
		{0, 2, "Criadoras: como promover seu conteúdo no fórum"},
		{4, 3, "Dicas para novas criadoras de conteúdo"},
		{5, 4, "Live especial de lançamento — 15 de maio"},
	}

	ids := make([]string, 0, len(topics))
	now := time.Now().UTC()

	for i, t := range topics {
		if t.catIdx >= len(categoryIDs) || t.authorIdx >= len(userIDs) {
			continue
		}

		id := uuid.New().String()
		slug := slugify(t.title)
		createdAt := now.Add(-time.Duration(len(topics)-i) * 48 * time.Hour)

		tx, err := s.db.Begin(ctx)
		if err != nil {
			slog.Warn("failed to begin tx for forum topic", "error", err)
			continue
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO forum_topics (id, category_id, author_id, title, slug, last_reply_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $6, $6)`,
			id, categoryIDs[t.catIdx], userIDs[t.authorIdx], t.title, slug, createdAt,
		)
		if err != nil {
			tx.Rollback(ctx)
			slog.Warn("failed to seed forum topic", "error", err)
			continue
		}

		// First post (the topic body)
		postID := uuid.New().String()
		_, err = tx.Exec(ctx,
			`INSERT INTO forum_posts (id, topic_id, author_id, content, is_first_post, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, true, $5, $5)`,
			postID, id, userIDs[t.authorIdx], fmt.Sprintf("Conteúdo do tópico: %s", t.title), createdAt,
		)
		if err != nil {
			tx.Rollback(ctx)
			slog.Warn("failed to seed forum first post", "error", err)
			continue
		}

		if err := tx.Commit(ctx); err != nil {
			slog.Warn("failed to commit forum topic", "error", err)
			continue
		}

		ids = append(ids, id)
	}

	slog.Info("forum topics seeded", "count", len(ids))
	return ids, nil
}

// --- Forum Posts (replies) ---

func (s *Seeder) seedForumPosts(ctx context.Context, userIDs, topicIDs []string) error {
	replyTexts := []string{
		"Concordo totalmente! 👍",
		"Obrigado pela dica, muito útil!",
		"Alguém sabe mais detalhes sobre isso?",
		"Show, vou testar aqui!",
		"Interessante, não sabia disso.",
		"Compartilhando minha experiência: foi ótimo!",
		"Pode contar comigo para ajudar!",
		"Que legal, obrigado por compartilhar!",
	}

	count := 0
	now := time.Now().UTC()

	for i, topicID := range topicIDs {
		nReplies := 2 + rand.Intn(4)
		for j := 0; j < nReplies; j++ {
			id := uuid.New().String()
			authorIdx := 5 + rand.Intn(len(userIDs)-5)
			if authorIdx >= len(userIDs) {
				authorIdx = len(userIDs) - 1
			}
			createdAt := now.Add(-time.Duration(len(topicIDs)-i) * 48 * time.Hour).Add(time.Duration(j) * 2 * time.Hour)

			_, err := s.db.Exec(ctx,
				`INSERT INTO forum_posts (id, topic_id, author_id, content, is_first_post, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, false, $5, $5)`,
				id, topicID, userIDs[authorIdx], replyTexts[j%len(replyTexts)], createdAt,
			)
			if err != nil {
				slog.Warn("failed to seed forum reply", "error", err)
				continue
			}
			count++
		}

		// Update reply_count on the topic
		s.db.Exec(ctx, `UPDATE forum_topics SET reply_count = (SELECT COUNT(*) FROM forum_posts WHERE topic_id = $1 AND is_first_post = false) WHERE id = $1`, topicID)
	}

	slog.Info("forum posts (replies) seeded", "count", count)
	return nil
}

// --- Chat Rooms ---

func (s *Seeder) seedChatRooms(ctx context.Context, userIDs []string) ([]string, error) {
	rooms := []struct {
		name        string
		roomType    string
		description string
		creatorIdx  int
	}{
		{"Sala Geral", "public", "Bate-papo livre para todos os membros", 0},
		{"Criadoras", "public", "Discussões entre criadoras de conteúdo", 2},
		{"Vip Lounge", "private", "Sala exclusiva para membros Premium", 0},
		{"Suporte", "public", "Dúvidas e suporte da plataforma", 1},
		{"CarlaSensual - Exclusivo", "private", "Chat exclusivo da CarlaSensual", 2},
		{"BiancaLux - Fans", "private", "Chat para fãs da BiancaLux", 3},
	}

	ids := make([]string, 0, len(rooms))
	now := time.Now().UTC()

	for i, r := range rooms {
		if r.creatorIdx >= len(userIDs) {
			continue
		}

		id := uuid.New().String()
		createdAt := now.Add(-time.Duration(len(rooms)-i) * 72 * time.Hour)

		tx, err := s.db.Begin(ctx)
		if err != nil {
			slog.Warn("failed to begin tx for chat room", "error", err)
			continue
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO chat_rooms (id, name, type, description, created_by, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
			id, r.name, r.roomType, r.description, userIDs[r.creatorIdx], createdAt,
		)
		if err != nil {
			tx.Rollback(ctx)
			slog.Warn("failed to seed chat room", "name", r.name, "error", err)
			continue
		}

		// Creator is admin member
		_, err = tx.Exec(ctx,
			`INSERT INTO chat_members (room_id, user_id, role, joined_at) VALUES ($1, $2, 'admin', $3)`,
			id, userIDs[r.creatorIdx], createdAt,
		)
		if err != nil {
			tx.Rollback(ctx)
			continue
		}

		// Add a few regular members to public rooms
		if r.roomType == "public" {
			for j := 5; j < len(userIDs) && j < 10; j++ {
				_, err = tx.Exec(ctx,
					`INSERT INTO chat_members (room_id, user_id, role, joined_at) VALUES ($1, $2, 'member', $3) ON CONFLICT DO NOTHING`,
					id, userIDs[j], createdAt.Add(time.Minute),
				)
				if err != nil {
					continue
				}
			}
		}

		if err := tx.Commit(ctx); err != nil {
			slog.Warn("failed to commit chat room", "error", err)
			continue
		}

		ids = append(ids, id)
	}

	slog.Info("chat rooms seeded", "count", len(ids))
	return ids, nil
}

// --- Chat Messages ---

func (s *Seeder) seedChatMessages(ctx context.Context, userIDs, roomIDs []string) error {
	messages := []string{
		"Oi pessoal! 👋",
		"Boa noite galera!",
		"Alguém online?",
		"Adorei a novidade de hoje!",
		"Quando é a próxima live?",
		"O conteúdo novo tá incrível 🔥",
		"Primeira vez aqui, tudo bem?",
		"Bom dia! Que dia lindo!",
		"Alguém de SP aqui?",
		"Vamos interagir mais!",
		"O fórum tá bombando hoje",
		"Acabei de assinar o Premium!",
		"Recomendo demais a plataforma",
		"Que vibe boa aqui 😊",
		"Boa tarde pessoal!",
	}

	count := 0
	now := time.Now().UTC()

	for i, roomID := range roomIDs {
		// Get members of this room
		memberIDs := []string{}
		rows, err := s.db.Query(ctx, `SELECT user_id FROM chat_members WHERE room_id = $1`, roomID)
		if err != nil {
			continue
		}
		for rows.Next() {
			var uid string
			if rows.Scan(&uid) == nil {
				memberIDs = append(memberIDs, uid)
			}
		}
		rows.Close()

		if len(memberIDs) == 0 {
			continue
		}

		// 5-15 messages per room
		nMessages := 5 + rand.Intn(11)
		for j := 0; j < nMessages; j++ {
			id := uuid.New().String()
			senderIdx := rand.Intn(len(memberIDs))
			createdAt := now.Add(-time.Duration(len(roomIDs)-i) * 72 * time.Hour).Add(time.Duration(j) * 3 * time.Minute)

			_, err := s.db.Exec(ctx,
				`INSERT INTO chat_messages (id, room_id, sender_id, content, created_at) VALUES ($1, $2, $3, $4, $5)`,
				id, roomID, memberIDs[senderIdx], messages[j%len(messages)], createdAt,
			)
			if err != nil {
				slog.Warn("failed to seed chat message", "error", err)
				continue
			}
			count++
		}
	}

	slog.Info("chat messages seeded", "count", count)
	return nil
}

// --- Helpers ---

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "!", "")
	s = strings.ReplaceAll(s, "?", "")
	s = strings.ReplaceAll(s, "—", "-")
	result := make([]byte, 0, len(s))
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result = append(result, c)
		}
	}
	return strings.Trim(string(result), "-")
}
