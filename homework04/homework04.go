package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"homework04/util"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"uniqueIndex;not null"`
	Password  string    `json:"-" gorm:"not null"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Post struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Title     string    `json:"title" gorm:"not null"`
	Content   string    `json:"content" gorm:"not null"`
	UserID    uint      `json:"user_id" gorm:"not null;index"`
	User      User      `json:"-" gorm:"constraint:OnDelete:RESTRICT;"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Comment struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Content   string    `json:"content" gorm:"not null"`
	UserID    uint      `json:"user_id" gorm:"not null;index"`
	User      User      `json:"-" gorm:"constraint:OnDelete:RESTRICT;"`
	PostID    uint      `json:"post_id" gorm:"not null;index"`
	Post      Post      `json:"-" gorm:"constraint:OnDelete:CASCADE;"`
	CreatedAt time.Time `json:"created_at"`
}

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

type application struct {
	db     *gorm.DB
	secret []byte
}

// NewRouter 自动迁移模型，并注册公开接口和需要 JWT 认证的接口。
func NewRouter(db *gorm.DB, secret string) (*gin.Engine, error) {
	if len(strings.TrimSpace(secret)) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must contain at least 32 bytes")
	}
	if db == nil {
		return nil, fmt.Errorf("database must not be nil")
	}
	if err := db.AutoMigrate(&User{}, &Post{}, &Comment{}); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	a := &application{db: db, secret: []byte(secret)}
	r := gin.New()
	if err := r.SetTrustedProxies(nil); err != nil {
		return nil, fmt.Errorf("configure trusted proxies: %w", err)
	}
	r.HandleMethodNotAllowed = true
	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.Info("http request", "method", c.Request.Method, "path", c.Request.URL.Path,
			"status", c.Writer.Status(), "duration", time.Since(start))
	})
	r.Use(func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("request panic", "error", recovered)
				respondError(c, http.StatusInternalServerError, "internal server error")
			}
		}()
		c.Next()
	})
	r.NoRoute(func(c *gin.Context) { respondError(c, http.StatusNotFound, "route not found") })
	r.NoMethod(func(c *gin.Context) { respondError(c, http.StatusMethodNotAllowed, "method not allowed") })
	r.POST("/register", a.register)
	r.POST("/login", a.login)
	r.GET("/posts", a.listPosts)
	r.GET("/posts/:id", a.getPost)
	r.GET("/posts/:id/comments", a.listComments)
	protected := r.Group("/", a.authenticate)
	protected.POST("/posts", a.createPost)
	protected.PUT("/posts/:id", a.updatePost)
	protected.DELETE("/posts/:id", a.deletePost)
	protected.POST("/posts/:id/comments", a.createComment)
	return r, nil
}

func respondError(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}

func databaseError(c *gin.Context, err error) {
	slog.Error("database operation failed", "method", c.Request.Method, "path", c.Request.URL.Path, "error", err)
	respondError(c, http.StatusInternalServerError, "internal server error")
}

func (a *application) register(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required,max=50"`
		Password string `json:"password" binding:"required,min=8"`
		Email    string `json:"email" binding:"required,email,max=254"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest, "username, valid email and password (at least 8 characters) are required")
		return
	}
	input.Username = strings.TrimSpace(input.Username)
	if input.Username == "" || strings.TrimSpace(input.Password) == "" || len(input.Password) > 72 {
		respondError(c, http.StatusBadRequest, "username and password must not be blank; password must be at most 72 bytes")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("hash password", "error", err)
		respondError(c, http.StatusInternalServerError, "internal server error")
		return
	}
	user := User{Username: input.Username, Password: string(hash), Email: strings.ToLower(input.Email)}
	if err := a.db.WithContext(c.Request.Context()).Create(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			respondError(c, http.StatusConflict, "username or email already exists")
		} else {
			databaseError(c, err)
		}
		return
	}
	c.JSON(http.StatusCreated, user)
}

func (a *application) login(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Username) == "" {
		respondError(c, http.StatusBadRequest, "username and password are required")
		return
	}
	var user User
	err := a.db.WithContext(c.Request.Context()).Where("username = ?", strings.TrimSpace(input.Username)).First(&user).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		databaseError(c, err)
		return
	}
	if err != nil || len(input.Password) > 72 || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)) != nil {
		respondError(c, http.StatusUnauthorized, "invalid username or password")
		return
	}
	now := time.Now()
	expires := now.Add(24 * time.Hour)
	claims := Claims{UserID: user.ID, RegisteredClaims: jwt.RegisteredClaims{
		Issuer: "homework04", Subject: strconv.FormatUint(uint64(user.ID), 10),
		IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expires),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.secret)
	if err != nil {
		slog.Error("sign token", "error", err)
		respondError(c, http.StatusInternalServerError, "internal server error")
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "token_type": "Bearer", "expires_at": expires})
}

func (a *application) authenticate(c *gin.Context) {
	parts := strings.Fields(c.GetHeader("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		respondError(c, http.StatusUnauthorized, "Authorization: Bearer <token> is required")
		return
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (any, error) {
		return a.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired(), jwt.WithIssuer("homework04"), jwt.WithIssuedAt())
	if err != nil || !token.Valid || claims.UserID == 0 {
		respondError(c, http.StatusUnauthorized, "invalid or expired token")
		return
	}
	var user User
	if err := a.db.WithContext(c.Request.Context()).Select("id").First(&user, claims.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusUnauthorized, "user no longer exists")
		} else {
			databaseError(c, err)
		}
		return
	}
	c.Set("user_id", user.ID)
	c.Next()
}

type postInput struct {
	Title   string `json:"title" binding:"required,max=200"`
	Content string `json:"content" binding:"required"`
}

func bindPost(c *gin.Context) (postInput, bool) {
	var input postInput
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Content) == "" {
		respondError(c, http.StatusBadRequest, "non-blank title (at most 200 characters) and content are required")
		return input, false
	}
	input.Title = strings.TrimSpace(input.Title)
	return input, true
}

func (a *application) createPost(c *gin.Context) {
	input, ok := bindPost(c)
	if !ok {
		return
	}
	post := Post{Title: input.Title, Content: input.Content, UserID: c.GetUint("user_id")}
	if err := a.db.WithContext(c.Request.Context()).Create(&post).Error; err != nil {
		databaseError(c, err)
		return
	}
	c.JSON(http.StatusCreated, post)
}

func (a *application) listPosts(c *gin.Context) {
	posts := make([]Post, 0)
	if err := a.db.WithContext(c.Request.Context()).Order("created_at DESC, id DESC").Find(&posts).Error; err != nil {
		databaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, posts)
}

func (a *application) findPost(c *gin.Context) (Post, bool) {
	var post Post
	id, err := strconv.ParseUint(c.Param("id"), 10, 63)
	if err != nil || id == 0 {
		respondError(c, http.StatusBadRequest, "post id must be a positive integer")
		return post, false
	}
	if err := a.db.WithContext(c.Request.Context()).First(&post, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "post not found")
		} else {
			databaseError(c, err)
		}
		return post, false
	}
	return post, true
}

func (a *application) getPost(c *gin.Context) {
	if post, ok := a.findPost(c); ok {
		c.JSON(http.StatusOK, post)
	}
}

func (a *application) ownedPost(c *gin.Context) (Post, bool) {
	post, ok := a.findPost(c)
	if !ok {
		return post, false
	}
	if post.UserID != c.GetUint("user_id") {
		respondError(c, http.StatusForbidden, "only the author can modify this post")
		return post, false
	}
	return post, true
}

func (a *application) updatePost(c *gin.Context) {
	post, ok := a.ownedPost(c)
	if !ok {
		return
	}
	input, ok := bindPost(c)
	if !ok {
		return
	}
	result := a.db.WithContext(c.Request.Context()).Model(&post).Where("user_id = ?", c.GetUint("user_id")).Updates(map[string]any{
		"title": input.Title, "content": input.Content,
	})
	if result.Error != nil {
		databaseError(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		respondError(c, http.StatusNotFound, "post not found")
		return
	}
	c.JSON(http.StatusOK, post)
}

func (a *application) deletePost(c *gin.Context) {
	post, ok := a.ownedPost(c)
	if !ok {
		return
	}
	// 不使用软删除，SQLite 的 ON DELETE CASCADE 会原子地删除关联评论。
	result := a.db.WithContext(c.Request.Context()).Where("user_id = ?", c.GetUint("user_id")).Delete(&post)
	if result.Error != nil {
		databaseError(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		respondError(c, http.StatusNotFound, "post not found")
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *application) createComment(c *gin.Context) {
	post, ok := a.findPost(c)
	if !ok {
		return
	}
	var input struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Content) == "" {
		respondError(c, http.StatusBadRequest, "non-blank comment content is required")
		return
	}
	comment := Comment{Content: input.Content, UserID: c.GetUint("user_id"), PostID: post.ID}
	if err := a.db.WithContext(c.Request.Context()).Create(&comment).Error; err != nil {
		if errors.Is(err, gorm.ErrForeignKeyViolated) {
			respondError(c, http.StatusNotFound, "post or user no longer exists")
		} else {
			databaseError(c, err)
		}
		return
	}
	c.JSON(http.StatusCreated, comment)
}

func (a *application) listComments(c *gin.Context) {
	post, ok := a.findPost(c)
	if !ok {
		return
	}
	comments := make([]Comment, 0)
	if err := a.db.WithContext(c.Request.Context()).Where("post_id = ?", post.ID).Order("created_at ASC, id ASC").Find(&comments).Error; err != nil {
		databaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, comments)
}

func run() error {
	secret := os.Getenv("JWT_SECRET")
	if len(strings.TrimSpace(secret)) < 32 {
		return fmt.Errorf("set JWT_SECRET to a random secret of at least 32 bytes")
	}
	db, err := util.OpenDB(util.DefaultDBPath)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	router, err := NewRouter(db, secret)
	if err != nil {
		return err
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	server := &http.Server{
		Addr: addr, Handler: router, ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second,
	}
	slog.Info("blog server starting", "address", addr, "database", util.DefaultDBPath)
	return server.ListenAndServe()
}

func main() {
	if err := run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("blog server stopped", "error", err)
		os.Exit(1)
	}
}
