package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type EventHub struct {
	routeService  *RouteService
	mongoClient   *mongo.Client
	chDriverMoved chan *DriverMovedEvent
	// chFrieghtCalculated chan *FreightCalculatedEvent
	freightWriter    *kafka.Writer
	simulationWriter *kafka.Writer
}

func NewEventHub(
	routeService *RouteService,
	mongoClient *mongo.Client,
	chDriverMoved chan *DriverMovedEvent,
	// chFreightCalculated chan *FreightCalculatedEvent,
	freightWriter *kafka.Writer,
	simulationWriter *kafka.Writer,
) *EventHub {
	return &EventHub{
		routeService:  routeService,
		mongoClient:   mongoClient,
		chDriverMoved: chDriverMoved,
		// chFrieghtCalculated: chFreightCalculated,
		freightWriter:    freightWriter,
		simulationWriter: simulationWriter,
	}
}

func (eh *EventHub) HandlerEvent(msg []byte) error {
	var baseEvent struct {
		EventName string `json:"event"`
	}
	err := json.Unmarshal(msg, &baseEvent)
	if err != nil {
		return fmt.Errorf("error unmarshalling event: %w", err)
	}

	switch baseEvent.EventName {
	case "RouteCreated":
		var event RouteCreatedEvent
		err := json.Unmarshal(msg, &event)
		if err != nil {
			return fmt.Errorf("error unmarshalling event: %w", err)
		}
		return eh.handlerRouteCreated(event)

	case "DeliveryStarted":
		var event DeliveryStartedEvent
		err := json.Unmarshal(msg, &event)
		if err != nil {
			return fmt.Errorf("error unmarshalling event: %w", err)
		}

	default:
		return fmt.Errorf("unknown event")
	}
	return nil
}

func (eh *EventHub) handlerRouteCreated(event RouteCreatedEvent) error {
	freightCalculatedEvent, err := RouteCreatedHandler(&event, eh.routeService)
	if err != nil {
		return err
	}
	value, err := json.Marshal(freightCalculatedEvent)
	if err != nil {
		return fmt.Errorf("error marshalling event: %w", err)
	}

	err = eh.freightWriter.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(freightCalculatedEvent.RouteID),
		Value: value,
	})
	if err != nil {
		return fmt.Errorf("error writing message: %w", err)
	}
	return nil
}

func (eh *EventHub) handlerDeliveryStarted(event DeliveryStartedEvent) error {
	err := DeliveryStartedHandler(&event, eh.routeService, eh.chDriverMoved)
	if err != nil {
		return err
	}
	go eh.sendDirections()
	// ler o canal e publicar no kafka
	return nil
}

func (eh *EventHub) sendDirections() {
	for {
		select {
		case movedEvent := <-eh.chDriverMoved:
			value, err := json.Marshal(movedEvent)
			if err != nil {
				return
			}

			err = eh.simulationWriter.WriteMessages(context.Background(), kafka.Message{
				Key:   []byte(movedEvent.RouteID),
				Value: value,
			})
			if err != nil {
				return
			}
		case <-time.After(500 * time.Millisecond):
			return
		}

	}
}
