package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"text/template"

	"github.com/joho/godotenv"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
)

type Config struct {
	PublicURL       string
	StripeKey       string
	StripeSecretKey string
	StripePriceID   string
}

func envConfig() *Config {
	return &Config{
		PublicURL:       os.Getenv("PUBLIC_URL"),
		StripeKey:       os.Getenv("STRIPE_KEY"),
		StripeSecretKey: os.Getenv("STRIPE_SECRET_KEY"),
		StripePriceID:   os.Getenv("STRIPE_PRICE_ID"),
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not present, this might be normal (production)")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	config := envConfig()

	stripe.Key = config.StripeSecretKey

	mux := http.NewServeMux()

	mux.HandleFunc("/", createHomeHandler(config))
	mux.HandleFunc("/order-success", createOrderSuccessHandler(config))
	mux.HandleFunc("/stripe-webhook", handleStripeWebHook)
	mux.HandleFunc("/checkout-session", handleCheckoutSession)
	mux.HandleFunc("/create-checkout-session", createHandleCreateCheckoutSession(config))

	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/favicon.ico")
	})

	mux.HandleFunc("/humans.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/humans.txt")
	})

	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/robots.txt")
	})

	log.Printf("listening on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
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

func emailIsValid(email string) bool {
	pattern := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	return pattern.MatchString(email)
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
		if !emailIsValid(reqBody.Email) {
			http.Error(w, fmt.Sprintf("You email is invalid, %s", http.StatusText(http.StatusBadRequest)), http.StatusBadRequest)
			return
		}

		fmt.Print(path.Join(config.PublicURL, "order-success?session_id={CHECKOUT_SESSION_ID}"))

		cancelURL, _ := url.Parse(config.PublicURL)
		cancelURL.Path = "/"

		successURL, _ := url.Parse(config.PublicURL)
		successURL.Path = "/order-success"
		successURL.Query().Add("session_id", "{CHECKOUT_SESSION_ID}")

		fmt.Print(stripe.String(successURL.String()))
		fmt.Print(successURL, cancelURL)
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
					Description: stripe.String(fmt.Sprintf("À récupérer le '%s' ou appelez au ‭(581) 999-6284‬", reqBody.Moment)),
				},
			},
			SuccessURL: stripe.String(successURL.String()),
			CancelURL:  stripe.String(cancelURL.String()),
			PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
				Metadata: map[string]string{"name": reqBody.Name, "phone": reqBody.Phone},
			},
		}

		session, err := session.New(params)
		if err != nil {
			log.Println("There was an error creating the checkout session", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		data := CreateCheckoutSessionResponse{
			SessionID: session.ID,
		}

		json.NewEncoder(w).Encode(&data)
	}
}

func createHomeHandler(config *Config) http.HandlerFunc {
	tmpl, err := template.ParseFiles("home.html.tmpl")

	if err != nil {
		log.Fatal("cannot parse template", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {

		a := struct {
			StripePublishableKey string
		}{StripePublishableKey: config.StripeKey}

		err = tmpl.Execute(w, &a)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func createOrderSuccessHandler(config *Config) http.HandlerFunc {
	tmpl, err := template.ParseFiles("order-success.html.tmpl")

	if err != nil {
		log.Fatal("cannot parse template", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		err = tmpl.Execute(w, nil)
		if err != nil {
			log.Fatal(err)
		}
	}
}
