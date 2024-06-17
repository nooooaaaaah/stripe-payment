package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"

	_ "github.com/joho/godotenv/autoload"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/paymentintent"
	"github.com/stripe/stripe-go/v72/webhook"
)

func main() {

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	http.Handle("/", http.FileServer(http.Dir(os.Getenv("STATIC_DIR"))))
	http.HandleFunc("/config", logRequest(handleConfig))
	http.HandleFunc("/webhook", logRequest(handleWebhook))
	http.HandleFunc("/create-payment-intent", logRequest(handleCreatePaymentIntent))
	http.HandleFunc("/update-payment-intent", logRequest(handleUpdatePaymentIntent)) // New endpoint

	log.Println("server running at 0.0.0.0:4242")
	if err := http.ListenAndServe("0.0.0.0:4242", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func logRequest(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Attempted request: %s %s", r.Method, r.URL.Path)
		handler(w, r)
	}
}

// ErrorResponseMessage represents the structure of the error
// object sent in failed responses.
type ErrorResponseMessage struct {
	Message string `json:"message"`
}

// ErrorResponse represents the structure of the error object sent
// in failed responses.
type ErrorResponse struct {
	Error *ErrorResponseMessage `json:"error"`
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, struct {
		PublishableKey string `json:"publishableKey"`
	}{
		PublishableKey: os.Getenv("STRIPE_PUBLISHABLE_KEY"),
	})
}

type updatePaymentIntentReq struct {
	ClientSecret string `json:"clientSecret"`
	Amount       int64  `json:"amount"`
}

func handleUpdatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request to update payment intent")
	req := updatePaymentIntentReq{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Error decoding request body: %v", err)
		writeJSONErrorMessage(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		log.Println("Invalid amount provided")
		writeJSONErrorMessage(w, "Invalid amount", http.StatusBadRequest)
		return
	}
	log.Printf("Updating amount to: %d", req.Amount)

	params := &stripe.PaymentIntentParams{
		Amount: stripe.Int64(req.Amount),
	}

	re := regexp.MustCompile(`pi_[A-Za-z0-9]+`)
	paymentIntentID := re.FindString(req.ClientSecret)

	pi, err := paymentintent.Update(paymentIntentID, params)
	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok {
			log.Printf("Stripe error: %v", stripeErr.Error())
			writeJSONErrorMessage(w, stripeErr.Error(), http.StatusBadRequest)
		} else {
			log.Printf("Server error: %v", err.Error())
			writeJSONErrorMessage(w, "Unknown server error", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, struct {
		ClientSecret string `json:"clientSecret"`
	}{
		ClientSecret: pi.ClientSecret,
	})
}

type paymentIntentCreateReq struct {
	Currency          string `json:"currency"`
	PaymentMethodType string `json:"paymentMethodType"`
	Amount            int64  `json:"amount"`
}

func handleCreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request to create payment intent")
	req := paymentIntentCreateReq{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Error decoding request body: %v", err)
		writeJSONErrorMessage(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		log.Println("Invalid amount provided")
		writeJSONErrorMessage(w, "Invalid amount", http.StatusBadRequest)
		return
	}
	log.Printf("Creating payment intent with amount: %d", req.Amount)

	var formattedPaymentMethodType []*string
	if req.PaymentMethodType == "link" {
		formattedPaymentMethodType = append(formattedPaymentMethodType, stripe.String("link"), stripe.String("card"))
	} else {
		formattedPaymentMethodType = append(formattedPaymentMethodType, stripe.String(req.PaymentMethodType))
	}

	params := &stripe.PaymentIntentParams{
		Amount:             stripe.Int64(req.Amount),
		Currency:           stripe.String(req.Currency),
		PaymentMethodTypes: formattedPaymentMethodType,
	}

	if req.PaymentMethodType == "acss_debit" {
		params.PaymentMethodOptions = &stripe.PaymentIntentPaymentMethodOptionsParams{
			ACSSDebit: &stripe.PaymentIntentPaymentMethodOptionsACSSDebitParams{
				MandateOptions: &stripe.PaymentIntentPaymentMethodOptionsACSSDebitMandateOptionsParams{
					PaymentSchedule: stripe.String("sporadic"),
					TransactionType: stripe.String("personal"),
				},
			},
		}
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok {
			log.Printf("Stripe error: %v", stripeErr.Error())
			writeJSONErrorMessage(w, stripeErr.Error(), http.StatusBadRequest)
		} else {
			log.Printf("Server error: %v", err.Error())
			writeJSONErrorMessage(w, "Unknown server error", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, struct {
		ClientSecret string `json:"clientSecret"`
	}{
		ClientSecret: pi.ClientSecret,
	})
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	log.Println("Received webhook event")
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	b, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("ioutil.ReadAll: %v", err)
		return
	}

	event, err := webhook.ConstructEvent(b, r.Header.Get("Stripe-Signature"), os.Getenv("STRIPE_WEBHOOK_SECRET"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("webhook.ConstructEvent: %v", err)
		return
	}

	if event.Type == "payment_intent.succeeded" {
		log.Println("Payment completed!")
	}

	writeJSON(w, nil)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("json.NewEncoder.Encode: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := io.Copy(w, &buf); err != nil {
		log.Printf("io.Copy: %v", err)
		return
	}
}

func writeJSONError(w http.ResponseWriter, v interface{}, code int) {
	w.WriteHeader(code)
	writeJSON(w, v)
	return
}

func writeJSONErrorMessage(w http.ResponseWriter, message string, code int) {
	resp := &ErrorResponse{
		Error: &ErrorResponseMessage{
			Message: message,
		},
	}
	writeJSONError(w, resp, code)
}
