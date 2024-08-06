package main

import (
	"fmt"
	"regexp"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/zishang520/socket.io/v2/socket"
)

// var (
// 	upgrader = websocket.Upgrader{}
// )

// func hello(c echo.Context) error {
// 	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
// 	if err != nil {
// 		return err
// 	}
// 	defer ws.Close()

// 	for {
// 		// Write
// 		err := ws.WriteMessage(websocket.TextMessage, []byte("Hello, Client!"))
// 		if err != nil {
// 			c.Logger().Error(err)
// 		}

// 		// Read
// 		_, msg, err := ws.ReadMessage()
// 		if err != nil {
// 			c.Logger().Error(err)
// 		}
// 		fmt.Printf("%s\n", msg)
// 	}
// }

// func main() {
// 	io := socket.NewServer(nil,nil)
// 	echoApp := echo.New()

// 	echoApp.Any("/socket.io/", io.ServeHandler(nil))

// 	// e.Use(middleware.Logger())
// 	// e.Use(middleware.Recover())func serveSocketIO(c echo.Context) error {
// 		io.ServeHandler(nil).ServeHTTP(c.Response(), c.Request())
// 		return nil
// 	}

// 	// e.Static("/", "../public")
// 	// e.GET("/ws", hello)
// 	// e.Logger.Fatal(e.Start(":3000"))
// }

// func serveSocketIO(c echo.Context) error {
// 	io.ServeHandler(nil).ServeHTTP(c.Response(), c.Request())
// 	return nil
// }

func main() {
	echoApp := echo.New()
	var socketio *socket.Server = socket.NewServer(nil, nil)
	socketio.Of(
		regexp.MustCompile(`/\w+`),
		nil,
	).Use(func(client *socket.Socket, next func(*socket.ExtendedError)) {
		fmt.Println("MId:", client.Connected())
		next(nil)
	}).On("connection", func(clients ...interface{}) {
		socketConnection := clients[0].(*socket.Socket)
		// fmt.Println("/ test Handshake：", socketConnection.Handshake())

		socketConnection.On("sendMsg", func(clients ...interface{}) {
			fmt.Println("/ test asd: ", clients)
			socketConnection.Emit("reply", clients[0])

			// client.Broadcast().Emit("reply", "hi test")
		})

		// client.On("welcome", func(data ...interface{}) {
		// 	fmt.Println("/ test welcome: ", data)
		// 	client.Emit("welcomeReply", data[0])

		socketConnection.On("offer", func(data ...interface{}) {
			targetSocketId := fmt.Sprint(data[1])
			fmt.Println("HAHAHAHAHAH", "Offer received on server from "+fmt.Sprint(socketConnection.Id())+" to be able to send to "+targetSocketId, data)

			socketConnection.To(socket.Room(targetSocketId)).Emit("offer", data[1], socketConnection.Id())
		})

		socketConnection.On("answer", func(data ...interface{}) {
			targetSocketId := fmt.Sprint(data[1])
			fmt.Println(
				`Answer received on server to be able to send to: `+targetSocketId,
				data[0],
			)

			socketConnection.To(socket.Room(targetSocketId)).Emit("answer", data[0], socketConnection.Id())
		})

		socketConnection.On("ice-candidate", func(data ...interface{}) {
			targetSocketId := fmt.Sprint(data[1])
			fmt.Println(`Ice candidate received on server to be able to send to: `+targetSocketId, data[0])

			socketConnection.To(socket.Room(targetSocketId)).Emit("ice-candidate", data[0], socketConnection.Id())
		})

		socketConnection.On("create-meet-link", func(data ...interface{}) {
			fmt.Println("Create meet link received on server")

			meetLink := fmt.Sprint(socketConnection.Id()) + `_` + fmt.Sprint(time.Now().UnixMilli())

			// haha := fmt.Sprintf("%s", socketConnection.Id())
			fmt.Println("Created meet link: ", meetLink)
			socketConnection.To(socket.Room(socketConnection.Id())).Emit("meet-link-created", meetLink)
		})

		socketConnection.On("join-meet-link", func(data ...interface{}) {
			fmt.Println("Join meet link received on server: ", data[0])
			meetLink := fmt.Sprint(data[0])

			// const maxUsersPerRoom = 3
			// get from database later or else
			// if (users.length >= maxUsersPerRoom) {
			//     io.to(socket.id).emit('room-full', meetLink)
			//     return
			// }

			socketConnection.Join(socket.Room(meetLink))

			// users.push(socket.id)
			// roomUsers.set(meetLink, [...new Set(users)])

			socketConnection.To(socket.Room(meetLink)).Emit("user-joined", socketConnection.Id())
		})

		socketConnection.On("disconnect", func(data ...interface{}) {
			meetLink := fmt.Sprint(data[0])
			fmt.Println("/test disconnect ID", socketConnection.Id(), data)

			socketConnection.To(socket.Room(meetLink)).Emit("user-left:", socketConnection.Id())
		})
	})

	// echoApp.Use(middleware.Logger())
	echoApp.Use(middleware.Recover())

	echoApp.Any("/socket.io/", echo.WrapHandler(socketio.ServeHandler(nil)), corsMiddleware)

	echoApp.Logger.Fatal(echoApp.Start(":9000"))
}

func corsMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		allowHeaders := "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization"

		c.Response().Header().Set("Content-Type", "application/json")
		c.Response().Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Response().Header().Set("Access-Control-Allow-Methods", "OPTIONS, POST, PUT, PATCH, GET, DELETE")
		c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Response().Header().Set("Access-Control-Allow-Credentials", "true")
		c.Response().Header().Set("Access-Control-Allow-Headers", allowHeaders)

		return next(c)
	}
}
