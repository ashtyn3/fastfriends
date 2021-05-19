package user

import (
	"locy/auth"
	"log"
)

var db, _ = auth.Auth().DB.Init()

type LinkData struct {
	AuthToken string
	Data      string
}

func GetLink(authToken string) LinkData {
	data := LinkData{}
	query := db.QueryRow(`select auth_token, data from linking where auth_token='` + authToken + `';`)
	qErr := query.Scan(&data.AuthToken, &data.Data)
	if qErr != nil {
		log.Println("could not get user ", data)
		return data
	}
	return data
}

func GetUser(hash string) User {
	usr := User{}
	query := db.QueryRow(`select id, username, email from users where signature='` + hash + `';`)
	qErr := query.Scan(&usr.Id, &usr.Username, &usr.Email)
	if qErr != nil {
		log.Println("could not get user ", qErr)
		return usr
	}
	return usr
}

type RoomData struct {
	Id   int    `json:"id"`
	With string `json:"to"`
}

func UserRooms(sig string) []RoomData {
	query, _ := db.Query(`select rooms.id, username from rooms inner join users on signature = any(members) and signature != '` + sig + `' where '` + sig + `'= any(members);`)
	rooms := []RoomData{}
	for query.Next() {
		roomData := RoomData{}
		query.Scan(&roomData.Id, &roomData.With)
		rooms = append(rooms, roomData)
	}
	return rooms
}
