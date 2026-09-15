package service

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/order-processing-system/payment-service/internal/repository"
	"github.com/order-processing-system/pkg/events"
	"github.com/rs/zerolog/log"
)

type PaymentService interface {
	ProcessPaymentSaga(ctx context.Context, payload events.PaymentRequestedPayload) error
	GetPaymentsByOrderID(ctx context.Context, orderID string) ([]repository.Payment, error)
	CreateSnapTransaction(ctx context.Context, orderID string, amount float64, customerEmail string) (*MidtransSnapResponse, error)
	HandleMidtransNotification(ctx context.Context, notif MidtransNotification) error
}

type Publisher interface {
	PublishPaymentSuccess(ctx context.Context, payload events.PaymentSuccessPayload) error
	PublishPaymentFailed(ctx context.Context, payload events.PaymentFailedPayload) error
}

type paymentService struct {
	repo       repository.Repository
	publisher  Publisher
	serverKey  string
	clientKey  string
	isProd     bool
	snapAPIURL string
}

func NewPaymentService(repo repository.Repository, publisher Publisher) PaymentService {
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	clientKey := os.Getenv("MIDTRANS_CLIENT_KEY")
	isProd := os.Getenv("MIDTRANS_IS_PRODUCTION") == "true"

	snapURL := "https://app.sandbox.midtrans.com/snap/v1/transactions"
	if isProd {
		snapURL = "https://app.midtrans.com/snap/v1/transactions"
	}

	return &paymentService{
		repo:       repo,
		publisher:  publisher,
		serverKey:  serverKey,
		clientKey:  clientKey,
		isProd:     isProd,
		snapAPIURL: snapURL,
	}
}

// ── Midtrans DTOs ────────────────────────────────────────────────────────────

type MidtransSnapRequest struct {
	TransactionDetails MidtransTransactionDetails `json:"transaction_details"`
	CustomerDetails    *MidtransCustomerDetails   `json:"customer_details,omitempty"`
}

type MidtransTransactionDetails struct {
	OrderID     string  `json:"order_id"`
	GrossAmount int64   `json:"gross_amount"`
}

type MidtransCustomerDetails struct {
	Email string `json:"email,omitempty"`
}

type MidtransSnapResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

type MidtransNotification struct {
	TransactionTime   string `json:"transaction_time"`
	TransactionStatus string `json:"transaction_status"` // settlement, capture, pending, deny, expire, cancel
	TransactionID     string `json:"transaction_id"`
	StatusCode        string `json:"status_code"`
	SignatureKey      string `json:"signature_key"`
	PaymentType       string `json:"payment_type"`
	OrderID           string `json:"order_id"`
	GrossAmount       string `json:"gross_amount"`
	FraudStatus       string `json:"fraud_status,omitempty"`
	StatusMessage     string `json:"status_message,omitempty"`
}

// ── Core Payment Processing ──────────────────────────────────────────────────

func (s *paymentService) ProcessPaymentSaga(ctx context.Context, payload events.PaymentRequestedPayload) error {
	orderUUID, err := uuid.Parse(payload.OrderID)
	if err != nil {
		return fmt.Errorf("invalid order id: %v", err)
	}

	// Idempotency check: Have we processed this payment already?
	payments, err := s.repo.GetPaymentsByOrder(ctx, orderUUID)
	if err == nil && len(payments) > 0 {
		return nil
	}

	amountNumeric := repository.Float64ToNumeric(payload.Amount)

	// 1. Create initial payment record as PENDING
	_, err = s.repo.CreatePayment(ctx, repository.CreatePaymentParams{
		OrderID: orderUUID,
		Amount:  amountNumeric,
		Status:  "PENDING",
	})
	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}

	// 2. Request Midtrans Snap Token
	snapResp, err := s.CreateSnapTransaction(ctx, payload.OrderID, payload.Amount, "customer@example.com")
	if err != nil {
		log.Warn().Err(err).Str("order_id", payload.OrderID).Msg("Midtrans Snap creation fallback")
	} else {
		log.Info().Str("order_id", payload.OrderID).Str("snap_token", snapResp.Token).Msg("Midtrans Snap Token generated")
	}

	// 3. Payment record remains PENDING waiting for Midtrans Webhook Notification or Customer Checkout
	log.Info().Str("order_id", payload.OrderID).Msg("Payment created as PENDING, awaiting Midtrans transaction settlement")
	return nil
}

func (s *paymentService) CreateSnapTransaction(ctx context.Context, orderID string, amount float64, customerEmail string) (*MidtransSnapResponse, error) {
	if s.serverKey != "" {
		reqBody := MidtransSnapRequest{
			TransactionDetails: MidtransTransactionDetails{
				OrderID:     orderID,
				GrossAmount: int64(amount),
			},
			CustomerDetails: &MidtransCustomerDetails{
				Email: customerEmail,
			},
		}

		jsonBody, _ := json.Marshal(reqBody)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.snapAPIURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, err
		}

		authHeader := base64.StdEncoding.EncodeToString([]byte(s.serverKey + ":"))
		req.Header.Set("Authorization", "Basic "+authHeader)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			defer resp.Body.Close()
			var snapResp MidtransSnapResponse
			if err := json.NewDecoder(resp.Body).Decode(&snapResp); err == nil {
				return &snapResp, nil
			}
		}
	}

	// Fallback simulation token
	mockToken := fmt.Sprintf("midtrans-snap-%s", orderID[:8])
	mockRedirectURL := fmt.Sprintf("https://app.sandbox.midtrans.com/snap/v2/vtweb/%s", mockToken)

	return &MidtransSnapResponse{
		Token:       mockToken,
		RedirectURL: mockRedirectURL,
	}, nil
}

func (s *paymentService) HandleMidtransNotification(ctx context.Context, notif MidtransNotification) error {
	orderUUID, err := uuid.Parse(notif.OrderID)
	if err != nil {
		return fmt.Errorf("invalid order id: %v", err)
	}

	// Verify signature if server key is configured
	if s.serverKey != "" && notif.SignatureKey != "" {
		rawSig := notif.OrderID + notif.StatusCode + notif.GrossAmount + s.serverKey
		hasher := sha512.New()
		hasher.Write([]byte(rawSig))
		expectedSig := hex.EncodeToString(hasher.Sum(nil))

		if notif.SignatureKey != expectedSig {
			log.Warn().Str("order_id", notif.OrderID).Msg("Midtrans signature mismatch")
		}
	}

	payments, err := s.repo.GetPaymentsByOrder(ctx, orderUUID)
	if err != nil || len(payments) == 0 {
		return fmt.Errorf("payment not found for order: %s", notif.OrderID)
	}
	payment := payments[0]

	status := strings.ToLower(notif.TransactionStatus)
	fraud := strings.ToLower(notif.FraudStatus)

	isSuccess := false
	isFailure := false
	failReason := ""

	if status == "capture" {
		if fraud == "challenge" {
			// Challenge status
		} else if fraud == "accept" {
			isSuccess = true
		}
	} else if status == "settlement" {
		isSuccess = true
	} else if status == "cancel" || status == "deny" || status == "expire" {
		isFailure = true
		failReason = fmt.Sprintf("Midtrans %s: %s", status, notif.StatusMessage)
	}

	if isSuccess {
		_, err = s.repo.UpdatePaymentStatus(ctx, repository.UpdatePaymentStatusParams{
			ID:                     payment.ID,
			Status:                 "SUCCESS",
			SimulatedFailureReason: pgtype.Text{Valid: false},
		})
		if err != nil {
			return err
		}
		return s.publisher.PublishPaymentSuccess(ctx, events.PaymentSuccessPayload{
			OrderID:   notif.OrderID,
			PaymentID: payment.ID.String(),
		})
	} else if isFailure {
		_, err = s.repo.UpdatePaymentStatus(ctx, repository.UpdatePaymentStatusParams{
			ID:                     payment.ID,
			Status:                 "FAILED",
			SimulatedFailureReason: pgtype.Text{String: failReason, Valid: true},
		})
		if err != nil {
			return err
		}
		return s.publisher.PublishPaymentFailed(ctx, events.PaymentFailedPayload{
			OrderID: notif.OrderID,
			Reason:  failReason,
		})
	}

	return nil
}

func (s *paymentService) GetPaymentsByOrderID(ctx context.Context, orderID string) ([]repository.Payment, error) {
	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		return nil, fmt.Errorf("invalid order id: %v", err)
	}
	return s.repo.GetPaymentsByOrder(ctx, orderUUID)
}
