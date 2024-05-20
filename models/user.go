package models

import "time"

type User struct {
	Id           uint   `gorm:"primaryKey"`
	UserName     string `gorm:"type:varchar(50);unique_index"`
	HashPassword string `gorm:"type:varchar(255)"`
	CreatedAt    time.Time
}

