package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"

	"github.com/joho/godotenv"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
)

var price = "price_1HfqQmFcGrFlPonTgI04pCZq"

type Config struct {
	StripeKey       string
	StripeSecretKey string
	StripePriceID   string
}

func envConfig() *Config {
	return &Config{
		StripeKey:       os.Getenv("STRIPE_KEY"),
		StripeSecretKey: os.Getenv("STRIPE_SECRET_KEY"),
		StripePriceID:   os.Getenv("STRIPE_PRICE_ID"),
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	config := envConfig()

	stripe.Key = config.StripeSecretKey

	http.HandleFunc("/", createHomeHandler(config))
	http.HandleFunc("/order-success", handleOrderSuccess)
	http.HandleFunc("/stripe-webhook", handleStripeWebHook)
	http.HandleFunc("/checkout-session", handleCheckoutSession)
	http.HandleFunc("/create-checkout-session", createHandleCreateCheckoutSession(config))

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/favicon.ico")
	})

	http.HandleFunc("/humans.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/humans.txt")
	})

	http.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/robots.txt")
	})

	http.ListenAndServe(":8080", nil)
}

func handleOrderSuccess(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
func handleStripeWebHook(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func handleCheckoutSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	sessionID := r.URL.Query().Get("sessionId")
	fmt.Println(sessionID)
	s, _ := session.Get(sessionID, nil)
	json.NewEncoder(w).Encode(s)
}

type CreateCheckoutSessionResponse struct {
	SessionID string `json:"id"`
}

type CreateCheckoutSessionRequest struct {
	Name   string `json:"name"`
	Moment string `json:"moment"`
	Phone  string `json:"phone"`
	Email  string `json:"email"`
}

func createHandleCreateCheckoutSession(config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		reqBody := CreateCheckoutSessionRequest{}
		json.NewDecoder(r.Body).Decode(&reqBody)
		fmt.Println(reqBody)

		// Validate email, receive it

		params := &stripe.CheckoutSessionParams{
			PaymentMethodTypes: stripe.StringSlice([]string{
				"card",
			}),
			CustomerEmail: stripe.String(reqBody.Email),
			Mode:          stripe.String(string(stripe.CheckoutSessionModePayment)),
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				{
					Price:       stripe.String(config.StripePriceID),
					Quantity:    stripe.Int64(1),
					Description: stripe.String(fmt.Sprintf("À récupérer le '%s' ou appeler au ‭(581) 999-6284‬", reqBody.Moment)),
				},
			},
			SuccessURL: stripe.String("http://localhost:8080/success?session_id={CHECKOUT_SESSION_ID}"),
			CancelURL:  stripe.String("http://localhost:8080/"),
			PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
				Metadata: map[string]string{"name": reqBody.Name, "phone": reqBody.Phone},
			},
		}

		session, _ := session.New(params)
		// if err != nil {
		// 	return err
		// }

		data := CreateCheckoutSessionResponse{
			SessionID: session.ID,
		}

		json.NewEncoder(w).Encode(&data)
	}
}
func createHomeHandler(config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("home.html.tmpl")

		if err != nil {
			log.Fatal("cannor parse template", err)
		}

		a := struct {
			StripePublishableKey string
		}{StripePublishableKey: config.StripeKey}

		err = tmpl.Execute(w, &a)
		if err != nil {
			log.Fatal(err)
		}
	}
}
