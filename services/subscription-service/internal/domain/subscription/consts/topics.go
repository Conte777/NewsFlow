package consts

const (
	// Legacy topics (for backward compatibility during migration)
	TopicSubscriptionCreated   = "subscription.created"
	TopicSubscriptionCancelled = "subscription.cancelled"
	TopicSubscriptionUpdated   = "subscription.updated"

	// Saga: Subscription flow
	TopicSubscriptionRequested = "subscription.requested" // bot -> subscription-service
	TopicSubscriptionPending   = "subscription.pending"   // subscription-service -> account-service
	TopicSubscriptionActivated = "subscription.activated" // account-service -> subscription-service
	TopicSubscriptionFailed    = "subscription.failed"    // account-service -> subscription-service
	TopicSubscriptionRejected  = "subscription.rejected"  // subscription-service -> bot-service

	// Saga: Unsubscription flow
	TopicUnsubscriptionRequested = "unsubscription.requested" // bot -> subscription-service
	TopicUnsubscriptionPending   = "unsubscription.pending"   // subscription-service -> account-service
	TopicUnsubscriptionCompleted = "unsubscription.completed" // account-service -> subscription-service
	TopicUnsubscriptionFailed    = "unsubscription.failed"    // account-service -> subscription-service
	TopicUnsubscriptionRejected  = "unsubscription.rejected"  // subscription-service -> bot-service
)

// ConsumerTopics lists all topics subscription-service consumes
var ConsumerTopics = []string{
	// Legacy topics
	TopicSubscriptionCreated,
	TopicSubscriptionCancelled,
	TopicSubscriptionUpdated,
	// Saga: subscription flow
	TopicSubscriptionRequested,
	TopicSubscriptionActivated,
	TopicSubscriptionFailed,
	// Saga: unsubscription flow
	TopicUnsubscriptionRequested,
	TopicUnsubscriptionCompleted,
	TopicUnsubscriptionFailed,
}
