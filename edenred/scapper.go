package edenred

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

const (
	// Debug get debug info
	Debug = false

	// BaseURL Edenred base url
	BaseURL = "https://www.myedenred.pt/edenred-customer/api"
)

// LoginReq login request for edenred
type LoginReq struct {
	UserID   string `json:"userId"`
	Password string `json:"password"`
}

// LoginRes login response for edenred
type LoginRes struct {
	Data struct {
		Token          string `json:"token"`
		OnBoardApplied bool   `json:"onBoardApplied"`
		Customer       struct {
			ID                     int         `json:"id"`
			RegVersion             int         `json:"regVersion"`
			Name                   string      `json:"name"`
			BirthDate              interface{} `json:"birthDate"`
			CellPhoneNumber        string      `json:"cellPhoneNumber"`
			WorkPostalCode         int         `json:"workPostalCode"`
			ResidencePostalCode    int         `json:"residencePostalCode"`
			Gender                 string      `json:"gender"`
			EmailStatus            string      `json:"emailStatus"`
			WorkPlace              interface{} `json:"workPlace"`
			ResidencePlace         interface{} `json:"residencePlace"`
			RegisterStatus         string      `json:"registerStatus"`
			LatCoordinateWork      interface{} `json:"latCoordinateWork"`
			LngCoordinateWork      interface{} `json:"lngCoordinateWork"`
			LatCoordinateResidence interface{} `json:"latCoordinateResidence"`
			LngCoordinateResidence interface{} `json:"lngCoordinateResidence"`
			PasswordStatus         string      `json:"passwordStatus"`
			Email                  string      `json:"email"`
		} `json:"customer"`
	} `json:"data"`
	Message []interface{} `json:"message"`
}

// TxMsg response message containing all transaction data
type TxMsg struct {
	Data struct {
		Account struct {
			Iban                interface{} `json:"iban"`
			CardNumber          string      `json:"cardNumber"`
			AvailableBalance    float64     `json:"availableBalance"`
			CardHolderFirstName string      `json:"cardHolderFirstName"`
			CardHolderLastName  string      `json:"cardHolderLastName"`
			CardActivated       bool        `json:"cardActivated"`
		} `json:"account"`
		MovementList []Movement `json:"movementList"`
	} `json:"data"`
	Message []string `json:"message"`
}

// Movement transaction
type Movement struct {
	TransactionDate string  `json:"transactionDate"`
	TransactionType int     `json:"transactionType"`
	TransactionName string  `json:"transactionName"`
	Amount          float64 `json:"amount"`
	Mcc             string  `json:"mcc"`
	Category        struct {
		ID          int    `json:"id"`
		Description string `json:"description"`
	} `json:"category"`
	CategoryID interface{} `json:"categoryId"`
	Balance    float64     `json:"balance"`
}

// Edenred main
type Edenred struct {
	client   *http.Client
	Username string
	Password string
}

// New returns a new client
func New(u, p string) *Edenred {
	return &Edenred{
		Username: u,
		Password: p,
	}
}

func (e *Edenred) doLogin(client *http.Client) (auth string, err error) {
	var (
		req      *http.Request
		res      *http.Response
		loginRes LoginRes
	)

	loginReq := LoginReq{
		UserID:   e.Username,
		Password: e.Password,
	}

	loginURL := fmt.Sprintf("%s%s", BaseURL, "/authenticate/default?appVersion=1.0&appType=PORTAL&channel=WEB")

	// Decode JSON do not escape HTML
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)

	if err = enc.Encode(&loginReq); err != nil {
		return auth, fmt.Errorf("error marshaling login request: %s", err.Error())
	}

	// Create and submit request
	if req, err = http.NewRequest("POST", loginURL, buf); err != nil {
		return auth, fmt.Errorf("error creating request: %s", err.Error())
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:73.0) Gecko/20100101 Firefox/73.0")
	req.Header.Set("Content-Type", "application/json")

	if res, err = client.Do(req); err != nil {
		return auth, fmt.Errorf("error executing request: %s", err.Error())
	}
	defer res.Body.Close()

	// Parse the response
	if res.StatusCode != 200 {
		return auth, fmt.Errorf("error from API: %v", res.StatusCode)
	}

	if err = json.NewDecoder(res.Body).Decode(&loginRes); err != nil {
		return auth, fmt.Errorf("error decoding json: %s", err.Error())
	}

	auth = loginRes.Data.Token

	return auth, nil
}

func (e *Edenred) getTransactions(client *http.Client, authorization string) (balance decimal.Decimal, err error) {
	var (
		req    *http.Request
		res    *http.Response
		txList TxMsg
	)

	txURL := fmt.Sprintf("%s%s", BaseURL, "/protected/card/537781/accountmovement")

	// Create and submit request
	if req, err = http.NewRequest("GET", txURL, nil); err != nil {
		return decimal.Decimal{}, fmt.Errorf("error creating request: %s", err.Error())
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:73.0) Gecko/20100101 Firefox/73.0")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authorization)

	if res, err = client.Do(req); err != nil {
		return decimal.Decimal{}, fmt.Errorf("error executing request: %s", err.Error())
	}
	defer res.Body.Close()

	if err = json.NewDecoder(res.Body).Decode(&txList); err != nil {
		return decimal.Decimal{}, fmt.Errorf("error decoding json: %s", err.Error())
	}

	for _, m := range txList.Data.MovementList {
		var movementBty []byte
		if movementBty, err = json.Marshal(m); err != nil {
			log.WithError(err).Errorln("error marshalling")
			continue
		}

		fmt.Println(string(movementBty))
	}

	return decimal.NewFromFloat(txList.Data.Account.AvailableBalance), err
}

// CheckBalance checks edenred balance
func (e *Edenred) CheckBalance() (err error) {
	var (
		client        *http.Client
		balance       decimal.Decimal
		authorization string
	)

	client = GetHTTPTransport()

	// do login
	if authorization, err = e.doLogin(client); err != nil {
		log.WithError(err).Errorln("error logging in")
		return
	}
	log.Println("logged in")

	// get account movements view
	if balance, err = e.getTransactions(client, authorization); err != nil {
		return
	}
	log.WithField("balance", balance.String()).Println("retrieved all transactions")

	return
}

// GetHTTPTransport http client setup
func GetHTTPTransport() *http.Client {
	var rtTransport http.RoundTripper = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 2 * time.Second,
		}).Dial,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	cookieJar, _ := cookiejar.New(nil)

	return &http.Client{
		Transport: rtTransport,
		Jar:       cookieJar,
	}
}
