package models

type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

type UserResponse struct {
	Data User `json:"data"`
}
