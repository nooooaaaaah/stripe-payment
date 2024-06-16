package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/joho/godotenv"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/paymentintent"
	"github.com/stripe/stripe-go/v72/webhook"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	// For sample support and debugging, not required for production:
	stripe.SetAppInfo(&stripe.AppInfo{
		Name:    "stripe-samples/your-sample-name",
		Version: "0.0.1",
		URL:     "https://github.com/stripe-samples",
	})

	http.Handle("/", http.FileServer(http.Dir(os.Getenv("STATIC_DIR"))))
	http.HandleFunc("/config", handleConfig)
	http.HandleFunc("/webhook", handleWebhook)
	http.HandleFunc("/create-payment-intent", handleCreatePaymentIntent)
	http.HandleFunc("/update-payment-intent", handleUpdatePaymentIntent) // New endpoint

	log.Println("server running at 0.0.0.0:4242")
	if err := http.ListenAndServe("0.0.0.0:4242", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
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
	req := updatePaymentIntentReq{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeJSONErrorMessage(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		writeJSONErrorMessage(w, "Invalid amount", http.StatusBadRequest)
		return
	}
	fmt.Println("updated ammount: ", req.Amount)

	params := &stripe.PaymentIntentParams{
		Amount: stripe.Int64(req.Amount),
	}

	re := regexp.MustCompile(`pi_[A-Za-z0-9]+`)
	paymentIntentID := re.FindString(req.ClientSecret)

	pi, err := paymentintent.Update(paymentIntentID, params)
	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok {
			fmt.Printf("Stripe error: %v\n", stripeErr.Error())
			writeJSONErrorMessage(w, stripeErr.Error(), http.StatusBadRequest)
		} else {
			fmt.Printf("Server error: %v\n", err.Error())
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
	req := paymentIntentCreateReq{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeJSONErrorMessage(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		writeJSONErrorMessage(w, "Invalid amount", http.StatusBadRequest)
		return
	}
	fmt.Println("amount: ", req.Amount)

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
			fmt.Printf("Stripe error: %v\n", stripeErr.Error())
			writeJSONErrorMessage(w, stripeErr.Error(), http.StatusBadRequest)
		} else {
			fmt.Printf("Server error: %v\n", err.Error())
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
		fmt.Println("Payment completed!")
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
