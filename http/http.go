package http

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func (s *Server) registerRoutes() {
	r := s.router
	r.GET("/welcome", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "welcome gopher",
		})
	})

	r.POST("/login", s.HandleLogin)
	r.POST("/register", s.HandleRegister)
	r.GET("/profile", s.auth(), s.HandleProfile)
}

type customJwtClaims struct {
	UserID   int
	UserName string
	jwt.StandardClaims
}

var ErrInvalidToken = errors.New("invalid token")

func (s *Server) genToken(id int, name string) (string, error) {
	c := customJwtClaims{
		id,
		name,
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Duration(s.config.JWT.TTL) * time.Minute).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString([]byte(s.config.JWT.Secret))
}

func (s *Server) parseToken(tokenString string) (*customJwtClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &customJwtClaims{}, func(token *jwt.Token) (i interface{}, err error) {
		return []byte(s.config.JWT.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*customJwtClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, ErrInvalidToken
}

func (s *Server) respondWithServerErr(c *gin.Context, err error, showErr bool) {
	msg := "server error"
	if showErr {
		msg = msg + " : " + err.Error()
	}
	c.JSON(http.StatusInternalServerError, gin.H{
		"err_msg": msg,
	})
}

func (s *Server) respondWithAuthErr(c *gin.Context, err error) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"err_msg": err.Error(),
	})
}

func (s *Server) respondWithErr(c *gin.Context, err error) {
	code := http.StatusBadRequest
	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		c.JSON(code, gin.H{
			"err_msg": err.Error(),
		})
		return
	}
	code = http.StatusUnprocessableEntity
	c.JSON(code, gin.H{
		"err_msg": removeTopStruct(errs.Translate(s.translator)),
	})
}

func removeTopStruct(fields map[string]string) map[string]string {
	res := map[string]string{}
	for field, err := range fields {
		res[field[strings.Index(field, ".")+1:]] = err
	}
	return res
}

func md5Str(str string) string {
	w := md5.New()
	io.WriteString(w, str)
	return fmt.Sprintf("%x", w.Sum(nil))
}
