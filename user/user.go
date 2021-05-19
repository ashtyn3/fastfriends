package user

type User struct {
	Id        int     `json:"id,omitempty"`
	Username  string  `json:"username,omitempty"`
	Email     string  `json:"email,omitempty"`
	Password  string  `json:"password,omitempty"`
	Signature string  `json:"signature,omitempty"`
	Status    bool    `json:"status,omitempty"`
	Lng       float64 `json:"lng,omitempty"`
	Lat       float64 `json:"lat,omitempty"`
}
