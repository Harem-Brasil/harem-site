/**
 * Gera api-harem-brasil.postman_collection.json (v2.1)
 * Executar: node gen-postman-collection.mjs
 */
import { writeFileSync } from "fs";
import { fileURLToPath } from "url";
import { dirname, join } from "path";

const __dirname = dirname(fileURLToPath(import.meta.url));
const out = join(__dirname, "api-harem-brasil.postman_collection.json");

const R = (name, method, path, opts = {}) => {
  const h = [{ key: "Accept", value: "application/json" }];
  if (opts.json)
    h.push({ key: "Content-Type", value: "application/json" });
  if (opts.bearer === "user")
    h.push({ key: "Authorization", value: "Bearer {{access_token}}" });
  if (opts.bearer === "admin")
    h.push({ key: "Authorization", value: "Bearer {{admin_access_token}}" });
  const body =
    opts.body === undefined
      ? undefined
      : {
          mode: "raw",
          raw:
            typeof opts.body === "string"
              ? opts.body
              : JSON.stringify(opts.body, null, 2),
          options: { raw: { language: "json" } },
        };
  const item = {
    name,
    request: {
      method,
      header: h,
      url: { raw: `{{baseUrl}}${path}` },
    },
  };
  if (body) item.request.body = body;
  if (opts.tests)
    item.event = [{ listen: "test", script: { type: "text/javascript", exec: opts.tests } }];
  return item;
};

const folder = (name, items) => ({ name, item: items });

const loginTests = [
  "try {",
  "  const j = pm.response.json();",
  "  pm.collectionVariables.set('access_token', j.access_token);",
  "  pm.collectionVariables.set('refresh_token', j.refresh_token);",
  "  if (j.user && j.user.id) pm.collectionVariables.set('user_id', j.user.id);",
  "} catch (e) { console.warn(e); }",
];

const registerTests = [
  "if (pm.response.code >= 200 && pm.response.code < 300) {",
  "  try {",
  "    const j = pm.response.json();",
  "    pm.collectionVariables.set('access_token', j.access_token);",
  "    pm.collectionVariables.set('refresh_token', j.refresh_token);",
  "    if (j.user && j.user.id) pm.collectionVariables.set('user_id', j.user.id);",
  "  } catch (e) {}",
  "}",
];

const refreshTests = [
  "try {",
  "  const j = pm.response.json();",
  "  pm.collectionVariables.set('access_token', j.access_token);",
  "  if (j.refresh_token) pm.collectionVariables.set('refresh_token', j.refresh_token);",
  "} catch (e) {}",
];

const adminLoginTests = [
  "try {",
  "  const j = pm.response.json();",
  "  pm.collectionVariables.set('admin_access_token', j.access_token);",
  "} catch (e) {}",
];

const createPostTests = [
  "try {",
  "  const j = pm.response.json();",
  "  if (j.id) pm.collectionVariables.set('last_post_id', j.id);",
  "} catch (e) {}",
];

const collection = {
  info: {
    name: "Harem Brasil API",
    description: [
      "Importar no Postman ou Insomnia (compatível com Collection v2.1).",
      "",
      "Token: execute primeiro «Login» (Auth público). Pedidos autenticados usam {{access_token}}.",
      "«Refresh» atualiza access_token e refresh_token. «Login como admin» preenche {{admin_access_token}} para /admin.",
      "",
      "Variáveis: baseUrl, email, password, adminEmail, adminPassword, targetUserId.",
      "Gerado por: scripts/gen-postman-collection.mjs",
    ].join("\n"),
    schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
  },
  variable: [
    { key: "baseUrl", value: "http://localhost:40080" },
    { key: "email", value: "teste.criador@exemplo.com" },
    { key: "password", value: "SenhaForte!1" },
    { key: "screenName", value: "TesteCriador" },
    { key: "termsVersion", value: "1.0" },
    { key: "adminEmail", value: "admin@exemplo.com" },
    { key: "adminPassword", value: "TroquePorSenhaAdmin" },
    { key: "targetUserId", value: "00000000-0000-0000-0000-000000000000" },
    { key: "access_token", value: "" },
    { key: "refresh_token", value: "" },
    { key: "user_id", value: "" },
    { key: "admin_access_token", value: "" },
    { key: "last_post_id", value: "" },
  ],
  item: [
    folder("Infra", [
      R("Health", "GET", "/health"),
      R("Healthz", "GET", "/healthz"),
      R("Readyz", "GET", "/readyz"),
      R("Version", "GET", "/version"),
    ]),
    folder("Auth público", [
      R("Registo", "POST", "/api/v1/auth/register", {
        json: true,
        body: {
          email: "{{email}}",
          screen_name: "{{screenName}}",
          password: "{{password}}",
          accept_terms_version: "{{termsVersion}}",
        },
        tests: registerTests,
      }),
      R("Login", "POST", "/api/v1/auth/login", {
        json: true,
        body: { email: "{{email}}", password: "{{password}}" },
        tests: loginTests,
      }),
      R("Refresh", "POST", "/api/v1/auth/refresh", {
        json: true,
        body: { refresh_token: "{{refresh_token}}" },
        tests: refreshTests,
      }),
      R("Logout", "POST", "/api/v1/auth/logout", {
        json: true,
        body: { refresh_token: "{{refresh_token}}" },
      }),
      R("Password forgot", "POST", "/api/v1/auth/password/forgot", {
        json: true,
        body: { email: "{{email}}" },
      }),
    ]),
    folder("Auth autenticado", [
      R("Logout todos dispositivos", "POST", "/api/v1/auth/logout-all", {
        bearer: "user",
      }),
    ]),
    folder("Perfil", [
      R("Eu (GET)", "GET", "/api/v1/me", { bearer: "user" }),
      R("Eu (PATCH)", "PATCH", "/api/v1/me", {
        bearer: "user",
        json: true,
        body: {
          screen_name: "{{screenName}}",
          bio: "Bio de teste via Postman",
        },
      }),
      R("Eu (DELETE)", "DELETE", "/api/v1/me", { bearer: "user" }),
    ]),
    folder("Utilizadores", [
      R("Listar", "GET", "/api/v1/users", { bearer: "user" }),
      R("Pesquisar", "GET", "/api/v1/users/search?q=teste&cursor=", {
        bearer: "user",
      }),
      R("Por ID", "GET", "/api/v1/users/{{user_id}}", { bearer: "user" }),
      R("Posts do utilizador", "GET", "/api/v1/users/{{user_id}}/posts?cursor=", {
        bearer: "user",
      }),
    ]),
    folder("Posts & feed", [
      R("Listar posts", "GET", "/api/v1/posts?cursor=", { bearer: "user" }),
      R("Criar post", "POST", "/api/v1/posts", {
        bearer: "user",
        json: true,
        body: {
          content: "Olá, este é um post de teste.",
          media_urls: [],
          visibility: "public",
        },
        tests: createPostTests,
      }),
      R("Obter post (último criado)", "GET", "/api/v1/posts/{{last_post_id}}", {
        bearer: "user",
      }),
      R("Obter post (ID manual)", "GET", "/api/v1/posts/POST_ID_AQUI", {
        bearer: "user",
      }),
      R("Feed home", "GET", "/api/v1/feed/home?cursor=", { bearer: "user" }),
      R("Atualizar post", "PATCH", "/api/v1/posts/POST_ID_AQUI", {
        bearer: "user",
        json: true,
        body: { content: "Conteúdo atualizado" },
      }),
      R("Apagar post", "DELETE", "/api/v1/posts/POST_ID_AQUI", {
        bearer: "user",
      }),
      R("Like", "POST", "/api/v1/posts/POST_ID_AQUI/like", { bearer: "user" }),
      R("Unlike", "DELETE", "/api/v1/posts/POST_ID_AQUI/like", {
        bearer: "user",
      }),
      R("Comentários", "GET", "/api/v1/posts/POST_ID_AQUI/comments?cursor=", {
        bearer: "user",
      }),
      R("Criar comentário", "POST", "/api/v1/posts/POST_ID_AQUI/comments", {
        bearer: "user",
        json: true,
        body: { content: "Comentário de teste" },
      }),
    ]),
    folder("Fórum", [
      R("Categorias", "GET", "/api/v1/forum/categories", { bearer: "user" }),
      R("Tópicos", "GET", "/api/v1/forum/topics?cursor=", { bearer: "user" }),
      R("Criar tópico", "POST", "/api/v1/forum/topics", {
        bearer: "user",
        json: true,
        body: {
          category_id: "00000000-0000-4000-8000-000000000001",
          title: "Tópico de teste",
          content: "Corpo do tópico",
        },
      }),
      R("Tópico por ID", "GET", "/api/v1/forum/topics/TOPIC_ID", {
        bearer: "user",
      }),
      R("Resposta no tópico", "POST", "/api/v1/forum/topics/TOPIC_ID/posts", {
        bearer: "user",
        json: true,
        body: { content: "Mensagem no fórum" },
      }),
    ]),
    folder("Chat", [
      R("Salas", "GET", "/api/v1/chat/rooms", { bearer: "user" }),
      R("Criar sala", "POST", "/api/v1/chat/rooms", {
        bearer: "user",
        json: true,
        body: {
          name: "Sala teste",
          type: "public",
          description: "Teste Postman",
        },
      }),
      R("Sala por ID", "GET", "/api/v1/chat/rooms/ROOM_ID", { bearer: "user" }),
      R("Mensagens", "GET", "/api/v1/chat/rooms/ROOM_ID/messages?cursor=", {
        bearer: "user",
      }),
      R("Entrar na sala", "POST", "/api/v1/chat/rooms/ROOM_ID/join", {
        bearer: "user",
      }),
    ]),
    folder("Notificações", [
      R("Listar", "GET", "/api/v1/notifications?cursor=", { bearer: "user" }),
      R("Não lidas", "GET", "/api/v1/notifications?unread=true&cursor=", {
        bearer: "user",
      }),
      R("Contagem não lidas", "GET", "/api/v1/notifications/unread-count", {
        bearer: "user",
      }),
      R("Marcar lida", "PATCH", "/api/v1/notifications/NOTIF_ID/read", {
        bearer: "user",
      }),
    ]),
    folder("Billing / subscrições", [
      R("Planos", "GET", "/api/v1/billing/plans", { bearer: "user" }),
      R("Subscrição (billing)", "GET", "/api/v1/billing/subscription", {
        bearer: "user",
      }),
      R("Subscrição (subscriptions/me)", "GET", "/api/v1/subscriptions/me", {
        bearer: "user",
      }),
      R("Checkout", "POST", "/api/v1/billing/checkout", {
        bearer: "user",
        json: true,
        body: { plan_id: "SUBSTITUA_UUID_PLANO_ATIVO" },
      }),
      R("Criar subscrição", "POST", "/api/v1/subscriptions", {
        bearer: "user",
        json: true,
        body: { plan_id: "SUBSTITUA_UUID_PLANO_ATIVO" },
      }),
      R("Cancelar subscrição", "POST", "/api/v1/billing/subscription/cancel", {
        bearer: "user",
      }),
      R("Retomar subscrição", "POST", "/api/v1/billing/subscription/resume", {
        bearer: "user",
      }),
    ]),
    folder("Media", [
      R("Criar sessão upload", "POST", "/api/v1/media/upload-sessions", {
        bearer: "user",
        json: true,
        body: {
          file_name: "test.jpg",
          content_type: "image/jpeg",
          size: 1024,
        },
      }),
      R("Completar upload", "POST", "/api/v1/media/upload-sessions/SESSION_ID/complete", {
        bearer: "user",
        json: true,
        body: { etag: '"etagopcional"' },
      }),
    ]),
    folder("Criador", [
      R("Candidatura", "POST", "/api/v1/creator/apply", {
        bearer: "user",
        json: true,
        body: {
          bio: "Sou criador de conteúdo e quero publicar no Harém.",
          social_links: [
            "https://twitter.com/exemplo",
            "https://instagram.com/exemplo",
          ],
        },
      }),
      R("Dashboard", "GET", "/api/v1/creator/dashboard", { bearer: "user" }),
      R("Ganhos", "GET", "/api/v1/creator/earnings", { bearer: "user" }),
      R(
        "Ganhos sumário",
        "GET",
        "/api/v1/creator/earnings/summary?from=2026-01-01&to=2026-02-01",
        { bearer: "user" },
      ),
      R("Catálogo", "GET", "/api/v1/creator/catalog?cursor=", { bearer: "user" }),
      R("Pedidos", "GET", "/api/v1/creator/orders?cursor=", { bearer: "user" }),
    ]),
    folder("Admin", [
      R("Login como admin", "POST", "/api/v1/auth/login", {
        json: true,
        body: { email: "{{adminEmail}}", password: "{{adminPassword}}" },
        tests: adminLoginTests,
      }),
      R("Listar utilizadores", "GET", "/api/v1/admin/users?cursor=", {
        bearer: "admin",
      }),
      R("Alterar role", "PATCH", "/api/v1/admin/users/{{targetUserId}}/role", {
        bearer: "admin",
        json: true,
        body: { role: "creator" },
      }),
      R("Apagar utilizador", "DELETE", "/api/v1/admin/users/{{targetUserId}}", {
        bearer: "admin",
      }),
      R("Estatísticas", "GET", "/api/v1/admin/stats", { bearer: "admin" }),
      R("Audit log", "GET", "/api/v1/admin/audit-log?cursor=", {
        bearer: "admin",
      }),
    ]),
  ],
};

writeFileSync(out, JSON.stringify(collection, null, 2), "utf8");
console.log("Escrito:", out);
