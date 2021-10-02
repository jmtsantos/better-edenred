package main

import (
	"os"

	"github.com/shopspring/decimal"

	edenred "better-edenred/edenred"

	log "github.com/sirupsen/logrus"
)

func main() {
	var (
		balance  decimal.Decimal
		username string
		password string
		err      error
	)

	log.Println("[observer][edenred] starting")

	username = os.Getenv("USER")
	password = os.Getenv("PASS")

	if username == "" || password == "" {
		log.Errorln("error: USER or PASS not set")
		return
	}

	clientEden := edenred.New(username, password)

	if err = clientEden.CheckBalance(); err != nil {
		log.WithError(err).Errorln("[observer][edenred] error processing balance")
		return
	}

	log.Println("[observer][edenred] current balance for Edenred Meal card", balance.String())
}
