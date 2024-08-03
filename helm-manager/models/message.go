package models

import "encoding/json"

type Message struct {
	// Http standard json response message
	Message map[string]string
}

func (e *Message) JsonMessage(message string) string {
	e.setMessage(message)
	e.Message["type"] = "message"
	return e.toJsonString()
}

func (e *Message) JsonError(message string) string {
	e.setMessage(message)
	e.Message["type"] = "error"
	return e.toJsonString()
}

func (e *Message) setMessage(message string) {
	e.Message = map[string]string{"message": message}
}

func (e *Message) JsonServerError() string {
	e.setMessage("Failed: Internal Server Error")
	e.Message["type"] = "serverError"
	return e.toJsonString()
}

func (e *Message) toJsonString() string {
	byte_string, err := json.Marshal(e.Message)
	if err != nil {
		return ""
	}
	return string(byte_string)
}
