package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v86"

	"github.com/FyrmForge/hamr-example/gather/internal/repo"
)

// ErrFreeEvent is returned when someone tries to buy a ticket for a free event.
var ErrFreeEvent = errors.New("event is free")

// currency is what every ticket is priced in.
// ponytail: one currency; store it per event if Gather sells outside the euro area.
const currency = "eur"

// TicketService sells tickets through Stripe Checkout. In development the
// stripe client points at the hamr Stripe mock (STRIPE_MOCK=true); the code
// is the same against the real API.
type TicketService struct {
	store  repo.Store
	stripe *stripe.Client
}

// NewTicketService creates a new ticket service.
func NewTicketService(store repo.Store, client *stripe.Client) *TicketService {
	return &TicketService{store: store, stripe: client}
}

// Checkout opens a Stripe Checkout session for one ticket and returns the
// URL to send the buyer to. eventURL is the event page's absolute URL;
// Stripe sends the buyer back there when they pay or cancel. It returns
// ("", nil) when the event doesn't exist.
func (s *TicketService) Checkout(ctx context.Context, eventID, userID, eventURL string) (string, error) {
	e, err := s.store.Events().GetByID(ctx, eventID)
	if err != nil || e == nil {
		return "", err
	}
	if e.PriceCents == 0 {
		return "", ErrFreeEvent
	}
	ticketID := uuid.New().String()
	cs, err := s.stripe.V1CheckoutSessions.Create(ctx, &stripe.CheckoutSessionCreateParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		ClientReferenceID: stripe.String(ticketID),
		SuccessURL:        stripe.String(eventURL + "?ticket=paid"),
		CancelURL:         stripe.String(eventURL + "?ticket=cancelled"),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{{
			Quantity: stripe.Int64(1),
			PriceData: &stripe.CheckoutSessionCreateLineItemPriceDataParams{
				Currency:   stripe.String(currency),
				UnitAmount: stripe.Int64(int64(e.PriceCents)),
				ProductData: &stripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{
					Name: stripe.String("Ticket: " + e.Title),
				},
			},
		}},
	})
	if err != nil {
		return "", fmt.Errorf("create checkout session: %w", err)
	}
	err = s.store.Tickets().Create(ctx, &repo.Ticket{
		ID:                ticketID,
		EventID:           e.ID,
		UserID:            userID,
		CheckoutSessionID: cs.ID,
		AmountCents:       e.PriceCents,
		Status:            "pending",
		CreatedAt:         time.Now(),
	})
	if err != nil {
		return "", fmt.Errorf("store ticket: %w", err)
	}
	return cs.URL, nil
}

// CheckoutCompleted marks the ticket bought through the session as paid.
// Called from the checkout.session.completed webhook.
func (s *TicketService) CheckoutCompleted(ctx context.Context, cs *stripe.CheckoutSession) error {
	if cs.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
		return nil // async payment methods settle later; nothing to do yet
	}
	return s.store.Tickets().SetStatus(ctx, cs.ID, "paid")
}

// CheckoutExpired marks the ticket of an abandoned session as expired.
// Called from the checkout.session.expired webhook.
func (s *TicketService) CheckoutExpired(ctx context.Context, cs *stripe.CheckoutSession) error {
	return s.store.Tickets().SetStatus(ctx, cs.ID, "expired")
}

// ParsePrice turns a price in euros ("", "5", "7.50") into cents.
func ParsePrice(v string) (int, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, nil
	}
	if i := strings.IndexByte(v, '.'); i >= 0 && len(v)-i-1 > 2 {
		return 0, fmt.Errorf("price %q has more than two decimals", v)
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < 0 || f > 10000 {
		return 0, fmt.Errorf("price %q is not between 0 and 10000", v)
	}
	return int(math.Round(f * 100)), nil
}

// FormatPrice renders cents as euros: 500 is "€5", 750 is "€7.50".
func FormatPrice(cents int) string {
	if cents%100 == 0 {
		return fmt.Sprintf("€%d", cents/100)
	}
	return fmt.Sprintf("€%d.%02d", cents/100, cents%100)
}
