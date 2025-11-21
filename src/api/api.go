package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Egor-Evsikov/crschback/src/db"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

// * Обьявление структур и констант
type ctxKey string

var (
	secret = []byte("yokosowatashinosoulsociety")
)

const ctxKeyLogin ctxKey = "login"

type HandlerDB struct {
	dtbs *db.Repo
}

func NewHandlerDB(d *db.Repo) *HandlerDB {
	return &HandlerDB{d}
}

// * Создание токена
func createToken(id int, login string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   id,
		"login": login,
		"exp":   time.Now().Add(24 * time.Hour * 31).Unix(),
		"iat":   time.Now().Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret)
}

// * Парсинг токена
func parseToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		// validate signing method
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

// * Обработка токена в аунтефикации на уровне middleware
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, "missing auth header", http.StatusUnauthorized)
			return
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "invalid auth header", http.StatusUnauthorized)
			return
		}
		claims, err := parseToken(parts[1])
		if err != nil {
			http.Error(w, "invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}
		loginVal, ok := claims["login"].(string)
		if !ok {
			http.Error(w, "token missing login", http.StatusUnauthorized)
			return
		}
		// положим login в контекст
		ctx := context.WithValue(r.Context(), ctxKeyLogin, loginVal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// * Работа с jwt, login и uid для других хенлеров
func loginFromCtx(ctx context.Context) (string, bool) {
	l, ok := ctx.Value(ctxKeyLogin).(string)
	return l, ok
}

// * Логин пользователя, обновялем ему jwt
func (h *HandlerDB) UserLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	var c db.User
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	exist, err := h.dtbs.CheckUser(r.Context(), c.Login, c.Password)
	if err != nil {
		http.Error(w, "db err: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !exist {
		http.Error(w, "auth error", http.StatusUnauthorized)
		return
	}

	// Получим id пользователя
	uid, err := h.dtbs.GetUserId(r.Context(), c.Login)
	if err != nil {
		http.Error(w, "db err: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tokenString, err := createToken(uid, c.Login)
	if err != nil {
		http.Error(w, "error generating token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	type tokenResp struct {
		Token string `json:"token"`
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(tokenResp{Token: tokenString})
}

// * Регистрация пользователя, отдаем jwt
func (h *HandlerDB) UserRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	// прочитаем тело для логирования и последующей декодировки
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}
	// логируем (fmt или logger)
	log.Printf("UserRegister body: %s\n", string(bodyBytes))

	// восстановим r.Body для json.Decoder
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	var user db.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid json", http.StatusUnsupportedMediaType)
		return
	}

	id, err := h.dtbs.SaveUser(r.Context(), user.Login, user.Password)
	if err != nil {
		// если SaveUser вернул ошибку, проверим сообщение (в идеале в db.SaveUser вернуть специальный тип ошибки)
		if err.Error() == "пользователь существует" {
			log.Println(err)
			http.Error(w, "user exists", http.StatusConflict)
			return
		}
		log.Println(err)
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if id == 0 {
		http.Error(w, "cannot create user", http.StatusInternalServerError)
		return
	}

	tokenString, err := createToken(id, user.Login)
	if err != nil {
		http.Error(w, "Error generating token ", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
}

// * Возврат всех директорий пользователя
func (h *HandlerDB) GetDirs(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodGet {
		login, ok := loginFromCtx(r.Context())
		if !ok {
			http.Error(w, "user has not been authorized", http.StatusUnauthorized)
			return
		}

		dirs, err := h.dtbs.GetDirectories(r.Context(), login)
		if err != nil {
			http.Error(w, "dirs not found", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dirs)

	}

}

// * Создание директории по пользователю
func (h *HandlerDB) Mkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		login, ok := loginFromCtx(r.Context())
		if !ok {
			http.Error(w, "user has not been authorized", http.StatusUnauthorized)
			return
		}

		var dir db.Dir
		err := json.NewDecoder(r.Body).Decode(&dir)
		if err != nil {
			http.Error(w, "Invalid json", http.StatusUnsupportedMediaType)
			return
		}

		_, err = h.dtbs.MakeDirectory(r.Context(), dir.Name, login)
		if err != nil {
			log.Println(err)
			http.Error(w, "Error making directory ", http.StatusConflict)
			return
		}
		w.WriteHeader(200)
		w.Write([]byte("mkdir success"))

	}
}

// * Удаление директории
func (h *HandlerDB) DeleteDir(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		login, ok := loginFromCtx(r.Context())
		if !ok {
			http.Error(w, "user has not been authorized", http.StatusUnauthorized)
			return
		}

		var dir db.Dir
		err := json.NewDecoder(r.Body).Decode(&dir)
		if err != nil {
			http.Error(w, "Invalid json", http.StatusUnsupportedMediaType)
			return
		}

		err = h.dtbs.DeleteDirectory(r.Context(), dir.Name, login)
		if err != nil {
			http.Error(w, "directory could not been deleted", http.StatusInternalServerError)
		}

		w.WriteHeader(200)
		w.Write([]byte("mkdir successfully deleted"))

	}
}

// * Добавление лекарства
func (h *HandlerDB) AddMedicine(w http.ResponseWriter, r *http.Request) {
	// URL содержит directory id
	idStr := chi.URLParam(r, "id")
	dirID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid directory id", http.StatusBadRequest)
		return
	}

	// авторизация — login из контекста
	requester, ok := loginFromCtx(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// проверка прав: владелец или участник
	dir, err := h.dtbs.GetDirectoryById(r.Context(), dirID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "directory not found", http.StatusNotFound)
			return
		}
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if dir.Owner != requester {
		member, err := h.dtbs.IsUserInDirectory(r.Context(), requester, dirID)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if !member {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	// Декодируем JSON прямо в db.Medicine (без промежуточного req)
	var m db.Medicine
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	// минимальная валидация
	if m.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

	// дата: если пришла пустая (zero), используем now
	if m.Date.IsZero() {
		m.Date = time.Now()
	}
	// cost/amount: если ноль — можно оставить ноль или задать дефолт
	// Записываем в БД через AddMedicineByDirID
	newID, err := h.dtbs.AddMedicine(r.Context(), m.Name, m.Date, m.Cost, m.Amount, dirID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]int{"id": newID})
}

// * Получение лекарства
func (h *HandlerDB) GetMedicine(w http.ResponseWriter, r *http.Request) {
	dirIdStr := chi.URLParam(r, "id")     // directory id
	medIdStr := chi.URLParam(r, "med_id") // medicine id

	dirID, err := strconv.Atoi(dirIdStr)
	if err != nil {
		http.Error(w, "invalid directory id", http.StatusBadRequest)
		return
	}
	medID, err := strconv.Atoi(medIdStr)
	if err != nil {
		http.Error(w, "invalid medicine id", http.StatusBadRequest)
		return
	}

	requester, ok := loginFromCtx(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// проверка: существует ли мед и принадлежит ли он директории
	med, err := h.dtbs.GetMedicineByID(r.Context(), medID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "medicine not found", http.StatusNotFound)
			return
		}
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if med.DirId != dirID {
		http.Error(w, "medicine does not belong to directory", http.StatusBadRequest)
		return
	}

	// проверка прав (владелец директории или member)
	dir, err := h.dtbs.GetDirectoryById(r.Context(), dirID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if dir.Owner != requester {
		member, err := h.dtbs.IsUserInDirectory(r.Context(), requester, dirID)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if !member {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(med)
}

// * Обновление лекарства
func (h *HandlerDB) UpdateMedicine(w http.ResponseWriter, r *http.Request) {
	dirIdStr := chi.URLParam(r, "id")
	medIdStr := chi.URLParam(r, "med_id")

	dirID, err := strconv.Atoi(dirIdStr)
	if err != nil {
		http.Error(w, "invalid directory id", http.StatusBadRequest)
		return
	}
	medID, err := strconv.Atoi(medIdStr)
	if err != nil {
		http.Error(w, "invalid medicine id", http.StatusBadRequest)
		return
	}

	requester, ok := loginFromCtx(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// проверка прав (как в GetMedicine) — пропущена здесь для краткости, нужно выполнить ту же логику:
	// 1) получить med, убедиться что med.DirId == dirID
	// 2) получить dir, проверить requester == dir.Owner || requester in dir_users

	medExisting, err := h.dtbs.GetMedicineByID(r.Context(), medID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "medicine not found", http.StatusNotFound)
			return
		}
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if medExisting.DirId != dirID {
		http.Error(w, "medicine does not belong to directory", http.StatusBadRequest)
		return
	}

	dir, err := h.dtbs.GetDirectoryById(r.Context(), dirID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if dir.Owner != requester {
		member, err := h.dtbs.IsUserInDirectory(r.Context(), requester, dirID)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if !member {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	// декодируем тело напрямую в db.Medicine, но игнорируем DirId и Id из тела
	var m db.Medicine
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if m.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if m.Date.IsZero() {
		m.Date = time.Now()
	}

	// обновляем
	if err := h.dtbs.UpdateMedicine(r.Context(), medID, m.Name, m.Date, m.Cost, m.Amount); err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// * Удаление лекарства
func (h *HandlerDB) DeleteMedicine(w http.ResponseWriter, r *http.Request) {
	dirIdStr := chi.URLParam(r, "id")
	medIdStr := chi.URLParam(r, "med_id")

	dirID, err := strconv.Atoi(dirIdStr)
	if err != nil {
		http.Error(w, "invalid directory id", http.StatusBadRequest)
		return
	}
	medID, err := strconv.Atoi(medIdStr)
	if err != nil {
		http.Error(w, "invalid medicine id", http.StatusBadRequest)
		return
	}

	requester, ok := loginFromCtx(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	med, err := h.dtbs.GetMedicineByID(r.Context(), medID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "medicine not found", http.StatusNotFound)
			return
		}
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if med.DirId != dirID {
		http.Error(w, "medicine does not belong to directory", http.StatusBadRequest)
		return
	}

	dir, err := h.dtbs.GetDirectoryById(r.Context(), dirID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if dir.Owner != requester {
		member, err := h.dtbs.IsUserInDirectory(r.Context(), requester, dirID)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if !member {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	if err := h.dtbs.DeleteMedicine(r.Context(), medID); err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// * Добавление пользователя в директорию
func (h *HandlerDB) AddUserToDir(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1) parse directory id
	idStr := chi.URLParam(r, "id")
	dirID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid directory id", http.StatusBadRequest)
		return
	}

	// 2) get requester login from context
	requester, ok := loginFromCtx(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// 3) ensure directory exists and get owner
	dir, err := h.dtbs.GetDirectoryById(ctx, dirID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "directory not found", http.StatusNotFound)
			return
		}
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 4) only owner can add users
	if dir.Owner != requester {
		http.Error(w, "forbidden: only owner can add users", http.StatusForbidden)
		return
	}

	// 5) decode body into map[string]string (no extra struct)
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	targetLogin := body["login"]
	if targetLogin == "" {
		http.Error(w, "login is required", http.StatusBadRequest)
		return
	}

	// 6) add user to directory via repo
	if err := h.dtbs.AddUserToDirectory(ctx, targetLogin, dirID); err != nil {
		// Возможные ошибки: пользователь не найден, db error
		// Здесь можно различать ошибки по типу/тексту, если хочется
		http.Error(w, "error adding user to directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte("user added"))
}

// * Удаление пользователя из директории
func (h *HandlerDB) RemoveUserFromDir(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	dirID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid directory id", http.StatusBadRequest)
		return
	}

	targetLogin := chi.URLParam(r, "login")
	if targetLogin == "" {
		http.Error(w, "target login is required", http.StatusBadRequest)
		return
	}

	requester, ok := loginFromCtx(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// get directory and owner
	dir, err := h.dtbs.GetDirectoryById(ctx, dirID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "directory not found", http.StatusNotFound)
			return
		}
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// only owner can remove users
	if dir.Owner != requester {
		http.Error(w, "forbidden: only owner can remove users", http.StatusForbidden)
		return
	}

	// prevent removing the owner
	if targetLogin == dir.Owner {
		http.Error(w, "cannot remove owner from directory", http.StatusBadRequest)
		return
	}

	// call repo to remove
	if err := h.dtbs.RemoveUserFromDirectory(ctx, targetLogin, dirID); err != nil {
		// различаем типичные ошибки (строковые сравнения — можно заменить на typed errors)
		switch err.Error() {
		case "user is not a member of directory":
			http.Error(w, "user is not a member", http.StatusNotFound)
			return
		case "directory not found":
			http.Error(w, "directory not found", http.StatusNotFound)
			return
		case "cannot remove owner from directory":
			http.Error(w, "cannot remove owner", http.StatusBadRequest)
			return
		default:
			http.Error(w, "error removing user from directory: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// * Получение всех лекарств
func (h *HandlerDB) GetMeds(w http.ResponseWriter, r *http.Request) {
	// parse dir id from URL
	idStr := chi.URLParam(r, "id")
	dirID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid directory id", http.StatusBadRequest)
		return
	}

	// auth: get requester login from context
	requester, ok := loginFromCtx(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// TODO: check access — owner or member
	// Предполагаем, что в Repo есть метод IsUserInDirectory(ctx, login, dirId) (bool, error)
	// и что GetDirectoryById(ctx, dirId) возвращает Dir с Owner id/login.
	hasAccess := false
	// try owner
	dir, err := h.dtbs.GetDirectoryById(r.Context(), dirID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "directory not found", http.StatusNotFound)
			return
		}
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if dir.Owner == requester {
		hasAccess = true
	} else {
		// если не владелец, проверить membership
		in, err := h.dtbs.IsUserInDirectory(r.Context(), requester, dirID)
		if err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if in {
			hasAccess = true
		}
	}

	if !hasAccess {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// получить лекарства
	meds, err := h.dtbs.GetMedicines(r.Context(), dirID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(meds)
}

// *
func (h *HandlerDB) GetUsersInDir(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1) directory ID из URL
	idStr := chi.URLParam(r, "id")
	dirID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid directory id", http.StatusBadRequest)
		return
	}

	// 2) логин пользователя из JWT
	requester, ok := loginFromCtx(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// 3) проверяем существует ли директория
	dir, err := h.dtbs.GetDirectoryById(ctx, dirID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "directory not found", http.StatusNotFound)
			return
		}
		http.Error(w, "db error: "+err.Error(), 500)
		return
	}

	// 4) Проверка прав: requester = owner или member
	if dir.Owner != requester {
		isInDir, err := h.dtbs.IsUserInDirectory(ctx, requester, dirID)
		if err != nil {
			http.Error(w, "db error: "+err.Error(), 500)
			return
		}
		if !isInDir {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	// 5) Получаем логины всех пользователей директории
	userLogins, err := h.dtbs.GetUsersByDirectory(ctx, dirID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), 500)
		return
	}

	// 6) Отдаём массив логинов
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userLogins)
}

/*

| Код     | Название            | Описание                                                         |
| ------- | ------------------- | ---------------------------------------------------------------- |
| **100** | Continue            | Сервер получил начальную часть запроса, клиент может продолжать. |
| **101** | Switching Protocols | Сервер переключает протокол (например, на WebSocket).            |
| **102** | Processing          | Сервер принял запрос, но обработка ещё не завершена (WebDAV).    |
| **103** | Early Hints         | Сервер отправляет заголовки до основного ответа (для preload).   |


| Код     | Название                      | Описание                                                  |
| ------- | ----------------------------- | --------------------------------------------------------- |
!| **200** | OK                            | Запрос успешно обработан, тело содержит результат.        |
!| **201** | Created                       | Успешное создание ресурса (например, через POST).         |
| **202** | Accepted                      | Запрос принят, но ещё не обработан.                       |
| **203** | Non-Authoritative Information | Ответ получен не от оригинального сервера.                |
| **204** | No Content                    | Успешно, но тело ответа отсутствует.                      |
| **205** | Reset Content                 | Клиенту следует сбросить форму (например, очистить поля). |
| **206** | Partial Content               | Частичный ответ на запрос диапазона (Range).              |
| **207** | Multi-Status                  | Несколько статусов для разных операций (WebDAV).          |
| **208** | Already Reported              | Элемент уже упоминался ранее (WebDAV).                    |
| **226** | IM Used                       | Использовано преобразование содержимого (Delta Encoding). |


| Код     | Название           | Описание                                                         |
| ------- | ------------------ | ---------------------------------------------------------------- |
| **300** | Multiple Choices   | Несколько возможных вариантов ответа.                            |
| **301** | Moved Permanently  | Ресурс окончательно перемещён (постоянный редирект).             |
| **302** | Found              | Ресурс временно перемещён (временный редирект).                  |
| **303** | See Other          | Следует обратиться по другому URL (часто после POST).            |
| **304** | Not Modified       | Контент не изменился с последнего запроса (используется с ETag). |
| **305** | Use Proxy          | Следует использовать прокси (устарел).                           |
| **306** | (Unused)           | Зарезервирован, не используется.                                 |
| **307** | Temporary Redirect | Временный редирект без изменения метода.                         |
| **308** | Permanent Redirect | Постоянный редирект без изменения метода.                        |


| Код     | Название                        | Описание                                                   |
| ------- | ------------------------------- | ---------------------------------------------------------- |
| **400** | Bad Request                     | Неверный запрос (ошибка синтаксиса).                       |
!| **401** | Unauthorized                    | Требуется авторизация.                                     |
| **402** | Payment Required                | Зарезервирован под оплату (редко используется).            |
| **403** | Forbidden                       | Доступ запрещён, даже с авторизацией.                      |
| **404** | Not Found                       | Ресурс не найден.                                          |
| **405** | Method Not Allowed              | Метод (GET, POST и т.д.) не поддерживается.                |
| **406** | Not Acceptable                  | Контент не подходит под условия Accept-заголовков.         |
| **407** | Proxy Authentication Required   | Требуется авторизация на прокси.                           |
| **408** | Request Timeout                 | Таймаут ожидания запроса.                                  |
!| **409** | Conflict                        | Конфликт при обработке (например, при обновлении ресурса). |
| **410** | Gone                            | Ресурс удалён навсегда.                                    |
| **411** | Length Required                 | Не указан заголовок `Content-Length`.                      |
| **412** | Precondition Failed             | Не выполнено условие (If-Match, If-Modified-Since).        |
| **413** | Payload Too Large               | Слишком большой размер тела запроса.                       |
| **414** | URI Too Long                    | Слишком длинный URI.                                       |
!| **415** | Unsupported Media Type          | Неподдерживаемый тип данных.                               |
| **416** | Range Not Satisfiable           | Некорректный диапазон в Range-заголовке.                   |
| **417** | Expectation Failed              | Условие из `Expect` не выполнено.                          |
| **418** | I'm a teapot                    | Пасхалка из RFC 2324 (код шутка).                          |
| **421** | Misdirected Request             | Запрос направлен на неправильный сервер.                   |
| **422** | Unprocessable Entity            | Семантически некорректные данные (часто в API).            |
| **423** | Locked                          | Ресурс заблокирован (WebDAV).                              |
| **424** | Failed Dependency               | Ошибка зависимого запроса (WebDAV).                        |
| **425** | Too Early                       | Сервер не хочет выполнять запрос слишком рано (HTTP/2).    |
| **426** | Upgrade Required                | Требуется переключение протокола (например, HTTPS).        |
| **428** | Precondition Required           | Требуются предусловия (If-Match и т.д.).                   |
| **429** | Too Many Requests               | Превышен лимит запросов (rate limiting).                   |
| **431** | Request Header Fields Too Large | Заголовки запроса слишком велики.                          |
| **451** | Unavailable For Legal Reasons   | Недоступно по юридическим причинам (например, цензура).    |


| Код     | Название                        | Описание                                                    |
| ------- | ------------------------------- | ----------------------------------------------------------- |
!| **500** | Internal Server Error           | Общая ошибка сервера.                                       |
| **501** | Not Implemented                 | Метод не реализован на сервере.                             |
| **502** | Bad Gateway                     | Ошибка прокси или шлюза.                                    |
| **503** | Service Unavailable             | Сервер временно недоступен (перегрузка, техработы).         |
| **504** | Gateway Timeout                 | Таймаут ожидания ответа от другого сервера.                 |
| **505** | HTTP Version Not Supported      | Версия HTTP не поддерживается.                              |
| **506** | Variant Also Negotiates         | Ошибка контентной договорённости.                           |
| **507** | Insufficient Storage            | Недостаточно места для операции (WebDAV).                   |
| **508** | Loop Detected                   | Обнаружена бесконечная петля (WebDAV).                      |
| **510** | Not Extended                    | Требуются дополнительные расширения.                        |
| **511** | Network Authentication Required | Требуется аутентификация в сети (например, captive portal). |


*/
