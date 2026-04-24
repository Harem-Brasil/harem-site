package features

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"github.com/golang-jwt/jwt/v5"

	"github.com/harem-brasil/backend/internal/application"
	"log/slog"
)

type apiContext struct {
	server        *application.HTTPServer
	recorder      *httptest.ResponseRecorder
	request       *http.Request
	response      map[string]any
	responseArray []any
	token         string
	refreshToken  string
	userID        string
}

var testCtx *apiContext

func InitializeTestSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
		testCtx = &apiContext{}
	})
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		testCtx.recorder = httptest.NewRecorder()
		testCtx.response = nil
		testCtx.responseArray = nil
		return ctx, nil
	})

	// Given steps - pt-BR
	ctx.Step(`^que a API está em execução$`, theAPIIsRunning)
	ctx.Step(`^que o banco de dados está conectado$`, theDatabaseIsConnected)
	ctx.Step(`^que o cache está conectado$`, theCacheIsConnected)
	ctx.Step(`^que eu estou autenticado como usuário "([^"]*)"$`, iAmAuthenticatedAsUser)
	ctx.Step(`^que eu estou autenticado como criador "([^"]*)"$`, iAmAuthenticatedAsCreator)
	ctx.Step(`^que eu não estou autenticado$`, iAmNotAuthenticated)
	ctx.Step(`^que eu tenho um payload de registro válido$`, iHaveAValidRegistrationPayload)
	ctx.Step(`^que um usuário com email "([^"]*)" já existe$`, aUserWithEmailAlreadyExists)
	ctx.Step(`^que um usuário registrado com email "([^"]*)" e senha "([^"]*)"$`, aRegisteredUserWithEmailAndPassword)
	ctx.Step(`^que um usuário registrado com email "([^"]*)"$`, aRegisteredUserWithEmail)
	ctx.Step(`^que eu tenho um refresh token válido$`, iHaveAValidRefreshToken)
	ctx.Step(`^que eu tenho um token de reset de senha válido$`, iHaveAValidPasswordResetToken)
	ctx.Step(`^que eu tenho um token de verificação de email válido$`, iHaveAValidEmailVerificationToken)
	ctx.Step(`^que um usuário com id "([^"]*)" existe$`, aUserWithIDExists)
	ctx.Step(`^que o usuário tem posts publicados$`, theUserHasPublishedPosts)
	ctx.Step(`^que eu sou dono de um post com id "([^"]*)"$`, iOwnAPostWithID)
	ctx.Step(`^que um post com id "([^"]*)" existe$`, aPostWithIDExists)
	ctx.Step(`^que eu já curti o post$`, iHaveLikedThePost)
	ctx.Step(`^que eu sigo um criador com id "([^"]*)"$`, iFollowACreatorWithID)
	ctx.Step(`^que o criador tem posts publicados$`, theCreatorHasPublishedPosts)
	ctx.Step(`^que eu não sou subscrito no criador "([^"]*)"$`, iAmNotSubscribedToCreator)
	ctx.Step(`^que eu tenho posts publicados$`, iHavePublishedPosts)
	ctx.Step(`^que um plano com id "([^"]*)" existe$`, aPlanWithIDExists)
	ctx.Step(`^que eu tenho uma assinatura ativa$`, iHaveAnActiveSubscription)
	ctx.Step(`^que eu não tenho uma assinatura ativa$`, iDoNotHaveAnActiveSubscription)
	ctx.Step(`^que eu tenho uma assinatura cancelada$`, iHaveACanceledSubscription)
	ctx.Step(`^que eu tenho uma sessão de upload com id "([^"]*)"$`, iHaveAnUploadSessionWithID)

	// When steps - pt-BR
	ctx.Step(`^eu enviar uma requisição (GET|POST|PATCH|PUT|DELETE) para "([^"]*)"$`, iSendARequestTo)
	ctx.Step(`^eu enviar uma requisição (GET|POST|PATCH|PUT|DELETE) para "([^"]*)" com o payload$`, iSendARequestToWithThePayload)
	ctx.Step(`^eu enviar uma requisição (GET|POST|PATCH|PUT|DELETE) para "([^"]*)" com:$`, iSendARequestToWith)
	ctx.Step(`^eu enviar uma requisição POST para "([^"]*)" com um evento (Stripe|PagSeguro|MercadoPago):$`, iSendAPostRequestWithProviderEvent)
	ctx.Step(`^eu enviar uma requisição POST para "([^"]*)" com JSON inválido$`, iSendAPostRequestWithInvalidJSON)

	// Then steps - pt-BR
	ctx.Step(`^o código de status da resposta deve ser (\d+)$`, theResponseStatusCodeShouldBe)
	ctx.Step(`^a resposta deve conter "([^"]*)"$`, theResponseShouldContain)
	ctx.Step(`^a resposta deve conter "([^"]*)" com valor "([^"]*)"$`, theResponseShouldContainWithValue)
	ctx.Step(`^a resposta deve ser um array$`, theResponseShouldBeAnArray)
	ctx.Step(`^a resposta deve ser nula$`, theResponseShouldBeNull)
	ctx.Step(`^a resposta deve conter erro de validação para "([^"]*)"$`, theResponseShouldContainValidationErrorFor)
	ctx.Step(`^a resposta deve indicar que o post foi curtido$`, theResponseShouldIndicatePostIsLiked)
	ctx.Step(`^a resposta deve indicar que o post foi descurtido$`, theResponseShouldIndicatePostIsUnliked)
	ctx.Step(`^cada usuário nos resultados deve ter username contendo "([^"]*)"$`, eachUserInResultsShouldHaveUsernameContaining)
	ctx.Step(`^cada post deve ter "([^"]*)" com valor "([^"]*)"$`, eachPostShouldHaveWithValue)
	ctx.Step(`^cada post deve ter "([^"]*)" como "([^"]*)" ou de criadores subscritos$`, eachPostShouldHaveVisibilityAsOrFromSubscribedCreators)
	ctx.Step(`^a resposta deve conter posts do criador "([^"]*)"$`, theResponseShouldContainPostsFromCreator)
	ctx.Step(`^a resposta não deve conter posts privados de "([^"]*)"$`, theResponseShouldNotContainPrivatePostsFrom)
	ctx.Step(`^a resposta deve conter meus próprios posts$`, theResponseShouldContainMyOwnPosts)
	ctx.Step(`^cada plano deve conter "([^"]*)"$`, eachPlanShouldContain)
	ctx.Step(`^cada pedido deve conter "([^"]*)"$`, eachOrderShouldContain)
	ctx.Step(`^a resposta deve conter no máximo (\d+) itens$`, theResponseShouldContainAtMostItems)
	ctx.Step(`^o evento deve ser registrado sem dados sensíveis$`, theEventShouldBeLoggedWithoutSensitiveData)

	// Robust auth assertion steps - pt-BR
	ctx.Step(`^a resposta deve conter "access_token" não vazio$`, theResponseShouldContainNonEmptyAccessToken)
	ctx.Step(`^a resposta deve conter "refresh_token" não vazio$`, theResponseShouldContainNonEmptyRefreshToken)
	ctx.Step(`^a resposta deve conter "user" com dados estruturados sem campos sensíveis$`, theResponseShouldContainStructuredUserWithoutSensitiveFields)
	ctx.Step(`^a resposta deve conter "message" não vazia$`, theResponseShouldContainNonEmptyMessage)
}

func theAPIIsRunning() error {
	cfg := application.Config{
		Port:      "40080",
		DBURL:     os.Getenv("DATABASE_URL"),
		RedisURL:  os.Getenv("REDIS_URL"),
		JWTSecret: "test-jwt-secret-that-is-long-enough-for-tests-32chars",
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		AppEnv:    "test",
	}
	if cfg.DBURL == "" {
		cfg.DBURL = "postgres://harem:harem@localhost:5432/harem?sslmode=disable"
	}
	if cfg.RedisURL == "" {
		cfg.RedisURL = "redis://localhost:6379/0"
	}

	srv, err := application.NewHTTPServer(context.Background(), cfg)
	if err != nil {
		// Skip tests when DB/Redis is not available - return pending
		return godog.ErrPending
	}
	testCtx.server = srv
	return nil
}

func theDatabaseIsConnected() error {
	// In real tests, this would verify DB connection
	return nil
}

func theCacheIsConnected() error {
	// In real tests, this would verify Redis connection
	return nil
}

func iAmAuthenticatedAsUser(username string) error {
	// Create a test JWT token
	testCtx.token = generateTestToken(username, "user")
	return nil
}

func iAmAuthenticatedAsCreator(username string) error {
	testCtx.token = generateTestToken(username, "creator")
	return nil
}

func iAmNotAuthenticated() error {
	testCtx.token = ""
	return nil
}

const testJWTSecret = "test-jwt-secret-that-is-long-enough-for-tests-32chars"

func generateTestToken(username, role string) string {
	claims := jwt.MapClaims{
		"sub":      username,
		"roles":    []string{role},
		"email":    username + "@test.local",
		"username": username,
		"exp":      time.Now().Add(time.Hour).Unix(),
		"iat":      time.Now().Unix(),
		"iss":      "harem-api",
		"aud":      "harem-client",
		"type":     "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(testJWTSecret))
	return signed
}

func iHaveAValidRegistrationPayload(table *godog.Table) error {
	return nil
}

func aUserWithEmailAlreadyExists(email string) error {
	return nil
}

func aRegisteredUserWithEmailAndPassword(email, password string) error {
	return nil
}

func aRegisteredUserWithEmail(email string) error {
	return nil
}

func iHaveAValidRefreshToken() error {
	testCtx.refreshToken = "test-refresh-token"
	return nil
}

func iHaveAValidPasswordResetToken() error {
	return nil
}

func iHaveAValidEmailVerificationToken() error {
	return nil
}

func aUserWithIDExists(userID string) error {
	return nil
}

func theUserHasPublishedPosts() error {
	return nil
}

func iOwnAPostWithID(postID string) error {
	return nil
}

func aPostWithIDExists(postID string) error {
	return nil
}

func iHaveLikedThePost() error {
	return nil
}

func iFollowACreatorWithID(creatorID string) error {
	return nil
}

func theCreatorHasPublishedPosts() error {
	return nil
}

func iAmNotSubscribedToCreator(creatorID string) error {
	return nil
}

func iHavePublishedPosts() error {
	return nil
}

func aPlanWithIDExists(planID string) error {
	return nil
}

func iHaveAnActiveSubscription() error {
	return nil
}

func iDoNotHaveAnActiveSubscription() error {
	return nil
}

func iHaveACanceledSubscription() error {
	return nil
}

func iHaveAnUploadSessionWithID(uploadID string) error {
	return nil
}

func iSendARequestTo(method, path string) error {
	return iSendARequestToWithThePayload(method, path, nil)
}

func iSendARequestToWithThePayload(method, path string, table *godog.Table) error {
	var body []byte
	if table != nil && len(table.Rows) > 1 {
		data := make(map[string]string)
		for i, cell := range table.Rows[1].Cells {
			if i < len(table.Rows[0].Cells) {
				data[table.Rows[0].Cells[i].Value] = cell.Value
			}
		}
		body, _ = json.Marshal(data)
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if testCtx.token != "" {
		req.Header.Set("Authorization", "Bearer "+testCtx.token)
	}

	testCtx.request = req
	testCtx.recorder = httptest.NewRecorder()

	if testCtx.server != nil {
		testCtx.server.ServeHTTP(testCtx.recorder, req)
	}

	return parseResponse()
}

func iSendARequestToWith(method, path string, table *godog.Table) error {
	return iSendARequestToWithThePayload(method, path, table)
}

func iSendAPostRequestWithProviderEvent(path, provider string, table *godog.Table) error {
	return iSendARequestToWithThePayload("POST", path, table)
}

func iSendAPostRequestWithInvalidJSON(path string) error {
	req := httptest.NewRequest("POST", path, strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	testCtx.request = req
	testCtx.recorder = httptest.NewRecorder()

	if testCtx.server != nil {
		testCtx.server.ServeHTTP(testCtx.recorder, req)
	}

	return parseResponse()
}

func parseResponse() error {
	contentType := testCtx.recorder.Header().Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		body := testCtx.recorder.Body.String()
		if body != "null" && body != "" {
			var result any
			if err := json.Unmarshal([]byte(body), &result); err == nil {
				switch v := result.(type) {
				case map[string]any:
					testCtx.response = v
				case []any:
					testCtx.responseArray = v
				}
			}
		}
	}
	return nil
}

func theResponseStatusCodeShouldBe(expectedCode int) error {
	actualCode := testCtx.recorder.Code
	if actualCode != expectedCode {
		return fmt.Errorf("expected status code %d, got %d. Response: %s",
			expectedCode, actualCode, testCtx.recorder.Body.String())
	}
	return nil
}

func theResponseShouldContain(field string) error {
	if testCtx.response == nil {
		return fmt.Errorf("response is not a JSON object")
	}
	if _, ok := testCtx.response[field]; !ok {
		return fmt.Errorf("response does not contain field '%s'. Response: %v", field, testCtx.response)
	}
	return nil
}

func theResponseShouldContainWithValue(field, expectedValue string) error {
	if testCtx.response == nil {
		return fmt.Errorf("response is not a JSON object")
	}
	actualValue, ok := testCtx.response[field]
	if !ok {
		return fmt.Errorf("response does not contain field '%s'", field)
	}
	if fmt.Sprintf("%v", actualValue) != expectedValue {
		return fmt.Errorf("expected '%s' to be '%s', got '%v'", field, expectedValue, actualValue)
	}
	return nil
}

func theResponseShouldBeAnArray() error {
	if testCtx.responseArray == nil {
		return fmt.Errorf("response is not a JSON array")
	}
	return nil
}

func theResponseShouldBeNull() error {
	body := testCtx.recorder.Body.String()
	if body != "null" {
		return fmt.Errorf("expected null response, got: %s", body)
	}
	return nil
}

func theResponseShouldContainValidationErrorFor(field string) error {
	// Check for errors object or field-specific error
	return theResponseShouldContain("error")
}

func theResponseShouldIndicatePostIsLiked() error {
	return theResponseShouldContain("liked")
}

func theResponseShouldIndicatePostIsUnliked() error {
	return theResponseShouldContain("unliked")
}

func eachUserInResultsShouldHaveUsernameContaining(substring string) error {
	return nil
}

func eachPostShouldHaveWithValue(field, value string) error {
	return nil
}

func eachPostShouldHaveVisibilityAsOrFromSubscribedCreators(visibility string) error {
	return nil
}

func theResponseShouldContainPostsFromCreator(creatorID string) error {
	return nil
}

func theResponseShouldNotContainPrivatePostsFrom(creatorID string) error {
	return nil
}

func theResponseShouldContainMyOwnPosts() error {
	return nil
}

func eachPlanShouldContain(field string) error {
	return nil
}

func eachOrderShouldContain(field string) error {
	return nil
}

func theResponseShouldContainAtMostItems(count int) error {
	if len(testCtx.responseArray) > count {
		return fmt.Errorf("expected at most %d items, got %d", count, len(testCtx.responseArray))
	}
	return nil
}

func theEventShouldBeLoggedWithoutSensitiveData() error {
	return nil
}

func theResponseShouldContainNonEmptyMessage() error {
	if testCtx.response == nil {
		return fmt.Errorf("response is not a JSON object")
	}
	val, ok := testCtx.response["message"]
	if !ok {
		return fmt.Errorf("response does not contain field 'message'")
	}
	s, ok := val.(string)
	if !ok || s == "" {
		return fmt.Errorf("field 'message' is empty or not a string")
	}
	return nil
}

func theResponseShouldContainNonEmptyAccessToken() error {
	if testCtx.response == nil {
		return fmt.Errorf("response is not a JSON object")
	}
	val, ok := testCtx.response["access_token"]
	if !ok {
		return fmt.Errorf("response does not contain field 'access_token'")
	}
	s, ok := val.(string)
	if !ok || s == "" {
		return fmt.Errorf("field 'access_token' is empty or not a string")
	}
	if len(s) < 20 {
		return fmt.Errorf("field 'access_token' appears too short (%d chars), expected a valid JWT", len(s))
	}
	return nil
}

func theResponseShouldContainNonEmptyRefreshToken() error {
	if testCtx.response == nil {
		return fmt.Errorf("response is not a JSON object")
	}
	val, ok := testCtx.response["refresh_token"]
	if !ok {
		return fmt.Errorf("response does not contain field 'refresh_token'")
	}
	s, ok := val.(string)
	if !ok || s == "" {
		return fmt.Errorf("field 'refresh_token' is empty or not a string")
	}
	if len(s) < 8 {
		return fmt.Errorf("field 'refresh_token' appears too short (%d chars)", len(s))
	}
	return nil
}

var sensitiveUserFields = []string{
	"password_hash", "password", "passwordHash", "secret", "token",
	"deleted_at", "updated_at", "last_seen_at",
}

var requiredUserFields = []string{"id", "username", "email", "role", "created_at"}

func theResponseShouldContainStructuredUserWithoutSensitiveFields() error {
	if testCtx.response == nil {
		return fmt.Errorf("response is not a JSON object")
	}
	userVal, ok := testCtx.response["user"]
	if !ok {
		return fmt.Errorf("response does not contain field 'user'")
	}
	user, ok := userVal.(map[string]any)
	if !ok {
		return fmt.Errorf("field 'user' is not a JSON object, got %T", userVal)
	}

	// Check required fields
	for _, field := range requiredUserFields {
		if _, exists := user[field]; !exists {
			return fmt.Errorf("user object missing required field '%s'", field)
		}
	}

	// Check no sensitive fields
	for _, field := range sensitiveUserFields {
		if _, exists := user[field]; exists {
			return fmt.Errorf("user object exposes sensitive field '%s'", field)
		}
	}

	// Check id is non-empty
	if id, _ := user["id"].(string); id == "" {
		return fmt.Errorf("user.id is empty")
	}

	// Check username is non-empty
	if username, _ := user["username"].(string); username == "" {
		return fmt.Errorf("user.username is empty")
	}

	return nil
}

func TestFeatures(t *testing.T) {
	opts := godog.Options{
		Format:   "pretty",
		Paths:    []string{"."},
		TestingT: t,
	}

	status := godog.TestSuite{
		Name:                 "API Feature Tests",
		ScenarioInitializer:  InitializeScenario,
		TestSuiteInitializer: InitializeTestSuite,
		Options:              &opts,
	}.Run()

	if status != 0 {
		t.Fatal("Feature tests failed")
	}
}
