package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
)

func Test_main(t *testing.T) {
	t.Helper()
	url := "http://127.0.0.1:8080/chatStreaming/bruno"
	app := fiber.New()
	jsonBody := []byte(`{"message": "hello, server!"}`)
	bodyReader := bytes.NewReader(jsonBody)

	req := httptest.NewRequest("POST", url, bodyReader)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJfaWQiOiI2NDhmYmJhYjMzMDA3N2ViNWNlNjUxN2UiLCJleHAiOjE2ODkyNjQzNjUsIm5hbWV1c2VyIjoiYnJ1bm8ifQ.XTgcFinxEgzaGWnpVjW15F5bBjRo49aQ9zw_VFA8aSs")
	resp, err := app.Test(req)
	responseBody, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(responseBody))
	utils.AssertEqual(t, nil, err, "app.Test(req)")
	utils.AssertEqual(t, 200, string(responseBody), "Status code")
}
