package models

type ProcessResponse struct {
	Response string
}

type ProcessResponseError struct {
	Error string
}

type Request struct {
	Id int `json:"id"`
}
