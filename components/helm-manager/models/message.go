package models

import "encoding/json"

type Message struct {
	// Http standard json response message
	Message map[string]string
}

func (e *Message) setMessage(message string) {
	if e.Message == nil {
		e.Message = make(map[string]string)
	}
	e.Message["message"] = message
}
func (e *Message) JsonMessage(message string) string {
	e.setMessage(message)
	e.Message["type"] = "message"
	return e.toJsonString()
}

func (e *Message) JsonError(messages ...interface{}) string {
	var finalMessage string
	for _, message := range messages {
		switch msg := message.(type) {
		case string:
			finalMessage = finalMessage + " " + msg
		case error:
			finalMessage = finalMessage + " " + msg.Error()
		}
	}
	e.setMessage(finalMessage)
	e.Message["type"] = "error"
	return e.toJsonString()
}

func (e *Message) JsonServerError() string {
	errorMessage := "Internal Server Error"
	e.setMessage(errorMessage)
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
