package main

import (
	"crypto/tls"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	gomail "gopkg.in/mail.v2"
)

//go:embed static/*
var static embed.FS

//go:embed templates/*
var templates embed.FS

func main() {
	r := gin.Default()
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Load HTML templates
	r.SetHTMLTemplate(template.Must(template.New("").ParseFS(templates, "templates/*")))

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Make sure environment variables are set
	requiredEnvVars := []string{"HOST", "HOST_PORT", "HOST_USERNAME", "HOST_PASSWORD", "RECIVER_EMAIL"}
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			log.Fatalf("Environment variable %s is not set", envVar)
		}
	}

	// Serve static files on /static
	r.StaticFS("/f", http.FS(static))

	// Define the root route
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{"title": "Go Contact System"})
	})

	r.POST("/api/contact", func(c *gin.Context) {
		message := gomail.NewMessage()

		message.SetAddressHeader("From", c.PostForm("email"), fmt.Sprintf("%s (Go Contact System)", c.PostForm("name")))
		message.SetHeader("To", os.Getenv("RECIVER_EMAIL"))
		message.SetHeader("Subject", c.PostForm("subject"))

		message.SetBody("text/plain", fmt.Sprintf("%s\n\nDelivered By Go Contact System", c.PostForm("message")))

		port, err := strconv.Atoi(os.Getenv("HOST_PORT"))
		if err != nil {
			fmt.Println("Error converting HOST_PORT to integer:", err)
			panic(err)
		}
		dialer := gomail.NewDialer(os.Getenv("HOST"), port, os.Getenv("HOST_USERNAME"), os.Getenv("HOST_PASSWORD"))
		dialer.TLSConfig = &tls.Config{
			ServerName: os.Getenv("HOST"),
		}

		if err := dialer.DialAndSend(message); err != nil {
			fmt.Println("Error:", err)
			c.JSON(500, gin.H{
				"status":  "error",
				"message": "Failed to send email",
			})
			panic(err)
		} else {
			redirectURL := "/"
			if os.Getenv("REDIRECT_URL") != "" {
				redirectURL = os.Getenv("REDIRECT_URL")
			}
			c.Redirect(302, redirectURL)
		}
	})

	// Start the server
	if err := r.Run(":8080"); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
