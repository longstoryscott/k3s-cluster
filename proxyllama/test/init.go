package test

import (
	"fmt"
	"proxyllama/config"
	"proxyllama/storage"
	"proxyllama/util"

	"github.com/sirupsen/logrus"
)

func Init() {
	confFile := "testdata/.config.yaml"
	conf := config.GetConfig(&confFile)

	psqlconn := fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		conf.Database.User,
		conf.Database.Password,
		conf.Database.Host,
		conf.Database.Port,
		conf.Database.DBName,
		conf.Database.SSLMode,
	)

	if err := storage.InitDB(psqlconn); err != nil {
		util.HandleError(err)
	}
	util.LogInfo("Connected to PostgreSQL database", logrus.Fields{
		"connection": psqlconn,
	})
}
