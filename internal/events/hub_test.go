package events

import "testing"

func TestSubscriptionReceivesOnlyItsOwnersEvents(t *testing.T) {
	h := NewHub()
	mine := h.Subscribe("owner-a")
	other := h.Subscribe("owner-b")
	defer mine.Close()
	defer other.Close()

	h.Publish(Event{Type: TypeMessageCreated, OwnerID: "owner-a", Data: "hi"})

	ev := <-mine.C
	if ev.Type != TypeMessageCreated || ev.Data != "hi" {
		t.Fatalf("got %+v, want the published message", ev)
	}
	select {
	case ev := <-other.C:
		t.Fatalf("other owner received %+v", ev)
	default:
	}
}

func TestSlowSubscriptionIsDroppedWithoutBlockingPublish(t *testing.T) {
	h := NewHub()
	slow := h.Subscribe("owner-a")

	for i := 0; i <= subscriptionBuffer; i++ {
		h.Publish(Event{Type: TypeRunStep, OwnerID: "owner-a"})
	}

	received := 0
	for range slow.C {
		received++
	}
	if received != subscriptionBuffer {
		t.Fatalf("received %d events before the drop, want %d", received, subscriptionBuffer)
	}
}

func TestCloseIsIdempotentAndSafeAfterDrop(t *testing.T) {
	h := NewHub()
	sub := h.Subscribe("owner-a")
	for i := 0; i <= subscriptionBuffer; i++ {
		h.Publish(Event{Type: TypeRunStep, OwnerID: "owner-a"})
	}
	sub.Close()
	sub.Close()
	h.Publish(Event{Type: TypeRunFinished, OwnerID: "owner-a"})
}
