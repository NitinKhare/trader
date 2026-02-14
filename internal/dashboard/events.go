package dashboard

import (
	"context"
	"log"
	"time"

	"github.com/lib/pq"
)

// EventListener listens for PostgreSQL notifications
type EventListener struct {
	dbURL      string
	logger     *log.Logger
	broadcaster *Broadcaster
	shutdown   chan struct{}
}

// NewEventListener creates a new EventListener
func NewEventListener(dbURL string, broadcaster *Broadcaster, logger *log.Logger) *EventListener {
	return &EventListener{
		dbURL:       dbURL,
		logger:      logger,
		broadcaster: broadcaster,
		shutdown:    make(chan struct{}),
	}
}

// Start begins listening for database notifications
func (el *EventListener) Start(ctx context.Context) {
	go el.listenLoop(ctx)
}

// listenLoop continuously listens for PostgreSQL notifications
func (el *EventListener) listenLoop(ctx context.Context) {
	defer el.logger.Println("event listener: shutting down")

	minRetryDelay := 100 * time.Millisecond
	maxRetryDelay := 10 * time.Second
	retryDelay := minRetryDelay

	for {
		select {
		case <-ctx.Done():
			return
		case <-el.shutdown:
			return
		default:
		}

		listener := pq.NewListener(el.dbURL, minRetryDelay, maxRetryDelay, func(ev pq.ListenerEventType, err error) {
			if err != nil {
				el.logger.Printf("event listener: %v", err)
			}
		})

		if err := el.setupListeners(listener); err != nil {
			el.logger.Printf("event listener: failed to setup listeners: %v", err)
			listener.Close()
			retryDelay = maxRetryDelay
			time.Sleep(retryDelay)
			continue
		}

		retryDelay = minRetryDelay

		// Listen for notifications
		if err := el.handleNotifications(ctx, listener); err != nil {
			el.logger.Printf("event listener: %v", err)
		}

		listener.Close()

		select {
		case <-ctx.Done():
			return
		case <-el.shutdown:
			return
		default:
			time.Sleep(retryDelay)
		}
	}
}

// setupListeners subscribes to PostgreSQL channels
func (el *EventListener) setupListeners(listener *pq.Listener) error {
	channels := []string{
		"trade_closed",
		"position_opened",
		"trade_executed",
		"metrics_updated",
	}

	for _, channel := range channels {
		if err := listener.Listen(channel); err != nil {
			return err
		}
		el.logger.Printf("event listener: listening on channel '%s'", channel)
	}

	return nil
}

// handleNotifications handles incoming PostgreSQL notifications
func (el *EventListener) handleNotifications(ctx context.Context, listener *pq.Listener) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-el.shutdown:
			return nil

		case notification := <-listener.Notify:
			if notification == nil {
				return nil
			}

			el.logger.Printf("event listener: received notification on channel '%s': %s", notification.Channel, notification.Extra)

			// Broadcast the event to WebSocket clients
			msg := WebSocketMessage{
				Type: notification.Channel,
				Data: map[string]interface{}{
					"event": notification.Extra,
				},
				Timestamp: time.Now().Format(time.RFC3339),
			}

			el.broadcaster.Broadcast(msg)
		}
	}
}

// Stop stops the event listener
func (el *EventListener) Stop() {
	close(el.shutdown)
}
