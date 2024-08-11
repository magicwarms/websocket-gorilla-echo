package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/goccy/go-json"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/zishang520/socket.io/v2/socket"
)

var listClientID []string
var (
	roomUsers       = make(map[string][]string)
	roomUsersMutex  = &sync.Mutex{}
	maxUsersPerRoom = 3 // Set your max users per room
)

func main() {
	echoApp := echo.New()
	var socIO *socket.Server = socket.NewServer(nil, nil)

	socIO.On("connection", func(clients ...any) {
		socketCon := clients[0].(*socket.Socket)

		listClientID = append(listClientID, string(socketCon.Id()))

		socketCon.On("offer", func(data ...any) {
			socketID := socketCon.Id()

			type Offer struct {
				SDP  string `json:"sdp"`
				Type string `json:"type"`
			}
			
			offerMap, ok := data[0].(map[string]interface{})
			if !ok {
				fmt.Println("Error: offer data is not in the correct format")
				return
			}

			offerJSON, err := json.Marshal(offerMap)
			if err != nil {
				fmt.Printf("Error marshalling offer to JSON (offerJSON): %v\n", err)
				return
			}

			offer := string(offerJSON)
			targetedSocketID := fmt.Sprint(data[1])
			fmt.Printf("Offer received on server from %s to be able to send to %s in offer (%s)  \n", socketID, targetedSocketID, offer)

			socIO.To(socket.Room(targetedSocketID)).Emit("offer", offer, socketID)
		})

		socketCon.On("answer", func(data ...any) {
			socketID := socketCon.Id()
			answerMap, ok := data[0].(map[string]interface{})
			if !ok {
				fmt.Println("Error: answer data is not in the correct format")
				return
			}

			answerJSON, err := json.Marshal(answerMap)
			if err != nil {
				fmt.Printf("Error marshalling offer to JSON (answerJSON): %v\n", err)
				return
			}

			answer := string(answerJSON)
			targetedSocketID := fmt.Sprint(data[1])
			fmt.Printf("Answer received on server to be able to send to %s in answer (%s)  \n", targetedSocketID, answer)

			socIO.To(socket.Room(targetedSocketID)).Emit("answer", answer, socketID)
		})

		socketCon.On("ice-candidate", func(data ...any) {
			socketID := socketCon.Id()

			iceCandidateMap, ok := data[0].(map[string]interface{})
			if !ok {
				fmt.Println("Error: iceCandidate data is not in the correct format")
				return
			}

			iceCandidateJSON, err := json.Marshal(iceCandidateMap)
			if err != nil {
				fmt.Printf("Error marshalling offer to JSON (iceCandidateJSON): %v\n", err)
				return
			}

			iceCandidate := string(iceCandidateJSON)
			targetedSocketID := fmt.Sprint(data[1])
			fmt.Printf("Ice candidate received on server to be able to send to %s in iceCandidate (%s) \n", targetedSocketID, iceCandidate)

			socIO.To(socket.Room(targetedSocketID)).Emit("ice-candidate", iceCandidate, socketID)
		})

		socketCon.On("create-meet-link", func(data ...any) {
			socketID := string(socketCon.Id())
			// Get the current timestamp
			timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

			// Combine socket ID and timestamp
			combinedString := socketID + "-" + timestamp

			// Get the last 5 characters
			meetLink := combinedString[len(combinedString)-5:]
			fmt.Printf("Create meet link received on server %s \n", meetLink)
			socIO.To(socket.Room(socketID)).Emit("meet-link-created", meetLink)
		})

		socketCon.On("join-meet-link", func(data ...any) {
			meetLink := fmt.Sprint(data[0])
			fmt.Printf("Join meet link received on server: (%s) \n", meetLink)

			roomUsersMutex.Lock()
			users := roomUsers[meetLink]

			// Check if the room is full
			if len(users) >= maxUsersPerRoom {
				socketCon.Emit("room-full", meetLink)
				roomUsersMutex.Unlock()
				return
			}

			// Add user to the room
			users = append(users, string(socketCon.Id()))
			roomUsers[meetLink] = removeDuplicates(users)

			socketCon.Join(socket.Room(meetLink))
			socketCon.To(socket.Room(meetLink)).Emit("user-joined", socketCon.Id())
			roomUsersMutex.Unlock()

		})

		socketCon.On("disconnect", func(data ...any) {
			roomUsersMutex.Lock()
			for meetLink, users := range roomUsers {
				for i, userID := range users {
					if userID == string(socketCon.Id()) {
						users = append(users[:i], users[i+1:]...)
						roomUsers[meetLink] = users
						socketCon.To(socket.Room(meetLink)).Emit("user-left", socketCon.Id())
						break
					}
				}
			}
			fmt.Println("After leave:", roomUsers)
			roomUsersMutex.Unlock()
		})

	})

	// echoApp.Use(middleware.Logger())
	echoApp.Use(middleware.Recover())

	echoApp.Use(corsMiddleware)
	echoApp.Any("/socket.io/", echo.WrapHandler(socIO.ServeHandler(nil)))
	echoApp.GET("/status", func(c echo.Context) error {
		return c.String(http.StatusOK, "Server is running")
	})
	echoApp.Logger.Fatal(echoApp.Start(":9000"))

	exit := make(chan struct{})
	SignalC := make(chan os.Signal)

	signal.Notify(SignalC, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for s := range SignalC {
			switch s {
			case os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				close(exit)
				return
			}
		}
	}()

	<-exit
	echoApp.Close()
	os.Exit(0)
}

func corsMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{echo.GET, echo.HEAD, echo.PUT, echo.PATCH, echo.POST, echo.DELETE},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowCredentials: true,
	})(next)
}

// Helper function to remove duplicates from a slice
func removeDuplicates(users []string) []string {
	userSet := make(map[string]struct{})
	var uniqueUsers []string

	for _, user := range users {
		if _, exists := userSet[user]; !exists {
			userSet[user] = struct{}{}
			uniqueUsers = append(uniqueUsers, user)
		}
	}
	return uniqueUsers
}
