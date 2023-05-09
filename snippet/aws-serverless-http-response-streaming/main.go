package main

import (
	"embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/net/websocket"
)

//go:embed index.html
var static embed.FS

func pingHandler(c echo.Context) error {
	return c.String(http.StatusOK, "pong")
}

func transferEncodingHandler(c echo.Context) error {
	c.Response().WriteHeader(http.StatusOK)

	for _, text := range []string{"foo", "bar", "baz"} {
		_, _ = io.WriteString(c.Response(), text+"\n")
		c.Response().Flush()

		time.Sleep(200 * time.Millisecond)
	}

	return nil
}

func sseHandler(c echo.Context) error {
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().WriteHeader(http.StatusOK)

	for _, text := range []string{"foo", "bar", "baz"} {
		_, _ = io.WriteString(c.Response(), "event: test\n")
		_, _ = io.WriteString(c.Response(), fmt.Sprintf("data: %s\n\n", text))
		c.Response().Flush()

		time.Sleep(200 * time.Millisecond)
	}

	return nil
}

func websocketHandler(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()

		for _, text := range []string{"foo", "bar", "baz"} {
			if err := websocket.Message.Send(ws, text+"\n"); err != nil {
				log.Println(err)
			}

			time.Sleep(200 * time.Millisecond)
		}
	}).ServeHTTP(c.Response(), c.Request())

	return nil
}

func buildEchoServer() *echo.Echo {
	e := echo.New()

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		c.Logger().Error(err)
	}

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/", func(c echo.Context) error {
		http.FileServer(http.FS(static)).ServeHTTP(c.Response(), c.Request())

		return nil
	})
	e.GET("/ping", pingHandler)
	e.GET("/transfer-encoding", transferEncodingHandler)
	e.GET("/sse", sseHandler)
	e.GET("/websocket", websocketHandler)

	return e
}

func main() {
	e := buildEchoServer()

	e.Logger.Fatal(e.Start(":1323"))
}
