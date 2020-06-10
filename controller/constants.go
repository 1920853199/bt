package controller

import (
	"time"
)

/* Specification constants */

const (
	RekeyAfterMessages     = (1 << 64) - (1 << 16) - 1
	RejectAfterMessages    = (1 << 64) - (1 << 4) - 1
	RekeyAfterTime         = time.Second * 120
	RekeyAfterTimeChange   = time.Second * 30
	RekeyAttemptTime       = time.Second * 90
	RekeyTimeout           = time.Second * 5
	RejectAfterTime        = time.Second * 180
	KeepaliveTimeout       = time.Second * 10
	CookieRefreshTime      = time.Second * 120
	HandshakeInitationRate = time.Second / 20
	PaddingMultiple        = 16
)

const (
	RekeyAfterTimeReceiving = RekeyAfterTime - KeepaliveTimeout - RekeyTimeout
	NewHandshakeTime        = KeepaliveTimeout + RekeyTimeout // upon failure to acknowledge transport message
)

/* Implementation specific constants */

const (
	QueueOutboundSize  = 20
	QueueInboundSize   = 20
	QueueHandshakeSize = 20
	MaxSegmentSize     = 1700                                  // largest possible UDP datagram
	MinMessageSize     = MessageKeepaliveSize                  // minimum size of transport message (keepalive)
	MaxMessageSize     = MaxSegmentSize                        // maximum size of transport message
	MaxContentSize     = MaxSegmentSize - MessageTransportSize // maximum size of transport message content
)

const (
	UnderLoadQueueSize = QueueHandshakeSize / 4
)
