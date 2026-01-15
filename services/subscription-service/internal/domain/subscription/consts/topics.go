package consts

const (
	TopicSubscriptionCreated   = "subscription.created"
	TopicSubscriptionCancelled = "subscription.cancelled"
	TopicSubscriptionUpdated   = "subscription.updated"
)

var ConsumerTopics = []string{
	TopicSubscriptionCreated,
	TopicSubscriptionCancelled,
	TopicSubscriptionUpdated,
}
