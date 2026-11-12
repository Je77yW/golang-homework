package main

import (
	"fmt"
	"log"

	"homework03/util"

	"gorm.io/gorm"
)

const (
	noComments  = "无评论"
	hasComments = "有评论"
)

type User struct {
	gorm.Model
	Name      string
	PostCount int
	Posts     []Post
}

type Post struct {
	gorm.Model
	UserID        uint `gorm:"not null;index"`
	Title         string
	Content       string
	CommentStatus string `gorm:"not null;default:无评论"`
	Comments      []Comment
}

type Comment struct {
	gorm.Model
	PostID  uint `gorm:"not null;index"`
	Content string
}

func (p *Post) AfterCreate(tx *gorm.DB) error {
	return tx.Model(&User{}).Where("id = ?", p.UserID).
		UpdateColumn("post_count", gorm.Expr("post_count + ?", 1)).Error
}

func (c *Comment) AfterCreate(tx *gorm.DB) error {
	return tx.Model(&Post{}).Where("id = ?", c.PostID).
		Update("comment_status", hasComments).Error
}

// AfterDelete 使用已加载的评论实例定位所属文章。
func (c *Comment) AfterDelete(tx *gorm.DB) error {
	if c.PostID == 0 {
		return fmt.Errorf("删除评论时需要提供已加载的评论实例")
	}

	var count int64
	if err := tx.Model(&Comment{}).Where("post_id = ?", c.PostID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return tx.Model(&Post{}).Where("id = ?", c.PostID).
			Update("comment_status", noComments).Error
	}
	return nil
}

func findUserPosts(db *gorm.DB, userID uint) ([]Post, error) {
	var posts []Post
	err := db.Preload("Comments").Where("user_id = ?", userID).Order("id").Find(&posts).Error
	return posts, err
}

func findMostCommentedPost(db *gorm.DB) (Post, error) {
	var post Post
	// 子查询使用 Comment 模型，自动排除软删除的评论。
	counts := db.Model(&Comment{}).Select("post_id, COUNT(*) AS comment_count").Group("post_id")
	err := db.Preload("Comments").
		Joins("LEFT JOIN (?) AS counts ON counts.post_id = posts.id", counts).
		Order("COALESCE(counts.comment_count, 0) DESC").
		Order("posts.id ASC").
		First(&post).Error
	return post, err
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	db, err := util.OpenSQLite()
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	if err := db.AutoMigrate(&User{}, &Post{}, &Comment{}); err != nil {
		return err
	}

	user := User{Name: "小明"}
	if err := db.Create(&user).Error; err != nil {
		return err
	}
	posts := []Post{
		{UserID: user.ID, Title: "GORM 模型"},
		{UserID: user.ID, Title: "GORM 关联查询"},
	}
	if err := db.Create(&posts).Error; err != nil {
		return err
	}
	comments := []Comment{
		{PostID: posts[1].ID, Content: "学到了！"},
		{PostID: posts[1].ID, Content: "期待下一篇。"},
	}
	if err := db.Create(&comments).Error; err != nil {
		return err
	}

	if err := db.First(&user, user.ID).Error; err != nil {
		return err
	}
	fmt.Printf("%s共发布 %d 篇文章\n", user.Name, user.PostCount)

	userPosts, err := findUserPosts(db, user.ID)
	if err != nil {
		return err
	}
	for _, post := range userPosts {
		fmt.Printf("《%s》有 %d 条评论\n", post.Title, len(post.Comments))
		for _, comment := range post.Comments {
			fmt.Printf("  - %s\n", comment.Content)
		}
	}

	topPost, err := findMostCommentedPost(db)
	if err != nil {
		return err
	}
	fmt.Printf("评论最多的文章是《%s》，共 %d 条评论\n", topPost.Title, len(topPost.Comments))

	for i := range topPost.Comments {
		if err := db.Delete(&topPost.Comments[i]).Error; err != nil {
			return err
		}
		var updated Post
		if err := db.First(&updated, topPost.ID).Error; err != nil {
			return err
		}
		fmt.Printf("删除第 %d 条评论后，文章状态为「%s」\n", i+1, updated.CommentStatus)
	}
	return nil
}
