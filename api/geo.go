package api

import (
	"encoding/json"
	"fmt"
	"locy/auth"
	"locy/crypto"
	"locy/user"
	"log"
	"math"
	"strings"

	"github.com/valyala/fasthttp"
)

type coords struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

func (c *coords) Distance(lat2 float64, lng2 float64, unit ...string) float64 {
	lat1 := c.Lat
	lng1 := c.Lng
	const PI float64 = 3.141592653589793

	radlat1 := float64(PI * lat1 / 180)
	radlat2 := float64(PI * lat2 / 180)

	theta := float64(lng1 - lng2)
	radtheta := float64(PI * theta / 180)

	dist := math.Sin(radlat1)*math.Sin(radlat2) + math.Cos(radlat1)*math.Cos(radlat2)*math.Cos(radtheta)

	if dist > 1 {
		dist = 1
	}

	dist = math.Acos(dist)
	dist = dist * 180 / PI
	dist = dist * 60 * 1.1515

	if len(unit) > 0 {
		if unit[0] == "K" {
			dist = dist * 1.609344
		} else if unit[0] == "N" {
			dist = dist * 0.8684
		}
	}

	return dist
}

type Active struct {
	Lng       float64 `json:"lng,omitempty"`
	Lat       float64 `json:"lat,omitempty"`
	Signature string  `json:"signature,omitempty"`
	Status    bool    `json:"status,omitempty"`
}

func (c *coords) getClosest() []user.User {
	db, _ := auth.Auth().DB.Init()
	query := fmt.Sprintf("select lat,lng,signature,username from users where cast(calculate_distance(lat, lng,%f, %f, 'M') as numeric(10,2)) > .25 and status = true;", c.Lat, c.Lng)
	rows, qErr := db.Query(query)
	if qErr != nil {
		log.Println(qErr)
	}
	users := []user.User{}
	for rows.Next() {
		usr := user.User{}
		rows.Scan(&usr.Lng, &usr.Lat, &usr.Signature, usr.Username)
		users = append(users, usr)
	}

	return users
}

func Endpoint(ctx *fasthttp.RequestCtx) {
	switch ctx.UserValue("op") {
	case "near":
		{
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
				log.Println("User does not exist.")
				fmt.Fprintf(ctx, "User does not exist.")
				return
			}

			body := ctx.Request.Body()
			location := coords{}
			err := json.Unmarshal(body, &location)
			if err != nil {
				log.Fatalln(err)
			}
			users := location.getClosest()
			data, _ := json.Marshal(users)

			fmt.Fprintf(ctx, string(data))
		}
	}
}
