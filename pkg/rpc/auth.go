package rpc

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"workout/pkg/db"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vmkteam/zenrpc/v2"
)

type AuthService struct {
	zenrpc.Service
	ur       db.UserRepo
	botToken string
}

func NewAuthService(dbo db.DB, botToken string) *AuthService {
	return &AuthService{ur: db.NewUserRepo(dbo.DB), botToken: botToken}
}

// TelegramLogin аутентифицирует пользователя через Telegram Login Widget.
//
//zenrpc:return JWT-токен для дальнейших RPC-запросов
//zenrpc:400 Невалидные или устаревшие данные Telegram
//zenrpc:500 Внутренняя ошибка
func (s AuthService) TelegramLogin(ctx context.Context,
	tgID int64, firstName, lastName, username, photoURL string,
	authDate int64, hash string) (*string, error) {

	// Проверяем свежесть данных (не старше 24 часов)
	if time.Since(time.Unix(authDate, 0)) > 24*time.Hour {
		return nil, zenrpc.NewStringError(http.StatusBadRequest, "данные Telegram устарели")
	}

	// Верифицируем подпись
	if !s.verifyTelegramHash(tgID, firstName, lastName, username, photoURL, authDate, hash) {
		return nil, zenrpc.NewStringError(http.StatusBadRequest, "неверная подпись Telegram")
	}

	// Ищем или создаём SiteUser по tgId
	tgIDInt := int(tgID)
	su, err := s.ur.OneSiteUser(ctx, &db.SiteUserSearch{TgID: &tgIDInt})
	if err != nil {
		return nil, zenrpc.NewStringError(http.StatusInternalServerError, err.Error())
	}
	if su == nil {
		enabled := db.StatusEnabled
		su, err = s.ur.AddSiteUser(ctx, &db.SiteUser{TgID: tgIDInt, StatusID: enabled})
		if err != nil || su == nil {
			return nil, zenrpc.NewStringError(http.StatusInternalServerError, "не удалось создать пользователя")
		}
	}

	// Генерируем JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": su.ID,
		"iat": time.Now().Unix(),
	})
	tokenStr, err := token.SignedString([]byte(s.botToken))
	if err != nil {
		return nil, zenrpc.NewStringError(http.StatusInternalServerError, "ошибка генерации токена")
	}

	// Сохраняем JWT в apiKey
	su.ApiKey = &tokenStr
	if _, err := s.ur.UpdateSiteUser(ctx, su, db.WithColumns(db.Columns.SiteUser.ApiKey)); err != nil {
		return nil, zenrpc.NewStringError(http.StatusInternalServerError, err.Error())
	}

	return &tokenStr, nil
}

func (s AuthService) verifyTelegramHash(tgID int64, firstName, lastName, username, photoURL string, authDate int64, hash string) bool {
	var pairs []string
	pairs = append(pairs, fmt.Sprintf("auth_date=%d", authDate))
	if firstName != "" {
		pairs = append(pairs, "first_name="+firstName)
	}
	pairs = append(pairs, fmt.Sprintf("id=%d", tgID))
	if lastName != "" {
		pairs = append(pairs, "last_name="+lastName)
	}
	if photoURL != "" {
		pairs = append(pairs, "photo_url="+photoURL)
	}
	if username != "" {
		pairs = append(pairs, "username="+username)
	}

	h := sha256.New()
	h.Write([]byte(s.botToken))
	secretKey := h.Sum(nil)

	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(strings.Join(pairs, "\n")))
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(hash))
}
