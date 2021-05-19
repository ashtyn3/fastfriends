package api

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"locy/crypto"
	"locy/user"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

var ledger map[string]string = make(map[string]string)

type Body struct {
	Token    string `json:"token,omitempty"`
	Tok_expr string `json:"tok_expr,omitempty"`
}

type userConfirm struct {
	Signature string `json:"signature"`
}

func Auth(ctx *fasthttp.RequestCtx) {
	switch ctx.UserValue("op") {
	case "grant":
		hash := sha256.New()
		base, _ := rand.Int(rand.Reader, big.NewInt(5000000000))
		hash.Write(base.Bytes())
		section := []byte(strconv.Itoa(len(ledger) - 1))
		bytes := hash.Sum(section)
		sha := sha1.New()
		sha.Write([]byte(base64.StdEncoding.EncodeToString(bytes)))
		id := hex.EncodeToString(sha.Sum(nil))
		ledger[id[0:32]] = time.Now().String()
		timein := time.Now().Local().AddDate(0, 0, 60)
		data, _ := json.Marshal(Body{Token: id[0:32], Tok_expr: timein.String()})
		fmt.Fprintf(ctx, string(data))
	case "new":
		body := ctx.Request.Body()
		var u user.User
		json.Unmarshal(body, &u)
		hash := sha256.New()
		content := u.Username + u.Password
		hash.Write([]byte(content))
		newHash := hash.Sum(nil)
		if hex.EncodeToString(newHash) != u.Signature {
			fmt.Fprintf(ctx, "signatures do not match")
			return
		}
		tok := ctx.Request.Header.Peek("Authorization")
		grant_tok := ledger[string(tok)]
		if grant_tok == "" {
			fmt.Fprintf(ctx, "Could not authorize request.")
			return
		}
		//newUser := usr.User{Email: user.Email, Firstname: user.First_name, Lastname: user.Last_name, Password: user.Password, Signature: user.Signature, Username: user.Username}
		//created := newUser.Create()
		u.Create()
		token := hex.EncodeToString(tok)[0:11]
		user.Link(string(tok), token, u.Signature)
		data, _ := json.Marshal(Body{Token: token})
		fmt.Fprintf(ctx, string(data))
	case "get":
		tok := ctx.Request.Header.Peek("Authorization")
		keys := strings.Split(string(tok), " ")
		grant_tok := ledger[keys[0]]
		if grant_tok == "" {
			log.Println("Could not authorize request.")
			fmt.Fprintf(ctx, "Could not authorize request.")
			return
		}

		enc := user.GetLink(keys[0])
		if enc.Data == "" {
			log.Println("Could not authorize request.")
			fmt.Fprintf(ctx, "Could not authorize request.")
			return
		}
		sig := crypto.Decrypt(keys[1], enc.Data)

		user := user.GetUser(sig)
		if user.Username == "" {
			log.Println("Could not get user.")
			fmt.Fprintf(ctx, "Could not get user.")
			return
		}

		userJson, _ := json.Marshal(user)
		fmt.Fprintf(ctx, string(userJson))
		return
	case "confirm":
		body := ctx.Request.Body()
		var token userConfirm
		json.Unmarshal(body, &token)
		tok := ctx.Request.Header.Peek("Authorization")
		key := string(tok)
		grant_tok := ledger[key]
		if grant_tok == "" {
			log.Println("Could not authorize request.")
			fmt.Fprintf(ctx, "Could not authorize request.")
			return
		}

		usr := user.GetUser(token.Signature)

		if usr.Username == "" {
			log.Println("User does not exist.")
			fmt.Fprintf(ctx, "User does not exist")
			return
		}
		connectToken := hex.EncodeToString(tok)[0:11]
		user.Link(string(tok), connectToken, token.Signature)
		json, _ := json.Marshal(Body{Token: connectToken})
		fmt.Fprintf(ctx, string(json))
	}

}

func RemoveExpired() {
	ticker := time.NewTicker(2 * time.Second)
	quit := make(chan string)
	go func() {
		for {
			select {
			case <-ticker.C:
				{
					for key, element := range ledger {
						then, _ := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", element)
						since := time.Since(then)
						if since.Hours() <= 1440 {
							delete(ledger, key)
							user.UnLink(key)
						}
					}
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}
func Run() {
	r := router.New()
	r.GET("/auth/{op}", Auth)
	r.GET("/geo/{op}", Endpoint)
	r.GET("/chat/{op}", ChatEndpoint)

	go log.Fatal(fasthttp.ListenAndServe(":8080", r.Handler))
	go RemoveExpired()
}
