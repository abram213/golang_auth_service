package storage

import (
	"auth_service/app"
	"auth_service/config"
	"auth_service/storage/postgres"
	"fmt"
)

type Storage interface {
	CreateUser(user *app.User) error
	GetUserByLogin(login string) (*app.User, error)
	GetUserByID(id uint) (*app.User, error)

	Close() error
}

func New(conf config.Config) (Storage, error) {
	uri := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		conf.Db.User, conf.Db.Pass, conf.Db.Host, conf.Db.Port, conf.Db.Name)

	return postgres.New(uri)
}
