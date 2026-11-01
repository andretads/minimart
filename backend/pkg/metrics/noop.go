package metrics

import "time"

// Noop is a Metrics that records nothing. Tests inject it so a use case can be
// exercised without a registry or a database pool; production never uses it.
type Noop struct{}

var _ Metrics = Noop{}

func (Noop) IncRequest(string, string, string)                        {}
func (Noop) ObserveLatency(string, string, time.Duration)             {}
func (Noop) IncInFlight()                                             {}
func (Noop) DecInFlight()                                             {}
func (Noop) IncOrderPlaced(string)                                    {}
func (Noop) ObserveOrderAmount(float64)                               {}
func (Noop) IncOrderCancelled(string)                                 {}
func (Noop) IncOrderPaymentApplied(string)                            {}
func (Noop) IncProductCreated(string)                                 {}
func (Noop) IncProductUpdated(string)                                 {}
func (Noop) IncStockReserved(string)                                  {}
func (Noop) IncStockReleased(string)                                  {}
func (Noop) IncNotificationSent(string)                               {}
func (Noop) IncNotificationPreferencesUpdated(string)                 {}
func (Noop) IncNotificationSuppressed(string)                         {}
func (Noop) IncKafkaPublished(string, string)                         {}
func (Noop) ObserveKafkaPublishLatency(string, time.Duration)         {}
func (Noop) IncKafkaConsumed(string, string, string)                  {}
func (Noop) ObserveKafkaConsumeLatency(string, string, time.Duration) {}
func (Noop) IncKafkaRetried(string, string)                           {}
func (Noop) IncKafkaDeadLettered(string, string, string)              {}
func (Noop) IncKafkaConsumerRestart(string, string)                   {}
func (Noop) SetKafkaConsumerLag(string, string, int64)                {}
func (Noop) IncKafkaBackpressureWait(string, string)                  {}
func (Noop) SetKafkaConsumerPaused(string, string, bool)              {}
func (Noop) IncKafkaDLQReplayParked(string)                           {}
func (Noop) IncRateLimited(string, string)                            {}
func (Noop) ObserveRateLimitRemaining(string, int)                    {}
func (Noop) IncRedisCommand(string, string)                           {}
func (Noop) ObserveRedisCommandLatency(string, time.Duration)         {}
func (Noop) IncIdempotency(string, string)                            {}
func (Noop) IncPayment(string)                                        {}
func (Noop) ObservePaymentAmount(float64)                             {}

func (Noop) IncCustomerEnsured(status string) {}

func (Noop) IncCustomerCreated(status string)     {}
func (Noop) IncCustomerUpdated(status string)     {}
func (Noop) IncCustomerDeactivated(status string) {}

func (Noop) IncCategoryCreated(status string) {}
func (Noop) IncCategoryUpdated(status string) {}
func (Noop) IncCategoryDeleted(status string) {}

func (Noop) IncReviewAdded(status string)       {}
func (Noop) IncReviewModerated(decision string) {}

func (Noop) IncCartCreated(status string)     {}
func (Noop) IncCartRetrieved(status string)   {}
func (Noop) IncCartItemAdded(status string)   {}
func (Noop) IncCartItemUpdated(status string) {}
func (Noop) IncCartItemRemoved(status string) {}
func (Noop) IncCartCleared(status string)     {}
func (Noop) IncCartCheckout(status string)    {}
func (Noop) IncCartConsumed(status string)    {}

func (Noop) IncWishlistCreated(status string)     {}
func (Noop) IncWishlistRetrieved(status string)   {}
func (Noop) IncWishlistItemAdded(status string)   {}
func (Noop) IncWishlistItemRemoved(status string) {}
func (Noop) IncWishlistCleared(status string)     {}

func (Noop) IncAddressCreated(status string)    {}
func (Noop) IncAddressFetched(status string)    {}
func (Noop) IncAddressListed(status string)     {}
func (Noop) IncAddressUpdated(status string)    {}
func (Noop) IncAddressDeleted(status string)    {}
func (Noop) IncAddressDefaultSet(status string) {}

func (Noop) IncAvatarUploaded(status string) {}
func (Noop) IncAvatarFetched(status string)  {}

func (Noop) IncShippingMethodsList(status string)  {}
func (Noop) IncShippingMethodGet(status string)    {}
func (Noop) IncShippingMethodCreate(status string) {}
func (Noop) IncShippingMethodUpdate(status string) {}

func (Noop) IncShippingZonesList(status string)  {}
func (Noop) IncShippingZoneGet(status string)    {}
func (Noop) IncShippingZoneCreate(status string) {}
func (Noop) IncShippingZoneUpdate(status string) {}

func (Noop) IncShippingRatesList(status string)  {}
func (Noop) IncShippingRateGet(status string)    {}
func (Noop) IncShippingRateCreate(status string) {}
func (Noop) IncShippingRateUpdate(status string) {}

func (Noop) IncShippingQuoteCreate(status string)   {}
func (Noop) IncShippingQuoteGet(status string)      {}
func (Noop) IncShippingQuoteValidate(status string) {}
func (Noop) IncShippingQuotePrice(status string)    {}
func (Noop) IncShippingQuotesExpire(status string)  {}

func (Noop) IncShipmentCreate(status string)              {}
func (Noop) IncShipmentCreateFromOrderPaid(status string) {}
func (Noop) IncShipmentGet(status string)                 {}
func (Noop) IncShipmentList(status string)                {}

func (Noop) IncShipmentProcess(status string)  {}
func (Noop) IncShipmentDispatch(status string) {}
func (Noop) IncShipmentDeliver(status string)  {}

func (Noop) IncTrackingEventCreate(status string) {}
func (Noop) IncTrackingEventList(status string)   {}

func (Noop) IncAccountCreated(status string)  {}
func (Noop) IncAuthenticated(status string)   {}
func (Noop) IncPasswordChanged(status string) {}
func (Noop) IncTokenRefreshed(status string)  {}

func (Noop) IncRoleAssigned(status string)      {}
func (Noop) IncPermissionsLoaded(status string) {}
func (Noop) IncAuthzDenied(status string)       {}
