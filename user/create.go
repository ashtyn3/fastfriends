package user

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"locy/auth"
	"locy/crypto"
	"log"
)

var pg, _ = auth.Auth().DB.Init()

func Link(authToken, connect, sig string) {
	res := crypto.Encrypt(connect, sig)
	pg.Exec("insert into linking (auth_token, data) values ('" + authToken + "','" + res + "')")
}

func UnLink(authToken string) {
	pg.Exec("delete from linking where auth_token =" + authToken)
}
func (u *User) Create() {
	sha := sha256.New()
	sha.Write([]byte(u.Username + u.Password))
	val := sha.Sum(nil)
	u.Signature = hex.EncodeToString(val)

	_, err := pg.Exec(fmt.Sprintf("insert into users (username, email, password, signature) values ('%s','%s','%s','%s')", u.Username, u.Email, u.Password, u.Signature))
	if err != nil {
		log.Println(err)
	}
}

func SetStatus(sig string, status bool) {
}
