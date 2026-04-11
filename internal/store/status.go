package store

// EventStatus is the lifecycle state of an event.
type EventStatus string

const (
	EventStatusDraft  EventStatus = "draft"
	EventStatusActive EventStatus = "active"
	EventStatusClosed EventStatus = "closed"
)

// ChannelStatus is the lifecycle state of a channel.
type ChannelStatus string

const (
	ChannelStatusInactive ChannelStatus = "inactive"
	ChannelStatusActive   ChannelStatus = "active"
	ChannelStatusClosed   ChannelStatus = "closed"
)
